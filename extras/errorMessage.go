package dataModel

import (
	"encoding/json"
)

type ErrorMessage struct {
	Attempted  string      `json:"attemped,omitempty"`
	Code       string      `json:"code"`
	Details    interface{} `json:"details"`
	ErrorNo    int         `json:"errorNo,omitempty"`
	Function   string      `json:"function,omitempty"`
	InnerError interface{} `json:"err,omitempty"`
	IPAddress  string      `json:"ipaddress,omitempty"`
	Message    string      `json:"message,omitempty"`
	Payload    interface{} `json:"payload,omitempty"`
	RequestUrl string      `jsono:"requestUrl,omitempty"`
	Stack      []string    `json:"stack,omitempty"`
	Timestamp  string      `json:"timestamp,omitempty"`
	Timezone   string      `json:"timezone,omitempty"`
}

func (e ErrorMessage) Error() string {
	bytes, err := json.Marshal(e)
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}
