package model

import (
	echobasicauth "github.com/etkecc/go-echo-basic-auth"
	"github.com/etkecc/go-msc1929"
)

// Config is MRS configuration model
type Config struct {
	Address      string              `yaml:"address"`
	Port         string              `yaml:"port"`
	SentryDSN    string              `yaml:"sentry_dsn"`
	Healthchecks *ConfigHealthchecks `yaml:"healthchecks"`
	Public       *ConfigPublic       `yaml:"public"`
	Matrix       *ConfigMatrix       `yaml:"matrix"`
	Search       *ConfigSearch       `yaml:"search"`
	Path         *ConfigPaths        `yaml:"path"`
	Batch        *ConfigBatch        `yaml:"batch"`
	Auth         *ConfigAuth         `yaml:"auth"`
	Cron         *ConfigCron         `yaml:"cron"`
	Cache        *ConfigCache        `yaml:"cache"`
	Workers      *ConfigWorkers      `yaml:"workers"`
	Webhooks     *ConfigWebhooks     `yaml:"webhooks"`
	Email        *ConfigEmail        `yaml:"email"`
	Plausible    *ConfigPlausible    `yaml:"plausible"`
	Languages    []string            `yaml:"languages"`
	Servers      []string            `yaml:"servers"`
	Blocklist    *ConfigBlocklist    `yaml:"blocklist"`
}

// ConfigHealthchecks - healthchecks.io configuration
type ConfigHealthchecks struct {
	URL  string `yaml:"url"`
	UUID string `yaml:"uuid"`
}

// ConfigPublic - instance public information
type ConfigPublic struct {
	Name string `yaml:"name"`
	UI   string `yaml:"ui"`
	API  string `yaml:"api"`
}

// ConfigSearch - search-related configuration
type ConfigSearch struct {
	Defaults   ConfigSearchDefaults     `yaml:"defaults"`
	Highlights []*ConfigSearchHighlight `yaml:"highlights"`
}

// ConfigSearchDefaults default params
type ConfigSearchDefaults struct {
	Limit  int    `yaml:"limit"`
	Offset int    `yaml:"offset"`
	SortBy string `yaml:"sort_by"`
}

// ConfigSearchHighlight - search highlight configuration
type ConfigSearchHighlight struct {
	Position int      `yaml:"position"`
	Servers  []string `yaml:"servers"`

	ID            string `yaml:"id"`
	Alias         string `yaml:"alias"`
	Name          string `yaml:"name"`
	Topic         string `yaml:"topic"`
	Avatar        string `yaml:"avatar"`
	AvatarURL     string `yaml:"avatar_url"`
	Server        string `yaml:"server"`
	Members       int    `yaml:"members"`
	Language      string `yaml:"language"`
	RoomType      string `yaml:"room_type"`
	JoinRule      string `yaml:"join_rule"`
	GuestJoinable bool   `yaml:"guest_can_join"`
	WorldReadable bool   `yaml:"world_readable"`
}

// Entry converts ConfigSearchHighlight to Entry
func (c *ConfigSearchHighlight) Entry() *Entry {
	return &Entry{
		ID:            c.ID,
		Alias:         c.Alias,
		Name:          c.Name,
		Topic:         c.Topic,
		Avatar:        c.Avatar,
		AvatarURL:     c.AvatarURL,
		Server:        c.Server,
		Members:       c.Members,
		Language:      c.Language,
		RoomType:      c.RoomType,
		JoinRule:      c.JoinRule,
		GuestJoinable: c.GuestJoinable,
		WorldReadable: c.WorldReadable,
	}
}

// ConfigPlausible - plausible analytics configuration
type ConfigPlausible struct {
	Host   string `yaml:"host"`
	Domain string `yaml:"domain"`
}

// ConfigCache - cache-related configuration
type ConfigCache struct {
	MaxAge       int `yaml:"max_age"`
	MaxAgeSearch int `yaml:"max_age_search"`
}

// ConfigAuth - auth-related configuration
type ConfigAuth struct {
	Admin      echobasicauth.Auth `yaml:"admin"`
	Metrics    echobasicauth.Auth `yaml:"metrics"`
	Catalog    echobasicauth.Auth `yaml:"catalog"`
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
	Media string `yaml:"media"`
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
	IPs     []string `json:"ips"`
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

// ConfigEmailTemplates - email templates config
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
