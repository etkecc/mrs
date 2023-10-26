package services

import (
	"net/http"

	"github.com/mattevans/postmark-go"

	"gitlab.com/etke.cc/mrs/api/config"
	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

// Email service
type Email struct {
	pm     *postmark.EmailService
	cfg    *config.Email
	public *config.Public
}

type emailVars struct {
	Public        *config.Public
	Room          *model.MatrixRoom
	Server        *model.MatrixServer
	RoomAliasOrID string
	Reason        string
}

// NewEmail creates new email service
func NewEmail(public *config.Public, email *config.Email) *Email {
	e := &Email{cfg: email, public: public}
	if e.validateConfig() {
		e.pm = postmark.NewClient(&http.Client{
			Transport: &postmark.AuthTransport{
				Token: e.cfg.Postmark.Token,
			},
		}).Email
	}

	return e
}

// SendReport sends report email
func (e *Email) SendReport(room *model.MatrixRoom, server *model.MatrixServer, reason string, emails []string) error {
	if len(emails) == 0 {
		return nil
	}
	if e.pm == nil {
		return nil
	}

	var aliasOrID string
	if room.Alias != "" {
		aliasOrID = room.Alias
	} else {
		aliasOrID = room.ID
	}

	vars := emailVars{Public: e.public, Room: room, Server: server, Reason: reason, RoomAliasOrID: aliasOrID}
	subject, err := utils.Template(e.cfg.Templates.Report.Subject, vars)
	if err != nil {
		return err
	}
	body, err := utils.Template(e.cfg.Templates.Report.Body, vars)
	if err != nil {
		return err
	}
	text, html := utils.MarkdownRender(body)
	reqs := e.buildPMReqs(subject, text, html, emails, &e.cfg.Postmark.Report)
	_, _, err = e.pm.SendBatch(reqs)

	return err
}

// validateConfig checks if all config vars are set
func (e *Email) validateConfig() bool {
	return !(e.cfg.Postmark.Token == "" ||
		e.cfg.Postmark.Report.From == "" ||
		e.cfg.Postmark.Report.Stream == "" ||
		e.cfg.Templates.Report.Body == "" ||
		e.cfg.Templates.Report.Subject == "")
}

// buildPMReqs builds postmark email for each email
func (e *Email) buildPMReqs(subject, text, html string, emails []string, cfg *config.EmailPostmarkType) []*postmark.Email {
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
