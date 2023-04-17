package model

import (
	"strings"

	"github.com/abadojack/whatlanggo"
)

// MatrixRoom from matrix client-server API
type MatrixRoom struct {
	ID      string `json:"room_id"`
	Alias   string `json:"canonical_alias"`
	Name    string `json:"name"`
	Topic   string `json:"topic"`
	Avatar  string `json:"avatar_url"`
	Members int    `json:"num_joined_members"`

	// Parsed (custom) fields
	Server    string `json:"server"`
	Language  string `json:"language"`
	AvatarURL string `json:"avatar_url_http"`
}

// Entry converts matrix room to search entry
func (r *MatrixRoom) Entry() *Entry {
	return &Entry{
		ID:        r.ID,
		Type:      "room",
		Alias:     r.Alias,
		Name:      r.Name,
		Topic:     r.Topic,
		Avatar:    r.Avatar,
		Server:    r.Server,
		Members:   r.Members,
		Language:  r.Language,
		AvatarURL: r.AvatarURL,
	}
}

// Parse matrix room info to prepare custom fields
func (r *MatrixRoom) Parse(serverURL string) {
	r.parseServer()
	r.parseLanguage()
	r.parseAvatar(serverURL)
}

// parseServer from room ID
func (r *MatrixRoom) parseServer() {
	parts := strings.SplitN(r.ID, ":", 2)
	if len(parts) > 1 {
		r.Server = parts[1]
	}
}

// parseLanguage tries to identify room language by room name and topic
func (r *MatrixRoom) parseLanguage() {
	name := whatlanggo.Detect(r.Name)
	topic := whatlanggo.Detect(r.Topic)
	if !name.IsReliable() {
		if !topic.IsReliable() {
			r.Language = "-"
			return
		}
		r.Language = topic.Lang.Iso6391()
		return
	}

	if name.Confidence > topic.Confidence {
		r.Language = name.Lang.Iso6391()
		return
	}
	r.Language = topic.Lang.Iso6391()
}

// parseAvatar builds HTTP URL to access room avatar
func (r *MatrixRoom) parseAvatar(serverURL string) {
	if r.Avatar == "" {
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.Avatar, "mxc://"), "/")
	if len(parts) != 2 {
		return
	}
	r.AvatarURL = serverURL + "/_matrix/media/v3/download/" + parts[0] + "/" + parts[1]
}
