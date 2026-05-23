package modelgeneratoroutput

import "strings"

type OutputMode string

const (
	OutputModeSingle  OutputMode = "single"
	OutputModeOmnibus OutputMode = "omnibus"
)

// DefaultOutputMode returns the default mode.
func DefaultOutputMode() OutputMode {
	return OutputModeSingle
}

// ParseOutputMode converts a string into an OutputMode.
// Unknown or empty values fall back to the default.
func ParseOutputMode(s string) OutputMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case string(OutputModeOmnibus):
		return OutputModeOmnibus

	case string(OutputModeSingle):
		return OutputModeSingle

	case "":
		return DefaultOutputMode()

	default:
		return DefaultOutputMode()
	}
}

// IsSingle returns true if the mode is single.
// Empty values are treated as the default.
func (m OutputMode) IsSingle() bool {
	if m == "" {
		return true
	}
	return m == OutputModeSingle
}

// IsOmnibus returns true if the mode is omnibus.
func (m OutputMode) IsOmnibus() bool {
	return m == OutputModeOmnibus
}

// String returns the string representation.
func (m OutputMode) String() string {
	if m == "" {
		return string(DefaultOutputMode())
	}
	return string(m)
}

// Valid reports whether the mode is recognised.
func (m OutputMode) Valid() bool {
	switch m {
	case OutputModeSingle, OutputModeOmnibus:
		return true
	default:
		return false
	}
}
