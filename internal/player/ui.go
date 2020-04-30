package player

import (
	"fmt"
	"github.com/fhs/gompd/v2/mpd"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/yktoo/ymuse/internal/util"
	"strconv"
	"time"
)

type MainWindow struct {
	// Application reference
	app *gtk.Application
	// Connector instance
	connector *Connector
	// Main window
	window *gtk.ApplicationWindow
	// Control widgets
	lblStatus       *gtk.Label
	lblPosition     *gtk.Label
	btnPrevious     *gtk.ToolButton
	btnPlayPause    *gtk.ToolButton
	btnNext         *gtk.ToolButton
	btnRandom       *gtk.ToggleToolButton
	btnRepeat       *gtk.ToggleToolButton
	btnConsume      *gtk.ToggleToolButton
	scPlayPosition  *gtk.Scale
	adjPlayPosition *gtk.Adjustment
	// Queue widgets
	lblQueueInfo *gtk.Label
	trvQueue     *gtk.TreeView
	lstQueue     *gtk.ListStore

	// Last reported MPD status
	mpdStatus mpd.Attrs

	// Playlist's track index (last) marked as current
	currentIndex int

	// Play position manual update flag
	playPosUpdating bool
	// Options update flag
	optionsUpdating bool
}

//noinspection GoSnakeCaseUsage
const (
	// Queue tree view column indices
	ColQueue_Artist int = iota
	ColQueue_Year
	ColQueue_Album
	ColQueue_Number
	ColQueue_Track
	ColQueue_Length
	ColQueue_FontWeight
	ColQueue_BgColor
)

const (
	FontWeightNormal      = 400
	FontWeightBold        = 700
	BackgroundColorNormal = "#ffffff"
	BackgroundColorActive = "#ffffe0"
)

func NewMainWindow(application *gtk.Application, mpdAddress string) (*MainWindow, error) {
	// Set up the window
	builder := NewBuilder("internal/player/player.glade")

	w := &MainWindow{
		app: application,
		// Find widgets
		window:          builder.getApplicationWindow("mainWindow"),
		lblStatus:       builder.getLabel("lblStatus"),
		lblPosition:     builder.getLabel("lblPosition"),
		btnPrevious:     builder.getToolButton("btnPrevious"),
		btnPlayPause:    builder.getToolButton("btnPlayPause"),
		btnNext:         builder.getToolButton("btnNext"),
		btnRandom:       builder.getToggleToolButton("btnRandom"),
		btnRepeat:       builder.getToggleToolButton("btnRepeat"),
		btnConsume:      builder.getToggleToolButton("btnConsume"),
		scPlayPosition:  builder.getScale("scPlayPosition"),
		adjPlayPosition: builder.getAdjustment("adjPlayPosition"),
		// Queue
		lblQueueInfo: builder.getLabel("lblQueueInfo"),
		trvQueue:     builder.getTreeView("trvQueue"),
		lstQueue:     builder.getListStore("lstQueue"),
	}

	// Map the handlers to callback functions
	builder.ConnectSignals(map[string]interface{}{
		"on_mainWindow_destroy":         w.onDestroy,
		"on_mainWindow_map":             w.onMap,
		"on_trvQueue_buttonPress":       w.onQueueTreeViewButtonPress,
		"on_btnPrevious_clicked":        w.onPreviousClicked,
		"on_btnStop_clicked":            w.onStopClicked,
		"on_btnPlayPause_clicked":       w.onPlayPauseClicked,
		"on_btnNext_clicked":            w.onNextClicked,
		"on_btnRandom_toggled":          w.onRandomToggled,
		"on_btnRepeat_toggled":          w.onRepeatToggled,
		"on_btnConsume_toggled":         w.onConsumeToggled,
		"on_scPlayPosition_buttonEvent": w.onPlayPositionButtonEvent,
	})

	// Create actions
	w.addAction("about", "F1", w.onAbout)
	w.addAction("prefs", "<Ctrl>comma", w.notImplemented)
	w.addAction("quit", "<Ctrl>Q", w.window.Close)

	// Register the main window with the app
	application.AddWindow(w.window)

	// Instantiate a connector
	w.connector = NewConnector(mpdAddress, w.onConnectorConnected, w.onConnectorHeartbeat, w.onConnectorSubsystemChange)
	return w, nil
}

// whenIdle() schedules a function call on GLib's main loop thread
func whenIdle(name string, f interface{}, args ...interface{}) {
	_, err := glib.IdleAdd(f, args...)
	errCheck(err, "glib.IdleAdd() failed for "+name)
}

func (w *MainWindow) notImplemented() {
	dlg := gtk.MessageDialogNew(w.window, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, "Function not implemented.")
	dlg.Run()
	dlg.Destroy()
}

func (w *MainWindow) addAction(name, shortcut string, onActivate interface{}) {
	action := glib.SimpleActionNew(name, nil)
	if _, err := action.Connect("activate", onActivate); err != nil {
		log.Fatalf("Failed to connect activate signal of action '%v': %v", name, err)
	}
	w.app.AddAction(action)
	if shortcut != "" {
		w.app.SetAccelsForAction("app."+name, []string{shortcut})
	}
}

func (w *MainWindow) onConnectorConnected() {
	whenIdle("onConnectorConnected()", w.updateAll)
}

func (w *MainWindow) onConnectorHeartbeat() {
	whenIdle("onConnectorHeartbeat()", func() {
		w.fetchStatus()
		w.updateSeekBar()
	})
}

func (w *MainWindow) onConnectorSubsystemChange(subsystem string) {
	log.Debugf("onSubsystemChange(%v)", subsystem)
	switch subsystem {
	case "options":
		whenIdle("updateOptions()", w.updateOptions, true)
	case "player":
		whenIdle("updatePlayer()", w.updatePlayer, true)
	case "playlist":
		whenIdle("updateQueue()", w.updateQueue)
	}
}

func (w *MainWindow) onAbout() {
	dlg, err := gtk.AboutDialogNew()
	if errCheck(err, "AboutDialogNew() failed") {
		return
	}
	dlg.SetLogoIconName("dialog-information")
	dlg.SetProgramName(util.AppName)
	dlg.SetCopyright("Written by Dmitry Kann")
	dlg.SetLicense(util.AppLicense)
	dlg.SetWebsite(util.AppWebsite)
	dlg.SetWebsiteLabel(util.AppWebsiteLabel)
	_, _ = dlg.Connect("response", func() { dlg.Destroy() })
	dlg.Run()
}

func (w *MainWindow) onMap() {
	log.Debug("onMap()")

	// Start connecting
	w.connector.Start()
}

func (w *MainWindow) onDestroy() {
	log.Debug("onDestroy()")

	// Shut the connector down
	w.connector.Stop()
}

func (w *MainWindow) onQueueTreeViewButtonPress(trv *gtk.TreeView, event *gdk.Event) {
	// Double click in the tree
	if gdk.EventButtonNewFromEvent(event).Type() == gdk.EVENT_DOUBLE_BUTTON_PRESS {
		log.Debug("onQueueTreeViewButtonPress: double click")
		if sel, err := trv.GetSelection(); err != nil {
			errCheck(err, "trvQueue.GetSelection() failed")
		} else {
			// Get selected node's index
			indices := sel.GetSelectedRows(nil).Data().(*gtk.TreePath).GetIndices()
			if len(indices) > 0 {
				// Start playback from the given index
				w.connector.IfConnected(
					func(client *mpd.Client) {
						errCheck(client.Play(indices[0]), "Play() failed")
					},
					nil)
			}
		}
	}
}

func (w *MainWindow) onPreviousClicked() {
	log.Debug("onPreviousClicked()")
	w.connector.IfConnected(
		func(client *mpd.Client) { errCheck(client.Previous(), "Previous() failed") },
		nil)
}

func (w *MainWindow) onStopClicked() {
	log.Debug("onStopClicked()")
	w.connector.IfConnected(
		func(client *mpd.Client) { errCheck(client.Stop(), "Stop() failed") },
		nil)
}

func (w *MainWindow) onPlayPauseClicked() {
	log.Debug("onPlayPauseClicked()")
	w.connector.IfConnected(
		func(client *mpd.Client) {
			errCheck(client.Pause(w.mpdStatus["state"] == "play"), "Pause() failed")
		},
		nil)
}

func (w *MainWindow) onNextClicked() {
	log.Debug("onNextClicked()")
	w.connector.IfConnected(
		func(client *mpd.Client) { errCheck(client.Next(), "Next() failed") },
		nil)
}

func (w *MainWindow) onRandomToggled() {
	if !w.optionsUpdating {
		log.Debug("onRandomToggled()")
		w.connector.IfConnected(
			func(client *mpd.Client) {
				errCheck(client.Random(w.mpdStatus["random"] == "0"), "Random() failed")
			},
			nil)
	}
}

func (w *MainWindow) onRepeatToggled() {
	if !w.optionsUpdating {
		log.Debug("onRepeatToggled()")
		w.connector.IfConnected(
			func(client *mpd.Client) {
				errCheck(client.Repeat(w.mpdStatus["repeat"] == "0"), "Repeat() failed")
			},
			nil)
	}
}

func (w *MainWindow) onConsumeToggled() {
	if !w.optionsUpdating {
		log.Debug("onConsumeToggled()")
		w.connector.IfConnected(
			func(client *mpd.Client) {
				errCheck(client.Consume(w.mpdStatus["consume"] == "0"), "Consume() failed")
			},
			nil)
	}
}

func (w *MainWindow) onPlayPositionButtonEvent(_ interface{}, event *gdk.Event) {
	switch gdk.EventButtonNewFromEvent(event).Type() {
	case gdk.EVENT_BUTTON_PRESS:
		w.playPosUpdating = true

	case gdk.EVENT_BUTTON_RELEASE:
		w.playPosUpdating = false
		log.Debug("onPlayPositionButtonEvent()")
		w.connector.IfConnected(
			func(client *mpd.Client) {
				d := time.Duration(w.adjPlayPosition.GetValue())
				errCheck(
					client.SeekCur(d*time.Second, false),
					"SeekCur() failed")
			},
			nil)
	}
}

// Show() shows the window and all its child widgets
func (w *MainWindow) Show() {
	w.window.ShowAll()
}

// setQueueSelection() selects or deselects an item in the Queue tree view at the given index
func (w *MainWindow) setQueueSelection(index int, selected bool) {
	if index >= 0 {
		if iter, err := w.lstQueue.GetIterFromString(strconv.Itoa(index)); err == nil {
			weight := FontWeightNormal
			bgColor := BackgroundColorNormal
			if selected {
				weight = FontWeightBold
				bgColor = BackgroundColorActive
			}
			errCheck(
				w.lstQueue.SetCols(iter, map[int]interface{}{
					ColQueue_FontWeight: weight,
					ColQueue_BgColor:    bgColor,
				}),
				"lstQueue.SetValue() failed")
		}
	}
}

// fetchStatus() updates stored MPD's status info
func (w *MainWindow) fetchStatus() {
	w.connector.IfConnected(
		func(client *mpd.Client) {
			// Request player status
			if status, err := client.Status(); err == nil {
				w.mpdStatus = status
			} else {
				// Error: provide an empty map
				errCheck(err, "Status() failed")
				w.mpdStatus = mpd.Attrs{}
			}
		},
		func() {
			// No connection: provide an empty map
			w.mpdStatus = mpd.Attrs{}
		})
}

// updateAll() updates all player's widgets and lists
func (w *MainWindow) updateAll() {
	w.fetchStatus()
	w.updateQueue()
	w.updateOptions(false)
	w.updatePlayer(false)
}

// updateOptions() updates player options widgets
func (w *MainWindow) updateOptions(fetchStatus bool) {
	// Fetch MPD status, if needed
	if fetchStatus {
		w.fetchStatus()
	}

	// Update option widgets
	w.optionsUpdating = true
	w.btnRandom.SetActive(w.mpdStatus["random"] == "1")
	w.btnRepeat.SetActive(w.mpdStatus["repeat"] == "1")
	w.btnConsume.SetActive(w.mpdStatus["consume"] == "1")
	w.optionsUpdating = false
}

// updatePlayer() updates player control widgets
func (w *MainWindow) updatePlayer(fetchStatus bool) {
	connected := true

	// Fetch MPD status, if needed
	if fetchStatus {
		w.fetchStatus()
	}

	// Process player state
	w.connector.IfConnected(
		// Connected
		func(client *mpd.Client) {
			// Update play/pause button's appearance
			switch w.mpdStatus["state"] {
			case "play":
				w.btnPlayPause.SetIconName("media-playback-pause")
			default:
				w.btnPlayPause.SetIconName("media-playback-start")
			}

			// Fetch current song
			curSong, err := client.CurrentSong()
			str := ""
			switch {
			case err != nil:
				errCheck(err, "CurrentSong() failed")
				str = fmt.Sprintf("Error: %v", err)
			case curSong["Artist"] != "" || curSong["Album"] != "" || curSong["Title"] != "":
				str = fmt.Sprintf("%v • %v • %v", curSong["Artist"], curSong["Album"], curSong["Title"])
			case curSong["Name"] != "":
				str = curSong["Name"]
			default:
				str = "(unknown)"
			}
			w.lblStatus.SetText(str)

			// Update queue selection
			if curIdx := util.AtoiDef(w.mpdStatus["song"], -1); w.currentIndex != curIdx {
				w.setQueueSelection(w.currentIndex, false)
				w.setQueueSelection(curIdx, true)
				w.currentIndex = curIdx
			}

			// Scroll the tree to the currently played item
			if w.currentIndex >= 0 {
				if path, err := gtk.TreePathNewFromString(strconv.Itoa(w.currentIndex)); err == nil {
					w.trvQueue.ScrollToCell(path, nil, true, 0.5, 0)
				}
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
	w.btnRandom.SetSensitive(connected)
	w.btnRepeat.SetSensitive(connected)
	w.btnConsume.SetSensitive(connected)
}

// updateQueue() updates the current play queue contents
func (w *MainWindow) updateQueue() {
	w.connector.IfConnected(
		// Connected
		func(client *mpd.Client) {
			status := ""
			// Fetch the current playlist
			if attrs, err := client.PlaylistInfo(-1, -1); err == nil {
				totalSecs := 0.0
				// Repopulate the queue tree view
				w.currentIndex = -1
				w.lstQueue.Clear()
				for _, a := range attrs {
					secs := util.ParseFloatDef(a["duration"], 0)
					// Add a row to the tree view
					errCheck(
						w.lstQueue.SetCols(
							w.lstQueue.Append(),
							map[int]interface{}{
								ColQueue_Artist:     a["Artist"],
								ColQueue_Year:       a["Date"],
								ColQueue_Album:      a["Album"],
								ColQueue_Number:     a["Track"],
								ColQueue_Track:      a["Title"],
								ColQueue_Length:     util.FormatSeconds(secs),
								ColQueue_FontWeight: FontWeightNormal,
								ColQueue_BgColor:    BackgroundColorNormal,
							}),
						"lstQueue.SetCols() failed")
					// Accumulate length
					totalSecs += secs
				}

				// Add number of tracks
				switch len(attrs) {
				case 0:
					status = "Queue is empty"
				case 1:
					status = "One track"
				default:
					status = fmt.Sprintf("%d tracks", len(attrs))
				}

				// Add playing time, if any
				if totalSecs > 0 {
					status += fmt.Sprintf(", playing time %s", util.FormatSeconds(totalSecs))
				}
			}
			// Update queue info
			w.lblQueueInfo.SetText(status)
		},
		// Disconnected - clear the queue
		func() {
			w.lstQueue.Clear()
			w.lblQueueInfo.SetText("")
		})
}

// updateSeekBar() updates the seek bar position and status
func (w *MainWindow) updateSeekBar() {
	// Ignore if the user is dragging the slider manually
	if w.playPosUpdating {
		return
	}

	seekable := false
	w.connector.IfConnected(
		// Connected
		func(client *mpd.Client) {
			// Update the seek bar position
			trackLen := util.ParseFloatDef(w.mpdStatus["duration"], -1)
			trackPos := util.ParseFloatDef(w.mpdStatus["elapsed"], -1)
			seekable = trackLen >= 0 && trackPos >= 0
			w.adjPlayPosition.SetUpper(trackLen)
			w.adjPlayPosition.SetValue(trackPos)

			// Update position text
			if seekable {
				w.lblPosition.SetText(util.FormatSeconds(trackPos) + " / " + util.FormatSeconds(trackLen))
			}
		},
		// Disconnected - send a ping
		nil)

	// Enable the seek bar based on status
	w.scPlayPosition.SetSensitive(seekable)

	// If not seekable, remove position text
	if !seekable {
		w.lblPosition.SetText("")
	}
}
