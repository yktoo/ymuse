package player

import (
	"fmt"
	"github.com/fhs/gompd/v2/mpd"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/yktoo/ymuse/internal/util"
	"strconv"
	"sync"
	"time"
)

type MainWindow struct {
	// Widgets
	window          *gtk.ApplicationWindow
	lblStatus       *gtk.Label
	btnPrevious     *gtk.Button
	btnPlayPause    *gtk.Button
	btnNext         *gtk.Button
	scPlayPosition  *gtk.Scale
	adjPlayPosition *gtk.Adjustment
	trvQueue        *gtk.TreeView
	lstQueue        *gtk.ListStore

	// Connector's connect channel
	chConnectorConnect chan int
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
		mpdAddress:         mpdAddress,
		chConnectorConnect: make(chan int),
		chConnectorQuit:    make(chan int),
		chWatcherStart:     make(chan int),
		chWatcherQuit:      make(chan int),
	}

	// Set up the window
	builder := NewBuilder("internal/player/player.glade")

	// Map the handlers to callback functions
	builder.ConnectSignals(map[string]interface{}{
		"on_mainWindow_destroy":         w.onDestroy,
		"on_mainWindow_map":             w.onMap,
		"on_btnPrevious_clicked":        w.onPreviousClicked,
		"on_btnPlayPause_clicked":       w.onPlayPauseClicked,
		"on_btnNext_clicked":            w.onNextClicked,
		"on_scPlayPosition_formatValue": w.onPlayPositionFormatValue,
	})

	// Find widgets
	w.window = builder.getApplicationWindow("mainWindow")
	w.lblStatus = builder.getLabel("lblStatus")
	w.btnPrevious = builder.getButton("btnPrevious")
	w.btnPlayPause = builder.getButton("btnPlayPause")
	w.btnNext = builder.getButton("btnNext")
	w.scPlayPosition = builder.getScale("scPlayPosition")
	w.adjPlayPosition = builder.getAdjustment("adjPlayPosition")
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

// whenIdle() schedules a function call on GLib's main loop thread
func whenIdle(name string, f interface{}, args ...interface{}) {
	_, err := glib.IdleAdd(f, args...)
	errCheck(err, "glib.IdleAdd() failed for "+name)
}

func (w *MainWindow) onDestroy() {
	log.Debug("onDestroy()")

	// Signal quit
	close(w.chConnectorQuit)
	close(w.chWatcherQuit)
}

func (w *MainWindow) onMap() {
	log.Debug("onMap()")

	// Start connecting
	w.startConnector()
}

func (w *MainWindow) onPreviousClicked() {
	log.Debug("onPreviousClicked()")
	w.ifConnected(
		func(client *mpd.Client) { errCheck(client.Previous(), "Previous() failed") },
		nil)
}

func (w *MainWindow) onPlayPauseClicked() {
	log.Debug("onPlayPauseClicked()")
	w.ifConnected(
		func(client *mpd.Client) {
			if status, err := client.Status(); err == nil {
				errCheck(client.Pause(status["state"] == "play"), "Pause() failed")
			} else {
				errCheck(err, "Status() failed")
			}
		},
		nil)
}

func (w *MainWindow) onNextClicked() {
	log.Debug("onNextClicked()")
	w.ifConnected(
		func(client *mpd.Client) { errCheck(client.Next(), "Next() failed") },
		nil)
}

func (w *MainWindow) onPlayPositionFormatValue(_ *gtk.Scale, v float64) string {
	return util.FormatSeconds(v) + "/" + util.FormatSeconds(w.adjPlayPosition.GetUpper())
}

// startConnector() signals the connector to initiate connection process
func (w *MainWindow) startConnector() {
	w.chConnectorConnect <- 1
}

// connect() takes care of establishing a connection to MPD
func (w *MainWindow) connect() {
	log.Debug("connect()")
	var keepaliveTicker = time.NewTicker(time.Second)
	for {
		select {
		// Request to connect
		case <-w.chConnectorConnect:
			log.Debug("Start connector")

			// If disconnected
			w.ifConnected(
				nil,
				func() {
					// Try to connect to MPD
					client, err := mpd.Dial("tcp", w.mpdAddress)
					if err != nil {
						errCheck(err, "Dial() failed")
						return
					}

					// Connection succeeded
					w.mpdClient = client

					// Notify the app
					whenIdle("updateAll()", w.updateAll)

					// Start the watcher
					w.chWatcherStart <- 1
				})

		// Keepalive tick
		case <-keepaliveTicker.C:
			w.ifConnected(
				func(client *mpd.Client) {
					// Connection lost
					if err := client.Ping(); err != nil {
						log.Debug("Ping(): connection to MPD lost", err)
						w.mpdClient = nil
						go w.startConnector()
					}
				},
				func() {
					go w.startConnector()
				})

			// Update the seek bar
			whenIdle("updateSeekBar()", w.updateSeekBar)

		// Request to quit
		case <-w.chConnectorQuit:
			// Kill the keepalive timer
			keepaliveTicker.Stop()

			// Close the connection to MPD, if any
			w.ifConnected(
				func(client *mpd.Client) {
					log.Debug("Stop connector")
					errCheck(client.Close(), "Close() failed")
					w.mpdClient = nil
				},
				nil)
			return
		}
	}
}

// watch() watches MPD subsystem changes
func (w *MainWindow) watch() {
	log.Debug("watch()")
	var rewatchTimer *time.Timer
	var eventChannel chan string = nil
	var errorChannel chan error = nil
	for {
		select {
		// Request to watch
		case <-w.chWatcherStart:
			log.Debug("Start watcher")

			// Remove the timer
			rewatchTimer = nil

			// If no watcher yet
			if w.mpdWatcher == nil {
				watcher, err := mpd.NewWatcher("tcp", w.mpdAddress, "")
				// Failed to connect
				if err != nil {
					log.Warning("Failed to watch MPD", err)
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
			whenIdle("onSubsystemChange()", w.onSubsystemChange, subsystem)

		// Watcher's error
		case err := <-errorChannel:
			log.Warning("Watcher error", err)

		// Request to quit
		case <-w.chWatcherQuit:
			// Kill the reconnection timer, if any
			if rewatchTimer != nil {
				rewatchTimer.Stop()
			}

			// Close the connection to MPD, if any
			if w.mpdWatcher != nil {
				log.Debug("Stop watcher")
				errCheck(w.mpdWatcher.Close(), "mpdWatcher.Close() failed")
				w.mpdWatcher = nil
			}
			return
		}
	}
}

// onSubsystemChange() is called whenever MPD's subsystem receives an event. Always on GTK+ main loop thread
func (w *MainWindow) onSubsystemChange(subsystem string) {
	log.Debugf("onSubsystemChange(%v)", subsystem)
	switch subsystem {
	case "player":
		w.updatePlayer()
	case "playlist":
		w.updateQueue()
	}
}

// ifConnected() runs MPD client code if there's a connection with MPD and/or code if there's no connection
func (w *MainWindow) ifConnected(funcIfConnected func(client *mpd.Client), funcIfDisconnected func()) {
	w.mpdClientMutex.Lock()
	defer w.mpdClientMutex.Unlock()
	switch {
	// Disconnected
	case w.mpdClient == nil && funcIfDisconnected != nil:
		funcIfDisconnected()
	// Connected
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
	connected := true
	w.ifConnected(
		// Connected
		func(client *mpd.Client) {
			// Request player status
			status, err := client.Status()
			if err != nil {
				errCheck(err, "Status() failed")
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
				errCheck(err, "CurrentSong() failed")
				w.lblStatus.SetText(fmt.Sprintf("Error: %v", err))
			} else {
				w.lblStatus.SetText(curSong["Title"])
			}
		},
		// Disconnected
		func() {
			w.lblStatus.SetText("(not connected)")
			connected = false
		})

	// Enable or disable widgets based on the connection status
	w.btnPrevious.SetSensitive(connected)
	w.btnPlayPause.SetSensitive(connected)
	w.btnNext.SetSensitive(connected)
}

// updateQueue() updates the current play queue contents
func (w *MainWindow) updateQueue() {
	w.ifConnected(
		// Connected
		func(client *mpd.Client) {
			if attrs, err := client.PlaylistInfo(-1, -1); err == nil {
				w.lstQueue.Clear()
				for _, a := range attrs {
					errCheck(
						w.lstQueue.SetCols(
							w.lstQueue.Append(),
							map[int]interface{}{
								0: a["Artist"],
								1: a["Date"],
								2: a["Album"],
								3: a["Track"],
								4: a["Title"],
								5: a["duration"],
							}),
						"lstQueue.SetCols() failed")
				}
				w.window.ShowAll()
			}
		},
		// Disconnected - clear the queue
		func() {
			w.lstQueue.Clear()
		})
}

// updateSeekBar() updates the seek bar position and status
func (w *MainWindow) updateSeekBar() {
	seekable := false
	w.ifConnected(
		// Connected
		func(client *mpd.Client) {
			status, err := client.Status()
			switch {
			case err != nil:
				errCheck(err, "Status() failed")
			case status["state"] == "play":
				seekable = true
				if f, err := strconv.ParseFloat(status["duration"], 32); err == nil {
					w.adjPlayPosition.SetUpper(f)
				}
				if f, err := strconv.ParseFloat(status["elapsed"], 32); err == nil {
					w.adjPlayPosition.SetValue(f)
				}
			}
		},
		// Disconnected - send a ping
		nil)
	w.scPlayPosition.SetSensitive(seekable)
}
