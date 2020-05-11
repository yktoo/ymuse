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

package config

import (
	"encoding/json"
	"fmt"
	"github.com/gotk3/gotk3/glib"
	"io/ioutil"
	"os"
	"path"
	"sync"
)

var AppMetadata = &struct {
	Version   string
	BuildDate string
	Name      string
	Icon      string
	Copyright string
	URL       string
	URLLabel  string
	Id        string
	License   string
}{
	Name:      "Ymuse",
	Icon:      "ymuse",
	Copyright: "Written by Dmitry Kann",
	URL:       "https://yktoo.com",
	URLLabel:  "yktoo.com",
	Id:        "com.yktoo.ymuse",
	License: "Licensed under the Apache License, Version 2.0 (the \"License\");\n" +
		"you may not use this file except in compliance with the License.\n" +
		"You may obtain a copy of the License at\n" +
		"    http://www.apache.org/licenses/LICENSE-2.0\n" +
		"\n" +
		"Unless required by applicable law or agreed to in writing, software\n" +
		"distributed under the License is distributed on an \"AS IS\" BASIS,\n" +
		"WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\n" +
		"See the License for the specific language governing permissions and\n" +
		"limitations under the License.\n",
}

type Dimensions struct {
	X, Y, Width, Height int
}

type Config struct {
	MpdHost                string     // MPD's IP address or hostname
	MpdPort                int        // MPD's port number
	MpdPassword            string     // MPD's password (optional)
	MpdAutoConnect         bool       // Whether to automatically connect to MPD on startup
	MpdAutoReconnect       bool       // Whether to automatically reconnect to MPD after connection is lost
	QueueColumnIds         []int      // Displayed queue columns
	DefaultSortAttrId      int        // ID of MPD attribute used as a default for queue sorting
	TrackDefaultReplace    bool       // Whether the default action for double-clicking a track is replace rather than append
	PlaylistDefaultReplace bool       // Whether the default action for double-clicking a playlist is replace rather than append
	PlayerTitleTemplate    string     // Track's title formatting template for the player
	MainWindowDimensions   Dimensions // Main window dimensions
}

// Config singleton with all the defaults
var config = &Config{
	MpdHost:                "",
	MpdPort:                6600,
	MpdPassword:            "",
	MpdAutoConnect:         true,
	MpdAutoReconnect:       true,
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
	MainWindowDimensions: Dimensions{-1, -1, -1, -1},
}
var once sync.Once

// GetConfig() returns a global Config instance
func GetConfig() *Config {
	// Load the config from the file
	once.Do(config.Load)
	return config
}

// Load() reads the config from the default file
func (c *Config) Load() {
	// Try to read the file
	file := c.getConfigFile()
	data, err := ioutil.ReadFile(file)
	if errCheck(err, "Couldn't read file") {
		return
	}

	// Unmarshal the config
	if errCheck(json.Unmarshal(data, &c), "json.Unmarshal() failed") {
		return
	}
	log.Debugf("Loaded configuration from %s", file)
}

// MpdAddress() returns the MPD address string constructed from host and port
func (c *Config) MpdAddress() string {
	return fmt.Sprintf("%s:%d", c.MpdHost, c.MpdPort)
}

// Save() writes out the config to the default file
func (c *Config) Save() {
	// Create the config directory if it doesn't exist
	if errCheck(os.MkdirAll(c.getConfigDir(), 0755), "MkdirAll() failed") {
		return
	}

	// Serialise the config
	data, err := json.MarshalIndent(c, "", "    ")
	if errCheck(err, "json.MarshalIndent() failed") {
		return
	}

	// Save the config
	file := c.getConfigFile()
	if !errCheck(ioutil.WriteFile(file, data, 0600), "WriteFile() failed") {
		log.Debugf("Saved configuration to %s", file)
	}
}

// getConfigDir() returns the full path to the config directory
func (c *Config) getConfigDir() string {
	return path.Join(glib.GetUserConfigDir(), "ymuse")
}

// getConfigFile() returns the full path of the config file
func (c *Config) getConfigFile() string {
	return path.Join(c.getConfigDir(), "config.json")
}
