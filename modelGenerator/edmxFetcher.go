package modelGenerator

import (
	"io"
	"net/http"
	"os"
	"strings"
)

func fetchEdmx(source string) (edmxDataServices, error) {
	var body []byte
	var err error

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		client := &http.Client{}
		request, err := http.NewRequest("GET", source, nil)
		if err != nil {
			return edmxDataServices{}, err
		}
		request.Header.Set("Accept", "application/atom+xml")
		request.Header.Set("DataServiceVersion", "4.0")
		response, err := client.Do(request)
		if err != nil {
			return edmxDataServices{}, err
		}
		defer func() { _ = response.Body.Close() }()
		body, err = io.ReadAll(response.Body)
		if err != nil {
			return edmxDataServices{}, err
		}
	} else {
		body, err = os.ReadFile(source)
		if err != nil {
			return edmxDataServices{}, err
		}
	}

	return parseEdmx(body)
}
