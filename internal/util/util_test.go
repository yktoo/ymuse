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
	"github.com/fhs/gompd/v2/mpd"
	"math"
	"reflect"
	"testing"
)

func TestAtoiDef(t *testing.T) {
	type args struct {
		s   string
		def int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"empty string", args{"", 777}, 777},
		{"positive numeric string", args{"42", -1}, 42},
		{"negative numeric string", args{"-120", 0}, -120},
		{"non-numeric string", args{"Zook", 16}, 16},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AtoiDef(tt.args.s, tt.args.def); got != tt.want {
				t.Errorf("AtoiDef() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatSeconds(t *testing.T) {
	type args struct {
		seconds float64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"zero seconds", args{0}, "0:00"},
		{"some seconds", args{42}, "0:42"},
		{"fractional seconds", args{4.2234514}, "0:04"},
		{"minute with seconds", args{218}, "3:38"},
		{"many minutes", args{2722.7}, "45:22"},
		{"an hour with minutes", args{3600 + 3*60 + 15}, "1:03:15"},
		{"almost a day", args{23*3600 + 59*60 + 59}, "23:59:59"},
		{"one day", args{1*24*3600 + 1*3600 + 8*60 + 47}, "one day 1:08:47"},
		{"many days", args{66*24*3600 + 15*3600 + 12*60 + 33}, "66 days 15:12:33"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatSeconds(tt.args.seconds); got != tt.want {
				t.Errorf("FormatSeconds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatSecondsStr(t *testing.T) {
	type args struct {
		seconds string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"empty value", args{""}, ""},
		{"invalid value", args{"boo"}, ""},
		{"zero seconds", args{"0"}, "0:00"},
		{"some seconds", args{"42"}, "0:42"},
		{"fractional seconds", args{"4.2234514"}, "0:04"},
		{"minute with seconds", args{"218"}, "3:38"},
		{"many minutes", args{"2722.7"}, "45:22"},
		{"an hour with minutes", args{"3795"}, "1:03:15"},
		{"almost a day", args{"86399"}, "23:59:59"},
		{"one day", args{"90527"}, "one day 1:08:47"},
		{"many days", args{"5757153"}, "66 days 15:12:33"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatSecondsStr(tt.args.seconds); got != tt.want {
				t.Errorf("FormatSecondsStr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseFloatDef(t *testing.T) {
	type args struct {
		s   string
		def float64
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{"empty string", args{"", 777.14}, 777.14},
		{"positive numeric string", args{"42.52", -1.234}, 42.52},
		{"negative numeric string", args{"-120.0001", 0}, -120.0001},
		{"non-numeric string", args{"Zook", 16.8899}, 16.8899},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compare the numbers with 1/1e6 tolerance to ignore rounding errors
			if got := ParseFloatDef(tt.args.s, tt.args.def); math.Abs(got-tt.want) > 0.000001 {
				t.Errorf("ParseFloatDef() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefault(t *testing.T) {
	type args struct {
		def   string
		value interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"nil is no value", args{"Foo", nil}, "Foo"},
		{"empty string is no value", args{"Foo", ""}, "Foo"},
		{"non-empty string is value", args{"Foo", "barr"}, "barr"},
		{"false is no value", args{"Foo", false}, "Foo"},
		{"true is value", args{"Foo", true}, "true"},
		{"struct is value", args{"Foo", struct{}{}}, "{}"},
		{"int 0 is no value", args{"Foo", 0}, "Foo"},
		{"positive int is value", args{"Foo", 14}, "14"},
		{"negative int is value", args{"Foo", -2}, "-2"},
		{"float 0 is no value", args{"Foo", 0.0}, "Foo"},
		{"positive float is value", args{"Foo", 14.0}, "14"},
		{"negative float is value", args{"Foo", -2.3}, "-2.3"},
		{"complex is value", args{"Foo", 3 + 2i}, "(3+2i)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Default(tt.args.def, tt.args.value); got != tt.want {
				t.Errorf("Default() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsStreamURI(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want bool
	}{
		{"empty is no stream", "", false},
		{"Name is no stream", "Name", false},
		{"http: is no stream", "http:", false},
		{"https: is no stream", "https:", false},
		{"http:// not at begin is no stream", "[http://whatev.er]", false},
		{"http-URL is a stream", "http://example.com", true},
		{"https-URL is a stream", "https://www.musicpd.org/", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStreamURI(tt.uri); got != tt.want {
				t.Errorf("IsStreamURI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapAttrsToSlice(t *testing.T) {
	type args struct {
		attrs []mpd.Attrs
		attr  string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"empty list", args{[]mpd.Attrs{}, "file"}, []string{}},
		{"single element", args{[]mpd.Attrs{{"file": "foo"}}, "file"}, []string{"foo"}},
		{"missing attribute", args{[]mpd.Attrs{{"whoppa": "hippa"}}, "file"}, []string{""}},
		{
			"multiple elements",
			args{
				[]mpd.Attrs{
					{"artist": "bar", "album": "baz", "file": "foo"},
					{"artist": "X-None", "file": "whoopsie"},
					{"artist": "Blase", "file": "snap"},
					{"album": "There"},
				},
				"file"},
			[]string{"foo", "whoopsie", "snap", ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MapAttrsToSlice(tt.args.attrs, tt.args.attr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapAttrsToSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}
