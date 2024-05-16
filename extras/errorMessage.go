package dataModel

import "encoding/json"

type ErrorMessage struct {
	Message   string      `json:"message"`
	Function  string      `json:"function,omitempty"`
	Attempted string      `json:"attemped,omitempty"`
	Details   interface{} `json:"details,omitempty"`
}

func (e ErrorMessage) Error() string {
	bytes, err := json.Marshal(e)
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}
