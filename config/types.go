package config

// Config structure
type Config struct {
	Port      string   `yaml:"port"`
	PublicURL string   `yaml:"public_url"`
	Path      Paths    `yaml:"path"`
	Batch     Batch    `yaml:"batch"`
	Auth      Auth     `yaml:"auth"`
	Cron      Cron     `yaml:"cron"`
	Cache     Cache    `yaml:"cache"`
	Workers   Workers  `yaml:"workers"`
	Languages []string `yaml:"languages"`
	Servers   []string `yaml:"servers"`
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

// Auth config
type Auth struct {
	Admin     AuthAdmin     `yaml:"admin"`
	Discovery AuthDiscovery `yaml:"discovery"`
}

// AuthAdmin config
type AuthAdmin struct {
	Login    string   `yaml:"login"`
	Password string   `yaml:"password"`
	IPs      []string `yaml:"ips"`
}

// AuthDiscovery config
type AuthDiscovery struct {
	Login    string `yaml:"login"`
	Password string `yaml:"password"`
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
