package utils

import "github.com/theovassiliou/base64url"

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
