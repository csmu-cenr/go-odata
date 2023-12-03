package odataClient

type Result struct {
	Context  string `json:"@odata.context"`
	Count    *int   `json:"@odata.count,omitempty"`
	Model    string `json:"model,omitempty"`
	Value    []any  `json:"value"`
	NextLink string `json:"@odata.nextLink,omitempty"`
}
