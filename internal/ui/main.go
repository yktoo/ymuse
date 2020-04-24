package ui

import (
	"fmt"
	"github.com/gotk3/gotk3/gtk"
	"github.com/yktoo/ymuse/internal"
	"log"
)

type MainWindow struct {
	win *gtk.ApplicationWindow
}

func NewMainWindow(application *gtk.Application) (*MainWindow, error) {
	mainWindow := &MainWindow{}

	// Initialize GTK without parsing any command line arguments.
	gtk.Init(nil)

	// Instantiate a GTK window
	win, err := gtk.ApplicationWindowNew(application)
	if err != nil {
		return nil, err
	}
	mainWindow.win = win

	// Set up the window
	mainWindow.setup()
	return mainWindow, nil
}

func (win *MainWindow) setup() {
	win.win.SetTitle("Ymuse")
	win.win.SetDefaultSize(800, 600)

	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)
	if err != nil {
		log.Fatal(err)
	}
	win.win.Add(box)

	textView, err := gtk.TextViewNew()
	if err != nil {
		log.Fatal(err)
	}
	textView.SetBorderWidth(12)
	textView.SetEditable(false)
	box.PackStart(textView, true, true, 0)

	buf, err := textView.GetBuffer()
	if err != nil {
		log.Fatal(err)
	}

	// Instantiate a player
	player, err := internal.NewPlayer("127.0.0.1:6600")
	if err != nil {
		log.Fatal(err)
	}

	// Print MPD version
	ver, err := player.Version()
	if err != nil {
		log.Fatal(err)
	}
	addText(buf, fmt.Sprintf("Connected to MPD version %v\n", ver))

	// Print out status
	status, err := player.Status()
	if err != nil {
		log.Fatal(err)
	}
	addText(buf, "MPD status:\n")
	for k, v := range status {
		addText(buf, fmt.Sprintf("  - %v: %v\n", k, v))
	}

	// Print out statistics
	stats, err := player.Stats()
	if err != nil {
		log.Fatal(err)
	}
	addText(buf, "MPD database statistics:\n")
	for k, v := range stats {
		addText(buf, fmt.Sprintf("  - %v: %v\n", k, v))
	}

	// Show the window
	win.win.ShowAll()
}

func addText(buf *gtk.TextBuffer, s string) {
	buf.Insert(buf.GetEndIter(), s)
}
