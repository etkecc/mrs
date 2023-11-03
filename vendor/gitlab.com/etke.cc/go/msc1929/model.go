package msc1929

import "net/mail"

// Contact details
type Contact struct {
	Email    string `json:"email_address,omitempty"`
	MatrixID string `json:"matrix_id,omitempty"`
	Role     string `json:"role"`
}

// Response of the MSC1929 support file
type Response struct {
	Contacts    []Contact `json:"contacts,omitempty"`     // Contacts list
	Admins      []Contact `json:"admins,omitempty"`       // Admins list
	SupportPage string    `json:"support_page,omitempty"` // SupportPage URL

	hasContent bool     `json:"-"` // flag indicated that response has at least one email, mxid or support page set
	emails     []string `json:"-"` // full list of parsed emails
	mxids      []string `json:"-"` // full list of parsed mxids
}

// IsEmpty checks if response contains at least one contact (either email or mxid) or SupportPage
func (r *Response) IsEmpty() bool {
	if r == nil {
		return true
	}
	return !r.hasContent
}

// Emails returns all listed emails
func (r *Response) Emails() []string {
	return r.emails
}

// MatrixIDs returns all listed MXIDs
func (r *Response) MatrixIDs() []string {
	return r.mxids
}

func (r *Response) hydrate() {
	r.emails = append(r.parseEmails(r.Contacts), r.parseMxids(r.Admins)...)
	r.mxids = append(r.parseMxids(r.Contacts), r.parseMxids(r.Admins)...)
	r.hasContent = len(r.emails) > 0 || len(r.mxids) > 0 || r.SupportPage != ""
}

func (r *Response) parseEmails(contacts []Contact) []string {
	emails := []string{}
	for _, contact := range contacts {
		if email, _ := mail.ParseAddress(contact.Email); email != nil {
			emails = append(emails, email.Address)
		}
	}
	return emails
}

func (r *Response) parseMxids(contacts []Contact) []string {
	mxids := []string{}
	for _, contact := range contacts {
		if contact.MatrixID != "" {
			mxids = append(mxids, contact.MatrixID)
		}
	}
	return mxids
}
