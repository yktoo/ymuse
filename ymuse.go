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
//go:generate resources/scripts/generate-mos

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
	version = "(dev)"
	commit  = "(?)"
	date    = "(?)"
)

func main() {
	// Initialise the gettext engine
	glib.InitI18n("ymuse", "/usr/share/locale/")

	// Process command line
	verbInfo := flag.Bool("v", false, glib.Local("verbose logging"))
	verbDebug := flag.Bool("vv", false, glib.Local("more verbose logging"))
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
	config.AppMetadata.BuildDate = date

	// Start the app
	log.Infof(glib.Local("Ymuse version %s; %s; released %s"), version, commit, date)

	// Create Gtk Application, change appID to your application domain name reversed.
	application, err := gtk.ApplicationNew(config.AppMetadata.ID, glib.APPLICATION_FLAGS_NONE)
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
