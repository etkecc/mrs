package services

import (
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"gitlab.com/etke.cc/go/fswatcher"
	"gopkg.in/yaml.v3"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

// Config service
type Config struct {
	mu   *sync.Mutex
	fsw  *fswatcher.Watcher
	path string
	cfg  *model.Config
}

type ConfigService interface {
	Get() *model.Config
}

// NewConfig creates new config service and loads the config
func NewConfig(path string) (*Config, error) {
	c := &Config{
		mu:   &sync.Mutex{},
		path: path,
	}
	if err := c.Read(); err != nil {
		return nil, err
	}

	var err error
	c.fsw, err = fswatcher.New([]string{path}, 0)
	if err != nil {
		return nil, err
	}
	go c.fsw.Start(func(_ fsnotify.Event) {
		if err := c.Read(); err != nil {
			utils.Logger.Error().Err(err).Msg("cannot reload config")
		}
	})

	return c, nil
}

// Get config
func (c *Config) Get() *model.Config {
	return c.cfg
}

// Read config
func (c *Config) Read() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	utils.Logger.Info().Msg("reading config")
	configb, err := os.ReadFile(c.path)
	if err != nil {
		return err
	}
	var config *model.Config
	err = yaml.Unmarshal(configb, &config)
	if err != nil {
		return err
	}

	c.cfg = config
	return nil
}

// Write config
func (c *Config) Write(cfg *model.Config) error {
	datab, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, datab, 0o666)
}
