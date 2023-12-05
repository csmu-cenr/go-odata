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
	Single(id string) (ModelT, error)
	List(queryOptions ODataQueryOptions) (<-chan Result, <-chan ModelT, <-chan error)
	Insert(model ModelT) (ModelT, error)
	Update(id string, model ModelT) (ModelT, error)
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

	options.ODataEditLink = queryParams.Get("$odataeditlink")
	options.ODataId = queryParams.Get("$odataidlink")
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

func (queryOptions ODataQueryOptions) toQueryString() string {
	queryStrings := url.Values{}
	if queryOptions.Select != "" {
		queryStrings.Add("$select", queryOptions.Select)
	}
	if queryOptions.Filter != "" {
		queryStrings.Add("$filter", queryOptions.Filter)
	}
	if queryOptions.Top != "" {
		queryStrings.Add("$top", queryOptions.Top)
	}
	if queryOptions.Skip != "" {
		queryStrings.Add("$skip", queryOptions.Skip)
	}
	if queryOptions.Count != "" {
		queryStrings.Add("$count", queryOptions.Count)
	}
	if queryOptions.OrderBy != "" {
		queryStrings.Add("orderby", queryOptions.OrderBy)
	}
	if queryOptions.Format != "" {
		queryStrings.Add("$format", queryOptions.Format)
	}
	result := queryStrings.Encode()
	result = strings.ReplaceAll(result, "+", "%20") // Using + for spaces causes issues - swap out to %20
	result = strings.ReplaceAll(result, "%24", "$") // sometimes %24 is not recognised as $ - make it explicitly $
	result = strings.ReplaceAll(result, "%2C", ",") // %2C stops odata from seeing the parameters, swap back to commas
	return result
}

func (dataSet odataDataSet[ModelT, Def]) getCollectionUrl() string {
	return dataSet.client.baseUrl + dataSet.modelDefinition.Url()
}

func (dataSet odataDataSet[ModelT, Def]) getSingleUrl(modelId string) string {
	return fmt.Sprintf("%s(%s)", dataSet.client.baseUrl+dataSet.modelDefinition.Url(), modelId)
}

type apiSingleResponse[T interface{}] struct {
	Value T `json:"value"`
}

type apiMultiResponse[T interface{}] struct {
	Value    []T    `json:"value"`
	Count    *int   `json:"@odata.count"`
	Context  string `json:"@odata.context"`
	NextLink string `json:"@odata.nextLink,omitempty"`
}

// Single model from the API by ID
func (dataSet odataDataSet[ModelT, Def]) Single(id string) (ModelT, error) {
	requestUrl := dataSet.getSingleUrl(id)
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

func selectFields(model interface{}, includeTags []string) {
	valueOfModel := reflect.ValueOf(model).Elem()
	typeOfModel := valueOfModel.Type()

	for i := 0; i < valueOfModel.NumField(); i++ {
		field := valueOfModel.Field(i)
		tag := typeOfModel.Field(i).Tag.Get("json")
		options := strings.Split(tag, ",")
		tag = options[0]

		if tag != "" && !contains(includeTags, tag) {
			// Set the field to nil
			if field.Kind() == reflect.Ptr {
				field.Set(reflect.Zero(field.Type()))
			}
		} else {
			// Make sure the nil value is returned as null
			// Check if the field is a pointer and is nil
			if field.Kind() == reflect.Ptr && field.IsNil() {
				// Allocate a new instance of the type and assign it to the field
				newInstance := reflect.New(field.Type().Elem())
				field.Set(newInstance)
			}
		}
	}
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
				if len(filters) > 0 {
					selectFields(&model, filters)
				}
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

// Insert a model to the API
func (dataSet odataDataSet[ModelT, Def]) Insert(model ModelT) (ModelT, error) {
	requestUrl := dataSet.getCollectionUrl()
	var result ModelT
	jsonData, err := json.Marshal(model)
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
func (dataSet odataDataSet[ModelT, Def]) Update(id string, model ModelT) (ModelT, error) {
	requestUrl := dataSet.getSingleUrl(id)
	var result ModelT
	jsonData, err := json.Marshal(model)
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
