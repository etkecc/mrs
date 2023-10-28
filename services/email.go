package services

import (
	"net/http"

	"github.com/mattevans/postmark-go"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
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
	return postmark.NewClient(&http.Client{
		Transport: &postmark.AuthTransport{
			Token: e.cfg.Get().Email.Postmark.Token,
		},
	}).Email
}

// SendReport sends report email
func (e *Email) SendReport(room *model.MatrixRoom, server *model.MatrixServer, reason string, emails []string) error {
	if len(emails) == 0 {
		return nil
	}
	client := e.getClient()
	if client == nil {
		return nil
	}

	var aliasOrID string
	if room.Alias != "" {
		aliasOrID = room.Alias
	} else {
		aliasOrID = room.ID
	}

	vars := emailVars{Public: e.cfg.Get().Public, Room: room, Server: server, Reason: reason, RoomAliasOrID: aliasOrID}
	subject, err := utils.Template(e.cfg.Get().Email.Templates.Report.Subject, vars)
	if err != nil {
		return err
	}
	body, err := utils.Template(e.cfg.Get().Email.Templates.Report.Body, vars)
	if err != nil {
		return err
	}
	text, html := utils.MarkdownRender(body)
	reqs := e.buildPMReqs(subject, text, html, emails, &e.cfg.Get().Email.Postmark.Report)
	_, _, err = client.SendBatch(reqs)

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
