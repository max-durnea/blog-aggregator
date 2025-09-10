package main

import (
	"fmt"
	"os"
	"database/sql"

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
	cmds.register("unfollow",middlewareLoggedIn(handlerUnfollow))
	cmds.register("browse",middlewareLoggedIn(handlerBrowse))
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