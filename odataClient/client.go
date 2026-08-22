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

type Error struct {
	Attempted  string   `json:"attempted,omitempty"`
	Body       any      `json:"body"`
	Code       string   `json:"code,omitempty"`
	Details    any      `json:"detail,omitempty"`
	ErrorNo    int      `json:"errorNo,omitempty"`
	Exit       string   `json:"exit,omitempty"`
	Function   string   `json:"function,omitempty"`
	InnerErr   error    `json:"innerErr"`
	Message    string   `json:"message"`
	RequestUrl string   `json:"requestUrl,omitempty"`
	Stack      []string `json:"stack"`
}

func (e Error) Error() string {
	bytes, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return fmt.Sprintf("Function: %s: Attempted: %s Details: %+v Body: %s", e.Function, e.Attempted, e.Details, e.Body)
	}
	return string(bytes)
}

type ODataQueryOptions struct {
	Table   string `json:"table,omitempty"`
	Select  string `json:"select,omitempty"`
	Filter  string `json:"filter,omitempty"`
	Count   string `json:"count,omitempty"`
	Top     string `json:"top,omitempty"`
	Skip    string `json:"skip,omitempty"`
	Limit   string `json:"limit,omitempty"`
	OrderBy string `json:"orderBy,omitempty"`
	Format  string `json:"format,omitempty"`
	Quoted  bool   `json:"quoted"`
	Escape  bool   `json:"escape"`

	Expand              string `json:"expand,omitempty"`
	ODataEditLink       string `json:"odataEditLink,omitempty"`
	ODataNavigationLink string `json:"odataNavigationLink,omitempty"`
	ODataEtag           string `json:"odataEtag,omitempty"`
	ODataId             string `json:"odataId,omitempty"`
	ODataReadLink       string `json:"odataReadLink,omitempty"`
	TimeOut             int    `json:"timeOut"`
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

func extract(err any) Error {
	in, e := json.Marshal(err)
	if e != nil {
		status := http.StatusBadRequest
		return Error{
			Details:  fmt.Sprintf(`Unable to extract %s`, err),
			ErrorNo:  status,
			Exit:     "f26a1d967175",
			InnerErr: e,
			Message:  http.StatusText(status),
		}
	}
	out := Error{}
	json.Unmarshal(in, &out)
	return out
}

func (client oDataClient) mapHeadersToRequest(req *http.Request) {
	for key, value := range client.headers {
		req.Header.Set(key, value)
	}
}

func executeHttpRequest[T any](client oDataClient, req *http.Request) (T, error) {

	function := `executeHttpRequest`
	stack := []string{function}
	link := getFullURL(req)

	client.mapHeadersToRequest(req)
	response, err := client.httpClient.Do(req)
	var responseData T
	if err != nil {
		httpClientDoError := Error{
			Attempted:  "client.httpClient.Do(req)",
			ErrorNo:    response.StatusCode,
			Exit:       "5e1cfbe5d9a3",
			Function:   function,
			InnerErr:   err,
			RequestUrl: link,
			Stack:      stack,
		}
		return responseData, httpClientDoError
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		message := Error{
			Attempted:  "body, err := io.ReadAll(response.Body)",
			ErrorNo:    response.StatusCode,
			Exit:       "f48ce3e25b2f",
			Function:   function,
			InnerErr:   err,
			RequestUrl: link,
			Stack:      stack,
		}
		return responseData, message
	}
	if response.StatusCode >= http.StatusBadRequest {
		m := Error{
			Attempted:  "response, err := client.httpClient.Do(req)",
			ErrorNo:    response.StatusCode,
			Exit:       "b3ef366b82ee",
			Function:   function,
			RequestUrl: link,
			Stack:      stack,
		}
		var data map[string]any
		err := json.Unmarshal(body, &data)
		if err != nil {
			m.Body = string(body)
			return responseData, m
		}
		// FileMaker Error Struct
		errVal, ok := data["error"]
		if ok {
			if errMap, ok := errVal.(map[string]interface{}); ok {
				code, _ := errMap["code"].(string)
				message, _ := errMap["message"].(string)
				m.Code = code
				m.Message = message
			}
		} else {
			m.Details = data
		}
		return responseData, m
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
			message := Error{
				Attempted: "err = json.Unmarshal(sanitised, &responseData)",
				Body:      string(sanitised),
				ErrorNo:   http.StatusInternalServerError,
				Exit:      "1ef3b68505bc",
				Function:  function,
				InnerErr:  err,
				Message:   fmt.Sprintf(`%v`, err),
				Stack:     stack,
			}
			return responseData, message
		}
	}

	return responseData, nil
}

// Function to get the full URL from the http.Request
func getFullURL(req *http.Request) string {

	scheme := req.URL.Scheme
	host := req.URL.Host
	path := req.URL.Path
	port := req.URL.Port()
	switch port {
	case "443", "80", "":
		port = NOTHING
	default:
		port = `:` + port
	}
	query := req.URL.RawQuery
	if len(query) > 0 {
		query = `?` + query
	}

	// Construct the full URL
	result := fmt.Sprintf("%s://%s%s%s%s", scheme, host, port, path, query)

	return result
}
