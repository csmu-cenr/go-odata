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
	Message    string             `json:"message,omitempty"`
	ErrorNo    int                `json:"errorNo"`
	Function   string             `json:"function,omitempty"`
	Attempted  string             `json:"attempted,omitempty"`
	Body       interface{}        `json:"body,omitempty"`
	Details    interface{}        `json:"details,omitempty"`
	Options    *ODataQueryOptions `json:"options,omitempty"`
	Payload    interface{}        `json:"payload"`
	InnerError interface{}        `json:"err,omitempty"`
	RequestUrl string             `json:"requestUrl,omitempty"`
}

func (ts ErrorMessage) Error() string {
	bytes, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		return fmt.Sprintf("Function: %s: Attempted: %s Details: %+v Body: %s", ts.Function, ts.Attempted, ts.Details, ts.Body)
	}
	return string(bytes)
}

// Function to get the full URL from the http.Request
func getFullURL(req *http.Request) string {
	// Determine the scheme (http or https)
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	// Construct the full URL
	return fmt.Sprintf("%s://%s%s", scheme, req.Host, req.RequestURI)
}

type ODataQueryOptions struct {
	Select  string `json:"select,omitempty"`
	Filter  string `json:"filter,omitempty"`
	Count   string `json:"count,omitempty"`
	Top     string `json:"top,omitempty"`
	Skip    string `json:"skip,omitempty"`
	OrderBy string `json:"orderBy,omitempty"`
	Format  string `json:"format,omitempty"`

	Expand              string `json:"expand,omitempty"`
	ODataEditLink       string `json:"odataEditLink,omitempty"`
	ODataNavigationLink string `json:"odataNavigationLink,omitempty"`
	ODataEtag           string `json:"odataEtag,omitempty"`
	ODataId             string `json:"odataId,omitempty"`
	ODataReadLink       string `json:"odataReadLink,omitempty"`
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

	functionName := `executeHttpRequest`
	link := getFullURL(req)

	client.mapHeadersToRequest(req)
	response, err := client.httpClient.Do(req)
	var responseData T
	if err != nil {
		httpClientDoError := ErrorMessage{
			Function:   functionName,
			Attempted:  "client.httpClient.Do(req)",
			InnerError: err,
			RequestUrl: link,
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
			RequestUrl: link,
			ErrorNo:    http.StatusInternalServerError}
		return responseData, message
	}
	if response.StatusCode >= http.StatusBadRequest {
		message := ErrorMessage{Function: "executeHttpRequest",
			Attempted:  "response, err := client.httpClient.Do(req)",
			RequestUrl: link,
			ErrorNo:    response.StatusCode}
		var data map[string]interface{}
		err := json.Unmarshal(body, &data)
		if err != nil {
			message.Details = string(body)
			return responseData, message
		}
		message.Details = data
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
				ErrorNo: http.StatusInternalServerError,
				Message: fmt.Sprintf(`%w`, err), Function: "odataClient.executeHttpRequest",
				Attempted: "err = json.Unmarshal(sanitised, &responseData)",
				Body:      string(sanitised), InnerError: err}
			return responseData, message
		}
	}

	return responseData, nil
}

func executeHttpRequestPayload[T interface{}](client oDataClient, req *http.Request, payload interface{}) (T, error) {

	functionName := `executeHttpRequestPayload`
	link := getFullURL(req)

	client.mapHeadersToRequest(req)
	response, err := client.httpClient.Do(req)
	var responseData T
	if err != nil {
		httpClientDoError := ErrorMessage{
			Function:   functionName,
			Attempted:  "client.httpClient.Do(req)",
			InnerError: err,
			RequestUrl: link,
			Payload:    payload,
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
			RequestUrl: link,
			Payload:    payload,
			ErrorNo:    http.StatusInternalServerError}
		return responseData, message
	}
	if response.StatusCode >= http.StatusBadRequest {
		message := ErrorMessage{Function: "executeHttpRequest",
			Attempted:  "response, err := client.httpClient.Do(req)",
			RequestUrl: link,
			Message:    UNEXPECTED_ERROR,
			Payload:    payload,
			ErrorNo:    response.StatusCode}
		var data map[string]interface{}
		err := json.Unmarshal(body, &data)
		if err != nil {
			message.Details = string(body)
			return responseData, message
		}
		message.Details = data
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
				ErrorNo: http.StatusInternalServerError,
				Message: fmt.Sprintf(`%w`, err), Function: "odataClient.executeHttpRequest",
				Attempted:  "err = json.Unmarshal(sanitised, &responseData)",
				Body:       string(sanitised),
				Payload:    payload,
				InnerError: err}
			return responseData, message
		}
	}

	return responseData, nil
}
