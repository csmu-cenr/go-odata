package dataModel

import (
	"encoding/json"
	"strconv"
	"time"
)

func formatDateTimeWithSeconds(t time.Time) string {
	// Use a custom formatting string to include milliseconds
	formattingString := "2006-01-02 15:04:05"
	return t.Format(formattingString)
}

func formatTimeWithMilliseconds(t time.Time) string {
	// Use a custom formatting string to include milliseconds
	formattingString := "2006-01-02 15:04:05.000"
	return t.Format(formattingString)
}

type NaiveTime time.Time

// This drops the timezone information.
func (nt NaiveTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(formatDateTimeWithSeconds(time.Time(nt)))
}

// Rename to MarshalJSON to keep timezone information
func (nt NaiveTime) MarshalJSONRFC3399() ([]byte, error) {
	timeValue := time.Time(nt)
	timeString := timeValue.Format(time.RFC3339)
	return json.Marshal(timeString)
}

func (nt *NaiveTime) UnmarshalJSON(data []byte) error {
	var timeString string
	if err := json.Unmarshal(data, &timeString); err != nil {
		return err
	}

	parsedTime, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		parsedTime, err = time.Parse("2006-01-02 15:04:05", timeString)
		if err != nil {
			i, err := strconv.ParseInt(timeString, 10, 64)
			if err != nil {
				return err
			}
			parsedTime = time.Unix(i, 0)
		}
	}

	*nt = NaiveTime(parsedTime)
	return nil
}
