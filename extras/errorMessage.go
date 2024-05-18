package dataModel

import "encoding/json"

type ErrorMessage struct {
	Attempted string      `json:"attemped,omitempty"`
	Err       interface{} `json:"err,omitempty"`
	Function  string      `json:"function,omitempty"`
	Message   string      `json:"message"`
}

func (e ErrorMessage) Error() string {
	bytes, err := json.Marshal(e)
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}
