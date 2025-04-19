package utils

import (
	"net/url"
)

// ValuesOrDefault returns the encoded values or the default encoded values
func ValuesOrDefault(values url.Values, defaultEncoded string) string {
	if len(values) == 0 {
		return defaultEncoded
	}
	return values.Encode()
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
	unescapedValue, err := url.PathUnescape(value)
	if err == nil {
		return unescapedValue
	}
	return value
}
