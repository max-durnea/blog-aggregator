package main

import (
	"fmt"
	"os"
	"database/sql"
	"time"
	"context"

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
		fmt.Println(err)
		os.Exit(1)
	}
	//Open Connection to the database
	db, err := sql.Open("postgres",st.cfg.DB_url)
	st.db = database.New(db)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	//Here we add new commands
	cmds.register("login",handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
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
		fmt.Println("ERROR: User not found!")
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
		fmt.Println("ERROR: User already exists!")
		os.Exit(1)
	}
	err = s.cfg.SetUser(cmd.args[0])
	if err != nil {
		fmt.Println("ERROR: User could not be changed!")
		os.Exit(1)
	}
	fmt.Println("User has been successfully created")
	fmt.Printf("%v\n",user)
	return nil
}

func handlerReset(s *state, cmd command) error{
	err := s.db.ResetUsers(context.Background())
	if err != nil {
		fmt.Println("ERROR: Failed to reset database")
		os.Exit(1)
	}
	fmt.Println("Database has been reset successfully.")
	return nil
}