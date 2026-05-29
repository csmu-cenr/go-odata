package dataModel

import "github.com/Uffe-Code/go-nullable/nullable"

func StructToMap(data any, tags []string) (map[string]any, error) {
	//nolint
	return nullable.StructToMap(data, tags)
}
