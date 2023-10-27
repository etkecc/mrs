package model

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/pemistahl/lingua-go"

	"gitlab.com/etke.cc/mrs/api/utils"
)

type BlocklistService interface {
	ByID(matrixID string) bool
	ByServer(server string) bool
}

// MatrixServer info
type MatrixServer struct {
	Name      string               `json:"name"`
	URL       string               `json:"url"`
	Online    bool                 `json:"online"`
	Indexable bool                 `json:"indexable"`
	Contacts  MatrixServerContacts `json:"contacts"` // Contacts as per MSC1929
	UpdatedAt time.Time            `json:"updated_at"`
}

// MatrixServerContacts - MSC1929
type MatrixServerContacts struct {
	Emails []string `json:"emails"`
	MXIDs  []string `json:"mxids"`
	URL    string   `json:"url"`
}

func (c MatrixServerContacts) IsEmpty() bool {
	return len(c.Emails) == 0 && len(c.MXIDs) == 0 && c.URL == ""
}

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
func (r *MatrixRoom) Parse(detector lingua.LanguageDetector, mrsPublicURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	r.Topic = utils.Truncate(r.Topic, 400)
	if ctx.Err() != nil {
		return
	}

	r.parseServer()
	if ctx.Err() != nil {
		return
	}

	r.parseAvatar(mrsPublicURL)
	if ctx.Err() != nil {
		return
	}

	r.parseLanguage(detector)
}

// parseServer from room ID
func (r *MatrixRoom) parseServer() {
	parts := strings.SplitN(r.ID, ":", 2)
	if len(parts) > 1 {
		r.Server = parts[1]
	}
}

// parseLanguage tries to identify room language by room name and topic
func (r *MatrixRoom) parseLanguage(detector lingua.LanguageDetector) {
	r.Language = utils.UnknownLang
	r.Language, _ = utils.DetectLanguage(detector, r.Name+" "+r.Topic)
}

// parseAvatar builds HTTP URL to access room avatar
func (r *MatrixRoom) parseAvatar(mrsPublicURL string) {
	if r.Avatar == "" {
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.Avatar, "mxc://"), "/")
	if len(parts) != 2 {
		return
	}
	base, err := url.Parse(mrsPublicURL)
	if err != nil {
		return
	}
	r.AvatarURL = base.JoinPath("/avatar", parts[0], parts[1]).String()
}
