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

// Write config file
func Write(cfg *Config, configPath string) error {
	datab, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, datab, 0o666)
}
