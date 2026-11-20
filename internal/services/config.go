package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/etkecc/go-apm/v2"
	echobasicauth "github.com/etkecc/go-echo-basic-auth/v2"
	"github.com/etkecc/go-fswatcher"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"

	"github.com/etkecc/mrs/internal/model"
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
	ctx := apm.NewContext()
	c := &Config{
		mu:   &sync.Mutex{},
		path: path,
	}
	c.Read(ctx)

	var err error
	c.fsw, err = fswatcher.New([]string{path}, 0)
	if err != nil {
		return nil, err
	}
	go c.fsw.Start(func(_ fsnotify.Event) { c.Read(ctx) })

	return c, nil
}

// Get config
func (c *Config) Get() *model.Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cfg
}

// Read config
func (c *Config) Read(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	log := apm.Log(ctx)

	log.Info("reading config")
	configb, err := os.ReadFile(c.path)
	if err != nil {
		log.Error("cannot read config", "error", err)
		return
	}
	var config *model.Config
	err = yaml.Unmarshal(configb, &config)
	if err != nil {
		log.Error("cannot unmarshal config", "error", err)
		return
	}

	c.cfg = config
}

// Write config
func (c *Config) Write(cfg *model.Config) error {
	datab, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, datab, 0o600)
}

// Validate reports auth allowlist entries that are neither an IP nor a CIDR, which would deny every peer
func (c *Config) Validate() error {
	cfg := c.Get()
	if cfg == nil || cfg.Auth == nil {
		return nil
	}
	named := []struct {
		name string
		auth *echobasicauth.Auth
	}{
		{"admin", &cfg.Auth.Admin},
		{"metrics", &cfg.Auth.Metrics},
		{"catalog", &cfg.Auth.Catalog},
		{"discovery", &cfg.Auth.Discovery},
		{"moderation", &cfg.Auth.Moderation},
	}

	errs := make([]error, 0, len(named))
	for _, entry := range named {
		if err := entry.auth.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("auth.%s: %w", entry.name, err))
		}
	}
	return errors.Join(errs...)
}
