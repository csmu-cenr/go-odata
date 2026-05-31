package dataModel

import "encoding/json"

type RecordIsOutOfDate struct {
	Model  string
	Uuid   string
	Id     float64
	Record any
}

func (r RecordIsOutOfDate) Error() string {
	data, err := json.Marshal(r)
	if err != nil {
		return err.Error()
	}
	return string(data)
}
