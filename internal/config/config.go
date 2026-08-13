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
		return "", fmt.Errorf("error getting home directory: %w", err)
	}
	return homeDirectory + "/.gatorconfig.json", nil
}

// Read loads the gator config from the user's home directory.
func Read() (Config, error) {
	configPath, err := GetConfigPath()
	cfg := Config{}

	if err != nil {
		return Config{}, err
	}

	// read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("error reading file: %w", err)
	}
	// UNMARSHAL -> cfg
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal: %w", err)
	}
	return cfg, nil
}

// Writes the username to the json on disk
func (c *Config) SetUser(username string) error {
	cfgPath, err := GetConfigPath()
    if err != nil {
        return err
    }

    updated := *c
    updated.CurrentUserName = username

    encoded, err := json.Marshal(updated)
    if err != nil {
        return fmt.Errorf("encoding config: %w", err)
    }

    if err := os.WriteFile(cfgPath, encoded, 0600); err != nil {
        return fmt.Errorf("writing config file: %w", err)
    }

    *c = updated
    return nil
}
