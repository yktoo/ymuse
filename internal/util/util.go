package util

import (
	"fmt"
	"strconv"
)

// FormatSeconds formats a number seconds as a string
func FormatSeconds(seconds float64) string {
	mins, secs := int(seconds)/60, int(seconds)%60
	if mins >= 60 {
		return fmt.Sprintf("%d:%02d:%02d", mins/60, mins, secs)
	}
	return fmt.Sprintf("%d:%02d", mins, secs)
}

// FormatSecondsStr formats a number seconds expressed as a string, into a string.
// If seconds is unparseable, returns an empty string
func FormatSecondsStr(seconds string) string {
	if f, err := strconv.ParseFloat(seconds, 32); err == nil {
		return FormatSeconds(f)
	}
	return ""
}
