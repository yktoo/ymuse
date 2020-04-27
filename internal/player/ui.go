package player

import (
	"fmt"
	"github.com/fhs/gompd/v2/mpd"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
	"log"
	"sync"
	"time"
)

type MainWindow struct {
	// Widgets
	window          *gtk.ApplicationWindow
	lblStatus       *gtk.Label
	lblCurrentTrack *gtk.Label
	btnPrevious     *gtk.Button
	btnPlayPause    *gtk.Button
	btnNext         *gtk.Button
	scPlayPosition  *gtk.Scale
	trvQueue        *gtk.TreeView
	lstQueue        *gtk.ListStore

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
	builder := NewBuilder("internal/player/player.glade")

	// Map the handlers to callback functions
	builder.ConnectSignals(map[string]interface{}{
		"on_mainWindow_destroy":   w.onDestroy,
		"on_mainWindow_map":       w.onMap,
		"on_btnPrevious_clicked":  w.onPreviousClicked,
		"on_btnPlayPause_clicked": w.onPlayPauseClicked,
		"on_btnNext_clicked":      w.onNextClicked,
	})

	// Find widgets
	w.window = builder.getApplicationWindow("mainWindow")
	w.lblStatus = builder.getLabel("lblStatus")
	w.lblCurrentTrack = builder.getLabel("lblCurrentTrack")
	w.btnPrevious = builder.getButton("btnPrevious")
	w.btnPlayPause = builder.getButton("btnPlayPause")
	w.btnNext = builder.getButton("btnNext")
	w.scPlayPosition = builder.getScale("scPlayPosition")
	w.trvQueue = builder.getTreeView("trvQueue")
	w.lstQueue = builder.getListStore("lstQueue")

	// Tweak buttons
	if icon, err := gtk.ImageNewFromIconName("media-skip-backward", gtk.ICON_SIZE_BUTTON); err == nil {
		w.btnPrevious.SetImage(icon)
	}
	if icon, err := gtk.ImageNewFromIconName("media-skip-forward", gtk.ICON_SIZE_BUTTON); err == nil {
		w.btnNext.SetImage(icon)
	}

	// Register the main window with the app
	application.AddWindow(w.window)

	// Start connector and watcher threads
	go w.connect()
	go w.watch()

	// Show the window
	w.window.ShowAll()
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

func (w *MainWindow) onPreviousClicked() {
	log.Println("onPreviousClicked")
	w.ifConnected(func(client *mpd.Client) { _ = client.Previous() }, nil)
}

func (w *MainWindow) onPlayPauseClicked() {
	log.Println("onPlayPauseClicked")
	w.ifConnected(
		func(client *mpd.Client) {
			if status, err := client.Status(); err == nil {
				_ = client.Pause(status["state"] == "play")
			}
		},
		nil)
}

func (w *MainWindow) onNextClicked() {
	log.Println("onNextClicked")
	w.ifConnected(func(client *mpd.Client) { _ = client.Next() }, nil)
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
					_, _ = glib.IdleAdd(w.updateAll)
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

// onSubsystemChange() is called whenever MPD's subsystem receives an event. Always on GTK+ main loop thread
func (w *MainWindow) onSubsystemChange(subsystem string) {
	log.Printf("onSubsystemChange(%v)", subsystem)
	switch subsystem {
	case "player":
		w.updatePlayer()
	case "playlist":
		w.updateQueue()
	}
}

// isConnected() returns whether a connection with MPD has been successfully established
func (w *MainWindow) isConnected() bool {
	w.mpdClientMutex.Lock()
	defer w.mpdClientMutex.Unlock()
	return w.mpdClient != nil
}

// ifConnected() runs MPD client code if there's a connection with MPD and/or code if there's no connection
func (w *MainWindow) ifConnected(funcIfConnected func(client *mpd.Client), funcIfDisconnected func()) {
	w.mpdClientMutex.Lock()
	defer w.mpdClientMutex.Unlock()
	switch {
	case w.mpdClient == nil && funcIfDisconnected != nil:
		funcIfDisconnected()
	case w.mpdClient != nil && funcIfConnected != nil:
		funcIfConnected(w.mpdClient)
	}
}

// updateAll() updates all player's widgets and lists
func (w *MainWindow) updateAll() {
	w.updatePlayer()
	w.updateQueue()
}

// updatePlayer() updates player control widgets
func (w *MainWindow) updatePlayer() {
	// Enable or disable widgets based on the connection status
	connected := w.isConnected()
	w.btnPrevious.SetSensitive(connected)
	w.btnPlayPause.SetSensitive(connected)
	w.btnNext.SetSensitive(connected)
	w.scPlayPosition.SetSensitive(connected)

	w.ifConnected(
		// Connected
		func(client *mpd.Client) {
			// Request player status
			status, err := client.Status()
			if err != nil {
				return
			}
			switch status["state"] {
			case "play":
				if icon, err := gtk.ImageNewFromIconName("media-playback-pause", gtk.ICON_SIZE_BUTTON); err == nil {
					w.btnPlayPause.SetImage(icon)
				}
			case "stop", "pause":
				if icon, err := gtk.ImageNewFromIconName("media-playback-start", gtk.ICON_SIZE_BUTTON); err == nil {
					w.btnPlayPause.SetImage(icon)
				}
			}

			// Fetch current song
			curSong, err := client.CurrentSong()
			if err != nil {
				w.lblCurrentTrack.SetText(fmt.Sprintf("Error: %v", err))
			} else {
				w.lblCurrentTrack.SetText(curSong["title"])
			}
		},
		// Disconnected
		func() {
			w.lblCurrentTrack.SetText("(not connected)")
		})
}

// updateQueue() updates the current play queue contents
func (w *MainWindow) updateQueue() {
	w.ifConnected(
		// Connected
		func(client *mpd.Client) {
			if attrs, err := client.PlaylistInfo(-1, -1); err == nil {
				w.lstQueue.Clear()
				for _, a := range attrs {
					w.lstQueue.SetCols(
						w.lstQueue.Append(),
						map[int]interface{}{
							0: a["Artist"],
							1: a["Date"],
							2: a["Album"],
							3: a["Track"],
							4: a["Title"],
							5: a["duration"],
						})
				}
				w.window.ShowAll()
			}
		},
		// Disconnected - clear the queue
		func() {
			w.lstQueue.Clear()
		})
}
