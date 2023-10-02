package utils

import "strings"

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
