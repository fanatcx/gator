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
	_, err := s.db.GetUser(context.Background(), cmd.arguments[0])
	if err != nil {
		return errors.New("user does not exist.")
	}

	if err = s.cfg.SetUser(cmd.arguments[0]); err != nil {
		return fmt.Errorf("Could not set user: %w", err)
	}

	fmt.Println("User is logged in sucessfully.")
	return nil
}

// needs to check for username (arg 2)
func handlerRegister(s *state, cmd command) error {
	if len(cmd.arguments) == 0 {
		return errors.New("username not supplied as an argument after command.")
	}

	// check if user exists ->
	_, err := s.db.GetUser(context.Background(), cmd.arguments[0])

	// not found, new user needed
	if err != nil {
		// create user
		u, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Name:      cmd.arguments[0],
		})
		if err != nil {
			return err
		}
		// set config locally
		if err = s.cfg.SetUser(u.Name); err != nil {
			return err
		}

		fmt.Printf("user: '%v' created successfully at %v and updated at %v. Current uuid: %v",
			u.Name,
			u.CreatedAt,
			u.UpdatedAt,
			u.ID)

		return nil
	}

	return errors.New("user already exists")
}

func handlerReset(s *state, cmd command) error {
	if err := s.db.DeleteUsers(context.Background()); err != nil {
		fmt.Print("failed to reset database: ")
		return err
	}
	fmt.Println("database has been successfully reset.")
	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		fmt.Print("unable to retrieve users: ")
		return err
	}
	for _, u := range users {
		if s.cfg.CurrentUserName == u {
			fmt.Printf("%s (current)\n", u)
			continue
		}
		fmt.Println(u)
	}
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
		return fmt.Errorf("error running command '%s': %w", cmd.name, err)
	}

	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.command[name] = f // assign to map
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error, not enough arguments. Please type a command.")
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
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmd := command{
		name:      os.Args[1],  // command name
		arguments: os.Args[2:], // rest of the arguments, if any
	}

	// self note: returns an error if failed to run a command through a handler
	if err := cmds.run(&stateST, cmd); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}
