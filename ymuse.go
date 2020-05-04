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

package main

import (
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/op/go-logging"
	"github.com/yktoo/ymuse/internal/player"
	"github.com/yktoo/ymuse/internal/util"
	"os"
)

var log *logging.Logger

func main() {
	// Init logging
	logging.SetFormatter(logging.MustStringFormatter(`%{time:15:04:05.000} %{level:-5s} %{module} %{message}`))
	log = logging.MustGetLogger("main")
	logging.SetLevel(util.GetConfig().LogLevel, "main")

	// Start the app
	log.Info("Ymuse version", util.AppVersion)

	// Create Gtk Application, change appID to your application domain name reversed.
	application, err := gtk.ApplicationNew(util.AppID, glib.APPLICATION_FLAGS_NONE)
	if err != nil {
		log.Fatal("Could not create application", err)
	}

	// Setup the application
	if _, err = application.Connect("activate", func() { onActivate(application) }); err != nil {
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
