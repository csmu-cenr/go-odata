package dataModel

import (
	"encoding/json"
)

type ErrorMessage struct {
	Attempted     string   `json:"attemped,omitempty"`
	Code          string   `json:"code"`
	Details       any      `json:"details"`
	ErrorNo       int      `json:"errorNo,omitempty"`
	Exit          string   `json:"exit"`
	FileName      string   `json:"filename,omitempty"`
	Function      string   `json:"function,omitempty"`
	InnerError    any      `json:"err,omitempty"`
	IPAddress     string   `json:"ipaddress,omitempty"`
	LineNumber    int      `json:"lineNumber,omitempty"`
	Link          string   `json:"link,omitempty"`
	Message       string   `json:"message,omitempty"`
	Path          string   `json:"path,omitempty"`
	Payload       any      `json:"payload,omitempty"`
	RequestUrl    string   `json:"requestUrl,omitempty"`
	Stack         []string `json:"stack,omitempty"`
	Timestamp     string   `json:"timestamp,omitempty"`
	Timezone      string   `json:"timezone,omitempty"`
	UnixTimestamp int64    `json:"unixTimestamp"`
	User          any      `json:"user,omitempty"`
}

func (e ErrorMessage) Error() string {
	bytes, err := json.Marshal(e)
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}
