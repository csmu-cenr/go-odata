package modelGenerator

import (
	"io"
	"net/http"
)

func fetchEdmx(url string) (edmxXmlData, edmxDataServices, error) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return edmxXmlData{}, edmxDataServices{}, err
	}
	request.Header.Set("Accept", "application/atom+xml")
	request.Header.Set("DataServiceVersion", "4.0")
	response, err := client.Do(request)
	if err != nil {
		return edmxXmlData{}, edmxDataServices{}, err
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return edmxXmlData{}, edmxDataServices{}, err
	}
	return parseEdmx(body)
}
