package services

import (
	"context"
	"net/http"

	"github.com/etkecc/go-kit/template"
	"github.com/mattevans/postmark-go"
	"github.com/rs/zerolog"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

// Email service
type Email struct {
	cfg ConfigService
}

type emailVars struct {
	Public        *model.ConfigPublic
	Room          *model.MatrixRoom
	Server        *model.MatrixServer
	RoomAliasOrID string
	Reason        string
}

// NewEmail creates new email service
func NewEmail(cfg ConfigService) *Email {
	return &Email{cfg: cfg}
}

func (e *Email) getClient() *postmark.EmailService {
	if !e.validateConfig() {
		return nil
	}
	return postmark.NewClient(postmark.WithClient(&http.Client{
		Transport: &postmark.AuthTransport{
			Token: e.cfg.Get().Email.Postmark.Token,
		},
	})).Email
}

// SendReport sends report email
func (e *Email) SendReport(ctx context.Context, room *model.MatrixRoom, server *model.MatrixServer, reason string, emails []string) error {
	log := zerolog.Ctx(ctx)
	if len(emails) == 0 {
		log.Info().Str("reason", "no recipients").Msg("email sending canceled")
		return nil
	}
	client := e.getClient()
	if client == nil {
		log.Info().Str("reason", "no sender").Msg("email sending canceled")
		return nil
	}

	var aliasOrID string
	if room.Alias != "" {
		aliasOrID = room.Alias
	} else {
		aliasOrID = room.ID
	}

	vars := emailVars{Public: e.cfg.Get().Public, Room: room, Server: server, Reason: reason, RoomAliasOrID: aliasOrID}
	subject, err := template.Execute(e.cfg.Get().Email.Templates.Report.Subject, vars)
	if err != nil {
		return err
	}
	body, err := template.Execute(e.cfg.Get().Email.Templates.Report.Body, vars)
	if err != nil {
		return err
	}
	text, html := utils.MarkdownRender(body)
	for _, req := range e.buildPMReqs(subject, text, html, emails, &e.cfg.Get().Email.Postmark.Report) {
		req.Tag = "report-msc1929"
		log.Info().Str("to", req.To).Msg("sending email")
		if _, _, err = client.Send(req); err != nil {
			log.Warn().Err(err).Str("to", req.To).Msg("sending email failed")
		}
	}

	return err
}

// SendModReport sends report email to MRS instance's moderators
func (e *Email) SendModReport(message, email string) error {
	subject := "New report from MRS instance"
	text, html := utils.MarkdownRender(message)
	req := e.buildPMReqs(subject, text, html, []string{email}, &e.cfg.Get().Email.Postmark.Report)[0]
	req.Tag = "report-mod"
	_, _, err := e.getClient().Send(req)
	return err
}

// validateConfig checks if all config vars are set
func (e *Email) validateConfig() bool {
	cfg := e.cfg.Get().Email
	return !(cfg.Postmark.Token == "" ||
		cfg.Postmark.Report.From == "" ||
		cfg.Postmark.Report.Stream == "" ||
		cfg.Templates.Report.Body == "" ||
		cfg.Templates.Report.Subject == "")
}

// buildPMReqs builds postmark email for each email
func (e *Email) buildPMReqs(subject, text, html string, emails []string, cfg *model.ConfigEmailPostmarkType) []*postmark.Email {
	reqs := make([]*postmark.Email, 0, len(emails))
	for _, to := range emails {
		reqs = append(reqs, &postmark.Email{
			From:          cfg.From,
			MessageStream: cfg.Stream,
			To:            to,
			Subject:       subject,
			TextBody:      text,
			HTMLBody:      html,
		})
	}
	return reqs
}
