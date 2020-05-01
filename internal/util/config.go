package util

import (
	"flag"
	"github.com/op/go-logging"
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
	TrackDefaultReplace    bool          // Whether the default action for double-clicking a track is replace rather than append
	PlaylistDefaultReplace bool          // Whether the default action for double-clicking a playlist is replace rather than append
}

// Config singleton with all the defaults
var config = &Config{
	LogLevel:               logging.WARNING,
	TrackDefaultReplace:    false,
	PlaylistDefaultReplace: true,
}
var once sync.Once

// GetConfig() returns a global Config instance
func GetConfig() *Config {
	once.Do(func() {
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
