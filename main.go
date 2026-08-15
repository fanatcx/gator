package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/fanatcx/gator/internal/config"
	"github.com/fanatcx/gator/internal/database"
	"github.com/google/uuid"

	_ "github.com/lib/pq"
)

// We are importing for side effects, not really using it

type state struct {
	cfg *config.Config
	db  *database.Queries
}

type command struct {
	// example: login command.
	// Name would be login, argument should be the username (slice)
	name      string
	arguments []string
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.arguments) < 1 {
		return errors.New("the login handler expects a username argument.")
	}

	if err := s.cfg.SetUser(cmd.arguments[0]); err != nil {
		return fmt.Errorf("Could not set user: %w", err)
	}

	fmt.Println("User has been set sucessfully.")
	return nil
}

// needs to check for username (arg 2)
func handlerRegister(s *state, cmd command) error {
	if len(cmd.arguments) == 0 {
		return errors.New("username not supplied as an argument after command.")
	}
	// empty user slot
	u, err := s.db.CreateUser(context.Background(), database.CreateUserParams{})
	if err != nil {
		return err
	}
	// user already exists? is that what err is? confused. Also, context? Do I pass in background or point to a parent. Not explained in lesson
	_, err = s.db.GetUser(context.Background(), u.Name) 
	if err != nil {

		return errors.New("user already exists.")
	}
	// user is new
	u.ID = uuid.New()
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	u.Name = cmd.arguments[0]
	s.cfg.CurrentUserName = u.Name

	fmt.Printf("user: '%v' created successfully at %v and updated at %v. Current uuid: %v", 
	u.Name, 
	u.CreatedAt, 
	u.UpdatedAt, 
	u.ID)
	
	return nil
}

type commands struct {
	command map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	valFunc, exists := c.command[cmd.name]
	if !exists {
		return errors.New("failed to run command, command does not exist.")
	}

	// run command, return err if not successful
	if err := valFunc(s, cmd); err != nil {
		return fmt.Errorf("error running command %s: %w", cmd.name, err)
	}
	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.command[name] = f // assign to map
}


func main() {
	if len(os.Args) < 2 {
		fmt.Print("Error, not enough arguments. Please type a command.")
		os.Exit(1)
	}

	var stateST state

	// read from json on disk. returns config object or error
	cfg, err := config.Read()
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}
	stateST.cfg = &cfg
	db, err := sql.Open("postgres", stateST.cfg.DbURL)

	// A pointer to queries?
	dbQueries := database.New(db)
	stateST.db = dbQueries
	
	// create the commands
	cmds := commands{
		command: make(map[string]func(*state, command) error),
	}
	// handler
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmd := command{
		name:      os.Args[1],  // command name
		arguments: os.Args[2:], // rest of the arguments, if any
	}

	// self note: returns an error if failed to run a command through a handler
	if err := cmds.run(&stateST, cmd); err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}


	

}
