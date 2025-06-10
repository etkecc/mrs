package model

import (
	"net/mail"
	"strings"

	"github.com/pemistahl/lingua-go"
)

const (
	// RoomConfigTagStart is the start of the room config substring in the topic
	RoomConfigTagStart = "(MRS-"
	// RoomConfigDelimiter is the delimiter used in the room config substring
	RoomConfigDelimiter = "|"
	// RoomConfigTagEnd is the end of the room config substring in the topic
	RoomConfigTagEnd = "-MRS)"
)

// RoomConfig contains configuration for a Matrix room from the room topic
type RoomConfig struct {
	Language string // language of the room, e.g. "EN", "DE", etc.
	Email    string // email of the room, e.g. "yourname@example.com"
	Noindex  bool   // if true, the room should not be indexed by MRS
}

// IsEmpty checks if the RoomConfig is empty.
func (cfg *RoomConfig) IsEmpty() bool {
	return cfg == nil || (cfg.Language == "" && cfg.Email == "" && !cfg.Noindex)
}

// ParseRoomConfig parses the room topic to extract the room configuration,
// removing the room config string from the topic if it exists.
func ParseRoomConfig(topic string) (string, *RoomConfig) {
	cfg := &RoomConfig{}
	if topic == "" {
		return "", cfg
	}

	start := strings.Index(topic, RoomConfigTagStart)
	if start == -1 {
		// No config found, return as is
		return topic, cfg
	}
	end := strings.Index(topic[start+len(RoomConfigTagStart):], RoomConfigTagEnd)
	if end == -1 {
		// No end tag found, return as is
		return topic, cfg
	}
	end = start + len(RoomConfigTagStart) + end

	// Extract config string and rest topic
	configStr := topic[start+len(RoomConfigTagStart) : end]
	topic = strings.TrimSpace(topic[:start] + topic[end+len(RoomConfigTagEnd):])
	cfg = parseRoomConfig(configStr)

	return topic, cfg
}

// parseRoomConfig parses a room configuration string into a RoomConfig struct.
func parseRoomConfig(configStr string) *RoomConfig {
	rcfg := &RoomConfig{}
	for _, pair := range strings.Split(configStr, RoomConfigDelimiter) {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key, value := strings.TrimSpace(kv[0]), strings.ToLower(strings.TrimSpace(kv[1]))
		switch strings.ToLower(key) {
		case "language":
			value = strings.ToUpper(value)
			if code := lingua.GetIsoCode639_1FromValue(value); code != lingua.UnknownIsoCode639_1 {
				rcfg.Language = value
			}
		case "email":
			if _, err := mail.ParseAddress(value); err == nil {
				rcfg.Email = value
			}
		case "noindex":
			rcfg.Noindex = value == "yes" || value == "true" || value == "1"
		}
	}
	return rcfg
}
