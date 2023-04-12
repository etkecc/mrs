package model

import "strings"

// MatrixRoom from matrix client-server API
type MatrixRoom struct {
	ID      string `json:"room_id"`
	Alias   string `json:"canonical_alias"`
	Name    string `json:"name"`
	Topic   string `json:"topic"`
	Avatar  string `json:"avatar_url"`
	Server  string `json:"server"` // custom
	Members int    `json:"num_joined_members"`
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
		ID:      r.ID,
		Type:    "room",
		Alias:   r.Alias,
		Name:    r.Name,
		Topic:   r.Topic,
		Avatar:  r.Avatar,
		Server:  r.ParseServer(),
		Members: r.Members,
	}
}
