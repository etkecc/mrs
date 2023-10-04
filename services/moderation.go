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

type EmailService interface {
	SendReport(room *model.MatrixRoom, server *model.MatrixServer, reason string, emails []string) error
}

// Moderation service
type Moderation struct {
	apiURL  *url.URL
	uiURL   *url.URL
	auth    config.AuthItem
	data    DataRepository
	mail    EmailService
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
func NewModeration(data DataRepository, index IndexRepository, mail EmailService, auth config.AuthItem, public config.Public, webhook string) (*Moderation, error) {
	parsedAPIURL, err := url.Parse(public.API)
	if err != nil {
		return nil, err
	}
	parsedUIURL, err := url.Parse(public.UI)
	if err != nil {
		return nil, err
	}

	return &Moderation{
		data:    data,
		auth:    auth,
		mail:    mail,
		index:   index,
		webhook: webhook,
		apiURL:  parsedAPIURL,
		uiURL:   parsedUIURL,
	}, nil
}

func (m *Moderation) getReportText(roomID, reason string, room *model.MatrixRoom, server *model.MatrixServer) string {
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

	text.WriteString(m.getServerContactsText(server.Contacts))

	var queryParams string
	hash, err := argon2pw.GenerateSaltedHash(m.auth.Login + m.auth.Password)
	if err != nil {
		log.Println("cannot generate auth hash:", err)
	} else {
		queryParams = "?auth=" + utils.URLSafeEncode(hash)
	}

	text.WriteString("[ban and erase](")
	text.WriteString(m.apiURL.JoinPath("/mod/ban", roomID).String() + queryParams)
	text.WriteString(") || [unban](")
	text.WriteString(m.apiURL.JoinPath("/mod/unban", roomID).String() + queryParams)
	text.WriteString(") || [list banned (all)](")
	text.WriteString(m.apiURL.JoinPath("/mod/list").String() + queryParams)
	text.WriteString(") || [list banned (" + room.Server + ")](")
	text.WriteString(m.apiURL.JoinPath("/mod/list/"+room.Server).String() + queryParams)
	text.WriteString(")")

	return text.String()
}

func (m *Moderation) getServerContactsText(contacts model.MatrixServerContacts) string {
	if contacts.IsEmpty() {
		return ""
	}
	var text strings.Builder
	emails := contacts.Emails
	mxids := contacts.MXIDs
	page := contacts.URL

	text.WriteString("**Server Contacts**\n\n")
	if len(emails) > 0 {
		text.WriteString("* Emails: " + utils.SliceToString(emails, ", ", utils.MarkdownEmail) + "\n")
	}
	if len(mxids) > 0 {
		text.WriteString("* MXIDs: " + utils.SliceToString(mxids, ", ", utils.MarkdownMXID) + "\n")
	}
	if page != "" {
		text.WriteString("* URL: " + utils.MarkdownLink(page) + "\n")
	}

	text.WriteString("\n---\n\n")

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
	server, err := m.data.GetServerInfo(room.Server)
	if err != nil {
		return err
	}
	if server == nil {
		return fmt.Errorf("server not found")
	}

	payload, err := json.Marshal(webhookPayload{
		Username: m.uiURL.Host,
		Markdown: m.getReportText(roomID, reason, room, server),
	})
	if err != nil {
		return err
	}

	if merr := m.mail.SendReport(room, server, reason, server.Contacts.Emails); merr != nil {
		log.Printf("email sending failed: %+v", merr)
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

// List returns full list of the banned rooms (optionally from specific server)
func (m *Moderation) List(serverName ...string) ([]string, error) {
	return m.data.GetBannedRooms(serverName...)
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
