package utils

import (
	"net/url"

	"github.com/theovassiliou/base64url"
)

// URLSafeEncode encodes url-unsafe string into url-safe form
func URLSafeEncode(unsafeString string) string {
	return base64url.Encode([]byte(unsafeString))
}

// URLSafeDecode decodes url-safe string into the original form
func URLSafeDecode(safeString string) string {
	unsafeBytes, err := base64url.Decode(safeString)
	if err != nil {
		return ""
	}
	return string(unsafeBytes)
}

// ParseURL parses a URL and returns a URL structure
func ParseURL(uri string) *url.URL {
	if uri == "" {
		return nil
	}
	u, err := url.Parse(uri)
	if err != nil {
		Logger.Warn().Err(err).Str("url", uri).Msg("Could not parse URL")
		return nil
	}
	return u
}
