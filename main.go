package main

import (
	"errors"
	"fmt"
	"github.com/fanatcx/gator/internal/config"
	"os"
)
type state struct {
	cfg *config.Config
}

type command struct {
	// example: login command. 
	// Name would be login, argument should be the username (slice)
	name string
	arguments []string
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.arguments) < 1 {
		return errors.New("Error: The login handler expects a username argument.\n")
	}

	if err := s.cfg.SetUser(cmd.arguments[0]); err != nil {
		return errors.New("Error setting user.\n")
	}

	fmt.Println("User has been set sucessfully.")
	return nil
}

type commands struct {
	command map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	valFunc, exists := c.command[cmd.name]
	if !exists {
		return errors.New("Command does not exist.\n")
	}

	// run command, return err if not successful 
	if err := valFunc(s, cmd); err != nil {
		return err
	}
	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.command[name] = f // assign to map
	fmt.Printf("%s method added successfully.\n", name)
	
}

//
func main() {

	if len(os.Args) < 2 {
			fmt.Println("Error, not enough arguments. Please type a command.")
			os.Exit(-1)
		}

	var stateST state

	cfg, err := config.Read()
	if err != nil {
		fmt.Println("Failed to read config file on disk.")
		os.Exit(-1)
	}
	stateST.cfg = &cfg

	cmds := commands{
		// handler function signature
		command: make(map[string]func(*state, command) error),
	}
	cmds.register("login", handlerLogin)


	cmd := command{
		name: os.Args[1], // command name
		arguments: os.Args[2:], // rest of the arguments SHOULD be 
	}
	if err = cmds.run(&stateST, cmd); err != nil {
		fmt.Printf("%v", err)
		os.Exit(-1) 
	}


}