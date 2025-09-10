package main

import (
	"fmt"
	"os"
	"time"
	"context"
	"net/http"
	"encoding/xml"
	"io"
	"html"
	"database/sql"
	"strings"

	"github.com/google/uuid"
	"github.com/max-durnea/blog-aggregator/internal/database"
)

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
	defer resp.Body.Close()
	
	//Read the bytes and unmarshal data into the RSSFeed struct
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR: Failed to read response body: %v\n", err)
		return nil, err
	}
	
	var rssFeed RSSFeed
	err = xml.Unmarshal(data,&rssFeed)
	if err != nil {
		fmt.Printf("ERROR: Failed to unmarshal XML from %s: %v\n", feedURL, err)
		fmt.Printf("Response content preview: %.200s...\n", string(data))
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


func parsePubDate(dateStr string) (time.Time, error) {
    layouts := []string{
        time.RFC1123,        
        time.RFC1123Z,       
        time.RFC3339,
    }

    for _, layout := range layouts {
        t, err := time.Parse(layout, dateStr)
        if err == nil {
            return t, nil
        }
    }

    return time.Time{}, fmt.Errorf("Unable to parse date: %s", dateStr)
}

func scrapeFeeds(s *state) error{
	nextFeed,err:= s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		fmt.Printf("ERROR: Failed to fetch next feed: %v\n",err)
		return nil
	}
	err=s.db.MarkFeedFetched(context.Background(), nextFeed.ID)
	if err != nil {
		fmt.Printf("ERROR: Failed to mark feed as fetched: %v\n",err)
		os.Exit(1)
	}
	feed,err := fetchFeed(context.Background(), nextFeed.Url)
	if err != nil {
		fmt.Printf("ERROR: Failed to fetch feed from web: %v\n",err)
		return nil
	}

	for _,item := range feed.Channel.Item{
		pubTime,err := parsePubDate(item.PubDate)
		if err != nil {
			fmt.Println("ERROR: %v\n",err)
		}
		params := database.CreatePostParams{
			ID : uuid.New(),
			CreatedAt : time.Now(),
			UpdatedAt : time.Now(),
			Title : sql.NullString{item.Title, item.Title != ""},
			Url : item.Link,
			Description : sql.NullString{item.Description, item.Description != ""},
			PublishedAt : sql.NullTime{pubTime, true},
			FeedID : nextFeed.ID,
		}
		_, err = s.db.CreatePost(context.Background(),params)
		if err != nil {
			if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
				//fmt.Println("Post URL already exists, ignoring...")
				continue
			}
			fmt.Println("ERROR: Could not insert new post: %v\n",err)
			continue
		}
	}
	
	return nil

}

func agg(s *state, cmd command) error{
	if len(cmd.args) != 1 {
		fmt.Println("ERROR: Please provide the time between requests like: 1s 1m 1h")
		os.Exit(1)
	}
	timeBetweenRequests, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		fmt.Println("ERROR: Error parsing duration: %v\n",err)
		os.Exit(1)
	}
	fmt.Printf("Scraping feeds every %v...\n",timeBetweenRequests)
	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
	return nil
}

//middleware for functions that have to ensure the user is logged in
func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error{
	//we return a new function where we simply fetch the current user before calling our handler
	return func(s *state,cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			fmt.Printf("ERROR: Could not fetch user: %v\n",err)
			os.Exit(1)
		}
		//the handlers need to accept the user struct
		return handler(s, cmd, user)
	}
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
func handlerReset(s *state, cmd command) error {
    // 1. Delete all posts first
    if err := s.db.ResetPosts(context.Background()); err != nil {
        fmt.Printf("ERROR: Failed to reset posts: %v\n", err)
        os.Exit(1)
    }

    // 2. Delete all feeds
    if err := s.db.ResetFeeds(context.Background()); err != nil {
        fmt.Printf("ERROR: Failed to reset feeds: %v\n", err)
        os.Exit(1)
    }

    // 3. Delete users
    if err := s.db.ResetUsers(context.Background()); err != nil {
        fmt.Printf("ERROR: Failed to reset users: %v\n", err)
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

func handlerUnfollow(s *state, cmd command, user database.User) error{
	if len(cmd.args) != 1 {
		fmt.Println("ERROR: Provide the URL\n")
		os.Exit(1)
	}
	params := database.DeleteFeedFollowParams{user.Name,cmd.args[0]}
	err:=s.db.DeleteFeedFollow(context.Background(),params)
	if err != nil {
		fmt.Printf("ERROR: Could not delete record: %v\n",err)
		os.Exit(1)
	}
	fmt.Printf("User unsubscribed from %v\n",cmd.args[0])
	return nil
}

