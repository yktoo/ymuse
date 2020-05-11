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

//go:generate resources/scripts/generate-resources

package main

import (
	"flag"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/op/go-logging"
	"github.com/yktoo/ymuse/internal/config"
	"github.com/yktoo/ymuse/internal/player"
	"os"
)

var log = logging.MustGetLogger("main")

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Process command line
	verbInfo := flag.Bool("v", false, "verbose logging")
	verbDebug := flag.Bool("vv", false, "more verbose logging")
	flag.Parse()

	// Init logging
	logLevel := logging.WARNING
	switch {
	case *verbDebug:
		logLevel = logging.DEBUG
	case *verbInfo:
		logLevel = logging.INFO
	}
	logging.SetFormatter(logging.MustStringFormatter(`%{time:15:04:05.000} %{level:-5s} %{module} %{message}`))
	logging.SetLevel(logLevel, "")

	// Init application metadata
	config.AppMetadata.Version = version
	config.AppMetadata.Commit = commit
	config.AppMetadata.BuildDate = date

	// Start the app
	log.Info("Ymuse version", version)

	// Create Gtk Application, change appID to your application domain name reversed.
	application, err := gtk.ApplicationNew(config.AppMetadata.Id, glib.APPLICATION_FLAGS_NONE)
	if err != nil {
		log.Fatal("Could not create application", err)
	}

	// Setup the application
	if _, err = application.Connect("activate", onActivate); err != nil {
		log.Fatal("Failed to connect activation signal", err)
	}

	// Run the application
	os.Exit(application.Run(nil))
}

func onActivate(application *gtk.Application) {
	// Create the main window
	if window, err := player.NewMainWindow(application); err != nil {
		log.Fatal("Could not create application window", err)
	} else {
		window.Show()
	}
}
