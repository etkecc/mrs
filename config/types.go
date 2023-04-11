package config

import "github.com/labstack/echo/v4/middleware"

// Config structure
type Config struct {
	Port    string                `yaml:"port"`
	Path    Paths                 `yaml:"path"`
	Admin   Admin                 `yaml:"admin"`
	CORS    middleware.CORSConfig `yaml:"cors"`
	Workers Workers               `yaml:"workers"`
	Servers []string              `yaml:"servers"`
}

// Admin config
type Admin struct {
	Login    string   `yaml:"login"`
	Password string   `yaml:"password"`
	IPs      []string `yaml:"ips"`
}

// Paths config
type Paths struct {
	Index string `yaml:"index"`
	Data  string `yaml:"data"`
}

// Workers config
type Workers struct {
	Discovery int `yaml:"discovery"`
	Parsing   int `yaml:"parsing"`
}
