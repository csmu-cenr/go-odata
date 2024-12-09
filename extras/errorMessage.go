package dataModel

import "encoding/json"

type ErrorMessage struct {
	Attempted string      `json:"attemped,omitempty"`
	Details   interface{} `json:"details"`
	Err       interface{} `json:"err,omitempty"`
	ErrorNo   int         `json:"errorNo"`
	Function  string      `json:"function,omitempty"`
	Message   string      `json:"message"`
	Payload   interface{} `json:"payload"`
}

func (e ErrorMessage) Error() string {
	bytes, err := json.Marshal(e)
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}
