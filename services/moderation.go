package services

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/goccy/go-json"
	"github.com/raja/argon2pw"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

type EmailService interface {
	SendReport(room *model.MatrixRoom, server *model.MatrixServer, reason string, emails []string) error
	SendModReport(text, email string) error
}

// Moderation service
type Moderation struct {
	cfg   ConfigService
	data  DataRepository
	mail  EmailService
	index IndexRepository
}

// webhookPayload for hookshot
// ref: https://matrix-org.github.io/matrix-hookshot/latest/setup/webhooks.html
type webhookPayload struct {
	Username string `json:"username"`
	Markdown string `json:"text"`
}

// NewModeration service
func NewModeration(cfg ConfigService, data DataRepository, index IndexRepository, mail EmailService) *Moderation {
	return &Moderation{
		cfg:   cfg,
		data:  data,
		mail:  mail,
		index: index,
	}
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
	hash, err := argon2pw.GenerateSaltedHash(m.cfg.Get().Auth.Moderation.Login + m.cfg.Get().Auth.Moderation.Password)
	if err != nil {
		utils.Logger.Error().Err(err).Msg("cannot generate auth hash")
	} else {
		queryParams = "?auth=" + utils.URLSafeEncode(hash)
	}

	apiURL, err := url.Parse(m.cfg.Get().Public.API)
	if err != nil {
		utils.Logger.Error().Err(err).Msg("cannot parse public api url")
		return text.String()
	}

	text.WriteString("[ban and erase](")
	text.WriteString(apiURL.JoinPath("/mod/ban", roomID).String() + queryParams)
	text.WriteString(") | [unban](")
	text.WriteString(apiURL.JoinPath("/mod/unban", roomID).String() + queryParams)
	text.WriteString(") | [list banned (all)](")
	text.WriteString(apiURL.JoinPath("/mod/list").String() + queryParams)
	text.WriteString(") | [list banned (" + room.Server + ")](")
	text.WriteString(apiURL.JoinPath("/mod/list/"+room.Server).String() + queryParams)
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

// sendWebhook sends a report to the configured webhook
func (m *Moderation) sendWebhook(room *model.MatrixRoom, server *model.MatrixServer, reason string) error {
	if m.cfg.Get().Webhooks.Moderation == "" {
		return nil
	}

	payload, err := json.Marshal(webhookPayload{
		Username: m.cfg.Get().Matrix.ServerName,
		Markdown: m.getReportText(room.ID, reason, room, server),
	})
	if err != nil {
		return err
	}

	if merr := m.mail.SendReport(room, server, reason, server.Contacts.Emails); merr != nil {
		utils.Logger.Warn().Err(merr).Msg("email sending failed")
	}

	req, err := http.NewRequest("POST", m.cfg.Get().Webhooks.Moderation, bytes.NewReader(payload))
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

// sendEmail sends a report to the configured moderators' email
func (m *Moderation) sendEmail(room *model.MatrixRoom, server *model.MatrixServer, reason string) error {
	if m.cfg.Get().Email.Moderation == "" {
		return nil
	}
	text := m.getReportText(room.ID, reason, room, server)
	return m.mail.SendModReport(text, m.cfg.Get().Email.Moderation)
}

// Report a room
func (m *Moderation) Report(roomID, reason string) error {
	if m.data.IsReported(roomID) {
		return nil
	}

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

	if err := m.sendWebhook(room, server, reason); err != nil {
		utils.Logger.Error().Err(err).Msg("cannot send moderation webhook")
	}

	if err := m.sendEmail(room, server, reason); err != nil {
		utils.Logger.Error().Err(err).Msg("cannot send moderation email")
	}

	return m.data.ReportRoom(roomID, reason)
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
