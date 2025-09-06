package main
import (
	"fmt"
	"os"
	"github.com/max-durnea/blog-aggregator/internal/config"
)
//maintain the state, here we have the Config struct which is built by reading the config file
type state struct{
	cfg *config.Config
}

type command struct{
	name string
	args []string
}
//maintain the commands in a map of name->function
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
	
	//Here we add new commands
	cmds.register("login",handlerLogin)

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
	//Write to the config file the new Username
	err:=s.cfg.SetUser(cmd.args[0])
	if err != nil {
		return err
	}
	fmt.Println("User has been set successfuly!")
	return nil
}