package odataClient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type odataDataSet[ModelT any, Def ODataModelDefinition[ModelT]] struct {
	client          *oDataClient
	modelDefinition ODataModelDefinition[ModelT]
}

type ODataDataSet[ModelT any, Def ODataModelDefinition[ModelT]] interface {
	Single(id string) (ModelT, error)
	List(queryOptions ODataQueryOptions) (<-chan Result, <-chan ModelT, <-chan error)
	Insert(model ModelT, selectFields []string) (ModelT, error)
	Update(id string, model ModelT, selectFields []string) (ModelT, error)
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

// Remove unwanted fields from a json object
func Extract(jsonInput string, fields []string) ([]byte, error) {

	var inputData map[string]interface{}
	err := json.Unmarshal([]byte(jsonInput), &inputData)
	if err != nil {
		return []byte{}, err
	}

	selectedData := make(map[string]interface{})
	for _, key := range fields {
		if value, ok := inputData[key]; ok {
			selectedData[key] = value
		}
	}

	selectedJSON, err := json.Marshal(selectedData)
	if err != nil {
		return []byte{}, err
	}

	return selectedJSON, err
}

// Insert a model to the API
func (dataSet odataDataSet[ModelT, Def]) Insert(model ModelT, selectFields []string) (ModelT, error) {
	requestUrl := dataSet.getCollectionUrl()
	var result ModelT
	jsonData, err := json.Marshal(model)
	if len(selectFields) > 0 {
		jsonData, err = Extract(string(jsonData), selectFields)
		if err != nil {
			return result, err
		}
	}
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
func (dataSet odataDataSet[ModelT, Def]) Update(id string, model ModelT, selectFields []string) (ModelT, error) {
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
