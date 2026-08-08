package main

import (
	"errors"
	"fmt"
	"github.com/fanatcx/gator/internal/config"
)
type state struct {
	cfg *config.Config
}

type command struct {
	// example: login command. 
	// Name would be login, argument should be the username
	name string
	arguments []string
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.arguments) < 1 || len(cmd.arguments) > 1 {
		return errors.New("The login handler expects a single argument: 'username'.")
	}

	if err := s.cfg.SetUser(cmd.arguments[0]); err != nil {
		return errors.New("Error setting user.")
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
		return errors.New("Command does not exist.")
	}

	// run command, return err if not successful 
	if err := valFunc(s, cmd); err != nil {
		return err
	}
	return nil
}

func (c *commands) register(name string, f func(*state, command) error) error {
	c.command[name] = f // assign to map
	fmt.Println("Method added successfully.")
	return nil
}

func main() {
	var ST state
	cfg, err := config.Read()
	if err != nil {
		errors.New("Failure to read config file.")
	}
	ST.cfg = &cfg


	
}