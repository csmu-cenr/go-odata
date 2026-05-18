package odataClient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
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
	Update(idOrEditLink string, model ModelT, fieldsToUpdate []string) (ModelT, error)
	UpdateByFilter(model ModelT, fieldsToUpdate []string, options ODataQueryOptions) error
	Delete(id string) error
	DeleteByFilter(options ODataQueryOptions) error

	getCollectionUrl() string
	getSingleUrl(modelId string) string
}

func NewDataSet[ModelT any, Def ODataModelDefinition[ModelT]](client ODataClient, modelDefinition Def) ODataDataSet[ModelT, Def] {
	return odataDataSet[ModelT, Def]{
		client:          client.(*oDataClient),
		modelDefinition: modelDefinition,
	}
}

func (options ODataQueryOptions) ApplyArguments(defaultFilter string, values url.Values) ODataQueryOptions {

	// Determine if the field names should be quoted
	if values.Has(QUOTED) {
		options.Quoted = values.Get(QUOTED) == TRUE
	} else {
		options.Quoted = true
	}

	if values.Has(ESCAPE) {
		options.Escape = values.Get(ESCAPE) == TRUE
	} else {
		options.Escape = false
	}

	// Quote the field names if requested
	if options.Quoted {
		options.Select = quoteCommaDelimited(values.Get(SELECT))
	} else {
		options.Select = values.Get(SELECT)
	}

	// Quote any fields in values["quote"]
	if values.Has(QUOTE) {
		found := false
		out := []string{}
		quote := values[QUOTE]
		fields := strings.Split(options.Select, COMMA)
		for _, f := range fields {
			if isDoubleQuoted(f) {
				out = append(out, f)
				continue
			}
			found = stringSliceContains(quote, f)
			if !found {
				out = append(out, f)
				continue
			}
			out = append(out, fmt.Sprintf(`"%s"`, f))
		}
		options.Select = strings.Join(out, COMMA)
	}

	// Remove quotes from fields that the odata provider rejects.
	if values.Has(DEQUOTE) {
		dequote := values[DEQUOTE]
		for _, v := range dequote {
			options.Select = strings.ReplaceAll(options.Select, fmt.Sprintf(`"%s"`, v), v)
		}
	}

	options.Count = values.Get(COUNT)
	options.Top = values.Get(TOP)
	options.Skip = values.Get(SKIP)
	options.OrderBy = values.Get(ORDERBY)

	options.Expand = values.Get(EXPAND)
	options.ODataEditLink = values.Get(ODATAEDITLINK)
	options.ODataNavigationLink = values.Get(ODATANAVIGATIONLINK)
	options.ODataEtag = values.Get(ODATAETAG)
	options.ODataId = values.Get(ODATAID)
	options.ODataReadLink = values.Get(ODATAREADLINK)

	filterValue := values.Get(FILTER)
	if defaultFilter == NOTHING && filterValue != NOTHING {
		options.Filter = filterValue
	}
	if defaultFilter != NOTHING && filterValue == NOTHING {
		options.Filter = defaultFilter
	}
	if defaultFilter != NOTHING && filterValue != NOTHING {
		if defaultFilter == filterValue {
			options.Filter = defaultFilter
		} else {
			options.Filter = fmt.Sprintf("(%s) and (%s)", defaultFilter, filterValue)
		}
	}

	// Quote the filter if options.Quoted
	if options.Filter != NOTHING && values.Has(QUOTE) {
		fields := []string{}
		fields = append(fields, values[QUOTE]...)
		options.Filter = quoteODataFields(options.Filter, fields)
	}

	format := values.Get((FORMAT))
	if format == NOTHING {
		options.Format = "json"
	} else {
		options.Format = format
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

func (options ODataQueryOptions) ToQueryString() string {
	values := url.Values{}
	if options.Select != NOTHING {
		values.Add(SELECT, options.Select)
	}
	if options.Filter != NOTHING {
		values.Add(FILTER, options.Filter)
	}
	if options.Top != NOTHING {
		values.Add(TOP, options.Top)
	}
	if options.Skip != NOTHING {
		values.Add(SKIP, options.Skip)
	}
	if options.Count != NOTHING {
		values.Add(COUNT, options.Count)
	}
	if options.OrderBy != NOTHING {
		values.Add(ORDERBY, options.OrderBy)
	}
	if options.Format != NOTHING {
		values.Add(FORMAT, options.Format)
	}
	if options.Expand != NOTHING {
		values.Add(EXPAND, options.Expand)
	}
	if options.ODataEditLink != NOTHING {
		values.Add(ODATAEDITLINK, options.ODataEditLink)
	}
	if options.ODataNavigationLink != NOTHING {
		values.Add(ODATANAVIGATIONLINK, options.ODataNavigationLink)
	}
	if options.ODataId != NOTHING {
		values.Add(ODATAID, options.ODataId)
	}
	if options.ODataReadLink != NOTHING {
		values.Add(ODATAREADLINK, options.ODataReadLink)
	}
	result := values.Encode()
	if options.Escape {
		result = strings.ReplaceAll(result, "%22", `"`) // %22 can stop odata from seeing the field name. swap back to "
	}
	result = strings.ReplaceAll(result, "%24", "$") // sometimes %24 is not recognised as $ - make it explicitly $
	result = strings.ReplaceAll(result, "%28", "(") // %28 can stop odata from seeing bracketed code swap back to (
	result = strings.ReplaceAll(result, "%29", ")") // %29 can stop odata from seeing bracketed code swap back to )
	result = strings.ReplaceAll(result, "%2C", ",") // %2C stops odata from seeing the parameters, swap back to commas
	result = strings.ReplaceAll(result, "%2F", "/") // %2F stops odata from seeing table identifiers swap back to slashes
	result = strings.ReplaceAll(result, "%3A", ":") // %3A can stop odata from makimg finds against timestamps- swap back to :
	result = strings.ReplaceAll(result, "%3D", "=") // %3D can stop odata from seeing equal signs swap back to =
	result = strings.ReplaceAll(result, "+", "%20") // Using + for spaces causes issues - swap out to %20
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

type apiDeleteResponse[T interface{}] struct {
	Count *int `json:"@odata.count"`
	Value []T  `json:"value"`
}

type apiUpdateResponse[T interface{}] struct {
	Count *int `json:"@odata.count"`
	Value []T  `json:"value"`
}

type apiSingleResponse[T interface{}] struct {
	Value T `json:"value"`
}

type apiMultiResponse[T interface{}] struct {
	Value    []T    `json:"value"`
	Count    *int   `json:"@odata.count"`
	Universe *int   `json:"universe,omitempty"` // total size of the set
	Context  string `json:"@odata.context"`
	NextLink string `json:"@odata.nextLink,omitempty"`
}

// Single model from the API by ID using the model json tags.
func (dataSet odataDataSet[ModelT, Def]) Single(id string, options ODataQueryOptions) (ModelT, error) {

	requestUrl := dataSet.getSingleUrl(id)
	urlArgments := options.ToQueryString()
	if urlArgments != NOTHING {
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
	urlArgments := options.ToQueryString()
	if urlArgments != NOTHING {
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

// func contains(slice []string, value string) bool {
// 	for _, v := range slice {
// 		if v == value {
// 			return true
// 		}
// 	}
// 	return false
// }

// List data from the API
func (dataSet odataDataSet[ModelT, Def]) List(options ODataQueryOptions) (<-chan Result, <-chan ModelT, <-chan error) {

	meta := make(chan Result)
	models := make(chan ModelT)
	errs := make(chan error)

	go func() {

		requestUrl := fmt.Sprintf("%s?%s",
			dataSet.getCollectionUrl(),
			options.ToQueryString())
		for requestUrl != NOTHING {
			request, err := http.NewRequest("GET", requestUrl, nil)
			if err != nil {
				newRequestError := ErrorMessage{
					Function:   "odataClient.List: Anonymous",
					Attempted:  `http.NewRequest GET`,
					RequestUrl: requestUrl,
					Details:    err}
				errs <- newRequestError
				close(meta)
				close(models)
				close(errs)
				return
			}
			responseData, err := executeHttpRequest[apiMultiResponse[ModelT]](*dataSet.client, request)
			if err != nil {
				executeHttpRequestError := ErrorMessage{
					Function:   "odataClient.List: Anonymous",
					Attempted:  "executeHttpRequest",
					RequestUrl: requestUrl,
					Details:    err}
				errs <- executeHttpRequestError
				close(meta)
				close(models)
				close(errs)
				return
			}
			close(errs) // defer(errs) was blocking.

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

// TODO Swap for official struct field is public function
func isFirstLetterCapital(s string) bool {
	// Check if the string is not empty
	if s == NOTHING {
		return false
	}

	// Get the first rune (Unicode character) in the string
	firstRune := []rune(s)[0]

	// Check if the first rune is uppercase
	return unicode.IsUpper(firstRune)
}

func StructToAny(data interface{}, fields []string) (interface{}, error) {
	result, err := StructToMap(data, fields)
	if err != nil {
		return nil, err
	}
	return result, nil
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

// StructListToInterface converts a list of structs to an interface
func StructListToInterface(data interface{}, fields []string) (interface{}, error) {
	return StructListToMapList(data, fields)
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

// Insert a model to the API
func (dataSet odataDataSet[ModelT, Def]) Insert(model ModelT, fields []string) (ModelT, error) {

	requestUrl := dataSet.getCollectionUrl()
	var result ModelT
	modelMap, err := StructToMap(model, fields)
	if err != nil {
		return result, err
	}
	payload, err := json.Marshal(modelMap)
	if err != nil {
		return result, err
	}
	request, err := http.NewRequest("POST", requestUrl, bytes.NewReader(payload))
	if err != nil {
		return result, err
	}
	request.Header.Set("Content-Type", "application/json;odata.metadata=minimal")
	request.Header.Set("Prefer", "return=representation")

	return executeHttpRequest[ModelT](*dataSet.client, request)
}

// Update a model in the API
func (dataSet odataDataSet[ModelT, Def]) Update(id string, model ModelT, fields []string) (ModelT, error) {

	requestUrl := dataSet.getSingleUrl(id)
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
	if response.Body != nil {
		response.Body.Close()
	}
	if response.StatusCode >= http.StatusBadRequest && err == nil {
		return fmt.Errorf(`{"StatusCode":%d, "Status":"%s"}`, response.StatusCode, response.Status)
	}
	if err != nil {
		return err
	}
	return nil
}

func (dataSet odataDataSet[ModelT, Def]) DeleteByFilter(options ODataQueryOptions) error {

	requestUrl := dataSet.getCollectionUrl()
	urlArgments := options.ToQueryString()
	if urlArgments != NOTHING {
		requestUrl = fmt.Sprintf("%s?%s", requestUrl, urlArgments)
	}
	request, err := http.NewRequest("DELETE", requestUrl, nil)
	dataSet.client.mapHeadersToRequest(request)
	if err != nil {
		return err
	}
	_, err = executeHttpRequest[apiDeleteResponse[ModelT]](*dataSet.client, request)
	if err != nil {
		return err
	}

	return nil
}

func (dataSet odataDataSet[ModelT, Def]) UpdateByFilter(model ModelT, fields []string, options ODataQueryOptions) error {

	requestUrl := dataSet.getCollectionUrl()
	urlArgments := options.ToQueryString()
	if urlArgments != NOTHING {
		requestUrl = fmt.Sprintf("%s?%s", requestUrl, urlArgments)
	}

	modelMap, err := StructToMap(model, fields)
	if err != nil {
		return err
	}
	jsonData, err := json.Marshal(modelMap)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PATCH", requestUrl, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	dataSet.client.mapHeadersToRequest(request)
	request.Header.Set("Content-Type", "application/json;odata.metadata=minimal")
	request.Header.Set("Prefer", "return=representation")

	_, err = executeHttpRequest[apiUpdateResponse[ModelT]](*dataSet.client, request)
	if err != nil {
		return err
	}

	return nil
}

// isDoubleQuoted checks if the first and last characters of the string are double quotes.
func isDoubleQuoted(field string) bool {
	// Must be at least 3 characters long to be quoted
	if len(field) < 2 {
		return false
	}

	// Get the first and last characters
	first := rune(field[0])
	last := rune(field[len(field)-1])

	// Return true if both are double quotes
	return first == '"' && last == '"'
}

// quoteCommaDelimited turns a comma delimited string into a double quoted comma delimited string.
// Such that 1,"2",3,4,""5"" is returned as "1","2","3","4","5"
func quoteCommaDelimited(input string) string {
	delimiter := `"`

	input = strings.ReplaceAll(input, `"`, ``)
	input = strings.ReplaceAll(input, `'`, ``)
	input = strings.ReplaceAll(input, `'`, ` `)

	// Split the string by commas
	parts := strings.Split(input, ",")

	// Process each part, strip existing quotes and enclose in double quotes
	for i, part := range parts {
		parts[i] = fmt.Sprintf(`%s%s%s`, delimiter, part, delimiter)
	}

	// Join the parts back together with commas
	return strings.Join(parts, ",")
}

// quoteODataFields quotes specified field names in standard OData expressions and function calls.
// 2025-06-18 Added support for functions - Keith John Hutchison
func quoteODataFields(query string, fields []string) string {
	if len(fields) == 0 {
		return query
	}

	// --- Step 1: Quote field names used with standard operators ---
	// List of supported OData operators
	operators := []string{
		"eq", "ne", "gt", "ge", "lt", "le",
		"and", "or", "not",
		"add", "sub", "mul", "div", "mod",
	}

	// Build regex pattern for fields followed by operators
	fieldPattern := `\b(` + strings.Join(fields, "|") + `)\b`
	operatorPattern := strings.Join(operators, "|")
	regex := regexp.MustCompile(`(?i)(\(?\s*)` + fieldPattern + `(\s+)(` + operatorPattern + `)\b`)

	// Replace matched fields with quoted versions
	result := regex.ReplaceAllStringFunc(query, func(match string) string {
		sub := regex.FindStringSubmatch(match)
		if len(sub) >= 4 {
			prefix := sub[1]
			field := sub[2]
			space := sub[3]
			operator := sub[4]
			return fmt.Sprintf("%s\"%s\"%s%s", prefix, field, space, operator)
		}
		return match
	})

	// --- Step 2: Quote field names used in function calls ---
	// e.g., contains(field, 'value') → contains("field", 'value')
	funcNames := []string{"startswith", "endswith", "contains", "length", "tolower", "toupper", "trim", "substring"}

	// Build regex to match function calls with the field as first argument
	funcPattern := `(?i)\b(` + strings.Join(funcNames, "|") + `)\(\s*(` + strings.Join(fields, "|") + `)\s*,`

	funcRegex := regexp.MustCompile(funcPattern)
	result = funcRegex.ReplaceAllStringFunc(result, func(match string) string {
		sub := funcRegex.FindStringSubmatch(match)
		if len(sub) >= 3 {
			funcName := sub[1]
			field := sub[2]
			return fmt.Sprintf(`%s("%s",`, funcName, field)
		}
		return match
	})

	return result
}

// stringSliceContains checks if a string slice stringSliceContains a specific element
func stringSliceContains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
