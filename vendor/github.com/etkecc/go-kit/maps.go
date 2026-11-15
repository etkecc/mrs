package kit

import (
	"cmp"
	"slices"
	"sort"
)

// MapFromSlice turns slice into a set (element to true) for O(1) membership checks; nil slice gives an empty map.
func MapFromSlice[T cmp.Ordered](slice []T) map[T]bool {
	data := make(map[T]bool, len(slice))
	for _, k := range slice {
		data[k] = true
	}
	return data
}

// MapKeys returns a map's keys as an ascending-sorted slice; empty or nil map gives an empty slice.
func MapKeys[T cmp.Ordered, V any](data map[T]V) []T {
	keys := make([]T, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}

	slices.Sort(keys)
	return keys
}

// MergeMapKeys gathers the keys across m and every add, deduplicated and sorted ascending; always non-nil.
func MergeMapKeys[V any](m map[string]V, adds ...map[string]V) []string {
	uniq := map[string]bool{}
	for k := range m {
		uniq[k] = true
	}
	for _, add := range adds {
		for k := range add {
			uniq[k] = true
		}
	}

	keys := make([]string, 0, len(uniq))
	for k := range uniq {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	return keys
}
