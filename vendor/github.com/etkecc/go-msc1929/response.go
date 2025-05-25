package msc1929

import (
	"net/url"
)

// Response of the MSC1929 support file
type Response struct {
	Contacts    []*Contact `json:"contacts,omitempty" yaml:"contacts,omitempty"`         // Contacts list
	Admins      []*Contact `json:"admins,omitempty" yaml:"admins,omitempty"`             // Admins list, deprecated since Nov 15, 2023, but still used by some servers
	SupportPage string     `json:"support_page,omitempty" yaml:"support_page,omitempty"` // SupportPage URL
	sanitized   bool       `json:"-"`                                                    // Flag to indicate if the response has been sanitized
}

// Sanitize ensures that all fields are valid, and removes those that are not
func (r *Response) Sanitize() {
	// ensure support page is a valid URL
	if r.SupportPage != "" {
		if _, err := url.Parse(r.SupportPage); err != nil {
			r.SupportPage = ""
		}
	}
	r.Contacts = sanitizeContacts(r.Contacts)
	r.Admins = sanitizeContacts(r.Admins)
	r.sanitized = true
}

// IsEmpty checks if response contains at least one contact (either email or mxid) or SupportPage
func (r *Response) IsEmpty() bool {
	if r == nil {
		return true
	}

	// to ensure that the response is sanitized before checking for emptiness,
	// we call Sanitize() if it hasn't been called before
	if !r.sanitized {
		r.Sanitize()
	}

	var hasContent bool
	for _, contact := range r.Contacts {
		if !contact.IsEmpty() {
			hasContent = true
			break
		}
	}

	if !hasContent {
		for _, contact := range r.Admins {
			if !contact.IsEmpty() {
				hasContent = true
				break
			}
		}
	}

	return !hasContent && r.SupportPage == ""
}

// Clone returns a deep copy of the response
func (r *Response) Clone() *Response {
	clone := &Response{}
	clone.Contacts = make([]*Contact, len(r.Contacts))
	for i, contact := range r.Contacts {
		clone.Contacts[i] = &Contact{
			Email:    contact.Email,
			MatrixID: contact.MatrixID,
			Role:     contact.Role,
		}
	}
	if r.Admins != nil {
		clone.Admins = make([]*Contact, len(r.Admins))
		for i, contact := range r.Admins {
			clone.Admins[i] = &Contact{
				Email:    contact.Email,
				MatrixID: contact.MatrixID,
				Role:     contact.Role,
			}
		}
	}
	clone.SupportPage = r.SupportPage
	return clone
}

// AdminEmails returns a list of admin emails
func (r *Response) AdminEmails() []string {
	var emails []string
	for _, contact := range r.Contacts {
		if contact.IsAdmin() && contact.Email != "" {
			emails = append(emails, contact.Email)
		}
	}
	for _, contact := range r.Admins {
		if contact.IsAdmin() && contact.Email != "" {
			emails = append(emails, contact.Email)
		}
	}
	return uniq(emails)
}

// AdminMatrixIDs returns a list of admin matrix IDs
func (r *Response) AdminMatrixIDs() []string {
	var mxids []string
	for _, contact := range r.Contacts {
		if contact.IsAdmin() && contact.MatrixID != "" {
			mxids = append(mxids, contact.MatrixID)
		}
	}
	for _, contact := range r.Admins {
		if contact.IsAdmin() && contact.MatrixID != "" {
			mxids = append(mxids, contact.MatrixID)
		}
	}
	return uniq(mxids)
}

// ModeratorEmails returns a list of moderator emails
func (r *Response) ModeratorEmails() []string {
	var emails []string
	for _, contact := range r.Contacts {
		if contact.IsModerator() && contact.Email != "" {
			emails = append(emails, contact.Email)
		}
	}
	for _, contact := range r.Admins {
		if contact.IsModerator() && contact.Email != "" {
			emails = append(emails, contact.Email)
		}
	}
	return uniq(emails)
}

// ModeratorMatrixIDs returns a list of moderator matrix IDs
func (r *Response) ModeratorMatrixIDs() []string {
	var mxids []string
	for _, contact := range r.Contacts {
		if contact.IsModerator() && contact.MatrixID != "" {
			mxids = append(mxids, contact.MatrixID)
		}
	}
	for _, contact := range r.Admins {
		if contact.IsModerator() && contact.MatrixID != "" {
			mxids = append(mxids, contact.MatrixID)
		}
	}
	return uniq(mxids)
}

// DPOEmails returns a list of DPO emails
func (r *Response) DPOEmails() []string {
	var emails []string
	for _, contact := range r.Contacts {
		if contact.IsDPO() && contact.Email != "" {
			emails = append(emails, contact.Email)
		}
	}
	for _, contact := range r.Admins {
		if contact.IsDPO() && contact.Email != "" {
			emails = append(emails, contact.Email)
		}
	}
	return uniq(emails)
}

// DPOMatrixIDs returns a list of DPO matrix IDs
func (r *Response) DPOMatrixIDs() []string {
	var mxids []string
	for _, contact := range r.Contacts {
		if contact.IsDPO() && contact.MatrixID != "" {
			mxids = append(mxids, contact.MatrixID)
		}
	}
	for _, contact := range r.Admins {
		if contact.IsDPO() && contact.MatrixID != "" {
			mxids = append(mxids, contact.MatrixID)
		}
	}
	return uniq(mxids)
}

// SecurityEmails returns a list of security emails
func (r *Response) SecurityEmails() []string {
	var emails []string
	for _, contact := range r.Contacts {
		if contact.IsSecurity() && contact.Email != "" {
			emails = append(emails, contact.Email)
		}
	}
	for _, contact := range r.Admins {
		if contact.IsSecurity() && contact.Email != "" {
			emails = append(emails, contact.Email)
		}
	}
	return uniq(emails)
}

// SecurityMatrixIDs returns a list of security matrix IDs
func (r *Response) SecurityMatrixIDs() []string {
	var mxids []string
	for _, contact := range r.Contacts {
		if contact.IsSecurity() && contact.MatrixID != "" {
			mxids = append(mxids, contact.MatrixID)
		}
	}
	for _, contact := range r.Admins {
		if contact.IsSecurity() && contact.MatrixID != "" {
			mxids = append(mxids, contact.MatrixID)
		}
	}
	return uniq(mxids)
}

// AllEmails returns a list of all emails
func (r *Response) AllEmails() []string {
	var emails []string
	for _, contact := range r.Contacts {
		if contact.Email != "" {
			emails = append(emails, contact.Email)
		}
	}
	for _, contact := range r.Admins {
		if contact.Email != "" {
			emails = append(emails, contact.Email)
		}
	}
	return uniq(emails)
}

// AllMatrixIDs returns a list of all matrix IDs
func (r *Response) AllMatrixIDs() []string {
	var mxids []string
	for _, contact := range r.Contacts {
		if contact.MatrixID != "" {
			mxids = append(mxids, contact.MatrixID)
		}
	}
	for _, contact := range r.Admins {
		if contact.MatrixID != "" {
			mxids = append(mxids, contact.MatrixID)
		}
	}
	return uniq(mxids)
}
