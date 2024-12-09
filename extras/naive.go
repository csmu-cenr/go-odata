package dataModel

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// formatDate returns a time as `yyyy-mm-dd` string
func formatDate(t time.Time) string {
	// Use a custom formatting string to include milliseconds
	return t.Format(time.DateOnly)
}

// formatDateTimeWithSeconds returns a time as `yyyymm-dd hh:mm:ss` string
func formatDateTimeWithSeconds(t time.Time) string {
	return t.Format(time.DateTime)
}

type NaiveDate time.Time

func (nt NaiveDate) MarshalJSON() ([]byte, error) {
	return json.Marshal(formatDate(time.Time(nt)))
}

func (ts *NaiveDate) UnmarshalJSON(data []byte) error {

	var timeString string
	if err := json.Unmarshal(data, &timeString); err != nil {
		return err
	}

	if timeString == "" {
		timeString = "0001-01-01"
	}

	var dateFormat string
	if strings.Contains(timeString, "/") {
		dateFormat = "01/02/2006"
	} else {
		dateFormat = "2006-01-02"
	}

	parsedTime, err := time.Parse(dateFormat, timeString)
	if err != nil {
		message := fmt.Errorf("NaiveDate: %+v", err)
		return message
	}

	*ts = NaiveDate(parsedTime)
	return nil
}

// Method to add seconds to NaiveTime
func (nt NaiveTime) AddSeconds(seconds float64) NaiveTime {
	t := time.Time(nt)
	duration := time.Duration(int(seconds)) * time.Second
	return NaiveTime(t.Add(duration))
}

func (nt NaiveTime) ToDate() NaiveDate {
	t := time.Time(nt)
	return NaiveDate(t)
}

// ToTime returns hours and minutes as a duration
func (nt NaiveTime) ToTime() NaiveDuration {
	t := time.Time(nt)
	h := time.Duration(t.Hour()) * time.Hour
	m := time.Duration(t.Minute()) * time.Minute
	r := h + m
	return NaiveDuration(r)
}

func NaiveTimesDoEqualDateHoursMinutes(left, right NaiveTime) bool {
	leftText := time.Time(left).Format(NAIVE_TIMESTAMP_YYYY_MM_DD_HH_MM)
	rightText := time.Time(right).Format(NAIVE_TIMESTAMP_YYYY_MM_DD_HH_MM)
	return strings.EqualFold(leftText, rightText)
}

func NaiveTimesDoNotEqualDateHoursMinutes(left, right NaiveTime) bool {
	return !NaiveTimesDoEqualDateHoursMinutes(left, right)
}

type NaiveDuration time.Duration

func NaiveDurationFromSeconds(seconds int) NaiveDuration {
	duration := time.Duration(seconds) * time.Second
	return NaiveDuration(duration)
}

// NaiveDuration MarshalJSON returns a duration in the form h:m:s.ms where leading zeros are used where necessary
func (nd NaiveDuration) MarshalJSON() ([]byte, error) {

	duration := time.Duration(nd)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60
	milliseconds := int(duration.Milliseconds()) % 1000

	// Use format specifiers to add leading zeros where required
	var text string

	if hours > 0 {
		text += fmt.Sprintf("%d:", hours)
	}

	if len(text) == 0 {
		text += fmt.Sprintf("%d:", minutes)
	} else {
		text += fmt.Sprintf("%02d:", minutes)
	}

	if len(text) == 0 {
		text += fmt.Sprintf("%d", seconds)
	} else {
		text += fmt.Sprintf("%02d", seconds)
	}

	if milliseconds > 0 {
		text += fmt.Sprintf(".%03d", milliseconds)
	}

	return json.Marshal(text)
}

func (o *NaiveDuration) UnmarshalJSON(data []byte) error {

	if data == nil {
		*o = NaiveDuration(0)
		return nil
	}

	if len(data) == 0 {
		*o = NaiveDuration(0)
		return nil
	}

	var durationString string
	if err := json.Unmarshal(data, &durationString); err != nil {
		return err
	}

	if durationString == "null" {
		*o = NaiveDuration(0)
		return nil
	}

	elements := strings.Split(durationString, ".")
	hhmmss := strings.Split(elements[0], ":")
	hh := 0
	mm := 0
	ss := 0
	resultString := ""
	switch len(hhmmss) {
	case 3:
		hh, _ = strconv.Atoi(hhmmss[0])
		mm, _ = strconv.Atoi(hhmmss[1])
		ss, _ = strconv.Atoi(hhmmss[2])
	case 2:
		mm, _ = strconv.Atoi(hhmmss[0])
		ss, _ = strconv.Atoi(hhmmss[1])
	case 1:
		ss, _ = strconv.Atoi(hhmmss[0])
	default:
	}
	resultString = fmt.Sprintf("%dh%dm%ds", hh, mm, ss)
	duration, err := time.ParseDuration(resultString)
	if err != nil {
		return err
	}
	*o = NaiveDuration(duration)

	return nil
}

type NaiveTime time.Time

// This drops the timezone information.
func (o NaiveTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(formatDateTimeWithSeconds(time.Time(o)))
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

	if timeString == "" {
		timeString = "0001-01-01 00:00:00"
	}

	var dateTimeFormat string
	switch {
	case strings.Contains(timeString, "/"):
		dateTimeFormat = "01/02/2006 15:04:05"
	case strings.Contains(timeString, "T"):
		dateTimeFormat = "2006-01-02T15:04:05-07:00"
	default:
		dateTimeFormat = "2006-01-02 15:04:05"
	}

	parsedTime, err := time.Parse(dateTimeFormat, timeString)
	if err != nil {
		message := fmt.Errorf("%+v", err)
		return message
	}

	*nt = NaiveTime(parsedTime)
	return nil
}

func (n NaiveTime) UnixTimestamp() float64 {
	return float64(time.Time(n).Unix())
}

func (n NaiveTime) UtcOffset(timezone string) (int, error) {

	function := `NaiveTime.UtcOffset`
	l, err := time.LoadLocation(timezone)
	if err != nil {
		m := ErrorMessage{
			Attempted: `time.LoadLocation(timezone)`,
			Details:   fmt.Sprintf(`Error loading location: %+v`, err),
			Function:  function,
			Message:   `bad request`,
			Payload:   timezone,
		}
		fmt.Println("Error loading location:", err)
		return 0, m
	}

	t := time.Time(n)

	localised := time.Date(t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), l)

	_, offset := localised.Zone()

	return offset, nil
}
