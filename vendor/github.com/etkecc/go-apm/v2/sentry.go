package apm

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/getsentry/sentry-go"
)

// applyEventFields maps log fields onto the sentry event as the zerolog integration did
func applyEventFields(event *sentry.Event, fields map[string]any) {
	event.Tags = make(map[string]string, len(fields))
	for key, value := range fields {
		switch key {
		case slog.MessageKey, slog.LevelKey, slog.TimeKey, slog.SourceKey:
			continue
		case "error", "err":
			if message := tagValue(value); message != "" {
				event.Exception = append(event.Exception, sentry.Exception{
					Value:      message,
					Stacktrace: sentry.NewStacktrace(),
				})
			}
		case "transaction":
			event.Transaction = tagValue(value)
		case "fingerprint":
			if fingerprint := stringSlice(value); fingerprint != nil {
				event.Fingerprint = fingerprint
			} else {
				addTags(event.Tags, key, value)
			}
		case "user":
			if user := sentryUser(value); user != nil {
				event.User = *user
			} else {
				addTags(event.Tags, key, value)
			}
		default:
			addTags(event.Tags, key, value)
		}
	}
}

// addTags writes value to tags, flattening nested objects into dotted keys
func addTags(tags map[string]string, key string, value any) {
	if nested, ok := value.(map[string]any); ok {
		for nestedKey, nestedValue := range nested {
			addTags(tags, key+"."+nestedKey, nestedValue)
		}
		return
	}
	tags[key] = tagValue(value)
}

// tagValue renders a log field value as a sentry tag value
func tagValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case nil:
		return ""
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(raw)
	}
}

// stringSlice converts a JSON array of strings into a string slice
func stringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil
		}
		result = append(result, text)
	}
	return result
}

// sentryUser decodes a JSON object value into a sentry user
func sentryUser(value any) *sentry.User {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	user := sentry.User{}
	if err := json.Unmarshal(raw, &user); err != nil {
		return nil
	}
	return &user
}

// sentryLevel maps a slog level to a sentry level
func sentryLevel(level slog.Level) sentry.Level {
	switch {
	case level >= slog.LevelError+4:
		return sentry.LevelFatal
	case level >= slog.LevelError:
		return sentry.LevelError
	case level >= slog.LevelWarn:
		return sentry.LevelWarning
	case level >= slog.LevelInfo:
		return sentry.LevelInfo
	default:
		return sentry.LevelDebug
	}
}
