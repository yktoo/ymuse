/*
 *   Copyright 2020 Dmitry Kann
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"fmt"
	"github.com/gotk3/gotk3/glib"
	"html/template"
	"strconv"
	"strings"
	"sync"
)

var (
	locDay  string
	locDays string
	locOnce sync.Once
)

// AtoiDef converts a string into an int, returning the given default value if conversion failed
func AtoiDef(s string, def int) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return def
}

// ParseFloatDef converts a string into a float64, returning the given default value if conversion failed
func ParseFloatDef(s string, def float64) float64 {
	if f, err := strconv.ParseFloat(s, 32); err == nil {
		return f
	}
	return def
}

// FormatSeconds formats a number seconds as a string
func FormatSeconds(seconds float64) string {
	// Make sure localised strings are fetched
	locOnce.Do(func() {
		locDay = glib.Local("one day")
		locDays = glib.Local("days")
	})

	minutes, secs := int(seconds)/60, int(seconds)%60
	hours, mins := minutes/60, minutes%60
	days, hrs := hours/24, hours%24
	switch {
	case days > 1:
		return fmt.Sprintf("%d %s %d:%02d:%02d", days, locDays, hrs, mins, secs)
	case days == 1:
		return fmt.Sprintf("%s %d:%02d:%02d", locDay, hrs, mins, secs)
	case hours >= 1:
		return fmt.Sprintf("%d:%02d:%02d", hrs, mins, secs)
	default:
		return fmt.Sprintf("%d:%02d", mins, secs)
	}
}

// FormatSecondsStr formats a number seconds as a string given string input
func FormatSecondsStr(seconds string) string {
	if f := ParseFloatDef(seconds, -1); f >= 0 {
		return FormatSeconds(f)
	}
	return ""
}

// Default returns a default value if no value is set
func Default(def string, value interface{}) string {
	if set, ok := template.IsTrue(value); ok && set {
		return fmt.Sprint(value)
	}
	return def
}

// IsStreamURI returns whether the given URI refers to an Internet stream
func IsStreamURI(uri string) bool {
	return strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://")
}
