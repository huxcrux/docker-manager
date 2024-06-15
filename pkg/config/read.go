package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Read config from file
func Read() (*Config, error) {

	ConfigFile := "config.yaml"

	// read ConfigFile from disk
	config, err := os.ReadFile(ConfigFile)
	if err != nil {
		return nil, err
	}

	// Marshal config into Config struct
	var cfg Config
	err = yaml.Unmarshal(config, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
