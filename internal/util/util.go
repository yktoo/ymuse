package util

import "fmt"

func FormatSeconds(seconds float64) string {
	mins, secs := int(seconds)/60, int(seconds)%60
	if mins >= 60 {
		return fmt.Sprintf("%d:%02d:%02d", mins/60, mins, secs)
	}
	return fmt.Sprintf("%d:%02d", mins, secs)
}
