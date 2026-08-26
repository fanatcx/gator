package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/fanatcx/gator/internal/config"
	"github.com/fanatcx/gator/internal/database"
	"github.com/google/uuid"

	_ "github.com/lib/pq"
)

// importing pq for side effects

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

// displays all feeds
func handlerFeeds(s *state, cmd command) error {
	if len(cmd.arguments) == 0 {
		f, err := s.db.DisplayFeeds(context.Background())
		if err != nil {
			return err
		}
		
		for _, feed := range f {
			u, err := s.db.GetUserById(context.Background(), feed.UserID)
			if err != nil {
				return err
			}
			fmt.Println(feed.Name)
			fmt.Println(feed.Url)
			fmt.Printf("created by user: %s\n", u.Name) // add feed user with sql 
			fmt.Println("------")
			
		}
		return nil
		
	}
	return errors.New("invalid arguments passed! 'feeds' takes no aruguments")
	
}

func addFeed(s *state, cmd command) error {
	// check number of arguments
	if len(cmd.arguments) == 2 {
		var feedInfo database.CreateFeedParams

		user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return err
		}

		feedName := cmd.arguments[0]
		url := cmd.arguments[1]
		//rss, err := fetchFeed(context.Background(), url)

		// NOTE: each "feed" has its own uuid auto generated
		feedInfo.CreatedAt = time.Now()
		feedInfo.Name = feedName
		feedInfo.Url = url
		feedInfo.UserID = user.ID
		feedInfo.UpdatedAt = time.Now()

		f, err := s.db.CreateFeed(context.Background(), feedInfo)
		if err != nil {
			return err
		}

		// print object to console
		fmt.Printf("'%s'\n", f.Name)
		fmt.Printf("'%s'\n", f.Url)
		return nil

	}
	return errors.New("invalid amount of arguments passed to addfeed. Provide a name and url for the feed")

}

// indicates that a user is now following a feed. To add a feed use addFeed
func handlerFollow(s *state, cmd command) error {
	if len(cmd.arguments) == 1 {
		feedURL := cmd.arguments[0]
		
		// current user
		u, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return err
		}
		
		// get feeds
		feed, err := s.db.GetFeedByUrl(context.Background(), feedURL)
		if err != nil {
			return errors.New("feed not found. you can add a new feed with 'addfeed'")
		}
		
		var feedFollowParams database.CreateFeedFollowParams
		
		feedFollowParams.CreatedAt = time.Now()
		feedFollowParams.UpdatedAt = time.Now()
		feedFollowParams.FeedID = feed.ID
		feedFollowParams.UserID = u.ID
		
		row, err := s.db.CreateFeedFollow(context.Background(), feedFollowParams)
		if err != nil {
			return err
		}
		
		fmt.Println(row.FeedName)
		fmt.Println(row.UserName)
		fmt.Println("-------------")

		return nil
	}
	
	return errors.New("invalid arguments passed as the follows handler expects a url argument")
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.arguments) < 1 {
		return errors.New("the login handler expects a username argument")
	}
	_, err := s.db.GetUser(context.Background(), cmd.arguments[0])
	if err != nil {
		return errors.New("user does not exist")
	}

	if err = s.cfg.SetUser(cmd.arguments[0]); err != nil {
		return fmt.Errorf("could not set user: %w", err)
	}

	fmt.Println("User is logged in successfully.")
	return nil
}

func handlerAggregator(s *state, cmd command) error {
	url := "https://www.wagslane.dev/index.xml"
	rss, err := fetchFeed(context.Background(), url)
	if err != nil {
		return err
	}

	fmt.Println(rss)

	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.arguments) == 0 {
		return errors.New("username not supplied as an argument after command")
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

		fmt.Printf("user: '%v' created successfully at %v and updated at %v. Current uuid: %v\n",
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
		return fmt.Errorf("resetting database: %w", err)
	}

	fmt.Println("database has been successfully reset")

	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())

	if err != nil {
		return fmt.Errorf("unable to retrieve users: %w", err)
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
		return errors.New("failed to run command, command does not exist")
	}

	if err := valFunc(s, cmd); err != nil {
		return fmt.Errorf("error running command '%s': %w", cmd.name, err)
	}

	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.command[name] = f // assign to map
}

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	var rss RSSFeed
	client := http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gator")
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client failed the request: %w", err)
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("fetching %s: %s", feedURL, res.Status)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %w", err)
	}

	// data->struct
	if err := xml.Unmarshal(data, &rss); err != nil {
		return nil, fmt.Errorf("could not unmarshal: %w", err)
	}

	// unescape
	rss.Channel.Title = html.UnescapeString(rss.Channel.Title)
	rss.Channel.Description = html.UnescapeString(rss.Channel.Description)
	for i, item := range rss.Channel.Item {
		rss.Channel.Item[i].Title = html.UnescapeString(item.Title)
		rss.Channel.Item[i].Description = html.UnescapeString(item.Description)
	}
	return &rss, nil
}



func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error, not enough arguments. Please type a command.")
		os.Exit(1)
	}

	var s state

	// read from json on disk. returns config object or error
	cfg, err := config.Read()
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}
	s.cfg = &cfg
	db, err := sql.Open("postgres", s.cfg.DbURL)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	if err := db.Ping(); err != nil {
		fmt.Printf("unable to ping the server: %v\n", err)
		os.Exit(1)
	}

	dbQueries := database.New(db)
	s.db = dbQueries

	// create the commands
	cmds := commands{
		command: make(map[string]func(*state, command) error),
	}
	// handlers
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAggregator)
	cmds.register("addfeed", addFeed)
	cmds.register("feeds", handlerFeeds)
	cmds.register("follow", handlerFollow)

	cmd := command{
		name:      os.Args[1],  // command name
		arguments: os.Args[2:], // rest of the arguments, if any
	}

	if err := cmds.run(&s, cmd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}
