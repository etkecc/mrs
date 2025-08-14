package utils

import (
	"regexp"
	"strings"

	"github.com/goccy/go-json"
	"github.com/matrix-org/gomatrixserverlib"
)

var aliasRegex = regexp.MustCompile(`^#[^:\x00]+:[a-zA-Z0-9.\-]+(:\d+)?$`)

// IsValidRoomAlias checks if the given alias is a valid matrix room alias
func IsValidAlias(alias string) bool {
	if alias == "" {
		return false
	}

	return aliasRegex.MatchString(alias)
}

// IsValidRoomID checks if the given ID is a valid matrix room ID
func IsValidID(id string) bool {
	if id == "" {
		return false
	}

	// v1-v11 roomID: !opaqueID:serverName
	// v12+ roomID: !31hneApxJ_1o-63DmFrpeqnkFfWppnzWso1JvH3ogLM
	// ref: https://github.com/matrix-org/matrix-spec-proposals/blob/matthew/msc4291/proposals/4291-room-ids-as-hashes.md
	return strings.HasPrefix(id, "!") && len(id) > 43 && len(id) < 256
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

// MustJSON marshals input into canonical json and panics on error
func MustJSON(input any) []byte {
	data, err := JSON(input)
	if err != nil {
		panic(err)
	}
	return data
}
