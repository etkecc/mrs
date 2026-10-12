package utils

import (
	"strings"

	"github.com/goccy/go-json"
	"github.com/matrix-org/gomatrixserverlib"
)

const maxIDLen = 255

// IsValidAlias checks if the given string is a valid matrix room alias (#localpart:domain).
func IsValidAlias(alias string) bool {
	return validNamespacedID(alias, '#', false)
}

// IsValidID checks if the given string is a valid matrix room ID (!localpart[:domain], domain-less since v12).
func IsValidID(id string) bool {
	return validNamespacedID(id, '!', true)
}

// ServerFrom returns the server name from a matrix ID (room ID, alias, user ID, etc).
func ServerFrom(matrixID string) string {
	_, domain, found := strings.Cut(matrixID, ":")
	if !found || domain == "" {
		return ""
	}
	return domain
}

// JSON marshals input into canonical json
func JSON(input any) ([]byte, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	return gomatrixserverlib.CanonicalJSON(data)
}

// MustJSON marshals input into canonical json and panics on error
func MustJSON(input any) []byte {
	data, err := JSON(input)
	if err != nil {
		panic(err)
	}
	return data
}

// validNamespacedID checks sigil, non-empty localpart without NUL, and a domain; aliases always have one.
func validNamespacedID(id string, sigil byte, domainless bool) bool {
	if id == "" || len(id) > maxIDLen {
		return false
	}
	if id[0] != sigil {
		return false
	}
	local, domain, hasDomain := strings.Cut(id[1:], ":")
	if local == "" || strings.ContainsRune(local, 0) {
		return false
	}
	if !hasDomain {
		return domainless
	}
	return validDomain(domain)
}

// validDomain checks the structural form of a server name: host, bracketed IPv6 host, or host:port.
func validDomain(domain string) bool {
	if domain == "" {
		return false
	}
	if strings.HasPrefix(domain, "[") {
		closer := strings.LastIndexByte(domain, ']')
		if closer <= 1 {
			return false
		}
		port := domain[closer+1:]
		return port == "" || (strings.HasPrefix(port, ":") && isDigits(port[1:]))
	}
	host, port, found := strings.Cut(domain, ":")
	if !found {
		return true
	}
	return host != "" && isDigits(port)
}

// isDigits reports whether s is a non-empty run of ASCII digits.
func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}
