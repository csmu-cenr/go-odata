package odataClient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

type odataDataSet[ModelT any, Def ODataModelDefinition[ModelT]] struct {
	client          *oDataClient
	modelDefinition ODataModelDefinition[ModelT]
}

type ODataDataSet[ModelT any, Def ODataModelDefinition[ModelT]] interface {
	Single(id string, options ODataQueryOptions) (ModelT, error)
	SingleValue(id string, options ODataQueryOptions) (ModelT, error)
	List(options ODataQueryOptions) (<-chan Result, <-chan ModelT, <-chan error)
	Insert(model ModelT, fields []string, quoted bool) (ModelT, error)
	Update(idOrEditLink string, model ModelT, fieldsToUpdate []string, quoted bool) (ModelT, error)
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

	options.Select = values.Get(SELECT)
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
	queryStrings := url.Values{}
	if options.Select != NOTHING {
		queryStrings.Add(SELECT, options.Select)
	}
	if options.Filter != NOTHING {
		queryStrings.Add(FILTER, options.Filter)
	}
	if options.Top != NOTHING {
		queryStrings.Add(TOP, options.Top)
	}
	if options.Skip != NOTHING {
		queryStrings.Add(SKIP, options.Skip)
	}
	if options.Count != NOTHING {
		queryStrings.Add(COUNT, options.Count)
	}
	if options.OrderBy != NOTHING {
		queryStrings.Add(ORDERBY, options.OrderBy)
	}
	if options.Format != NOTHING {
		queryStrings.Add(FORMAT, options.Format)
	}
	if options.Expand != NOTHING {
		queryStrings.Add(EXPAND, options.Expand)
	}
	if options.ODataEditLink != NOTHING {
		queryStrings.Add(ODATAEDITLINK, options.ODataEditLink)
	}
	if options.ODataNavigationLink != NOTHING {
		queryStrings.Add(ODATANAVIGATIONLINK, options.ODataNavigationLink)
	}
	if options.ODataId != NOTHING {
		queryStrings.Add(ODATAID, options.ODataId)
	}
	if options.ODataReadLink != NOTHING {
		queryStrings.Add(ODATAREADLINK, options.ODataReadLink)
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

	functionName := `odataDataSet[ModelT, Def]) Single`
	var responseModel ModelT

	requestUrl := dataSet.getSingleUrl(id)
	urlArgments := options.ToQueryString()
	if urlArgments != NOTHING {
		requestUrl = fmt.Sprintf("%s?%s", requestUrl, urlArgments)
	}
	request, err := http.NewRequest("GET", requestUrl, nil)

	if err != nil {
		message := ErrorMessage{ErrorNo: http.StatusInternalServerError,
			Message:    UNEXPECTED_ERROR,
			Function:   functionName,
			RequestUrl: requestUrl,
			Options:    &options,
			Details:    fmt.Sprintf(`%+v`, err)}
		return responseModel, message
	}
	responseData, err := executeHttpRequest[ModelT](*dataSet.client, request)
	if err != nil {
		message := ErrorMessage{ErrorNo: http.StatusBadRequest,
			Function:   functionName,
			RequestUrl: requestUrl,
			Options:    &options,
			InnerError: err}
		switch e := err.(type) {
		case *ErrorMessage:
			message.ErrorNo = e.ErrorNo
		case ErrorMessage:
			message.ErrorNo = e.ErrorNo
		default:
		}
		return responseModel, message
	}

	return responseData, nil
}

// Single model from the API using a Value tag, then model tags, by ID
func (dataSet odataDataSet[ModelT, Def]) SingleValue(id string, options ODataQueryOptions) (ModelT, error) {

	functionName := `odataDataSet[ModelT, Def]) SingleValue`

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
		message := ErrorMessage{ErrorNo: http.StatusBadRequest, Function: functionName,
			RequestUrl: requestUrl,
			InnerError: err}
		switch e := err.(type) {
		case *ErrorMessage:
			message.ErrorNo = e.ErrorNo
		case ErrorMessage:
			message.ErrorNo = e.ErrorNo
		default:
		}
		return responseModel, message
	}
	return responseData.Value, nil
}

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
					Payload:    options,
					InnerError: err,
					ErrorNo:    http.StatusInternalServerError}
				errs <- newRequestError
				close(meta)
				close(models)
				close(errs)
				return
			}
			responseData, err := executeHttpRequest[apiMultiResponse[ModelT]](*dataSet.client, request)
			if err != nil {
				executeHttpRequestError := ErrorMessage{
					ErrorNo:    http.StatusInternalServerError,
					Function:   "odataClient.List: Anonymous",
					Attempted:  "executeHttpRequest",
					RequestUrl: requestUrl,
					Options:    &options,
					InnerError: err}
				// get the internal error number
				switch e := err.(type) {
				case *ErrorMessage:
					executeHttpRequestError.ErrorNo = e.ErrorNo
				case ErrorMessage:
					executeHttpRequestError.ErrorNo = e.ErrorNo
				default:
				}

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

// isFieldExported checks if a struct field is public (exported).
func isFieldExported(field reflect.StructField) bool {
	return field.PkgPath == ""
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
			if isFieldExported(field) {
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
		if isFieldExported(field) {
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

// removeEmptyKeys resolves an issue when the odata source sends back an invalid editlink with an empty string key
func removeEmptyKeys(requestUrl string) string {

	if strings.Contains(requestUrl, DOUBLE_SINGLE_QUOTE) {
		elements := strings.Split(requestUrl, LEFT_BRACKET)
		keys := strings.Split(elements[1], COMMA)
		valid := []string{}
		for _, k := range keys {
			key := strings.Split(k, RIGHT_BRACKET)[0]
			if strings.EqualFold(key, DOUBLE_SINGLE_QUOTE) {
				continue
			}
			valid = append(valid, key)
		}
		requestUrl = fmt.Sprintf(`%s(%s)`, elements[0], strings.Join(valid, COMMA))
	}

	return requestUrl
}

// Insert a model to the API
func (dataSet odataDataSet[ModelT, Def]) Insert(model ModelT, fields []string, quoted bool) (ModelT, error) {

	functionName := `odataDataSet[ModelT, Def]) Insert`

	requestUrl := dataSet.getCollectionUrl()
	var result ModelT
	modelMap, err := StructToMap(model, fields)
	if err != nil {
		return result, err
	}
	if quoted {
		quotedMap := map[string]interface{}{}
		for k, v := range modelMap {
			quotedMap[fmt.Sprintf(`"%s"`, k)] = v
		}
		modelMap = quotedMap
	}
	data, err := json.Marshal(modelMap)
	if err != nil {
		message := ErrorMessage{ErrorNo: http.StatusInternalServerError,
			Attempted:  `http.NewRequest`,
			Function:   functionName,
			RequestUrl: requestUrl,
			Message:    UNEXPECTED_ERROR,
			Payload:    modelMap,
			Details:    fmt.Sprintf(`%+v`, err)}
		return result, message
	}
	request, err := http.NewRequest("POST", requestUrl, bytes.NewReader(data))
	if err != nil {
		message := ErrorMessage{ErrorNo: http.StatusInternalServerError,
			Attempted:  `http.NewRequest`,
			Function:   functionName,
			RequestUrl: requestUrl,
			Message:    UNEXPECTED_ERROR,
			Payload:    modelMap,
			Details:    fmt.Sprintf(`%+v`, err)}
		return result, message
	}
	request.Header.Set("Content-Type", "application/json;odata.metadata=minimal")
	request.Header.Set("Prefer", "return=representation")

	return executeHttpRequestPayload[ModelT](*dataSet.client, request, modelMap)
}

// Update a model in the API
func (dataSet odataDataSet[ModelT, Def]) Update(id string, model ModelT, fields []string, quoted bool) (ModelT, error) {

	functionName := `odataDataSet[ModelT, Def]) Update`

	requestUrl := removeEmptyKeys(dataSet.getSingleUrl(id))

	var result ModelT
	modelMap, err := StructToMap(model, fields)
	if err != nil {
		return result, err
	}
	if quoted {
		quotedMap := map[string]interface{}{}
		for k, v := range modelMap {
			quotedMap[fmt.Sprintf(`"%s"`, k)] = v
		}
		modelMap = quotedMap
	}
	data, err := json.Marshal(modelMap)
	if err != nil {
		message := ErrorMessage{ErrorNo: http.StatusInternalServerError,
			Attempted:  `json.Marshal`,
			Function:   functionName,
			RequestUrl: requestUrl, Message: UNEXPECTED_ERROR,
			Payload: modelMap,
			Details: fmt.Sprintf(`%+v`, err)}
		return result, message
	}
	request, err := http.NewRequest("PATCH", requestUrl, bytes.NewReader(data))
	if err != nil {
		message := ErrorMessage{ErrorNo: http.StatusInternalServerError,
			Attempted:  `http.NewRequest`,
			Function:   functionName,
			RequestUrl: requestUrl, Message: UNEXPECTED_ERROR,
			Payload: modelMap,
			Details: fmt.Sprintf(`%+v`, err)}
		return result, message
	}
	request.Header.Set("Content-Type", "application/json;odata.metadata=minimal")
	request.Header.Set("Prefer", "return=representation")

	return executeHttpRequestPayload[ModelT](*dataSet.client, request, modelMap)
}

// Delete a model from the API
func (dataSet odataDataSet[ModelT, Def]) Delete(id string) error {

	functionName := `odataDataSet[ModelT, Def]) Delete`

	requestUrl := dataSet.getSingleUrl(id)
	request, err := http.NewRequest("DELETE", requestUrl, nil)
	if err != nil {
		message := ErrorMessage{ErrorNo: http.StatusInternalServerError,
			Attempted: `http.NewRequest`,
			Function:  functionName, RequestUrl: requestUrl, Message: UNEXPECTED_ERROR,
			Details: fmt.Sprintf(`%+v`, err)}
		return message
	}
	dataSet.client.mapHeadersToRequest(request)
	response, err := dataSet.client.httpClient.Do(request)
	if response.Body != nil {
		defer response.Body.Close()
	}
	if err != nil {
		message := ErrorMessage{ErrorNo: response.StatusCode,
			Attempted: `dataSet.client.httpClient.Do`,
			Function:  functionName, RequestUrl: requestUrl, Message: UNEXPECTED_ERROR,
			Details: fmt.Sprintf(`%+v`, err)}
		return message
	}

	return nil
}

func (dataSet odataDataSet[ModelT, Def]) DeleteByFilter(options ODataQueryOptions) error {

	functionName := `odataDataSet[ModelT, Def]) DeleteByFilter`

	requestUrl := dataSet.getCollectionUrl()
	urlArgments := options.ToQueryString()
	if urlArgments != NOTHING {
		requestUrl = fmt.Sprintf("%s?%s", requestUrl, urlArgments)
	}
	request, err := http.NewRequest("DELETE", requestUrl, nil)
	if err != nil {
		message := ErrorMessage{ErrorNo: http.StatusInternalServerError,
			Attempted: `http.NewRequest`,
			Function:  functionName, RequestUrl: requestUrl, Message: UNEXPECTED_ERROR,
			Options: &options,
			Details: fmt.Sprintf(`%+v`, err)}
		return message
	}
	dataSet.client.mapHeadersToRequest(request)
	_, err = executeHttpRequest[apiDeleteResponse[ModelT]](*dataSet.client, request)
	if err != nil {
		message := ErrorMessage{ErrorNo: http.StatusInternalServerError,
			Attempted:  `executeHttpRequest`,
			Function:   functionName,
			RequestUrl: requestUrl, Message: UNEXPECTED_ERROR,
			Options: &options,
			Details: fmt.Sprintf(`%+v`, err)}
		return message
	}

	return nil
}

func (dataSet odataDataSet[ModelT, Def]) UpdateByFilter(model ModelT, fields []string, options ODataQueryOptions) error {

	functionName := `odataDataSet[ModelT, Def]) UpdateByFilter`

	requestUrl := dataSet.getCollectionUrl()
	urlArgments := options.ToQueryString()
	if urlArgments != NOTHING {
		requestUrl = fmt.Sprintf("%s?%s", requestUrl, urlArgments)
	}

	modelMap, err := StructToMap(model, fields)
	if err != nil {
		message := ErrorMessage{ErrorNo: http.StatusInternalServerError,
			Attempted:  `StructToMap`,
			Function:   functionName,
			RequestUrl: requestUrl, Message: UNEXPECTED_ERROR,
			Payload: fields,
			Options: &options,
			Details: fmt.Sprintf(`%+v`, err)}
		return message
	}
	if options.Quoted {
		quoted := map[string]interface{}{}
		for k, v := range modelMap {
			quoted[fmt.Sprintf(`"%s"`, k)] = v
		}
		modelMap = quoted
	}
	data, err := json.Marshal(modelMap)

	if err != nil {
		message := ErrorMessage{ErrorNo: http.StatusInternalServerError,
			Attempted:  `json.Marshal`,
			Function:   functionName,
			RequestUrl: requestUrl, Message: UNEXPECTED_ERROR,
			Payload: fields,
			Options: &options,
			Details: fmt.Sprintf(`%+v`, err)}
		return message
	}
	request, err := http.NewRequest("PATCH", requestUrl, bytes.NewReader(data))
	if err != nil {
		message := ErrorMessage{ErrorNo: http.StatusInternalServerError,
			Attempted:  `json.Marshal`,
			Function:   functionName,
			RequestUrl: requestUrl, Message: UNEXPECTED_ERROR,
			Payload: modelMap,
			Options: &options,
			Details: fmt.Sprintf(`%+v`, err)}
		return message
	}
	dataSet.client.mapHeadersToRequest(request)
	request.Header.Set("Content-Type", "application/json;odata.metadata=minimal")
	request.Header.Set("Prefer", "return=representation")

	_, err = executeHttpRequest[apiUpdateResponse[ModelT]](*dataSet.client, request)
	if err != nil {
		message := ErrorMessage{ErrorNo: http.StatusInternalServerError,
			Attempted:  `json.Marshal`,
			Function:   functionName,
			RequestUrl: requestUrl, Message: UNEXPECTED_ERROR,
			Payload:    modelMap,
			Options:    &options,
			InnerError: err}
		return message
	}

	return nil
}
