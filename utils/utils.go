package utils

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
