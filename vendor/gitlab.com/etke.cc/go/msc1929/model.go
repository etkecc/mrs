package msc1929

import (
	"net/mail"
	"net/url"
	"strings"
)

const (
	// RoleAdmin is catch-all user for any queries
	RoleAdmin = "m.role.admin"
	// RoleSecurity is intended for sensitive requests
	RoleSecurity = "m.role.security"
)

// Contact details
type Contact struct {
	Email    string `json:"email_address,omitempty" yaml:"email_address,omitempty"`
	MatrixID string `json:"matrix_id,omitempty" yaml:"matrix_id,omitempty"`
	Role     string `json:"role,omitempty" yaml:"role,omitempty"`
}

// IsEmpty checks if contact contains at least one contact (either email or mxid)
func (c *Contact) IsEmpty() bool {
	if c == nil {
		return true
	}
	return c.Email == "" && c.MatrixID == ""
}

// IsAdmin checks if contact has admin role
func (c *Contact) IsAdmin() bool {
	return c.Role == RoleAdmin
}

// IsSecurity checks if contact has security role
func (c *Contact) IsSecurity() bool {
	return c.Role == RoleSecurity
}

// Response of the MSC1929 support file
type Response struct {
	Contacts    []*Contact `json:"contacts,omitempty" yaml:"contacts,omitempty"`         // Contacts list
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

	contacts := []*Contact{}
	for _, contact := range r.Contacts {
		if contact.Email != "" {
			if email, _ := mail.ParseAddress(contact.Email); email != nil {
				contact.Email = email.Address
			} else {
				contact.Email = ""
			}
		}
		if contact.MatrixID != "" { // TODO: perform actual validation, use https://github.com/mautrix/go/blob/master/id/userid.go as reference
			if !strings.Contains(contact.MatrixID, "@") || !strings.Contains(contact.MatrixID, ":") {
				contact.MatrixID = ""
			}
		}
		if !contact.IsEmpty() {
			contacts = append(contacts, contact)
		}
	}
	r.Contacts = contacts
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
	return emails
}

// AdminMatrixIDs returns a list of admin matrix IDs
func (r *Response) AdminMatrixIDs() []string {
	var mxids []string
	for _, contact := range r.Contacts {
		if contact.IsAdmin() && contact.MatrixID != "" {
			mxids = append(mxids, contact.MatrixID)
		}
	}
	return mxids
}

// SecurityEmails returns a list of security emails
func (r *Response) SecurityEmails() []string {
	var emails []string
	for _, contact := range r.Contacts {
		if contact.IsSecurity() && contact.Email != "" {
			emails = append(emails, contact.Email)
		}
	}
	return emails
}

// SecurityMatrixIDs returns a list of security matrix IDs
func (r *Response) SecurityMatrixIDs() []string {
	var mxids []string
	for _, contact := range r.Contacts {
		if contact.IsSecurity() && contact.MatrixID != "" {
			mxids = append(mxids, contact.MatrixID)
		}
	}
	return mxids
}

// AllEmails returns a list of all emails
func (r *Response) AllEmails() []string {
	var emails []string
	for _, contact := range r.Contacts {
		if contact.Email != "" {
			emails = append(emails, contact.Email)
		}
	}
	return emails
}

// AllMatrixIDs returns a list of all matrix IDs
func (r *Response) AllMatrixIDs() []string {
	var mxids []string
	for _, contact := range r.Contacts {
		if contact.MatrixID != "" {
			mxids = append(mxids, contact.MatrixID)
		}
	}
	return mxids
}
