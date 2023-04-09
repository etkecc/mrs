package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Read config from file system
func Read(configPath string) (*Config, error) {
	configb, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(configb, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
