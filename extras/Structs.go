package dataModel

import (
	"fmt"
	"net/http"
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

func hasField(typ reflect.Type, fieldName string) bool {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == fieldName {
			return true
		}
	}
	return false
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
func StructListToInterface(data interface{}, fields []string) (interface{}, error) {
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

// StructToMap
// Move/cop to nullable
func StructToMap(data interface{}, fields []string) (map[string]interface{}, error) {

	function := `dmdata.StructToMap`

	result := map[string]interface{}{}

	if data == nil {
		m := ErrorMessage{
			Details:  `data cannot be nil`,
			ErrorNo:  http.StatusBadRequest,
			Function: function,
			Message:  "bad request",
		}
		return result, m
	}

	// Dereference pointers if necessary
	check := reflect.ValueOf(data)
	if check.Kind() == reflect.Ptr && !check.IsNil() {
		check = check.Elem()
		data = check
	}

	// Get the type and value of the input data
	dataType := reflect.TypeOf(data)
	dataValue := reflect.ValueOf(data)

	// Ensure the input is a struct
	if dataType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input is not a struct")
	}

	// If fields is empty, map all fields in the struct
	if len(fields) == 0 {
		for i := 0; i < dataType.NumField(); i++ {
			field := dataType.Field(i)
			if isFirstLetterCapital(field.Name) {
				fieldValue := dataValue.Field(i).Interface()
				jsonTag := field.Tag.Get("json")
				if jsonTag == "" {
					jsonTag = field.Name
				} else {
					jsonTag = strings.Split(jsonTag, ",")[0]
				}
				result[jsonTag] = fieldValue
			}
		}
		return result, nil
	}

	// Iterate over the fields to be selected
	for _, fieldName := range fields {
		// Find the field by JSON tag
		field, found := findFieldByJSONTag(dataType, fieldName)
		if !found {
			// Ignore fields not found in the struct
			continue
		}

		// field names must be exported to access the value
		if isFirstLetterCapital(field.Name) {
			// Get the field value
			fieldValue := dataValue.FieldByName(field.Name).Interface()

			// Add the field and value to the result map
			result[fieldName] = fieldValue
		} else {
			result[fieldName] = nil
		}
	}

	return result, nil
}
