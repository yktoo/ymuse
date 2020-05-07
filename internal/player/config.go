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

package player

import (
	"flag"
	"fmt"
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
const AppLicense = `Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
`

type Config struct {
	LogLevel               logging.Level // The logging level
	MpdHost                string        // MPD's IP address or hostname
	MpdPort                int           // MPD's port number
	MpdPassword            string        // MPD's password (optional)
	QueueColumnIds         []int         // Displayed queue columns
	DefaultSortAttrId      int           // ID of MPD attribute used as a default for queue sorting
	TrackDefaultReplace    bool          // Whether the default action for double-clicking a track is replace rather than append
	PlaylistDefaultReplace bool          // Whether the default action for double-clicking a playlist is replace rather than append
	PlayerTitleTemplate    string        // Track's title formatting template for the player
}

// Config singleton with all the defaults
var config = &Config{
	LogLevel:               logging.WARNING,
	MpdHost:                "",
	MpdPort:                6600,
	MpdPassword:            "",
	QueueColumnIds:         []int{MTA_Artist, MTA_Year, MTA_Album, MTA_Disc, MTA_Number, MTA_Track, MTA_Length, MTA_Genre},
	DefaultSortAttrId:      MTA_Path,
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

// MpdAddress() returns the MPD address string constructed from host and port
func (c *Config) MpdAddress() string {
	return fmt.Sprintf("%s:%d", c.MpdHost, c.MpdPort)
}
