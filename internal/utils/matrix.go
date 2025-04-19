package utils

import (
	"regexp"
	"strings"

	"github.com/goccy/go-json"
	"github.com/matrix-org/gomatrixserverlib"
)

var aliasRegex = regexp.MustCompile(`^#[a-zA-Z0-9_.-]+:[a-zA-Z0-9_.-]+$`)

// IsValidRoomAlias checks if the given alias is a valid matrix room alias
func IsValidAlias(alias string) bool {
	if alias == "" {
		return false
	}

	return aliasRegex.MatchString(alias)
}

// Server returns server name from the matrix ID (room id/alias, user ID, etc)
func ServerFrom(matrixID string) string {
	idx := strings.LastIndex(matrixID, ":")
	if idx == -1 {
		return ""
	}
	if idx+2 == len(matrixID) { // "wrongid:"
		return ""
	}
	return matrixID[idx+1:]
}

// JSON marshals input into canonical json
func JSON(input any) ([]byte, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	return gomatrixserverlib.CanonicalJSON(data)
}
