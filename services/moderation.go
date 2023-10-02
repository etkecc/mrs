package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/raja/argon2pw"

	"gitlab.com/etke.cc/mrs/api/config"
	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

// Moderation service
type Moderation struct {
	url     *url.URL
	auth    config.AuthItem
	data    DataRepository
	index   IndexRepository
	webhook string
}

// webhookPayload for hookshot
// ref: https://matrix-org.github.io/matrix-hookshot/latest/setup/webhooks.html
type webhookPayload struct {
	Username string `json:"username"`
	Markdown string `json:"text"`
}

// NewModeration service
func NewModeration(data DataRepository, index IndexRepository, auth config.AuthItem, publicURL, webhook string) (*Moderation, error) {
	parsedURL, err := url.Parse(publicURL)
	if err != nil {
		return nil, err
	}

	return &Moderation{
		data:    data,
		auth:    auth,
		index:   index,
		webhook: webhook,
		url:     parsedURL,
	}, nil
}

func (m *Moderation) getReportText(roomID, reason string, room *model.MatrixRoom) string {
	var roomtxt string
	roomb, err := json.MarshalIndent(room, "", "    ")
	if err == nil {
		roomtxt = string(roomb)
	} else {
		roomtxt = fmt.Sprintf("%+v", room)
	}

	var text strings.Builder
	text.WriteString("**New report**\n\n")

	text.WriteString("* ID: [")
	text.WriteString(roomID)
	text.WriteString("](https://matrix.to/#/")
	text.WriteString(roomID)
	text.WriteString(")\n")

	text.WriteString("* Reason: ")
	text.WriteString(reason)

	text.WriteString("\n\n---\n\n")

	text.WriteString("```json\n")
	text.WriteString(roomtxt)
	text.WriteString("\n```")

	text.WriteString("\n\n---\n\n")

	var queryParams string
	hash, err := argon2pw.GenerateSaltedHash(m.auth.Login + m.auth.Password)
	if err != nil {
		log.Println("cannot generate auth hash:", err)
	} else {
		queryParams = "?auth=" + utils.URLSafeEncode(hash)
	}

	text.WriteString("[ban and erase](")
	text.WriteString(m.url.JoinPath("/mod/ban", roomID).String() + queryParams)
	text.WriteString(") || [unban](")
	text.WriteString(m.url.JoinPath("/mod/unban", roomID).String() + queryParams)
	text.WriteString(") || [list banned](")
	text.WriteString(m.url.JoinPath("/mod/list").String() + queryParams)

	return text.String()
}

func (m *Moderation) Report(roomID, reason string) error {
	room, err := m.data.GetRoom(roomID)
	if err != nil {
		return err
	}
	if room == nil {
		return fmt.Errorf("room not found")
	}

	payload, err := json.Marshal(webhookPayload{
		Username: m.url.Host,
		Markdown: m.getReportText(roomID, reason, room),
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", m.webhook, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		return fmt.Errorf("backend returned HTTP %d: %s %v", resp.StatusCode, string(body), err)
	}

	return nil
}

// List returns full list of the banned rooms
func (m *Moderation) List() ([]string, error) {
	return m.data.GetBannedRooms()
}

// Ban a room
func (m *Moderation) Ban(roomID string) error {
	if err := m.data.BanRoom(roomID); err != nil {
		return err
	}
	return m.index.Delete(roomID)
}

// Unban a room
func (m *Moderation) Unban(roomID string) error {
	return m.data.UnbanRoom(roomID)
}
