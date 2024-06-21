package odataClient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type oDataClient struct {
	baseUrl         string
	headers         map[string]string
	httpClient      *http.Client
	defaultPageSize int
}

type ErrorMessage struct {
	Message    string      `json:"message"`
	Function   string      `json:"function,omitempty"`
	Attempted  string      `json:"attempted,omitempty"`
	Body       interface{} `json:"body"`
	Code       int         `json:"code,omitempty"`
	Details    interface{} `json:"detail,omitempty"`
	RequestUrl string      `json:"requestUrl,omitempty"`
}

func (ts ErrorMessage) Error() string {
	bytes, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		return fmt.Sprintf("Function: %s: Attempted: %s Details: %+v Body: %s", ts.Function, ts.Attempted, ts.Details, ts.Body)
	}
	return string(bytes)
}

type ODataQueryOptions struct {
	Select  string
	Filter  string
	Count   string
	Top     string
	Skip    string
	OrderBy string
	Format  string

	Expand        string
	ODataEditLink string
	ODataEtag     string
	ODataId       string
	ODataReadLink string
}

func (o *ODataQueryOptions) Fields() []string {
	return strings.Split(o.Select, ",")
}

func (o *ODataQueryOptions) FieldsWithODataTags() []string {
	fields := o.Fields()
	if o.ODataId == "true" {
		fields = append(fields, "@odata.id")
	}
	if o.ODataEditLink == "true" {
		fields = append(fields, "@odata.editLink")
	}
	if o.ODataEtag == "true" {
		fields = append(fields, "@odata.etag")
	}
	if o.ODataReadLink == "true" {
		fields = append(fields, "@odata.readLink")
	}
	return fields
}

// ODataClient represents a connection to the OData REST API
type ODataClient interface {
	Wrapper
	AddHeader(key string, value string)
	ODataQueryOptions() ODataQueryOptions
}

// Wrapper represents a wrapper around the OData client if you have build own code around the OData itself, for authentication etc
type Wrapper interface {
	ODataClient() ODataClient
}

func (client *oDataClient) ODataQueryOptions() ODataQueryOptions {
	return ODataQueryOptions{}
}

func New(baseUrl string) ODataClient {
	client := &oDataClient{
		baseUrl: strings.TrimRight(baseUrl, "/") + "/",
		headers: map[string]string{
			"DataServiceVersion": "4.0",
			"OData-Version":      "4.0",
			"Accept":             "application/json",
		},
		defaultPageSize: 1000,
	}

	httpTransport := &http.Transport{}
	client.httpClient = &http.Client{
		Transport: httpTransport,
	}

	return client
}

// AddHeader will add a custom HTTP Header to the API requests
func (client *oDataClient) AddHeader(key string, value string) {
	client.headers[strings.ToLower(key)] = value
}

// ODataClient will return self, so it also works as a wrapper in case we don't have a wrapper
func (client *oDataClient) ODataClient() ODataClient {
	return client
}

func (client oDataClient) mapHeadersToRequest(req *http.Request) {
	for key, value := range client.headers {
		req.Header.Set(key, value)
	}
}

func executeHttpRequest[T interface{}](client oDataClient, req *http.Request) (T, error) {

	client.mapHeadersToRequest(req)
	response, err := client.httpClient.Do(req)
	var responseData T
	if err != nil {
		httpClientDoError := ErrorMessage{
			Function:  "executeHttpRequest",
			Attempted: "response, err := client.httpClient.Do(req)",
			Details:   err}
		return responseData, httpClientDoError
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		ioReadAllError := ErrorMessage{
			Function:  "executeHttpRequest",
			Attempted: "body, err := io.ReadAll(response.Body)",
			Details:   err}
		return responseData, ioReadAllError
	}
	if response.StatusCode >= 400 {
		executeHttpRequestError := ErrorMessage{Function: "executeHttpRequest",
			Attempted: "response, err := client.httpClient.Do(req)",
			Code:      response.StatusCode}
		var data map[string]interface{}
		err := json.Unmarshal(body, &data)
		if err != nil {
			executeHttpRequestError.Details = string(body)
			return responseData, executeHttpRequestError
		} else {
			executeHttpRequestError.Details = data
		}
		return responseData, executeHttpRequestError
	}
	if response.StatusCode == 204 {
		// No Data - but no error
		return responseData, nil
	}
	colonSpaceQuestion := []byte(`": ?`)
	colonNull := []byte(`": null`)
	sanitised := bytes.ReplaceAll(body, colonSpaceQuestion, colonNull)
	if err != nil {
		modelError := ErrorMessage{
			Message:   err.Error(),
			Function:  "odataClient.executeHttpRequest",
			Attempted: "json.MarshalIndent",
			Body:      string(sanitised),
			Details:   err}
		return responseData, modelError
	}
	err = json.Unmarshal(sanitised, &responseData)
	if err != nil {
		modelError := ErrorMessage{
			Message:   err.Error(),
			Function:  "odataClient.executeHttpRequest",
			Attempted: "json.Unmarshal(body, &responseData)",
			Body:      string(sanitised),
			Details:   err}
		return responseData, modelError
	}

	return responseData, nil
}
