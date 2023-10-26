package config

// Config structure
type Config struct {
	Port       string     `yaml:"port"`
	Public     Public     `yaml:"public"`
	Matrix     Matrix     `yaml:"matrix"`
	Search     Search     `yaml:"search"`
	Path       Paths      `yaml:"path"`
	Batch      Batch      `yaml:"batch"`
	Auth       Auth       `yaml:"auth"`
	Cron       Cron       `yaml:"cron"`
	Cache      Cache      `yaml:"cache"`
	Workers    Workers    `yaml:"workers"`
	Moderation Moderation `yaml:"moderation"` // deprecated
	Webhooks   Webhooks   `yaml:"webhooks"`
	Email      Email      `yaml:"email"`
	Languages  []string   `yaml:"languages"`
	Servers    []string   `yaml:"servers"`
	Blocklist  Blocklist  `yaml:"blocklist"`
}

type Public struct {
	Name string `yaml:"name"`
	UI   string `yaml:"ui"`
	API  string `yaml:"api"`
}

// Search config
type Search struct {
	Defaults     SearchDefaults `yaml:"defaults"`
	EmptyResults []*SearchStub  `yaml:"empty_results"`
}

// SearchDefaults default params
type SearchDefaults struct {
	Limit  int    `yaml:"limit"`
	Offset int    `yaml:"offset"`
	SortBy string `yaml:"sort_by"`
}

// SearchStub is model.Entry, but for yaml
type SearchStub struct {
	ID        string `yaml:"id"`
	Alias     string `yaml:"alias"`
	Name      string `yaml:"name"`
	Topic     string `yaml:"topic"`
	Avatar    string `yaml:"avatar"`
	AvatarURL string `yaml:"avatar_url"`
	Server    string `yaml:"server"`
	Members   int    `yaml:"members"`
	Language  string `yaml:"language"`
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
	Admin      AuthItem `yaml:"admin"`
	Metrics    AuthItem `yaml:"metrics"`
	Discovery  AuthItem `yaml:"discovery"`
	Moderation AuthItem `yaml:"moderation"`
}

// AuthItem config (generic)
type AuthItem struct {
	Login    string   `yaml:"login"`
	Password string   `yaml:"password"`
	IPs      []string `yaml:"ips"`
}

// Webhooks config
type Webhooks struct {
	Moderation string `json:"moderation"`
	Stats      string `json:"stats"`
}

// Moderation config
// deprecated
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

// Blocklist config
type Blocklist struct {
	Servers []string `json:"servers"`
	Queries []string `json:"queries"`
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

// Matrix config
type Matrix struct {
	ServerName string   `yaml:"server_name"`
	Keys       []string `yaml:"keys"`
	OldKeys    []string `yaml:"old_keys"`
}
