package utils

import (
	"crypto/subtle"
	"strconv"
	"strings"
)

// MapKeys returns keys of the map
func MapKeys[K comparable, V any](datamap map[K]V) []K {
	keys := make([]K, 0, len(datamap))
	for key := range datamap {
		keys = append(keys, key)
	}

	return keys
}

// MergeSlices and remove duplicates
func MergeSlices[K comparable](slices ...[]K) []K {
	uniq := make(map[K]bool, 0)
	for _, slice := range slices {
		for _, item := range slice {
			uniq[item] = true
		}
	}

	result := make([]K, 0, len(uniq))
	for item := range uniq {
		result = append(result, item)
	}

	return result
}

// RemoveFromSlice removes elements of toRemove from base slice
func RemoveFromSlice[K comparable](base []K, toRemove []K) []K {
	include := map[K]bool{}
	for _, remove := range toRemove {
		include[remove] = false
	}
	for _, item := range base {
		if _, ok := include[item]; !ok {
			include[item] = true
		}
	}

	items := []K{}
	for item, ok := range include {
		if ok {
			items = append(items, item)
		}
	}

	return items
}

// Truncate string
func Truncate(s string, length int) string {
	if len(s) == 0 {
		return s
	}

	wb := strings.Split(s, "")
	if length > len(wb) {
		length = len(wb)
	}

	out := strings.Join(wb[:length], "")
	if s == out {
		return s
	}
	return out + "..."
}

// Uniq removes duplicates from slice
func Uniq[T comparable](slice []T) []T {
	uniq := map[T]struct{}{}
	uniqSlice := []T{}

	for _, item := range slice {
		if _, ok := uniq[item]; ok {
			continue
		}
		uniq[item] = struct{}{}
		uniqSlice = append(uniqSlice, item)
	}

	return uniqSlice
}

func StringToInt(value string, optionalDefaultValue ...int) int {
	defaultValue := 0
	if len(optionalDefaultValue) > 0 {
		defaultValue = optionalDefaultValue[0]
	}

	vInt, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return vInt
}

func StringToSlice(value string, defaultValue string) []string {
	value = strings.TrimSpace(value)
	if idx := strings.Index(value, ","); idx == -1 {
		value = defaultValue
	}
	return strings.Split(value, ",")
}

// SliceToString converts slice of strings into single string (using strings.Join) with optional hook
func SliceToString(slice []string, delimiter string, hook func(string) string) string {
	adjusted := make([]string, 0, len(slice))
	for _, item := range slice {
		if hook != nil {
			item = hook(item)
		}
		adjusted = append(adjusted, item)
	}

	return strings.Join(adjusted, delimiter)
}

// ConstantTimeEq checks if 2 strings are equal in constant time
func ConstantTimeEq(s1, s2 string) bool {
	b1 := []byte(s1)
	b2 := []byte(s2)
	return subtle.ConstantTimeEq(int32(len(b1)), int32(len(b2))) == 1 && subtle.ConstantTimeCompare(b1, b2) == 1
}
