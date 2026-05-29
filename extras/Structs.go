package dataModel

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// findFieldByJSONTag finds a struct field by its JSON tag.
func findFieldByJSONTag(dataType reflect.Type, jsonTag string) (reflect.StructField, bool) {
	for i := 0; i < dataType.NumField(); i++ {
		field := dataType.Field(i)
		tag := strings.Split(field.Tag.Get("json"), ",")[0]
		if tag == jsonTag {
			return field, true
		}
	}
	return reflect.StructField{}, false
}

// TODO Swap for official struct field is public function
func isFirstLetterCapital(s string) bool {
	// Check if the string is not empty
	if s == "" {
		return false
	}

	// Get the first rune (Unicode character) in the string
	firstRune := []rune(s)[0]

	// Check if the first rune is uppercase
	return unicode.IsUpper(firstRune)
}

// StructListToInterface converts a list of structs to an interface
func StructListToInterface(data any, fields []string) (interface{}, error) {
	return StructListToMapList(data, fields)
}

func MapListToMapMapInterface(data interface{}, fields []string) (map[string][]map[string]interface{}, error) {
	result := map[string][]map[string]interface{}{}

	// Get the type and value of the input data
	dataType := reflect.TypeOf(data)
	dataValue := reflect.ValueOf(data)

	// Ensure the input is a slice
	if dataType.Kind() != reflect.Map {
		return nil, fmt.Errorf("input is not a map")
	}

	// Iterate over the keys and values of the map
	for _, key := range dataValue.MapKeys() {
		elements := dataValue.MapIndex(key).Interface()

		elementMap, err := StructListToMapList(elements, fields)
		if err != nil {
			return result, err
		}

		// Assign the converted map to the result under the original map key
		result[fmt.Sprintf("%v", key)] = elementMap
	}

	return result, nil
}

func MapStructToMapMapInterface(data interface{}, fields []string) (map[string]map[string]interface{}, error) {
	result := map[string]map[string]interface{}{}

	// Get the type and value of the input data
	dataType := reflect.TypeOf(data)
	dataValue := reflect.ValueOf(data)

	// Ensure the input is a slice
	if dataType.Kind() != reflect.Map {
		return nil, fmt.Errorf("input is not a map")
	}

	// Iterate over the keys and values of the map
	for _, key := range dataValue.MapKeys() {
		element := dataValue.MapIndex(key).Interface()

		// Convert the struct to a map with selected fields
		elementMap, err := StructToMap(element, fields)
		if err != nil {
			return nil, fmt.Errorf("error converting struct to map: %v", err)
		}

		// Assign the converted map to the result under the original map key
		result[fmt.Sprintf("%v", key)] = elementMap
	}

	return result, nil
}

func StructListToInterfaceList(data interface{}, fields []string) ([]interface{}, error) {
	var result []interface{}

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
		result = append(result, element)
	}

	return result, nil
}

// StructListToMapList converts a list of structs to a list of maps with selected fields.
func StructListToMapList(data interface{}, fields []string) ([]map[string]interface{}, error) {
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

func StructJsonTags(data interface{}) ([]string, error) {

	// Get the type and value of the input data
	dataType := reflect.TypeOf(data)

	// Ensure the input is a struct
	if dataType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input is not a struct")
	}

	results := []string{}
	for i := 0; i < dataType.NumField(); i++ {
		field := dataType.Field(i)
		tag := strings.Split(field.Tag.Get("json"), ",")[0]
		if tag != "" {
			results = append(results, tag)
		}
	}
	return results, nil

}
