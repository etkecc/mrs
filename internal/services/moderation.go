package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/etkecc/go-apm"
	"github.com/etkecc/go-kit"
	"github.com/goccy/go-json"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

type EmailService interface {
	SendReport(ctx context.Context, room *model.MatrixRoom, server *model.MatrixServer, reason string, emails []string) error
	SendModReport(text, email string) error
}

// Moderation service
type Moderation struct {
	cfg   ConfigService
	data  DataRepository
	media MediaService
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
func NewModeration(cfg ConfigService, data DataRepository, media MediaService, index IndexRepository, mail EmailService) *Moderation {
	return &Moderation{
		cfg:   cfg,
		data:  data,
		mail:  mail,
		media: media,
		index: index,
	}
}

func (m *Moderation) getReportText(ctx context.Context, roomID, reason, fromIP string, room *model.MatrixRoom, server *model.MatrixServer) string {
	log := apm.Log(ctx)
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

	text.WriteString("* From IP: ")
	text.WriteString(fromIP)
	text.WriteString("\n")

	text.WriteString("* Reason: ")
	text.WriteString(reason)

	text.WriteString("\n\n---\n\n")

	text.WriteString("```json\n")
	text.WriteString(roomtxt)
	text.WriteString("\n```")

	text.WriteString("\n\n---\n\n")

	text.WriteString(m.getServerContactsText(server.Contacts))

	apiURL, err := url.Parse(m.cfg.Get().Public.API)
	if err != nil {
		log.Error().Err(err).Msg("cannot parse public api url")
		return text.String()
	}

	text.WriteString("[ban and erase](")
	text.WriteString(apiURL.JoinPath("/mod/ban", roomID).String())
	text.WriteString(") | [unban](")
	text.WriteString(apiURL.JoinPath("/mod/unban", roomID).String())
	text.WriteString(") | [list banned (all)](")
	text.WriteString(apiURL.JoinPath("/mod/list").String())
	text.WriteString(") | [list banned (" + room.Server + ")](")
	text.WriteString(apiURL.JoinPath("/mod/list/" + room.Server).String())
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
		text.WriteString("* Emails: " + kit.SliceToString(emails, ", ", utils.MarkdownEmail) + "\n")
	}
	if len(mxids) > 0 {
		text.WriteString("* MXIDs: " + kit.SliceToString(mxids, ", ", utils.MarkdownMXID) + "\n")
	}
	if page != "" {
		text.WriteString("* URL: " + utils.MarkdownLink(page) + "\n")
	}

	text.WriteString("\n---\n\n")

	return text.String()
}

// sendWebhook sends a report to the configured webhook
func (m *Moderation) sendWebhook(ctx context.Context, room *model.MatrixRoom, server *model.MatrixServer, fromIP, reason string) error {
	if m.cfg.Get().Webhooks.Moderation == "" {
		return nil
	}

	payload, err := json.Marshal(webhookPayload{
		Username: m.cfg.Get().Matrix.ServerName,
		Markdown: m.getReportText(ctx, room.ID, reason, fromIP, room, server),
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, m.cfg.Get().Webhooks.Moderation, bytes.NewReader(payload))
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
		return fmt.Errorf("backend returned HTTP %d: %s %w", resp.StatusCode, string(body), err)
	}
	return nil
}

// sendEmail sends a report to the configured moderators' email
func (m *Moderation) sendEmail(ctx context.Context, room *model.MatrixRoom, server *model.MatrixServer, reason, fromIP string) error {
	if m.cfg.Get().Email.Moderation == "" {
		return nil
	}
	text := m.getReportText(ctx, room.ID, reason, fromIP, room, server)
	return m.mail.SendModReport(text, m.cfg.Get().Email.Moderation)
}

// Report a room
func (m *Moderation) Report(ctx context.Context, fromIP, roomID, reason string, noMSC1929 bool) error {
	if m.data.IsReported(ctx, roomID) {
		return nil
	}

	log := apm.Log(ctx)
	room, err := m.data.GetRoom(ctx, roomID)
	if err != nil {
		return err
	}
	if room == nil {
		return fmt.Errorf("room not found")
	}
	server, err := m.data.GetServerInfo(ctx, room.Server)
	if err != nil {
		log.Error().Err(err).Msg("cannot get server info")
	}
	if server == nil {
		server = &model.MatrixServer{Name: room.Server}
	}

	if err := m.sendWebhook(ctx, room, server, reason, fromIP); err != nil {
		log.Error().Err(err).Msg("cannot send moderation webhook")
	}

	emails := server.Contacts.Emails
	if room.Email != "" {
		emails = append(emails, room.Email)
		emails = kit.Uniq(emails)
	}

	if !noMSC1929 {
		if merr := m.mail.SendReport(ctx, room, server, reason, emails); merr != nil {
			log.Warn().Err(merr).Msg("cannot send report to the server's owner")
		}
	}

	if err := m.sendEmail(ctx, room, server, reason, fromIP); err != nil {
		log.Error().Err(err).Msg("cannot send moderation email")
	}

	return m.data.ReportRoom(ctx, fromIP, roomID, reason)
}

// List returns full list of the banned rooms (optionally from specific server)
func (m *Moderation) List(ctx context.Context, serverName ...string) ([]string, error) {
	return m.data.GetBannedRooms(ctx, serverName...)
}

// Ban a room
func (m *Moderation) Ban(ctx context.Context, roomID string) error {
	room, err := m.data.GetRoom(ctx, roomID)
	if err != nil {
		return fmt.Errorf("cannot get room %s to ban: %w - room could be already banned", roomID, err)
	}
	if err := m.data.BanRoom(ctx, roomID); err != nil {
		return err
	}
	m.data.RemoveRoomMapping(ctx, room.ID, room.Alias)
	if err := m.index.Delete(roomID); err != nil {
		return err
	}

	if room.Avatar == "" {
		return nil
	}

	parts := strings.Split(strings.TrimPrefix(room.Avatar, "mxc://"), "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid mxc avatar: %s", room.Avatar)
	}
	m.media.Delete(ctx, parts[0], parts[1])
	return nil
}

// Unban a room
func (m *Moderation) Unban(ctx context.Context, roomID string) error {
	return m.data.UnbanRoom(ctx, roomID)
}
