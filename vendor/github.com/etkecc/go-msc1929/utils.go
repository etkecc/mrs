package msc1929

import (
	"net/mail"
	"net/url"
	"regexp"
)

// regexMXID is a regex to validate Matrix IDs, according to the https://spec.matrix.org/v1.18/appendices/#user-identifiers
var regexMXID = regexp.MustCompile(`^@[a-z0-9._=/+\-]+:[^:]+$`)

// uniq removes duplicates from slice
func uniq(slice []string) []string {
	uniq := map[string]struct{}{}
	result := []string{}
	for _, k := range slice {
		if _, ok := uniq[k]; !ok {
			uniq[k] = struct{}{}
			result = append(result, k)
		}
	}

	return result
}

// sanitizeContact sanitizes a single contact in-place
func sanitizeContact(c *Contact) {
	if c.Email != "" {
		email, err := mail.ParseAddress(c.Email)
		if err == nil && email != nil {
			c.Email = email.Address
		} else {
			c.Email = ""
		}
	}
	if c.MatrixID != "" {
		if !regexMXID.MatchString(c.MatrixID) {
			c.MatrixID = ""
		}
	}
	if c.PGPKey != "" {
		u, err := url.Parse(c.PGPKey)
		if err != nil || u.Scheme == "" {
			c.PGPKey = ""
		}
	}
}

// sanitizeContacts sanitizes a list of contacts by removing invalid email addresses and matrix IDs
func sanitizeContacts(rawContacts []*Contact) []*Contact {
	if len(rawContacts) == 0 {
		return nil
	}

	contacts := []*Contact{}
	for _, contact := range rawContacts {
		sanitizeContact(contact)
		if !contact.IsEmpty() {
			contacts = append(contacts, contact)
		}
	}
	return contacts
}
