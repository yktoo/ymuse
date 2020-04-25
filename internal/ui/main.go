package ui

import (
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
	//"github.com/yktoo/ymuse/internal"
	"log"
)

type MainWindow struct {
	GtkWindow *gtk.ApplicationWindow
}

func NewMainWindow(application *gtk.Application) (*MainWindow, error) {
	mainWindow := &MainWindow{}

	// Initialize GTK without parsing any command line arguments.
	gtk.Init(nil)

	// Set up the window
	builder, err := gtk.BuilderNewFromFile("internal/ui/player.glade")
	if err != nil {
		return nil, errors.Errorf("Failed to create GtkBuilder: %v", err)
	}

	// Map the handlers to callback functions
	signals := map[string]interface{}{
		"on_mainWindow_destroy": onMainWindowDestroy,
		"on_mainWindow_map":     onMainWindowMap,
	}
	builder.ConnectSignals(signals)

	// Find the app window
	obj, err := builder.GetObject("mainWindow")
	if err != nil {
		return nil, errors.Errorf("Failed to find mainWindow widget: %v", err)
	}

	// Validate its type
	gtkAppWindow, ok := obj.(*gtk.ApplicationWindow)
	if !ok {
		return nil, errors.New("mainWindow is not a gtk.ApplicationWindow")
	}

	mainWindow.GtkWindow = gtkAppWindow
	application.AddWindow(mainWindow.GtkWindow)

	// Show the window
	mainWindow.GtkWindow.ShowAll()
	return mainWindow, nil
}

func onMainWindowDestroy() {
	log.Println("onMainWindowDestroy")
}

func onMainWindowMap() {
	log.Println("onMainWindowMap")
	/*TODO
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
	*/
}
