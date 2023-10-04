package postmark

import (
	"errors"
	"time"
)

const (
	emailAPIPath             = "email"
	emailBatchAPIPath        = "email/batch"
	emailWithTemplateAPIPath = "email/withTemplate"
)

// EmailService handles communication with the email related methods of the
// Postmark API (http://developer.postmarkapp.com/developer-api-email.html)
type EmailService service

// Email is the set of parameters that can be used when sending an email.
type Email struct {
	From          string                 `json:",omitempty"`
	To            string                 `json:",omitempty"`
	Cc            string                 `json:",omitempty"`
	Bcc           string                 `json:",omitempty"`
	Subject       string                 `json:",omitempty"`
	Tag           string                 `json:",omitempty"`
	TemplateID    int                    `json:",omitempty"`
	TemplateModel map[string]interface{} `json:",omitempty"`
	HTMLBody      string                 `json:",omitempty"`
	TextBody      string                 `json:",omitempty"`
	ReplyTo       string                 `json:",omitempty"`
	Headers       []EmailHeader          `json:",omitempty"`
	Attachments   []EmailAttachment      `json:",omitempty"`
	TrackOpens    bool                   `json:",omitempty"`
	MessageStream string                 `json:",omitempty"`
	Metadata      map[string]string      `json:",omitempty"`
}

// EmailHeader represents the values for an email header.
type EmailHeader struct {
	Name  string
	Value string
}

// EmailAttachment represents the values for an email attachment.
type EmailAttachment struct {
	Name        string
	Content     []byte
	ContentType *string
}

// EmailResponse is the set of parameters that is used in response to a send
// request
type EmailResponse struct {
	To          string
	SubmittedAt time.Time
	MessageID   string
}

// Send will build and execute request to send an email via the API.
func (s *EmailService) Send(emailRequest *Email) (*EmailResponse, *Response, error) {
	if emailRequest == nil {
		return nil, nil, errors.New("The email request cannot be nil")
	}

	// If we have a template ID, use the Postmark template API endpoint.
	requestPath := emailAPIPath
	if emailRequest.TemplateID != 0 {
		requestPath = emailWithTemplateAPIPath
	}

	request, err := s.client.NewRequest("POST", requestPath, emailRequest)
	if err != nil {
		return nil, nil, err
	}

	email := &EmailResponse{}
	response, err := s.client.Do(request, email)
	if err != nil {
		return nil, response, err
	}

	return email, response, nil
}

// SendBatch will build and execute request to send batch emails via the API.
func (s *EmailService) SendBatch(emailRequests []*Email) ([]*EmailResponse, *Response, error) {
	if len(emailRequests) < 1 {
		return nil, nil, errors.New("You must pass a minimum of one email to SendBatch()")
	}

	request, err := s.client.NewRequest("POST", emailBatchAPIPath, emailRequests)
	if err != nil {
		return nil, nil, err
	}

	email := []*EmailResponse{}
	response, err := s.client.Do(request, email)
	if err != nil {
		return nil, response, err
	}

	return email, response, nil
}
