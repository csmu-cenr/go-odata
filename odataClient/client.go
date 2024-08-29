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
	Message    string      `json:"message,omitempty"`
	ErrorNo    int         `json:"errorNo"`
	Function   string      `json:"function,omitempty"`
	Attempted  string      `json:"attempted,omitempty"`
	Body       interface{} `json:"body,omitempty"`
	Details    interface{} `json:"detail,omitempty"`
	InnerError interface{} `json:"err,omitempty"`
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

	Expand              string
	ODataEditLink       string
	ODataNavigationLink string
	ODataEtag           string
	ODataId             string
	ODataReadLink       string
}

func (options *ODataQueryOptions) Fields() []string {
	return strings.Split(options.Select, ",")
}

func (options *ODataQueryOptions) FieldsWithODataTags() []string {
	fields := options.Fields()
	if options.ODataId == TRUE {
		fields = append(fields, "@odata.id")
	}
	if options.ODataEditLink == TRUE {
		fields = append(fields, "@odata.editLink")
	}
	if options.ODataEtag == TRUE {
		fields = append(fields, "@odata.etag")
	}
	if options.ODataNavigationLink == TRUE {
		fields = append(fields, "@odata.navigationLink")
	}
	if options.ODataReadLink == TRUE {
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
			Function:   "executeHttpRequest",
			Attempted:  "response, err := client.httpClient.Do(req)",
			InnerError: err,
			ErrorNo:    http.StatusInternalServerError}
		return responseData, httpClientDoError
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		message := ErrorMessage{
			Function:   "executeHttpRequest",
			Attempted:  "body, err := io.ReadAll(response.Body)",
			InnerError: err,
			ErrorNo:    http.StatusInternalServerError}
		return responseData, message
	}
	if response.StatusCode >= http.StatusBadRequest {
		message := ErrorMessage{Function: "executeHttpRequest",
			Attempted: "response, err := client.httpClient.Do(req)",
			ErrorNo:   response.StatusCode}
		var data map[string]interface{}
		err := json.Unmarshal(body, &data)
		if err != nil {
			message.Details = string(body)
			return responseData, message
		} else {
			message.Details = data
		}
		return responseData, message
	}
	if response.StatusCode == http.StatusNoContent {
		return responseData, nil
	}

	err = json.Unmarshal(body, &responseData)
	if err != nil {

		// Might be dirty data.
		sanitised := body
		// Had some dirty data being returned by an odata source where : null was being returned as : ?
		questionMark := []byte(`": ?`)
		null := []byte(`": null`)
		sanitised = bytes.ReplaceAll(sanitised, questionMark, null)
		// Had some dirty data being returned by an odata source where -0.5 was being returned as -.5 - which is invalid for a number
		minus := []byte(`": -.`)
		zero := []byte(`": -0.`)
		sanitised = bytes.ReplaceAll(sanitised, minus, zero)

		err = json.Unmarshal(sanitised, &responseData)
		if err != nil {
			message := ErrorMessage{
				Message: fmt.Sprintf(`%w`, err), Function: "odataClient.executeHttpRequest",
				Attempted: "err = json.Unmarshal(sanitised, &responseData)",
				Body:      string(sanitised), InnerError: err}
			return responseData, message
		}
	}

	return responseData, nil
}
