package odataClient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"unicode"
)

type odataDataSet[ModelT any, Def ODataModelDefinition[ModelT]] struct {
	client          *oDataClient
	modelDefinition ODataModelDefinition[ModelT]
}

type ODataDataSet[ModelT any, Def ODataModelDefinition[ModelT]] interface {
	Single(id string, options ODataQueryOptions) (ModelT, error)
	SingleValue(id string, options ODataQueryOptions) (ModelT, error)
	List(options ODataQueryOptions) (<-chan Result, <-chan ModelT, <-chan error)
	Insert(model ModelT, fields []string) (ModelT, error)
	Update(id string, model ModelT, fields []string) (ModelT, error)
	Delete(id string) error

	getCollectionUrl() string
	getSingleUrl(modelId string) string
}

func NewDataSet[ModelT any, Def ODataModelDefinition[ModelT]](client ODataClient, modelDefinition Def) ODataDataSet[ModelT, Def] {
	return odataDataSet[ModelT, Def]{
		client:          client.(*oDataClient),
		modelDefinition: modelDefinition,
	}
}

func (options ODataQueryOptions) ApplyArguments(defaultFilter string, queryParams url.Values) ODataQueryOptions {

	options.Select = queryParams.Get("$select")
	options.Count = queryParams.Get("$count")
	options.Top = queryParams.Get("$top")
	options.Skip = queryParams.Get("$skip")
	options.OrderBy = queryParams.Get("$orderby")

	options.Expand = queryParams.Get("$expand")
	options.ODataEditLink = queryParams.Get("$odataeditlink")
	options.ODataId = queryParams.Get("$odataid")
	options.ODataReadLink = queryParams.Get("$odatareadlink")

	filterValue := queryParams.Get("$filter")
	if defaultFilter == "" && filterValue != "" {
		options.Filter = filterValue
	}
	if defaultFilter != "" && filterValue == "" {
		options.Filter = defaultFilter
	}
	if defaultFilter != "" && filterValue != "" {
		options.Filter = fmt.Sprintf("(%s) and (%s)", defaultFilter, filterValue)
	}

	formatValue := queryParams.Get(("$format"))
	if formatValue == "" {
		options.Format = "json"
	} else {
		options.Format = formatValue
	}

	return options
}

func ConvertInterfaceToBytes(data interface{}) ([]byte, error) {
	// Use json.Marshal to convert the interface to a JSON-formatted byte slice
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error converting interface to bytes: %v", err)
	}
	return bytes, nil
}

func (options ODataQueryOptions) toQueryString() string {
	queryStrings := url.Values{}
	if options.Select != "" {
		queryStrings.Add("$select", options.Select)
	}
	if options.Filter != "" {
		queryStrings.Add("$filter", options.Filter)
	}
	if options.Top != "" {
		queryStrings.Add("$top", options.Top)
	}
	if options.Skip != "" {
		queryStrings.Add("$skip", options.Skip)
	}
	if options.Count != "" {
		queryStrings.Add("$count", options.Count)
	}
	if options.OrderBy != "" {
		queryStrings.Add("$orderby", options.OrderBy)
	}
	if options.Format != "" {
		queryStrings.Add("$format", options.Format)
	}
	if options.Expand != "" {
		queryStrings.Add("$expand", options.Expand)
	}
	if options.ODataEditLink != "" {
		queryStrings.Add("$odataeditlink", options.ODataEditLink)
	}
	if options.ODataId != "" {
		queryStrings.Add("$odataid", options.ODataId)
	}
	if options.ODataReadLink != "" {
		queryStrings.Add("$odatareadlink", options.ODataReadLink)
	}
	result := queryStrings.Encode()
	result = strings.ReplaceAll(result, "+", "%20") // Using + for spaces causes issues - swap out to %20
	result = strings.ReplaceAll(result, "%24", "$") // sometimes %24 is not recognised as $ - make it explicitly $
	result = strings.ReplaceAll(result, "%2C", ",") // %2C stops odata from seeing the parameters, swap back to commas
	result = strings.ReplaceAll(result, "%2F", "/") // %3D stops odata from seeing table identifiers swap back to slashes
	result = strings.ReplaceAll(result, "%28", "(") // %28 can stop odata from seeing bracketed code swap back to (
	result = strings.ReplaceAll(result, "%29", ")") // %29 can stop odata from seeing bracketed code swap back to )
	result = strings.ReplaceAll(result, "%3D", "=") // %3D can stop odata from seeing equal signs swap back to =
	return result
}

func (dataSet odataDataSet[ModelT, Def]) getCollectionUrl() string {
	return dataSet.client.baseUrl + dataSet.modelDefinition.Url()
}

func (dataSet odataDataSet[ModelT, Def]) getSingleUrl(modelId string) string {
	leftBracket := strings.Contains(modelId, "(")
	rightBracket := strings.Contains(modelId, ")")
	if leftBracket && rightBracket {
		return modelId
	}
	return fmt.Sprintf("%s(%s)", dataSet.client.baseUrl+dataSet.modelDefinition.Url(), modelId)
}

type apiSingleResponse[T interface{}] struct {
	Value T `json:"value"`
}

type apiMultiResponse[T interface{}] struct {
	Value    []T    `json:"value"`
	Count    *int   `json:"@odata.count"`
	Universe int    `json:"universe"` // total size of the set
	Context  string `json:"@odata.context"`
	NextLink string `json:"@odata.nextLink,omitempty"`
}

// Single model from the API by ID using the model json tags.
func (dataSet odataDataSet[ModelT, Def]) Single(id string, options ODataQueryOptions) (ModelT, error) {
	requestUrl := dataSet.getSingleUrl(id)
	urlArgments := options.toQueryString()
	if urlArgments != "" {
		requestUrl = fmt.Sprintf("%s?%s", requestUrl, urlArgments)
	}
	request, err := http.NewRequest("GET", requestUrl, nil)
	var responseModel ModelT
	if err != nil {
		return responseModel, err
	}
	responseData, err := executeHttpRequest[ModelT](*dataSet.client, request)
	if err != nil {
		return responseModel, err
	}
	return responseData, nil
}

// Single model from the API using a Value tag, then model tags, by ID
func (dataSet odataDataSet[ModelT, Def]) SingleValue(id string, options ODataQueryOptions) (ModelT, error) {
	requestUrl := dataSet.getSingleUrl(id)
	urlArgments := options.toQueryString()
	if urlArgments != "" {
		requestUrl = fmt.Sprintf("%s?%s", requestUrl, urlArgments)
	}
	request, err := http.NewRequest("GET", requestUrl, nil)
	var responseModel ModelT
	if err != nil {
		return responseModel, err
	}
	responseData, err := executeHttpRequest[apiSingleResponse[ModelT]](*dataSet.client, request)
	if err != nil {
		return responseModel, err
	}
	return responseData.Value, nil
}

func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// List data from the API
func (dataSet odataDataSet[ModelT, Def]) List(options ODataQueryOptions) (<-chan Result, <-chan ModelT, <-chan error) {

	meta := make(chan Result)
	models := make(chan ModelT)
	errs := make(chan error)

	go func() {

		requestUrl := fmt.Sprintf("%s?%s",
			dataSet.getCollectionUrl(),
			options.toQueryString())
		for requestUrl != "" {
			request, err := http.NewRequest("GET", requestUrl, nil)
			if err != nil {
				errs <- err
				close(meta)
				close(models)
				close(errs)
				return
			}
			responseData, err := executeHttpRequest[apiMultiResponse[ModelT]](*dataSet.client, request)
			if err != nil {
				errs <- err
				close(meta)
				close(models)
				close(errs)
				return
			}
			close(errs) // defer(errs) was blocking.
			var filters []string
			if options.Select != "" {
				filters = strings.Split(options.Select, ",")
			}
			if options.ODataEditLink == "true" {
				filters = append(filters, "@odata.editLink")
			}
			if options.ODataId == "true" {
				filters = append(filters, "@odata.id")
			}
			if options.ODataReadLink == "true" {
				filters = append(filters, "@odata.readLink")
			}
			result := Result{}
			result.Context = responseData.Context
			if options.Count == "true" {
				result.Count = responseData.Count
			}
			result.Model = dataSet.modelDefinition.Url()
			result.NextLink = responseData.NextLink
			meta <- result
			close(meta)
			for _, model := range responseData.Value {
				models <- model
			}
			defer close(models)
			if len(responseData.Value) < dataSet.client.defaultPageSize {
				return
			}
			requestUrl = responseData.NextLink
		}
	}()

	return meta, models, errs
}

// Makes a byte slice into an any slice
func MakeBytesAny(bytes []byte) []any {
	anySlice := make([]interface{}, len(bytes))
	for i, v := range bytes {
		anySlice[i] = v
	}
	return anySlice
}

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

func StructToMap(data interface{}, fields []string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

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
				result[field.Name] = fieldValue
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

// Select fields from a list of json objects
func SelectFieldsList(jsonInput string, fields []string) ([]byte, error) {

	var inputData []map[string]interface{}
	err := json.Unmarshal([]byte(jsonInput), &inputData)
	if err != nil {
		return []byte{}, err

	}

	selectedData := make([]map[string]interface{}, len(inputData))
	for i, item := range inputData {
		selectedItem := make(map[string]interface{})
		for _, key := range fields {
			if value, ok := item[key]; ok {
				selectedItem[key] = value
			}
		}
		selectedData[i] = selectedItem
	}

	selectedList, err := json.Marshal(selectedData)
	if err != nil {
		return []byte{}, err
	}

	return selectedList, err
}

// Insert a model to the API
func (dataSet odataDataSet[ModelT, Def]) Insert(model ModelT, fields []string) (ModelT, error) {
	requestUrl := dataSet.getCollectionUrl()
	var result ModelT
	modelMap, err := StructToMap(model, fields)
	if err != nil {
		return result, err
	}
	jsonData, err := json.Marshal(modelMap)
	if err != nil {
		return result, err
	}
	request, err := http.NewRequest("POST", requestUrl, bytes.NewReader(jsonData))
	if err != nil {
		return result, err
	}
	request.Header.Set("Content-Type", "application/json;odata.metadata=minimal")
	request.Header.Set("Prefer", "return=representation")
	return executeHttpRequest[ModelT](*dataSet.client, request)
}

// Update a model in the API
func (dataSet odataDataSet[ModelT, Def]) Update(id string, model ModelT, fields []string) (ModelT, error) {
	requestUrl := ""
	leftBracket := strings.Contains(id, "(")
	rightBracket := strings.Contains(id, ")")
	if !leftBracket && !rightBracket {
		requestUrl = dataSet.getSingleUrl(id)
	} else {
		requestUrl = id
	}
	var result ModelT
	modelMap, err := StructToMap(model, fields)
	if err != nil {
		return result, err
	}
	jsonData, err := json.Marshal(modelMap)
	if err != nil {
		return result, err
	}
	request, err := http.NewRequest("PATCH", requestUrl, bytes.NewReader(jsonData))
	if err != nil {
		return result, err
	}
	request.Header.Set("Content-Type", "application/json;odata.metadata=minimal")
	request.Header.Set("Prefer", "return=representation")
	return executeHttpRequest[ModelT](*dataSet.client, request)
}

// Delete a model from the API
func (dataSet odataDataSet[ModelT, Def]) Delete(id string) error {
	requestUrl := dataSet.getSingleUrl(id)
	request, err := http.NewRequest("DELETE", requestUrl, nil)
	if err != nil {
		return err
	}
	dataSet.client.mapHeadersToRequest(request)
	response, err := dataSet.client.httpClient.Do(request)
	if err != nil {
		return err
	}
	_ = response.Body.Close()
	return nil
}
