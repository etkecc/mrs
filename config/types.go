package config

import "github.com/labstack/echo/v4/middleware"

// Config structure
type Config struct {
	Port      string                `yaml:"port"`
	Path      Paths                 `yaml:"path"`
	Batch     Batch                 `yaml:"batch"`
	Admin     Admin                 `yaml:"admin"`
	Cron      Cron                  `yaml:"cron"`
	CORS      middleware.CORSConfig `yaml:"cors"`
	Cache     Cache                 `yaml:"cache"`
	Workers   Workers               `yaml:"workers"`
	Languages []string              `yaml:"languages"`
	Servers   []string              `yaml:"servers"`
}

// Cache config
type Cache struct {
	MaxAge int        `yaml:"max_age"`
	Bunny  CacheBunny `yaml:"bunny"`
}

// CacheBunny BunnyCDN cache purging config
type CacheBunny struct {
	URL string `yaml:"url"`
	Key string `yaml:"key"`
}

// Admin config
type Admin struct {
	Login    string   `yaml:"login"`
	Password string   `yaml:"password"`
	IPs      []string `yaml:"ips"`
}

// Cron config
type Cron struct {
	Discovery string `yaml:"discovery"`
	Parsing   string `yaml:"parsing"`
	Indexing  string `yaml:"indexing"`
	Full      string `yaml:"full"`
}

// Paths config
type Paths struct {
	Index string `yaml:"index"`
	Data  string `yaml:"data"`
}

// Batch config
type Batch struct {
	Rooms int `yaml:"rooms"`
}

// Workers config
type Workers struct {
	Discovery int `yaml:"discovery"`
	Parsing   int `yaml:"parsing"`
}
