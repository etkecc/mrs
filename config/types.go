package config

// Config structure
type Config struct {
	Port       string     `yaml:"port"`
	Public     Public     `yaml:"public"`
	Proxy      Proxy      `yaml:"proxy"`
	Path       Paths      `yaml:"path"`
	Batch      Batch      `yaml:"batch"`
	Auth       Auth       `yaml:"auth"`
	Cron       Cron       `yaml:"cron"`
	Cache      Cache      `yaml:"cache"`
	Workers    Workers    `yaml:"workers"`
	Moderation Moderation `yaml:"moderation"`
	Email      Email      `yaml:"email"`
	Languages  []string   `yaml:"languages"`
	Servers    []string   `yaml:"servers"`
	Blocklist  []string   `yaml:"blocklist"`
}

type Public struct {
	Name string `yaml:"name"`
	UI   string `yaml:"ui"`
	API  string `yaml:"api"`
}

type Proxy struct {
	Server string `yaml:"server"`
	Token  string `yaml:"token"`
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
	Admin      AuthAdmin         `yaml:"admin"`
	Discovery  AuthItem          `yaml:"discovery"`
	Moderation AuthItem          `yaml:"moderation"`
	Search     map[string]string `yaml:"search"`
}

// AuthItem config (generic)
type AuthItem struct {
	Login    string `yaml:"login"`
	Password string `yaml:"password"`
}

// AuthAdmin config
type AuthAdmin struct {
	Login    string   `yaml:"login"`
	Password string   `yaml:"password"`
	IPs      []string `yaml:"ips"`
}

// Moderation config
type Moderation struct {
	Webhook string `json:"webhook"`
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

// Email config
type Email struct {
	Postmark  EmailPostmark  `yaml:"postmark"`
	Templates EmailTemplates `yaml:"templates"`
}

// EmailPostmark is Postmark config
type EmailPostmark struct {
	Token  string            `yaml:"server_token"`
	Report EmailPostmarkType `yaml:"report"`
}

// EmailPostmarkType config
type EmailPostmarkType struct {
	Stream string `yaml:"message_stream"`
	From   string `yaml:"from"`
}

// EmailTemplates config
type EmailTemplates struct {
	Report EmailTemplate `yaml:"report"`
}

// EmailTemplate config
type EmailTemplate struct {
	Subject string `yaml:"subject"`
	Body    string `yaml:"body"`
}
