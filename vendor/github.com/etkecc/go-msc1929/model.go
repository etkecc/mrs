package msc1929

import (
	"net/mail"
	"net/url"
	"strings"
)

const (
	// RoleAdmin is catch-all user for any queries
	RoleAdmin = "m.role.admin"
	// RoleModerator is intended for moderation requests
	// TODO: currently unused, as MSC4121 mandates the use of RoleModeratorUnstable until it is merged into the spec,
	// ref: https://github.com/FSG-Cat/matrix-spec-proposals/blob/FSG-Cat-Moderation-Role-well-known-support-record/proposals/4121-m.role.moderator.md#unstable-prefix
	RoleModerator = "m.role.moderator"
	// RoleModeratorUnstable is intended for moderation requests, used until MSC4121 is merged into the spec
	RoleModeratorUnstable = "support.feline.msc4121.role.moderator"
	// RoleSecurity is intended for sensitive requests
	RoleSecurity = "m.role.security"
	// RoleDPO is intended for data protection officer contacts
	// TODO: currently unused, as MSC4265 mandates the use of RoleDPOUnstable until it is merged into the spec,
	// ref: bitbucket.org/helloticketscode/curator/internal/services/emailnotifier
	RoleDPO = "m.role.dpo"
	// RoleDPOUnstable is intended for data protection officer contacts, used until MSC4265 is merged into the spec
	RoleDPOUnstable = "org.matrix.msc4265.role.dpo"
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

// IsModerator checks if contact has moderator role
func (c *Contact) IsModerator() bool {
	return c.Role == RoleModeratorUnstable || c.Role == RoleModerator
}

// IsDPO checks if contact has DPO role
func (c *Contact) IsDPO() bool {
	return c.Role == RoleDPOUnstable || c.Role == RoleDPO
}

// IsSecurity checks if contact has security role
func (c *Contact) IsSecurity() bool {
	return c.Role == RoleSecurity
}

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
	r.Contacts = r.sanitizeContacts(r.Contacts)
	r.Admins = r.sanitizeContacts(r.Admins)
	r.sanitized = true
}

func (r *Response) sanitizeContacts(rawContacts []*Contact) []*Contact {
	if len(rawContacts) == 0 {
		return nil
	}

	contacts := []*Contact{}
	for _, contact := range rawContacts {
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
	return contacts
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
	for _, contact := range r.Admins {
		if contact.IsAdmin() && contact.MatrixID != "" {
			mxids = append(mxids, contact.MatrixID)
		}
	}
	return mxids
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
	return emails
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
	return mxids
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
	return emails
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
	for _, contact := range r.Admins {
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
	for _, contact := range r.Admins {
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
	for _, contact := range r.Admins {
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
	for _, contact := range r.Admins {
		if contact.MatrixID != "" {
			mxids = append(mxids, contact.MatrixID)
		}
	}
	return mxids
}
