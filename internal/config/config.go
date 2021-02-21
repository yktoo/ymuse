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
	"github.com/yktoo/ymuse/internal/util"
	"io/ioutil"
	"os"
	"path"
	"sync"
)

// AppMetadata stores application-wide metadata such as version, license etc.
var AppMetadata = &struct {
	Version   string
	BuildDate string
	Name      string
	Icon      string
	Copyright string
	URL       string
	URLLabel  string
	ID        string
	License   string
}{
	Name:      "Ymuse",
	Icon:      "ymuse",
	Copyright: "Written by Dmitry Kann",
	URL:       "https://yktoo.com",
	URLLabel:  "yktoo.com",
	ID:        "com.yktoo.ymuse",
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

// Dimensions represents window dimensions
type Dimensions struct {
	X, Y, Width, Height int
}

// ColumnSpec describes settings for a queue column
type ColumnSpec struct {
	ID    int // Column ID
	Width int // Column width, if differs from the default, otherwise 0
}

// StreamSpec describes settings for an Internet stream
type StreamSpec struct {
	Name string // Stream name
	URI  string // Stream URI
}

// Config represents (storable) application configuration
type Config struct {
	MpdNetwork             string       // Network to use to connect to MPD, either 'tcp' or 'unix'
	MpdSocketPath          string       // Path to the MPD's Unix socket (only if MpdNetwork == 'unix')
	MpdHost                string       // MPD's IP address or hostname (only if MpdNetwork == 'tcp')
	MpdPort                int          // MPD's port number (only if MpdNetwork == 'tcp')
	MpdPassword            string       // MPD's password (optional)
	MpdAutoConnect         bool         // Whether to automatically connect to MPD on startup
	MpdAutoReconnect       bool         // Whether to automatically reconnect to MPD after connection is lost
	QueueColumns           []ColumnSpec // Displayed queue columns
	QueueToolbar           bool         // Whether the queue toolbar is visible
	DefaultSortAttrID      int          // ID of MPD attribute used as a default for queue sorting
	TrackDefaultReplace    bool         // Whether the default action for double-clicking a track is replace rather than append
	PlaylistDefaultReplace bool         // Whether the default action for double-clicking a playlist is replace rather than append
	StreamDefaultReplace   bool         // Whether the default action for double-clicking a stream is replace rather than append
	PlayerTitleTemplate    string       // Track's title formatting template for the player
	PlayerAlbumArtTracks   bool         // Whether to display the current track's album art in the player
	PlayerAlbumArtStreams  bool         // Whether to display the current stream's album art in the player
	MaxSearchResults       int          // Maximum number of displayed search results
	Streams                []StreamSpec // Registered stream specifications
	LibraryPath            string       // Last selected library path

	MainWindowDimensions Dimensions // Main window dimensions
}

// Config singleton with all the defaults
var config *Config
var once sync.Once

// GetConfig returns a global Config instance
func GetConfig() *Config {
	// Load the config from the file
	once.Do(func() {
		// Instantiate a config
		config = newConfig()
		// Load the config from the default file, if any
		config.Load()
	})
	return config
}

// newConfig initialises and returns a config instance with all the defaults
func newConfig() *Config {
	return &Config{
		MpdNetwork:       "tcp",
		MpdSocketPath:    os.Getenv("XDG_RUNTIME_DIR") + "/mpd/socket",
		MpdHost:          os.Getenv("MPD_HOST"),
		MpdPort:          util.AtoiDef(os.Getenv("MPD_PORT"), 6600),
		MpdPassword:      "",
		MpdAutoConnect:   true,
		MpdAutoReconnect: true,
		QueueColumns: []ColumnSpec{
			{ID: MTAttrArtist},
			{ID: MTAttrYear},
			{ID: MTAttrAlbum},
			{ID: MTAttrDisc},
			{ID: MTAttrNumber},
			{ID: MTAttrTrack},
			{ID: MTAttrLength},
			{ID: MTAttrGenre},
		},
		QueueToolbar:           true,
		DefaultSortAttrID:      MTAttrPath,
		TrackDefaultReplace:    false,
		PlaylistDefaultReplace: true,
		StreamDefaultReplace:   true,
		PlayerTitleTemplate: glib.Local(
			"{{- if or .Title .Album | or .Artist -}}\n" +
				"<big><b>{{ .Title | default \"(unknown title)\" }}</b></big>\n" +
				"by <b>{{ .Artist | default \"(unknown artist)\" }}</b> from <b>{{ .Album | default \"(unknown album)\" }}</b>\n" +
				"{{- else if .Name -}}\n" +
				"<big><b>{{ .Name }}</b></big>\n" +
				"{{- else if .file -}}\n" +
				"File <big><b>{{ .file | basename }}</b></big>\n" +
				"from <b>{{ .file | dirname }}</b>\n" +
				"{{- else -}}\n" +
				"<i>(no track)</i>\n" +
				"{{- end -}}\n"),
		PlayerAlbumArtTracks:  true,
		PlayerAlbumArtStreams: false,
		MaxSearchResults:      500,
		Streams: []StreamSpec{
			{Name: "BBC World News", URI: "http://bbcwssc.ic.llnwd.net/stream/bbcwssc_mp1_ws-einws"},
		},
		MainWindowDimensions: Dimensions{-1, -1, -1, -1},
	}
}

// Load reads the config from the default file
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

// MpdNetworkAddress returns the MPD network and the address string
func (c *Config) MpdNetworkAddress() (string, string) {
	if c.MpdNetwork == "unix" {
		return "unix", c.MpdSocketPath
	}
	return "tcp", fmt.Sprintf("%s:%d", c.MpdHost, c.MpdPort)
}

// Save writes out the config to the default file
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

// getConfigDir returns the full path to the config directory
func (c *Config) getConfigDir() string {
	return path.Join(glib.GetUserConfigDir(), "ymuse")
}

// getConfigFile returns the full path of the config file
func (c *Config) getConfigFile() string {
	return path.Join(c.getConfigDir(), "config.json")
}
