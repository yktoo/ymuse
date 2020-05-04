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
	"flag"
	"github.com/op/go-logging"
	"os"
	"strings"
	"sync"
)

const AppVersion = "0.01"
const AppName = "Ymuse"
const AppWebsite = "https://yktoo.com"
const AppWebsiteLabel = "yktoo.com"
const AppID = "com.yktoo.ymuse"
const AppLicense = `This program is free software: you can redistribute it and/or modify it
under the terms of the GNU General Public License version 3, as published
by the Free Software Foundation.

This program is distributed in the hope that it will be useful, but
WITHOUT ANY WARRANTY; without even the implied warranties of
MERCHANTABILITY, SATISFACTORY QUALITY, or FITNESS FOR A PARTICULAR
PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along
with this program. If not, see http://www.gnu.org/licenses/`

type Config struct {
	LogLevel               logging.Level // The logging level
	MpdAddress             string        // MPD's IP address or hostname and port number
	MpdPassword            string        // MPD's password (optional)
	TrackDefaultReplace    bool          // Whether the default action for double-clicking a track is replace rather than append
	PlaylistDefaultReplace bool          // Whether the default action for double-clicking a playlist is replace rather than append
	PlayerTitleTemplate    string        // Track's title formatting template for the player
}

// Config singleton with all the defaults
var config = &Config{
	LogLevel:               logging.WARNING,
	MpdAddress:             ":6600",
	MpdPassword:            "",
	TrackDefaultReplace:    false,
	PlaylistDefaultReplace: true,
	PlayerTitleTemplate: `{{- if or .Title .Album | or .Artist -}}
<big><b>{{ .Title | default "(unknown title)" }}</b></big>
by <b>{{ .Artist | default "(unknown artist)" }}</b> from <b>{{ .Album | default "(unknown album)" }}</b>
{{- else if .Name -}}
<big><b>{{ .Name }}</b></big>
{{- else if .file -}}
File <big><b>{{ .file | basename }}</b></big>
from <b>{{ .file | dirname }}</b>
{{- else -}}
<i>(no track)</i>
{{- end -}}`,
}
var once sync.Once

// GetConfig() returns a global Config instance
func GetConfig() *Config {
	once.Do(func() {
		// Ignore if we're testing
		for _, arg := range os.Args {
			if strings.Contains(arg, "-test.") {
				return
			}
		}

		// Process command line
		verbInfo := flag.Bool("v", false, "verbose logging")
		verbDebug := flag.Bool("vv", false, "more verbose logging")
		flag.Parse()

		// Update Config
		switch {
		case *verbDebug:
			config.LogLevel = logging.DEBUG
		case *verbInfo:
			config.LogLevel = logging.INFO
		}
	})
	return config
}
