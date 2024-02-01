package dataModel

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type NaiveDuration time.Duration

// NaiveDuration MarshalJSON returns a duration in the form h:m:s.ms where leading zeros are used where necessary
func (nd NaiveDuration) MarshalJSON() ([]byte, error) {
	totalNanoSeconds := time.Duration(nd)
	totalSeconds := totalNanoSeconds / time.Second

	hours := totalSeconds / 3600
	minutes := (totalSeconds - hours*3600) / 60
	seconds := (totalSeconds - hours*3600 - minutes*60)
	milliseconds := totalNanoSeconds.Milliseconds()

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
