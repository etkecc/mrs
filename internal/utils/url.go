package utils

import (
	"encoding/hex"
	"hash/crc64"
	"net/url"
	"sort"
	"strings"
)

// ValuesOrDefault returns the values (only for keys present in the defaultValues) or the default encoded values
func ValuesOrDefault(values, defaultValues url.Values) url.Values {
	if len(values) == 0 {
		return defaultValues
	}

	for k := range defaultValues {
		if _, ok := values[k]; ok {
			defaultValues[k] = values[k]
		}
	}

	return defaultValues
}

// ParseURL parses a URL and returns a URL structure
func ParseURL(uri string) *url.URL {
	if uri == "" {
		return nil
	}
	u, err := url.Parse(uri)
	if err != nil {
		return nil
	}
	return u
}

// Unescape unescapes a URL-encoded string, e.g. a path param ("%23" -> "#")
// if unescaping fails, it returns the original value
func Unescape(value string) string {
	unescapedValue, err := url.QueryUnescape(value)
	if err == nil {
		return unescapedValue
	}
	return value
}

// HashURLValues returns the CRC64-ISO hash of url.Values as a hex string
// It uses a canonical ordering for the keys and values to ensure deterministic output
// It is intended to be fast and not cryptographically secure
func HashURLValues(values url.Values) string {
	table := crc64.MakeTable(crc64.ISO)
	h := crc64.New(table)

	// Canonical ordering for deterministic hash
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		vals := values[k]
		sort.Strings(vals)
		h.Write([]byte(k))
		h.Write([]byte("="))
		h.Write([]byte(strings.Join(vals, ",")))
		h.Write([]byte("&"))
	}
	hashBytes := make([]byte, 8)
	crc64Val := h.Sum64()
	// Encode uint64 CRC as big-endian bytes for hex encoding
	for i := uint(0); i < 8; i++ {
		hashBytes[7-i] = byte(crc64Val >> (i * 8))
	}
	return hex.EncodeToString(hashBytes)
}
