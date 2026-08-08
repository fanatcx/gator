package config	

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	DbURL           string `json:"db_url"`
	CurrentUserName string `json:"currentUserName"`
}



func GetConfigPath() (string, error) {
	homeDirectory, err := os.UserHomeDir() 
	if err != nil {
		return "", err
	}
	return homeDirectory + "/gatorconfig.json", nil
}



// Returns a nil Config object if failure
func Read() (cfg Config, err error) {
	configPath, err := GetConfigPath()

	if err != nil {
		fmt.Print("Config path reading raised an error: ")
		return cfg, err
	}
	// read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Print("Error reading JSON in home directory: ")
		return cfg, err
	}
	// UNMARSHAL -> cfg
	if err = json.Unmarshal(data, &cfg); err != nil {
		fmt.Println("Error unmarshalling.")
		return cfg, err
	}
	//

	return cfg, nil
}

// Writes the username to the json on disk
func (c *Config) SetUser(username string) error {
	c.CurrentUserName = username
	encodedJson, err := json.Marshal(c)
	cfgPath, err := GetConfigPath()

	// failure
	if err != nil {
		return err
	}

	if err := os.WriteFile(cfgPath, encodedJson, 0644); err != nil {
		return err
	}
	
	return nil
}
