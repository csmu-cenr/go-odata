package dataModel

import "strings"

// RemoveQuotesFromString Slice removes quotes from fields used in StructToMap
// 2025-02-13 Created:	- Keith John Hutchison
func RemoveEnclosingQuotes(in []string) []string {

	out := []string{}
	for _, field := range in {
		field = strings.ReplaceAll(field, "\"", "")
		out = append(out, field)
	}
	return out

}
