package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	RefreshInterval int      `json:"refresh_interval"`
	Modules         []string `json:"modules"`
	Colors          Colors   `json:"colors"`
}

type Colors struct {
	Primary string `json:"primary"`
	Surface string `json:"surface"`
	Text    string `json:"text"`
}

func loadConfig() (*Config, error) {
	configPath := filepath.Join(os.Getenv("HOME"), ".config", "tui-statusbar", "config.json")

	file, err := os.Open(configPath)
	if err != nil {
		return defaultConfig(), nil
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func defaultConfig() *Config {
	return &Config{
		RefreshInterval: 1,
		Modules:         []string{"workspaces", "clock", "cpu", "memory", "battery"},
		Colors: Colors{
			Primary: "#D7BAFF",
			Surface: "#16121B",
			Text:    "#E9DFEE",
		},
	}
}
