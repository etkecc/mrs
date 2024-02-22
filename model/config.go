package model

import (
	echobasicauth "gitlab.com/etke.cc/go/echo-basic-auth"
	"gitlab.com/etke.cc/go/msc1929"
)

// Config is MRS configuration model
type Config struct {
	Port        string            `yaml:"port"`
	SentryDSN   string            `yaml:"sentry_dsn"`
	Public      *ConfigPublic     `yaml:"public"`
	Matrix      *ConfigMatrix     `yaml:"matrix"`
	Search      *ConfigSearch     `yaml:"search"`
	Path        *ConfigPaths      `yaml:"path"`
	Batch       *ConfigBatch      `yaml:"batch"`
	Auth        *ConfigAuth       `yaml:"auth"`
	Cron        *ConfigCron       `yaml:"cron"`
	Cache       *ConfigCache      `yaml:"cache"`
	Workers     *ConfigWorkers    `yaml:"workers"`
	Webhooks    *ConfigWebhooks   `yaml:"webhooks"`
	Email       *ConfigEmail      `yaml:"email"`
	Languages   []string          `yaml:"languages"`
	Servers     []string          `yaml:"servers"`
	Blocklist   *ConfigBlocklist  `yaml:"blocklist"`
	Experiments ConfigExperiments `yaml:"experiments"`
}

// ConfigPublic - instance public information
type ConfigPublic struct {
	Name string `yaml:"name"`
	UI   string `yaml:"ui"`
	API  string `yaml:"api"`
}

// ConfigSearch - search-related configuration
type ConfigSearch struct {
	Defaults ConfigSearchDefaults `yaml:"defaults"`
}

// ConfigSearchDefaults default params
type ConfigSearchDefaults struct {
	Limit  int    `yaml:"limit"`
	Offset int    `yaml:"offset"`
	SortBy string `yaml:"sort_by"`
}

// ConfigCache - cache-related configuration
type ConfigCache struct {
	MaxAge       int              `yaml:"max_age"`
	MaxAgeSearch int              `yaml:"max_age_search"`
	Bunny        ConfigCacheBunny `yaml:"bunny"`
}

// ConfigCacheBunny BunnyCDN cache purging config
type ConfigCacheBunny struct {
	URL string `yaml:"url"`
	Key string `yaml:"key"`
}

// ConfigAuth - auth-related configuration
type ConfigAuth struct {
	Admin      echobasicauth.Auth `yaml:"admin"`
	Metrics    echobasicauth.Auth `yaml:"metrics"`
	Discovery  echobasicauth.Auth `yaml:"discovery"`
	Moderation echobasicauth.Auth `yaml:"moderation"`
}

// ConfigWebhooks - webhooks related config
type ConfigWebhooks struct {
	Moderation string `json:"moderation"`
	Stats      string `json:"stats"`
}

// ConfigCron - cronjobs config
type ConfigCron struct {
	Discovery string `yaml:"discovery"`
	Parsing   string `yaml:"parsing"`
	Indexing  string `yaml:"indexing"`
	Full      string `yaml:"full"`
}

// ConfigPaths - paths configuration
type ConfigPaths struct {
	Index string `yaml:"index"`
	Data  string `yaml:"data"`
}

// ConfigBatch - batches related configuration
type ConfigBatch struct {
	Rooms int `yaml:"rooms"`
}

// ConfigWorkers - workers related configuration
type ConfigWorkers struct {
	Discovery int `yaml:"discovery"`
	Parsing   int `yaml:"parsing"`
}

// ConfigBlocklist - blocklist related configuration
type ConfigBlocklist struct {
	Servers []string `json:"servers"`
	Queries []string `json:"queries"`
}

// ConfigEmail - email related configuration
type ConfigEmail struct {
	Postmark   ConfigEmailPostmark  `yaml:"postmark"`
	Moderation string               `yaml:"moderation"`
	Templates  ConfigEmailTemplates `yaml:"templates"`
}

// ConfigEmailPostmark is Postmark config
type ConfigEmailPostmark struct {
	Token  string                  `yaml:"server_token"`
	Report ConfigEmailPostmarkType `yaml:"report"`
}

// ConfigEmailPostmarkType is postmark config for specific email type
type ConfigEmailPostmarkType struct {
	Stream string `yaml:"message_stream"`
	From   string `yaml:"from"`
}

// ConfigEmailTemplates - email temaplates config
type ConfigEmailTemplates struct {
	Report ConfigEmailTemplate `yaml:"report"`
}

// ConfigEmailTemplate - email template config
type ConfigEmailTemplate struct {
	Subject string `yaml:"subject"`
	Body    string `yaml:"body"`
}

// ConfigMatrix - matrix server config
type ConfigMatrix struct {
	ServerName string            `yaml:"server_name"`
	Support    *msc1929.Response `yaml:"support"`
	Keys       []string          `yaml:"keys"`
	OldKeys    []string          `yaml:"old_keys"`
}

// ConfigExperiments - experimental features
type ConfigExperiments struct {
	InMemoryIndex bool `yaml:"in_memory_index"`
	FakeAliases   bool `yaml:"fake_aliases"`
}
