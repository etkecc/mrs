package model

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/etkecc/go-kit"
	"github.com/pemistahl/lingua-go"

	"github.com/etkecc/mrs/internal/utils"
)

type BlocklistService interface {
	ByID(matrixID string) bool
	ByServer(server string) bool
}

// MatrixError model
type MatrixError struct {
	HTTP    string `json:"-"`       // HTTP Status e.g., 401 Unauthorized
	Code    string `json:"errcode"` // Matrix error code, e.g M_UNAUTHORIZED
	Message string `json:"error"`   // Matrix error message
}

// Error string
func (e MatrixError) Error() string {
	return fmt.Sprintf("%s (%s): %s", e.HTTP, e.Code, e.Message)
}

// MatrixServer info
type MatrixServer struct {
	Name      string               `json:"name"`
	URL       string               `json:"url"`
	Online    bool                 `json:"online"`
	Indexable bool                 `json:"indexable"`
	Contacts  MatrixServerContacts `json:"contacts"` // Contacts as per MSC1929
	OnlineAt  time.Time            `json:"online_at"`
	UpdatedAt time.Time            `json:"updated_at"` // Deprecated
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
	ID            string `json:"room_id"`
	Name          string `json:"name"`
	Topic         string `json:"topic"`
	Alias         string `json:"canonical_alias"`
	Avatar        string `json:"avatar_url"`
	Members       int    `json:"num_joined_members"`
	RoomType      string `json:"room_type"`
	JoinRule      string `json:"join_rule"`
	GuestJoinable bool   `json:"guest_can_join"`
	WorldReadable bool   `json:"world_readable"`

	// Parsed (custom) fields
	Server    string    `json:"server"`
	Email     string    `json:"email"`
	Language  string    `json:"language"`
	AvatarURL string    `json:"avatar_url_http"`
	ParsedAt  time.Time `json:"parsed_at"`
}

// Entry converts matrix room to search entry
func (r *MatrixRoom) Entry() *Entry {
	return &Entry{
		ID:            r.ID,
		Type:          "room",
		Alias:         r.Alias,
		Name:          r.Name,
		Topic:         r.Topic,
		Avatar:        r.Avatar,
		Server:        r.Server,
		Members:       r.Members,
		Language:      r.Language,
		AvatarURL:     r.AvatarURL,
		RoomType:      r.RoomType,
		JoinRule:      r.JoinRule,
		GuestJoinable: r.GuestJoinable,
		WorldReadable: r.WorldReadable,
	}
}

// DirectoryEntry converts matrix room into matrix room directory entry
func (r *MatrixRoom) DirectoryEntry() *RoomDirectoryRoom {
	return &RoomDirectoryRoom{
		ID:            r.ID,
		Name:          r.Name,
		Alias:         r.Alias,
		Topic:         r.Topic,
		Avatar:        r.Avatar,
		Members:       r.Members,
		RoomType:      r.RoomType,
		JoinRule:      r.JoinRule,
		GuestJoinable: r.GuestJoinable,
		WorldReadable: r.WorldReadable,
	}
}

// Parse matrix room info to prepare custom fields
func (r *MatrixRoom) Parse(detector lingua.LanguageDetector, media mediaURLService, mrsServerName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	r.ParsedAt = time.Now().UTC()
	if ctx.Err() != nil {
		return
	}

	r.Email = r.parseContact(mrsServerName, "email")
	if ctx.Err() != nil {
		return
	}

	r.parseLanguage(detector, mrsServerName)
	if ctx.Err() != nil {
		return
	}

	r.parseServer()
	if ctx.Err() != nil {
		return
	}

	r.parseAvatar(media)
}

// Servers returns all servers from the room object, except own server
func (r *MatrixRoom) Servers(ownServerName string) []string {
	servers := []string{}
	if server := utils.ServerFrom(r.ID); server != ownServerName {
		servers = append(servers, server)
	}
	if server := utils.ServerFrom(r.Alias); server != ownServerName {
		servers = append(servers, server)
	}
	if server := r.Server; server != ownServerName {
		servers = append(servers, server)
	}

	return kit.Uniq(servers)
}

// parseServer from room ID
func (r *MatrixRoom) parseServer() {
	parts := strings.SplitN(r.ID, ":", 2)
	if len(parts) > 1 {
		r.Server = parts[1]
	}
}

// parseContact tries to parse contact info from room topic
// the contact should be in the form of "<matrix.server_name from MRS config>:<field>:<value>" string, example:
// "example.com:email:admin@example.com"
func (r *MatrixRoom) parseContact(mrsServerName, field string) string {
	if r.Topic == "" {
		return ""
	}

	token := fmt.Sprintf("%s:%s:", mrsServerName, field)
	if !strings.Contains(r.Topic, token) {
		return ""
	}
	parts := strings.Split(r.Topic, token)
	if len(parts) < 2 || parts[1] == "" {
		return ""
	}
	parts = strings.Split(parts[1], " ")
	if len(parts) < 1 {
		return ""
	}
	parts = strings.Split(parts[0], "\n")
	if len(parts) < 1 {
		return ""
	}

	rawContact := parts[0]
	contact := strings.ToLower(strings.TrimSpace(parts[0]))

	// TODO: currently it works for email only, because MRS itself works with emails only for reports.
	_, err := mail.ParseAddress(contact)
	if err != nil {
		return ""
	}

	// cleanup the contact, as it is a purely technical workaround and not meant to be indexed and/or searched
	r.Topic = strings.ReplaceAll(r.Topic, token+rawContact, "")

	return contact
}

// parseLanguage tries to identify room language by room name and topic
func (r *MatrixRoom) parseLanguage(detector lingua.LanguageDetector, mrsServerName string) {
	r.Language = utils.UnknownLang
	if language := r.parseLanguageOption(mrsServerName); language != "" {
		r.Language = language
		return
	}

	r.Language, _ = utils.DetectLanguage(detector, r.Name+" "+r.Topic)
}

// parseLanguageOption tries to parse language option from room topic
func (r *MatrixRoom) parseLanguageOption(mrsServerName string) string {
	if r.Topic == "" {
		return ""
	}

	token := fmt.Sprintf("%s:%s:", mrsServerName, "language")
	if !strings.Contains(r.Topic, token) {
		return ""
	}
	parts := strings.Split(r.Topic, token)
	if len(parts) < 2 || parts[1] == "" {
		return ""
	}
	parts = strings.Split(parts[1], " ")
	if len(parts) < 1 {
		return ""
	}
	parts = strings.Split(parts[0], "\n")
	if len(parts) < 1 {
		return ""
	}

	// cleanup the language, as it is a purely technical workaround and not meant to be indexed and/or searched
	r.Topic = strings.ReplaceAll(r.Topic, token+parts[0], "")
	language := strings.ToUpper(strings.TrimSpace(parts[0]))

	// if the language is unknown/invalid, return empty string
	if code := lingua.GetIsoCode639_1FromValue(language); code == lingua.UnknownIsoCode639_1 {
		return ""
	}
	return language
}

type mediaURLService interface {
	GetURL(serverName, mediaID string) string
}

// parseAvatar builds HTTP URL to access room avatar
func (r *MatrixRoom) parseAvatar(media mediaURLService) {
	if r.Avatar == "" {
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.Avatar, "mxc://"), "/")
	if len(parts) != 2 {
		return
	}
	r.AvatarURL = media.GetURL(parts[0], parts[1])
}
