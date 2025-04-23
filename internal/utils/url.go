package utils

import (
	"encoding/binary"
	"encoding/hex"
	"hash/crc64"
	"net/url"
	"sort"
)

var crc64Table = crc64.MakeTable(crc64.ISO)

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
// The function already highly optimized for performance (552.5 ns/op, 112 B/op, 3 allocs/op),
// do NOT change the implementation unless you have a very good reason
func HashURLValues(values url.Values) string {
	h := crc64.New(crc64Table)

	// Sort keys for deterministic order
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		vals := values[k]
		sort.Strings(vals)

		h.Write([]byte(k))
		h.Write([]byte("="))

		for j, v := range vals {
			h.Write([]byte(v))
			if j < len(vals)-1 {
				h.Write([]byte(","))
			}
		}

		if i < len(keys)-1 {
			h.Write([]byte("&"))
		}
	}

	var hashBytes [8]byte
	binary.BigEndian.PutUint64(hashBytes[:], h.Sum64())
	return hex.EncodeToString(hashBytes[:])
}
