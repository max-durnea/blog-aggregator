package main

import (
	"fmt"
	"os"
	"database/sql"
	"time"
	"context"
	"net/http"
	"encoding/xml"
	"io"
	"html"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/max-durnea/blog-aggregator/internal/config"
	"github.com/max-durnea/blog-aggregator/internal/database"

)
//maintain the state, here we have the Config struct which is built by reading the config file
type state struct{
	db *database.Queries
	cfg *config.Config
}

type command struct{
	name string
	args []string
}
//store the commands in a map of name->function
type commands struct{
	handlers map[string]func(*state, command) error
}

type RSSFeed struct {
	Channel struct {
		Title string `xml:"title"`
		Link string `xml:"link"`
		Description string `xml:"description"`
		Item []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error){
	//Create the request with the provided URL and Context
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		fmt.Printf("ERROR: Failed to create request: %v\n", err)
		return nil, err
	}
	req.Header.Add("User-Agent","gator")
	//Create a client and do the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: Failed to make HTTP request: %v\n", err)
		return nil, err
	}
	//Read the bytes and unmarshal data into the RSSFeed struct
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR: Failed to read response body: %v\n", err)
		return nil, err
	}
	var rssFeed RSSFeed
	err = xml.Unmarshal(data,&rssFeed)
	if err != nil {
		fmt.Printf("ERROR: Failed to unmarshal XML: %v\n", err)
		return nil, err
	}
	//Unescape strings
	rssFeed.Channel.Title = html.UnescapeString(rssFeed.Channel.Title)
	rssFeed.Channel.Description = html.UnescapeString(rssFeed.Channel.Description)

	for i := range rssFeed.Channel.Item {
		rssFeed.Channel.Item[i].Title = html.UnescapeString(rssFeed.Channel.Item[i].Title)
		rssFeed.Channel.Item[i].Description = html.UnescapeString(rssFeed.Channel.Item[i].Description)
	}
	return &rssFeed,nil
	
}

//run a command
func (c *commands) run(s *state, cmd command){
	handler,ok := c.handlers[cmd.name]
	if !ok {
		fmt.Printf("ERROR: The provided command does not exist\n")
		return
	}
	err:=handler(s,cmd)
	if err != nil{
		fmt.Printf("%v\n",err)
		os.Exit(1)
	}
}
//register a new command
func (c *commands) register(name string, f func(*state, command) error){
	c.handlers[name]=f
}
func main(){
	//initialize config, state, commands structs to use them later
	var cfg config.Config
	st := state{}
	st.cfg = &cfg
	cmds := commands{
    	handlers: map[string]func(*state, command) error{},
	}
	//read the config file into the Config struct
	cfg,err:=config.Read()
	if err!=nil {
		fmt.Printf("ERROR: Failed to read config: %v\n", err)
		os.Exit(1)
	}
	//Open Connection to the database
	db, err := sql.Open("postgres",st.cfg.DB_url)
	st.db = database.New(db)
	if err != nil {
		fmt.Printf("ERROR: Failed to open database connection: %v\n", err)
		os.Exit(1)
	}

	//Here we add new commands
	cmds.register("login",handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", agg)
	cmds.register("addfeed",middlewareLoggedIn(handlerFeed))
	cmds.register("feeds",handlerAllFeeds)
	cmds.register("follow",middlewareLoggedIn(handlerFollow))
	cmds.register("following",middlewareLoggedIn(handlerFollows))
	//Get the command line arguments
	args:=os.Args
	if(len(args)<2){
		fmt.Println("ERROR: No arguments provided!")
		os.Exit(1)
	}
	//Execute the command
	cmd := command{name : args[1], args : args[2:]}
	cmds.run(&st,cmd)
	
}

func handlerLogin(s *state, cmd command) error{
	if len(cmd.args)==0 {
		return fmt.Errorf("ERROR: Username not provided")
	}
	user,err:=s.db.GetUser(context.Background(),cmd.args[0])
	if err != nil {
		fmt.Printf("ERROR: User not found: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(user);
	//Write to the config file the new Username
	err=s.cfg.SetUser(cmd.args[0])
	if err != nil {
		return err
	}
	fmt.Println("User has been set successfuly!")
	return nil
}

func handlerRegister(s *state, cmd command) error{
	if len(cmd.args)==0 {
		return fmt.Errorf("ERROR: Username not provided")
	}
	//build the param struct for a new user
	params := database.CreateUserParams{uuid.New(),time.Now(),time.Now(),cmd.args[0]}
	// use an empty context and create the user
	user,err:=s.db.CreateUser(context.Background(),params)
	if err != nil {
		fmt.Printf("ERROR: User already exists: %v\n", err)
		os.Exit(1)
	}
	// set the user session inside the config file
	err = s.cfg.SetUser(cmd.args[0])
	if err != nil {
		fmt.Printf("ERROR: User could not be changed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("User has been successfully created")
	fmt.Printf("%v\n",user)
	return nil
}
//function to reset database
func handlerReset(s *state, cmd command) error{
	err := s.db.ResetUsers(context.Background())
	if err != nil {
		fmt.Printf("ERROR: Failed to reset database: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Database has been reset successfully.")
	return nil
}
//list all registered users from the database
func handlerUsers(s *state, cmd command) error{
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		fmt.Printf("ERROR: Could not get users: %v\n", err)
		os.Exit(1)
	}
	for _,user := range users {
		fmt.Printf("* %v",user.Name)
		if s.cfg.CurrentUserName == user.Name{
			fmt.Println(" (current)")
		}else{
			fmt.Println()
		}
		
	}
	return nil
}

func agg(s *state, cmd command) error{
	fmt.Println(fetchFeed(context.Background(),"https://www.wagslane.dev/index.xml"))
	return nil
}

func handlerFeed(s *state, cmd command, user database.User) error{
	if len(cmd.args)!=2{
		fmt.Println("ERROR: Wrong arguments, provide name and url!")
		os.Exit(1)
	}
	/*user,err:=s.db.GetUser(context.Background(),s.cfg.CurrentUserName)
	if err != nil {
		fmt.Printf("ERROR: Could not get current user: %v\n", err)
		os.Exit(1)
	}*/
	name:=cmd.args[0]
	url:=cmd.args[1]
	
	params:=database.CreateFeedParams{uuid.New(),time.Now(),time.Now(),name,url,user.ID}
	res,err:=s.db.CreateFeed(context.Background(),params)
	if err != nil {
		fmt.Printf("ERROR: Could not add feed entry: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(res)
	handlerFollow(s,command{args: []string{url}},user)
	return nil

}

func handlerAllFeeds(s *state, cmd command) error{
	feeds,err:=s.db.GetFeeds(context.Background())
	if err != nil {
		fmt.Printf("ERROR: Could not fetch feeds: %v\n", err)
		os.Exit(1)
	}
	for _, feed := range feeds {
		user,err:=s.db.GetUserById(context.Background(),feed.UserID)
		if err != nil {
			fmt.Printf("ERROR: Could not fetch user by id: %v\n", err)
			continue
		}
		fmt.Printf(" * %v\n * %v\n * %v\n---\n",feed.Name, feed.Url,user.Name)
	}
	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error{
	if len(cmd.args) != 1 {
		fmt.Printf("ERROR: Wrong argument, provide the URL\n")
		os.Exit(1)
	}
	feed,err:=s.db.GetFeedByUrl(context.Background(),cmd.args[0])
	if err != nil {
		fmt.Printf("ERROR: Could not fetch feed: %v\n",err)
		os.Exit(1)
	}
	/*user,err:=s.db.GetUser(context.Background(),s.cfg.CurrentUserName)
	if err != nil {
		fmt.Printf("ERROR: Could not fetch current user: %v\n",err)
		os.Exit(1)
	}*/

	params := database.CreateFeedFollowParams{uuid.New(),time.Now(),time.Now(),user.ID,feed.ID}
	feed_follow,err:=s.db.CreateFeedFollow(context.Background(),params)
	if err != nil {
		fmt.Printf("ERROR: Could not create feed follow: %v\n",err)
		os.Exit(1)
	}
	fmt.Printf("Added feed %v for user %v\n",feed_follow.FeedName,feed_follow.UserName)
	return nil
}

func handlerFollows(s *state,cmd command, user database.User) error{
	/*user,err:= s.db.GetUser(context.Background(),s.cfg.CurrentUserName)
	if err != nil {
		fmt.Printf("ERROR: Could not fetch user: %v\n", err)
		os.Exit(1)
	}*/
	feeds, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		fmt.Printf("ERROR: Could not fetch feeds: %v\n",err)
		os.Exit(1)
	}
	fmt.Printf(" - %v\n",user.Name)
	for _,feed := range feeds {
		fmt.Printf(" * %v\n",feed.FeedName)
	}
	return nil

}
//middleware for functions that have to ensure the user is logged in
func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error{
	//we return a new function where we simply fetch the current user before calling our handler
	return func(s *state,cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			fmt.Println("ERROR: Could not fetch user: %v\n",err)
			os.Exit(1)
		}
		//the handlers need to accept the user struct
		return handler(s, cmd, user)
	}
}




