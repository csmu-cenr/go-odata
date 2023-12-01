package modelGenerator

import (
	"sort"
	"strings"
)

type ordered interface {
	int | int64 | string
}

func sortedKeys[V any, K ordered](m map[K]V) []K {

	keys := make([]K, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	return keys
}

func sortedCaseInsensitiveStringKeys[V any](m map[string]V) []string {

	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}

	// A case insensitive sort
	sort.Slice(keys, func(i, j int) bool {
		left := strings.ToLower(keys[i])
		right := strings.ToLower(keys[j])
		return left < right
	})

	return keys
}
