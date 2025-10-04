package dataModel

import (
	"fmt"
	"reflect"
)

// StructListToMapList converts a list of structs to a list of maps with selected fields.
func StructMultipleToMapMultiple(data interface{}, fields []string) ([]map[string]interface{}, error) {
	var result []map[string]interface{}

	// Get the type and value of the input data
	dataType := reflect.TypeOf(data)
	dataValue := reflect.ValueOf(data)

	// Ensure the input is a slice
	if dataType.Kind() != reflect.Slice {
		return nil, fmt.Errorf("input is not a slice")
	}

	// Iterate over the elements of the slice
	for i := 0; i < dataValue.Len(); i++ {
		element := dataValue.Index(i).Interface()

		// Convert the struct to a map with selected fields
		elementMap, err := StructToMap(element, fields)
		if err != nil {
			return nil, fmt.Errorf("error converting struct to map: %v", err)
		}

		// Append the map to the result slice
		result = append(result, elementMap)
	}

	return result, nil
}
