package main

import (
	"fmt"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/yktoo/ymuse/internal/ui"
	"log"
	"os"
)

const appVersion = "0.01"
const appID = "com.yktoo.ymuse"

func main() {
	fmt.Printf("Ymuse version %s\n", appVersion)

	// Create Gtk Application, change appID to your application domain name reversed.
	application, err := gtk.ApplicationNew(appID, glib.APPLICATION_FLAGS_NONE)
	if err != nil {
		log.Fatal("Could not create application", err)
	}

	// Setup the application
	if _, err = application.Connect("activate", func() { onActivate(application) }); err != nil {
		log.Fatal("Failed to connect activation signal", err)
	}

	// Run the application
	os.Exit(application.Run(os.Args))
}

func onActivate(application *gtk.Application) {
	// Create the main window
	if _, err := ui.NewMainWindow(application); err != nil {
		log.Fatal("Could not create application window", err)
	}
}
