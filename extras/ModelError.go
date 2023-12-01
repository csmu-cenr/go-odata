package dataModel

import (
	"fmt"
)

type ModelError struct {
	Status     string      `json:"status"`
	StatusCode int         `json:"statusCode"`
	Details    interface{} `json:"details"`
}

func (e *ModelError) Error() string {
	return fmt.Sprintf("Status: %s StatusCode: %d Details: %+v", e.Status, e.StatusCode, e.Details)
}
