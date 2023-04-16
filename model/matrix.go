package model

import (
	"strings"

	"github.com/abadojack/whatlanggo"
)

// MatrixRoom from matrix client-server API
type MatrixRoom struct {
	ID       string `json:"room_id"`
	Alias    string `json:"canonical_alias"`
	Name     string `json:"name"`
	Topic    string `json:"topic"`
	Avatar   string `json:"avatar_url"`
	Members  int    `json:"num_joined_members"`
	Server   string `json:"server"`   // custom
	Language string `json:"language"` // custom
}

// ParseServer from room ID
func (r *MatrixRoom) ParseServer() string {
	if r.Server != "" {
		return r.Server
	}

	parts := strings.SplitN(r.ID, ":", 2)
	if len(parts) > 1 {
		r.Server = parts[1]
		return parts[1]
	}
	return ""
}

// Entry converts matrix room to search entry
func (r *MatrixRoom) Entry() *Entry {
	return &Entry{
		ID:       r.ID,
		Type:     "room",
		Alias:    r.Alias,
		Name:     r.Name,
		Topic:    r.Topic,
		Avatar:   r.Avatar,
		Server:   r.ParseServer(),
		Members:  r.Members,
		Language: r.Language,
	}
}

// ParseLanguage tries to identify room language by room name and topic
func (r *MatrixRoom) ParseLanguage() string {
	if r.Language != "" {
		return r.Language
	}

	name := whatlanggo.Detect(r.Name)
	topic := whatlanggo.Detect(r.Topic)
	r.Language = chooseLang(name, topic)

	return r.Language
}

func chooseLang(name whatlanggo.Info, topic whatlanggo.Info) string {
	if !name.IsReliable() {
		if !topic.IsReliable() {
			return "-"
		}
		return topic.Lang.Iso6391()
	}

	if name.Confidence > topic.Confidence {
		return name.Lang.Iso6391()
	}
	return topic.Lang.Iso6391()
}
