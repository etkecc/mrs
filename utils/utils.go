package utils

import "strings"

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
