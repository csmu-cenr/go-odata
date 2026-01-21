package odataClient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	netutil "github.com/Uffe-Code/go-odata/netutil"
	"github.com/Uffe-Code/go-odata/typename"
)

const (
	BAD_REQUEST = `bad request`
)

type oDataClient struct {
	baseUrl         string
	headers         map[string]string
	httpClient      *http.Client
	defaultPageSize int
}

type ErrorMessage struct {
	Attempted     string             `json:"attempted,omitempty"`
	Body          any                `json:"body,omitempty"`
	Code          string             `json:"code,omitempty"`
	Details       any                `json:"details,omitempty"`
	ErrorNo       int                `json:"errorNo,omitempty"`
	Exit          string             `json:"exit,omitempty"`
	Function      string             `json:"function,omitempty"`
	InnerError    any                `json:"err,omitempty"`
	Message       string             `json:"message,omitempty"`
	Options       *ODataQueryOptions `json:"options,omitempty"`
	Payload       any                `json:"payload,omitempty"`
	RequestUrl    string             `json:"requestUrl,omitempty"`
	Stack         []string           `json:"stack,omitempty"`
	UnixTimestamp int64              `json:"unixTimestamp,omitempty"`
}

func (e ErrorMessage) Error() string {
	bytes, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return fmt.Sprintf("Function: %s: Attempted: %s Details: %+v Body: %s", e.Function, e.Attempted, e.Details, e.Body)
	}
	return string(bytes)
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

type ODataQueryOptions struct {
	Arguments struct {
		DefaultFilter string     `json:"defaultFilter"`
		Values        url.Values `json:"url.Values"`
	}
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

	TimeoutSeconds float64 `json:"timeoutSeconds"`
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

// executeHttpRequest
func executeHttpRequest[T any](client oDataClient, req *http.Request) (T, error) {

	name := typename.ShortTypeName[T]()
	function := fmt.Sprintf(`executeHttpRequest.%s`, name)
	link := getFullURL(req)

	client.mapHeadersToRequest(req)
	response, err := client.httpClient.Do(req)
	var t T
	if err != nil {
		errorNo := http.StatusInternalServerError
		if response != nil {
			errorNo = response.StatusCode
		}
		message := UNEXPECTED_ERROR
		details := fmt.Sprintf(`%+v`, err)
		isTimeout, description := netutil.GetTimeoutInfo(err)
		if isTimeout {
			details = description
			message = GATEWAY_TIMEOUT
			errorNo = http.StatusGatewayTimeout
		}
		m := ErrorMessage{
			Attempted:  "client.httpClient.Do(req)",
			Details:    details,
			ErrorNo:    errorNo,
			Exit:       "4e9df8a7b421",
			Function:   function,
			InnerError: err,
			Message:    message,
			RequestUrl: link,
		}
		return t, m
	}
	if response == nil {
		errorNo := http.StatusInternalServerError
		m := ErrorMessage{
			Attempted:  "client.httpClient.Do(req)",
			ErrorNo:    errorNo,
			Exit:       "a4509ba97f1b",
			Function:   function,
			InnerError: nil,
			Message:    UNEXPECTED_ERROR,
			RequestUrl: link,
		}
		return t, m
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		m := ErrorMessage{
			Attempted:  "body, err := io.ReadAll(response.Body)",
			Code:       "",
			Details:    string(body),
			ErrorNo:    response.StatusCode,
			Exit:       "83cc31d60828",
			Function:   function,
			InnerError: err,
			Message:    UNEXPECTED_ERROR,
			RequestUrl: link,
		}
		return t, m
	}
	if response.StatusCode >= http.StatusBadRequest {
		var codeMessage struct {
			CodeMessage struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		m := ErrorMessage{
			Attempted:  "response, err := client.httpClient.Do(req)",
			Code:       codeMessage.CodeMessage.Code,
			Details:    string(body),
			ErrorNo:    response.StatusCode,
			Exit:       "220bc130aa31",
			Function:   function,
			Message:    BAD_REQUEST,
			RequestUrl: link,
		}
		var data map[string]any
		err := json.Unmarshal(body, &data)
		if err != nil {
			m.Body = string(body)
			return t, m
		}
		// FileMaker Error Struct
		if errVal, ok := data["error"]; ok {
			if errMap, ok := errVal.(map[string]any); ok {
				code, _ := errMap["code"].(string)
				message, _ := errMap["message"].(string)
				m.Code = code
				m.Details = message
			}
		}
		m.Payload = data

		return t, m
	}
	if response.StatusCode == http.StatusNoContent {
		return t, nil
	}

	err = json.Unmarshal(body, &t)
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

		err = json.Unmarshal(sanitised, &t)
		if err != nil {
			message := ErrorMessage{
				Attempted:  "err = json.Unmarshal(sanitised, &responseData)",
				Body:       string(sanitised),
				Details:    fmt.Sprintf(`%+v`, err),
				ErrorNo:    http.StatusInternalServerError,
				Exit:       "338a01774b7c",
				Function:   "odataClient.executeHttpRequest",
				InnerError: err,
				Message:    UNEXPECTED_ERROR,
				Payload:    string(sanitised),
			}
			return t, message
		}
	}

	return t, nil
}

func MapTo[T any](m map[string]any) (T, error) {
	var out T
	b, err := json.Marshal(m)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return out, err
	}
	return out, nil
}

func (r *apiSingleResponse[T]) UnmarshalJSON(b []byte) error {
	var v T
	if err := json.Unmarshal(b, &v); err == nil {
		r.Value = v
		return nil
	}

	// Fallback: try as a map, maybe adjust, then re-marshal into T.
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		// nothing we can do
		return err
	}

	// You can tweak m here if needed (strip @odata.*, etc.)

	// Convert map back to T via JSON
	converted, err := MapTo[T](m)
	if err != nil {
		return err
	}

	r.Value = converted
	return nil
}

func executeHttpRequestPayload[T any](client oDataClient, req *http.Request, payload any) (T, error) {

	functionName := `executeHttpRequestPayload`
	link := getFullURL(req)

	client.mapHeadersToRequest(req)
	response, err := client.httpClient.Do(req)
	var responseData T
	if err != nil {
		httpClientDoError := ErrorMessage{
			Attempted:  "client.httpClient.Do(req)",
			ErrorNo:    http.StatusInternalServerError,
			Exit:       "a8d0e3fe6855",
			Function:   functionName,
			InnerError: err,
			Payload:    payload,
			RequestUrl: link,
		}
		return responseData, httpClientDoError
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		message := ErrorMessage{
			Attempted:  "body, err := io.ReadAll(response.Body)",
			ErrorNo:    http.StatusInternalServerError,
			Exit:       "cbe8572fbb5a",
			Function:   "executeHttpRequest",
			InnerError: err,
			Payload:    payload,
			RequestUrl: link,
		}
		return responseData, message
	}
	if response.StatusCode >= http.StatusBadRequest {
		message := ErrorMessage{
			Attempted:  "response, err := client.httpClient.Do(req)",
			ErrorNo:    response.StatusCode,
			Exit:       "0ffdf16f81c7",
			Function:   "executeHttpRequest",
			Message:    UNEXPECTED_ERROR,
			Payload:    payload,
			RequestUrl: link,
		}
		var data map[string]any
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
				Attempted:  "err = json.Unmarshal(sanitised, &responseData)",
				Body:       string(sanitised),
				ErrorNo:    http.StatusInternalServerError,
				Exit:       "e7d9f55a3cec",
				Function:   "odataClient.executeHttpRequest",
				InnerError: err,
				Message:    fmt.Sprintf(`%+v`, err),
				Payload:    payload,
			}
			return responseData, message
		}
	}

	return responseData, nil
}
