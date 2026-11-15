package kit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Truncate cuts s to length runes and appends "..." only when it cut something; length<=0 or empty s gives "".
func Truncate(s string, length int) string {
	if length <= 0 || s == "" {
		return ""
	}

	if length >= utf8.RuneCountInString(s) {
		return s
	}

	var buffer bytes.Buffer
	count := 0
	for i := 0; i < len(s); {
		if count >= length {
			break
		}
		r, width := utf8.DecodeRuneInString(s[i:])
		buffer.WriteRune(r)
		i += width
		count++
	}

	truncated := buffer.String()

	if truncated != s {
		return truncated + "..."
	}

	return truncated
}

// Unquote is strconv.Unquote with the error swallowed: returns the original string if it wasn't validly quoted.
func Unquote(s string) string {
	unquoted, err := strconv.Unquote(s)
	if err != nil {
		return s
	}
	return unquoted
}

// Hash returns the SHA-256 of str as lowercase hex, always 64 characters; a one-way digest.
func Hash(str string) string {
	h := sha256.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// StringToInt parses value as an int, trimmed first; empty or unparseable falls back to the optional default, else 0.
func StringToInt(value string, optionalDefaultValue ...int) int {
	defaultValue := 0
	if len(optionalDefaultValue) > 0 {
		defaultValue = optionalDefaultValue[0]
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue
	}

	vInt, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return vInt
}

// StringToSlice splits a comma-separated value into trimmed parts; empty input gives []string{default}.
func StringToSlice(value string, optionalDefaultValue ...string) []string {
	var defaultValue string
	if len(optionalDefaultValue) > 0 {
		defaultValue = optionalDefaultValue[0]
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return []string{defaultValue}
	}
	if !strings.Contains(value, ",") {
		return []string{value}
	}

	parts := strings.Split(value, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}

	return parts
}

// SliceToString joins slice with delimiter, running each element through hook first if non-nil; empty slice gives "".
func SliceToString(slice []string, delimiter string, hook func(string) string) string {
	if len(slice) == 0 {
		return ""
	}

	adjusted := make([]string, 0, len(slice))
	for _, item := range slice {
		if hook != nil {
			item = hook(item)
		}
		adjusted = append(adjusted, item)
	}

	return strings.Join(adjusted, delimiter)
}
