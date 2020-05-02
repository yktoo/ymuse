package util

import (
	"fmt"
	"strconv"
)

// AtoiDef() converts a string into an int, returning the given default value if conversion failed
func AtoiDef(s string, def int) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return def
}

// ParseFloatDef() converts a string into a float64, returning the given default value if conversion failed
func ParseFloatDef(s string, def float64) float64 {
	if f, err := strconv.ParseFloat(s, 32); err == nil {
		return f
	}
	return def
}

// FormatSeconds() formats a number seconds as a string
func FormatSeconds(seconds float64) string {
	minutes, secs := int(seconds)/60, int(seconds)%60
	hours, mins := minutes/60, minutes%60
	days, hrs := hours/24, hours%24
	switch {
	case days > 1:
		return fmt.Sprintf("%d days %d:%02d:%02d", days, hrs, mins, secs)
	case days == 1:
		return fmt.Sprintf("One day %d:%02d:%02d", hrs, mins, secs)
	case hours >= 1:
		return fmt.Sprintf("%d:%02d:%02d", hrs, mins, secs)
	default:
		return fmt.Sprintf("%d:%02d", mins, secs)
	}
}
