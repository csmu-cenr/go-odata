package dataModel

import (
	"encoding/json"
	"fmt"
	"net/http"
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

func (nd NaiveDate) AddDays(days int) NaiveDate {
	if time.Time(nd).IsZero() {
		return nd
	}
	t := time.Time(nd)
	return NaiveDate(t.AddDate(0, 0, days))
}

func (nd NaiveDate) SubstractDays(days int) NaiveDate {
	if time.Time(nd).IsZero() {
		return nd
	}
	return nd.AddDays(days * -1)
}

// GreaterThan returns true if left is after right
func (nd NaiveDate) After(right NaiveDate) bool {
	l := time.Time(nd)
	r := time.Time(right)
	if r.Year() > l.Year() {
		return true
	}
	if r.Month() > l.Month() {
		return true
	}
	if r.Day() > l.Day() {
		return true
	}
	return false
}

func (nd NaiveDate) TimeAtLocation(tz string) (time.Time, error) {

	function := `NaiveDate) TimeAtLocation`

	loc, err := time.LoadLocation(tz)
	if err != nil {
		status := http.StatusBadRequest
		m := ErrorMessage{
			Details:  fmt.Sprintf(`invalid timezone %+s %+v`, tz, err),
			ErrorNo:  http.StatusInternalServerError,
			Exit:     "5559e9d2cd2f",
			Function: function,
			Message:  http.StatusText(status),
		}
		return time.Time{}, m
	}
	t := time.Time(nd)

	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc), nil
}

// Before returns true if left is before right
func (nd NaiveDate) Before(right NaiveDate) bool {
	l := time.Time(nd)
	r := time.Time(right)
	if r.Year() < l.Year() {
		return true
	}
	if r.Month() < l.Month() {
		return true
	}
	if r.Day() < l.Day() {
		return true
	}
	return false
}

// Equals returns true if left and right are the same time
func (nd NaiveDate) Equals(right NaiveDate) bool {
	l := time.Time(nd)
	r := time.Time(right)
	if r.Year() == l.Year() {
		if r.Month() == l.Month() {
			if r.Day() == l.Day() {
				return true
			}
		}
	}
	return false
}

// Day returns the day of the month specified by nd.
func (nd NaiveDate) Day() int {
	date := time.Time(nd)
	return date.Day()
}

func (nd *NaiveDate) Parse(timeString string) error {

	if timeString == "" {
		timeString = "0001-01-01"
	}

	if len(timeString) > len(`2006-01-02`) {
		timeString = timeString[0:10]
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

	*nd = NaiveDate(parsedTime)
	return nil
}

func (nd NaiveDate) MarshalJSON() ([]byte, error) {
	return json.Marshal(formatDate(time.Time(nd)))
}

func (nd *NaiveDate) UnmarshalJSON(data []byte) error {

	var timeString string
	if err := json.Unmarshal(data, &timeString); err != nil {
		return err
	}

	err := nd.Parse(timeString)
	return err
}

// Weekday returns the day of the week specified by t.
func (nd NaiveDate) Weekday(weekStart int) int {
	date := time.Time(nd)
	result := int(date.Weekday()) + weekStart
	return result
}

// YearDay returns the day of the year specified by nd, in the range [1,365] for non-leap years,
// and [1,366] in leap years.
func (nd NaiveDate) YearDay() int {
	date := time.Time(nd)
	return date.YearDay()
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
	layout := `2006-01-02 15:04`
	leftText := time.Time(left).Format(layout)
	rightText := time.Time(right).Format(layout)
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

func (nd NaiveDuration) Hours() int {
	duration := time.Duration(nd)
	hours := int(duration.Hours())
	return hours
}

func (nd NaiveDuration) Minutes() int {
	duration := time.Duration(nd)
	minutes := int(duration.Minutes()) % 60
	return minutes
}

func (nd NaiveDuration) Seconds() int {
	duration := time.Duration(nd)
	seconds := int(duration.Seconds()) % 60
	return seconds
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

func (nd *NaiveDuration) UnmarshalJSON(data []byte) error {

	if data == nil {
		*nd = NaiveDuration(0)
		return nil
	}

	if len(data) == 0 {
		*nd = NaiveDuration(0)
		return nil
	}

	var durationString string
	if err := json.Unmarshal(data, &durationString); err != nil {
		return err
	}

	if durationString == "null" {
		*nd = NaiveDuration(0)
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
	*nd = NaiveDuration(duration)

	return nil
}

type NaiveTime time.Time

func (nt NaiveTime) Time() time.Time {
	return time.Time(nt)
}

func (nt *NaiveTime) Parse(timeString string) error {

	if timeString == "" {
		timeString = "0001-01-01 00:00:00"
	}

	var dateTimeFormat string
	switch {
	case strings.Contains(timeString, "T") && strings.Contains(timeString, "Z"):
		dateTimeFormat = "2006-01-02T15:04:05Z"
	case strings.Contains(timeString, "T") && len(timeString) == len("2006-01-02T15:04:05-07:00"):
		dateTimeFormat = "2006-01-02T15:04:05-07:00"
	case strings.Contains(timeString, "T") && len(timeString) == len("2006-01-02T15:04:05Z"):
		dateTimeFormat = "2006-01-02T15:04:05Z"
	case strings.Contains(timeString, "T") && len(timeString) == len("2006-01-02T15:04:05"):
		dateTimeFormat = "2006-01-02T15:04:05"
	case strings.Contains(timeString, "T") && len(timeString) == len("2006-01-02T15:04"):
		dateTimeFormat = "2006-01-02T15:04"
	case strings.Contains(timeString, "T") && len(timeString) == len("2006-01-02T15"):
		dateTimeFormat = "2006-01-02T15"
	case strings.Contains(timeString, "/"):
		dateTimeFormat = "01/02/2006 15:04:05"
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
	return nt.Parse(timeString)
}

func (nt NaiveTime) UnixTimestamp() float64 {
	return float64(time.Time(nt).Unix())
}

func (nt NaiveTime) UtcOffset(timezone string) (int, error) {

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

	t := time.Time(nt)

	localised := time.Date(t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), l)

	_, offset := localised.Zone()

	return offset, nil
}

func (nt NaiveTime) Hour() int {
	t := time.Time(nt)
	return t.Hour()
}

func (nt NaiveTime) Minute() int {
	t := time.Time(nt)
	return t.Minute()
}

func (nt NaiveTime) Second() int {
	t := time.Time(nt)
	return t.Second()
}

func (nt NaiveTime) ISO8601() string {
	return formatDateTimeWithSeconds(time.Time(nt))
}
