package player

import (
	"fmt"
	"github.com/fhs/gompd/v2/mpd"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
	"sync"
	"time"

	//"github.com/yktoo/ymuse/internal"
	"log"
)

type MainWindow struct {
	GtkWindow *gtk.ApplicationWindow
	lblStatus *gtk.Label
	// Connector's start channel
	chConnectorStart chan int
	// Connector's quit channel
	chConnectorQuit chan int
	// Watcher's start channel
	chWatcherStart chan int
	// Watcher's quit channel
	chWatcherQuit chan int
	// MPD's IP address
	mpdAddress string
	// Status channel that show whether there's a connection to MPD
	chStatus chan bool
	// MPD client instance
	mpdClient      *mpd.Client
	mpdClientMutex sync.Mutex
	// MPD watcher instance
	mpdWatcher *mpd.Watcher
}

func NewMainWindow(application *gtk.Application, mpdAddress string) (*MainWindow, error) {
	w := &MainWindow{
		mpdAddress:       mpdAddress,
		chConnectorStart: make(chan int),
		chConnectorQuit:  make(chan int),
		chWatcherStart:   make(chan int),
		chWatcherQuit:    make(chan int),
	}

	// Initialize GTK without parsing any command line arguments.
	gtk.Init(nil)

	// Set up the window
	builder, err := gtk.BuilderNewFromFile("internal/player/player.glade")
	if err != nil {
		return nil, errors.Errorf("Failed to create GtkBuilder: %v", err)
	}

	// Map the handlers to callback functions
	builder.ConnectSignals(map[string]interface{}{
		"on_mainWindow_destroy": w.onDestroy,
		"on_mainWindow_map":     w.onMap,
	})

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
	w.GtkWindow = gtkAppWindow
	application.AddWindow(w.GtkWindow)

	// Map widgets
	obj, err = builder.GetObject("lblStatus")
	if err != nil {
		return nil, errors.Errorf("Failed to find lblStatus widget: %v", err)
	}
	w.lblStatus, _ = obj.(*gtk.Label)

	// Start connector and watcher threads
	go w.connect()
	go w.watch()

	// Show the window
	w.GtkWindow.ShowAll()
	return w, nil
}

func (w *MainWindow) onDestroy() {
	log.Println("onDestroy")

	// Signal quit
	close(w.chConnectorQuit)
	close(w.chWatcherQuit)
}

func (w *MainWindow) onMap() {
	log.Println("onMap")

	// Start connecting
	w.chConnectorStart <- 1
}

// connect() establishes a connection to MPD
func (w *MainWindow) connect() {
	var reconnectTimer *time.Timer
	for {
		select {
		// Request to connect
		case <-w.chConnectorStart:
			log.Println("Start connecting")

			// Remove the timer
			reconnectTimer = nil

			// If no client yet
			w.mpdClientMutex.Lock()
			if w.mpdClient == nil {
				client, err := tryConnect(w.mpdAddress)
				// Failed to connect
				if err != nil {
					log.Printf("Failed to connect to MPD: %v", err)
					// Schedule a reconnection
					reconnectTimer = time.AfterFunc(3*time.Second, func() {
						w.chConnectorStart <- 1
					})

					// Connection succeeded
				} else {
					w.mpdClient = client
					// Notify the app
					_, _ = glib.IdleAdd(w.onConnectStatus, true)
					// Start the watcher
					w.chWatcherStart <- 1
				}
			}
			w.mpdClientMutex.Unlock()

		// Request to quit
		case <-w.chConnectorQuit:
			// Kill the reconnection timer, if any
			if reconnectTimer != nil {
				reconnectTimer.Stop()
			}

			// Close the connection to MPD, if any
			w.mpdClientMutex.Lock()
			if w.mpdClient != nil {
				log.Println("Stop connection")
				_ = w.mpdClient.Close()
				w.mpdClient = nil
			}
			w.mpdClientMutex.Unlock()
			return
		}
	}
}

func tryConnect(address string) (*mpd.Client, error) {
	log.Printf("tryConnect(%v)\n", address)

	// Try to connect to MPD
	client, err := mpd.Dial("tcp", address)
	if err != nil {
		return nil, errors.Errorf("Dial() failed: %v", err)
	}

	// Client is available, validate connection status
	_, err = client.Status()
	if err != nil {
		return nil, errors.Errorf("Status() failed: %v", err)
	}

	// All OK
	return client, nil
}

// watch() watches MPD subsystem changes
func (w *MainWindow) watch() {
	var rewatchTimer *time.Timer
	var eventChannel chan string = nil
	var errorChannel chan error = nil
	for {
		select {
		// Request to watch
		case <-w.chWatcherStart:
			log.Println("Start watching")

			// Remove the timer
			rewatchTimer = nil

			// If no watcher yet
			if w.mpdWatcher == nil {
				watcher, err := mpd.NewWatcher("tcp", w.mpdAddress, "")
				// Failed to connect
				if err != nil {
					log.Printf("Failed to watch MPD: %v", err)
					// Schedule a reconnection
					rewatchTimer = time.AfterFunc(3*time.Second, func() {
						w.chWatcherStart <- 1
					})

					// Connection succeeded
				} else {
					w.mpdWatcher = watcher
					eventChannel = watcher.Event
					errorChannel = watcher.Error
				}
			}

		// Watcher's event
		case subsystem := <-eventChannel:
			// Notify the app
			_, _ = glib.IdleAdd(w.onSubsystemChange, subsystem)

		// Watcher's error
		case err := <-errorChannel:
			log.Printf("Watcher error: %v", err)

		// Request to quit
		case <-w.chWatcherQuit:
			// Kill the reconnection timer, if any
			if rewatchTimer != nil {
				rewatchTimer.Stop()
			}

			// Close the connection to MPD, if any
			if w.mpdWatcher != nil {
				log.Println("Stop watcher")
				_ = w.mpdWatcher.Close()
				w.mpdWatcher = nil
			}
			return
		}
	}
}

// onConnectStatus() is called whenever connection status changes. Always on GTK+ main loop thread
func (w *MainWindow) onConnectStatus(connected bool) {
	log.Printf("onConnectStatus(%v)", connected)
	w.lblStatus.SetText(fmt.Sprintf("Connected: %v", connected))
}

// onSubsystemChange() is called whenever MPD's subsystem receives an event. Always on GTK+ main loop thread
func (w *MainWindow) onSubsystemChange(subsystem string) {
	log.Printf("onSubsystemChange(%v)", subsystem)
}
