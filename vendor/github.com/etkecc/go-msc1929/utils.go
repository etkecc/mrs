package msc1929

import (
	"net/mail"
	"regexp"
)

// regexMXID is a regex to validate Matrix IDs, according to the https://spec.matrix.org/v1.14/appendices/#user-identifiers
var regexMXID = regexp.MustCompile(`^@[a-z0-9._=\/+\-]+:[^:]+$`)

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

// sanitizeContacts sanitizes a list of contacts by removing invalid email addresses and matrix IDs
func sanitizeContacts(rawContacts []*Contact) []*Contact {
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
		if contact.MatrixID != "" {
			if !regexMXID.MatchString(contact.MatrixID) {
				contact.MatrixID = ""
			}
		}
		if !contact.IsEmpty() {
			contacts = append(contacts, contact)
		}
	}
	return contacts
}
