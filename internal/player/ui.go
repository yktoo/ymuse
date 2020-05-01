package player

import (
	"fmt"
	"github.com/fhs/gompd/v2/mpd"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/yktoo/ymuse/internal/util"
	"strconv"
	"strings"
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
	// Library widgets
	bxLibraryPath *gtk.Box
	lbxLibrary    *gtk.ListBox
	// Playlists widgets
	lbxPlaylists *gtk.ListBox

	// Last reported MPD status
	mpdStatus mpd.Attrs

	// Playlist's track index (last) marked as current
	currentIndex int

	// Current library path, separated by slashes
	currentLibPath string

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
		// Library
		bxLibraryPath: builder.getBox("bxLibraryPath"),
		lbxLibrary:    builder.getListBox("lbxLibrary"),
		// Playlists
		lbxPlaylists: builder.getListBox("lbxPlaylists"),
	}

	// Map the handlers to callback functions
	builder.ConnectSignals(map[string]interface{}{
		"on_mainWindow_destroy":         w.onDestroy,
		"on_mainWindow_map":             w.onMap,
		"on_trvQueue_buttonPress":       w.onQueueTreeViewButtonPress,
		"on_trvQueue_keyPress":          w.onQueueTreeViewKeyPress,
		"on_lbxLibrary_buttonPress":     w.onLibraryListBoxButtonPress,
		"on_lbxLibrary_keyPress":        w.onLibraryListBoxKeyPress,
		"on_lbxPlaylists_buttonPress":   w.onPlaylistListBoxButtonPress,
		"on_lbxPlaylists_keyPress":      w.onPlaylistListBoxKeyPress,
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

// notImplemented() show a "function not implemented" message dialog
func (w *MainWindow) notImplemented() {
	dlg := gtk.MessageDialogNew(w.window, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, "Function not implemented.")
	dlg.Run()
	dlg.Destroy()
}

// addAction() add a new application action, with an optional keyboard shortcut
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
	case "database":
		whenIdle("updateLibrary()", w.updateLibrary, 0)
	case "options":
		whenIdle("updateOptions()", w.updateOptions, true)
	case "player":
		whenIdle("updatePlayer()", w.updatePlayer, true)
	case "playlist":
		whenIdle("updateQueue()", w.updateQueue)
	case "stored_playlist":
		whenIdle("updatePlaylists()", w.updatePlaylists)
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

func (w *MainWindow) onQueueTreeViewButtonPress(_ *gtk.TreeView, event *gdk.Event) {
	if gdk.EventButtonNewFromEvent(event).Type() == gdk.EVENT_DOUBLE_BUTTON_PRESS {
		// Double click in the tree
		w.applyQueueSelection()
	}
}

func (w *MainWindow) onQueueTreeViewKeyPress(_ *gtk.TreeView, event *gdk.Event) {
	if gdk.EventKeyNewFromEvent(event).KeyVal() == gdk.KEY_Return {
		// Enter key in the tree
		w.applyQueueSelection()
	}
}

func (w *MainWindow) onLibraryListBoxButtonPress(_ *gtk.ListBox, event *gdk.Event) {
	if gdk.EventButtonNewFromEvent(event).Type() == gdk.EVENT_DOUBLE_BUTTON_PRESS {
		// Double click in the list box
		w.applyLibrarySelection(util.GetConfig().TrackDefaultReplace)
	}
}

func (w *MainWindow) onLibraryListBoxKeyPress(_ *gtk.ListBox, event *gdk.Event) {
	ek := gdk.EventKeyNewFromEvent(event)
	switch ek.KeyVal() {
	// Enter: we need to go deeper
	case gdk.KEY_Return:
		w.applyLibrarySelection(util.GetConfig().TrackDefaultReplace)

	// Backspace: go level up
	case gdk.KEY_BackSpace:
		idx := strings.LastIndexByte(w.currentLibPath, '/')
		if idx < 0 {
			w.setLibraryPath("")
		} else {
			w.setLibraryPath(w.currentLibPath[:idx])
		}
	}
}

func (w *MainWindow) onPlaylistListBoxButtonPress(_ *gtk.ListBox, event *gdk.Event) {
	if gdk.EventButtonNewFromEvent(event).Type() == gdk.EVENT_DOUBLE_BUTTON_PRESS {
		// Double click in the list box
		w.applyPlaylistSelection(util.GetConfig().PlaylistDefaultReplace)
	}
}

func (w *MainWindow) onPlaylistListBoxKeyPress(_ *gtk.ListBox, event *gdk.Event) {
	ek := gdk.EventKeyNewFromEvent(event)
	if ek.KeyVal() == gdk.KEY_Return {
		w.applyPlaylistSelection(util.GetConfig().PlaylistDefaultReplace)
	}
}

func (w *MainWindow) onPreviousClicked() {
	log.Debug("onPreviousClicked()")
	w.connector.IfConnected(func(client *mpd.Client) {
		errCheck(client.Previous(), "Previous() failed")
	})
}

func (w *MainWindow) onStopClicked() {
	log.Debug("onStopClicked()")
	w.connector.IfConnected(func(client *mpd.Client) {
		errCheck(client.Stop(), "Stop() failed")
	})
}

func (w *MainWindow) onPlayPauseClicked() {
	log.Debug("onPlayPauseClicked()")
	w.connector.IfConnected(func(client *mpd.Client) {
		switch w.mpdStatus["state"] {
		case "pause":
			errCheck(client.Pause(false), "Pause(false) failed")
		case "play":
			errCheck(client.Pause(true), "Pause(true) failed")
		default:
			errCheck(client.Play(-1), "Play() failed")
		}
	})
}

func (w *MainWindow) onNextClicked() {
	log.Debug("onNextClicked()")
	w.connector.IfConnected(func(client *mpd.Client) {
		errCheck(client.Next(), "Next() failed")
	})
}

func (w *MainWindow) onRandomToggled() {
	if !w.optionsUpdating {
		log.Debug("onRandomToggled()")
		w.connector.IfConnected(func(client *mpd.Client) {
			errCheck(client.Random(w.mpdStatus["random"] == "0"), "Random() failed")
		})
	}
}

func (w *MainWindow) onRepeatToggled() {
	if !w.optionsUpdating {
		log.Debug("onRepeatToggled()")
		w.connector.IfConnected(func(client *mpd.Client) {
			errCheck(client.Repeat(w.mpdStatus["repeat"] == "0"), "Repeat() failed")
		})
	}
}

func (w *MainWindow) onConsumeToggled() {
	if !w.optionsUpdating {
		log.Debug("onConsumeToggled()")
		w.connector.IfConnected(func(client *mpd.Client) {
			errCheck(client.Consume(w.mpdStatus["consume"] == "0"), "Consume() failed")
		})
	}
}

func (w *MainWindow) onPlayPositionButtonEvent(_ interface{}, event *gdk.Event) {
	switch gdk.EventButtonNewFromEvent(event).Type() {
	case gdk.EVENT_BUTTON_PRESS:
		w.playPosUpdating = true

	case gdk.EVENT_BUTTON_RELEASE:
		w.playPosUpdating = false
		log.Debug("onPlayPositionButtonEvent()")
		w.connector.IfConnected(func(client *mpd.Client) {
			d := time.Duration(w.adjPlayPosition.GetValue())
			errCheck(client.SeekCur(d*time.Second, false), "SeekCur() failed")
		})
	}
}

// applyLibrarySelection() navigates into the folder or adds or replaces the content of the queue with the currently
// selected items in the library
func (w *MainWindow) applyLibrarySelection(replace bool) {
	// If there's selection
	row := w.lbxLibrary.GetSelectedRow()
	if row == nil {
		return
	}

	// Extract path, which is stored in the row's name
	s, err := row.GetName()
	if errCheck(err, "row.GetName() failed") {
		return
	}

	// Calculate final path
	path := w.currentLibPath
	if len(path) > 0 {
		path += "/"
	}
	path += s[2:]

	switch {
	// Directory - navigate inside it
	case strings.HasPrefix(s, "d:"):
		w.setLibraryPath(path)

	// File - append/replace the queue
	case strings.HasPrefix(s, "f:"):
		w.queueOne(replace, path)
	}
}

// applyPlaylistSelection() adds or replaces the content of the queue with the currently selected playlist
func (w *MainWindow) applyPlaylistSelection(replace bool) {
	// If there's selection
	row := w.lbxPlaylists.GetSelectedRow()
	if row == nil {
		return
	}

	// Extract playlist's name, which is stored in the row's name
	name, err := row.GetName()
	if errCheck(err, "row.GetName() failed") {
		return
	}

	// Queue the playlist
	w.queuePlaylist(replace, name)
}

// applyQueueSelection() starts playing from the currently selected track
func (w *MainWindow) applyQueueSelection() {
	// Get the tree's selection
	sel, err := w.trvQueue.GetSelection()
	if errCheck(err, "trvQueue.GetSelection() failed") {
		return
	}

	// Get selected node's index
	indices := sel.GetSelectedRows(nil).Data().(*gtk.TreePath).GetIndices()
	if len(indices) == 0 {
		return
	}

	// Start playback from the given index
	w.connector.IfConnected(func(client *mpd.Client) {
		errCheck(client.Play(indices[0]), "Play() failed")
	})
}

// fetchStatus() updates stored MPD's status info
func (w *MainWindow) fetchStatus() {
	// Provide an empty map as fallback
	w.mpdStatus = mpd.Attrs{}

	// Request player status if there's a connection
	w.connector.IfConnected(func(client *mpd.Client) {
		status, err := client.Status()
		if errCheck(err, "Status() failed") {
			return
		}
		w.mpdStatus = status
	})
}

// queue() adds or replaces the content of the queue with the specified URIs
func (w *MainWindow) queue(replace bool, uris []string) {
	w.connector.IfConnected(func(client *mpd.Client) {
		commands := client.BeginCommandList()

		// Clear the queue, if needed
		if replace {
			commands.Clear()
		}

		// Add the URIs
		for _, uri := range uris {
			commands.Add(uri)
		}

		// Run the commands
		errCheck(commands.End(), "CommandList execution failed")
	})
}

// queueOne() adds or replaces the content of the queue with one specified URI
func (w *MainWindow) queueOne(replace bool, uri string) {
	w.queue(replace, []string{uri})
}

// queuePlaylist() adds or replaces the content of the queue with the specified playlist
func (w *MainWindow) queuePlaylist(replace bool, playlistName string) {
	w.connector.IfConnected(func(client *mpd.Client) {
		commands := client.BeginCommandList()

		// Clear the queue, if needed
		if replace {
			commands.Clear()
		}

		// Add the content of the playlist
		commands.PlaylistLoad(playlistName, -1, -1)

		// Run the commands
		errCheck(commands.End(), "CommandList execution failed")
	})
}

// Show() shows the window and all its child widgets
func (w *MainWindow) Show() {
	w.window.ShowAll()
}

// setLibraryPath() sets the current library path selector and updates its widget and the current library list
func (w *MainWindow) setLibraryPath(path string) {
	w.currentLibPath = path
	w.updateLibraryPath()
	w.updateLibrary(0)
	w.lbxLibrary.GrabFocus()
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

// updateAll() updates all player's widgets and lists
func (w *MainWindow) updateAll() {
	w.fetchStatus()
	w.updateQueue()
	w.updateLibraryPath()
	w.updateLibrary(0)
	w.updatePlaylists()
	w.updateOptions(false)
	w.updatePlayer(false)
}

// updateLibrary() updates the current library list contents
func (w *MainWindow) updateLibrary(indexToSelect int) {
	// Clear the library list
	clearChildren(w.lbxLibrary.Container)

	// Update the library list if there's a connection
	w.connector.IfConnected(func(client *mpd.Client) {
		// Fetch the current library content
		attrs, err := client.ListInfo(w.currentLibPath)
		if errCheck(err, "ListInfo() failed") {
			return
		}

		// pathPrefix will need to be removed from element names
		pathPrefix := w.currentLibPath + "/"

		// Repopulate the library list
		var rowToSelect *gtk.ListBoxRow
		idxRow := 0
		for _, a := range attrs {
			// Pick files and directories only
			uri, iconName, prefix := "", "", ""
			if dir, ok := a["directory"]; ok {
				uri = dir
				iconName = "folder"
				prefix = "d:"
			} else if file, ok := a["file"]; ok {
				uri = file
				iconName = "audio-x-generic"
				prefix = "f:"
			} else {
				continue
			}

			// Add a new list box row
			name := strings.TrimPrefix(uri, pathPrefix)
			row, hbx, err := newListBoxRow(w.lbxLibrary, name, prefix+name, iconName)
			if errCheck(err, "newListBoxRow() failed") {
				return
			}
			if indexToSelect == idxRow {
				rowToSelect = row
			}

			// Add replace/append buttons
			hbx.PackEnd(newButton("", "Append to the queue", "", "list-add", func() { w.queueOne(false, uri) }), false, false, 0)
			hbx.PackEnd(newButton("", "Replace the queue", "", "edit-paste", func() { w.queueOne(true, uri) }), false, false, 0)

			// Add a label with track length, if any
			if secs := util.ParseFloatDef(a["duration"], 0); secs > 0 {
				lbl, err := gtk.LabelNew(util.FormatSeconds(secs))
				if errCheck(err, "LabelNew() failed") {
					return
				}
				hbx.PackEnd(lbl, false, false, 0)
			}
			idxRow++
		}

		// Show all rows
		w.lbxLibrary.ShowAll()

		// Select the required row
		if rowToSelect != nil {
			w.lbxLibrary.SelectRow(rowToSelect)
		}
	})
}

// updateLibraryPath() updates the current library path selector
func (w *MainWindow) updateLibraryPath() {
	// Remove all buttons from the box
	clearChildren(w.bxLibraryPath.Container)

	// Create buttons if there's a connection
	w.connector.IfConnected(func(client *mpd.Client) {
		// Create a button for "root"
		newBoxToggleButton(w.bxLibraryPath, "Files", "", "drive-harddisk", w.currentLibPath == "", func() { w.setLibraryPath("") })

		// Create buttons for path elements
		if len(w.currentLibPath) > 0 {
			path := ""
			for i, s := range strings.Split(w.currentLibPath, "/") {
				// Accumulate path
				if i > 0 {
					path += "/"
				}
				path += s

				// Create a local (in-loop) copy of path to use in the click event closure below
				pathCopy := path

				// Create a button. The last button must be depressed
				newBoxToggleButton(w.bxLibraryPath, s, "", "folder", path == w.currentLibPath, func() { w.setLibraryPath(pathCopy) })
			}
		}

		// Show all buttons
		w.bxLibraryPath.ShowAll()
	})
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
	w.connector.IfConnectedElse(
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

// updatePlaylists() updates the current playlists list contents
func (w *MainWindow) updatePlaylists() {
	// Clear the playlists list
	clearChildren(w.lbxPlaylists.Container)

	// Update playlists if there's a connection
	w.connector.IfConnected(func(client *mpd.Client) {
		// Fetch the current playlist
		attrs, err := client.ListPlaylists()
		if errCheck(err, "ListPlaylists() failed") {
			return
		}

		// Repopulate the playlists list
		for _, a := range attrs {
			name := a["playlist"]
			_, hbx, err := newListBoxRow(w.lbxPlaylists, name, name, "format-justify-left")
			if errCheck(err, "newListBoxRow() failed") {
				return
			}

			// Add replace/append buttons
			hbx.PackEnd(newButton("", "Append to the queue", "", "list-add", func() { w.queuePlaylist(false, name) }), false, false, 0)
			hbx.PackEnd(newButton("", "Replace the queue", "", "edit-paste", func() { w.queuePlaylist(true, name) }), false, false, 0)
		}

		// Show all rows
		w.lbxPlaylists.ShowAll()
	})
}

// updateQueue() updates the current play queue contents
func (w *MainWindow) updateQueue() {
	// Clear the queue
	w.lstQueue.Clear()

	// Update the queue if there's a connection
	status := ""
	w.connector.IfConnected(func(client *mpd.Client) {
		// Fetch the current playlist
		attrs, err := client.PlaylistInfo(-1, -1)
		if errCheck(err, "PlaylistInfo() failed") {
			return
		}

		// Repopulate the queue tree view
		totalSecs := 0.0
		w.currentIndex = -1
		for _, a := range attrs {
			secs := util.ParseFloatDef(a["duration"], 0)

			// Prepare row values
			rowData := map[int]interface{}{
				ColQueue_Artist:     a["Artist"],
				ColQueue_Year:       a["Date"],
				ColQueue_Album:      a["Album"],
				ColQueue_Number:     a["Track"],
				ColQueue_Track:      a["Title"],
				ColQueue_FontWeight: FontWeightNormal,
				ColQueue_BgColor:    BackgroundColorNormal,
			}

			// Add duration, if any
			if secs > 0 {
				rowData[ColQueue_Length] = util.FormatSeconds(secs)
			}

			// Add a row to the tree view
			errCheck(
				w.lstQueue.SetCols(w.lstQueue.Append(), rowData),
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
	})

	// Update the queue info
	w.lblQueueInfo.SetText(status)
}

// updateSeekBar() updates the seek bar position and status
func (w *MainWindow) updateSeekBar() {
	// Ignore if the user is dragging the slider manually
	if w.playPosUpdating {
		return
	}

	// Update the seek bar position if there's a connection
	seekable := false
	w.connector.IfConnected(func(client *mpd.Client) {
		trackLen := util.ParseFloatDef(w.mpdStatus["duration"], -1)
		trackPos := util.ParseFloatDef(w.mpdStatus["elapsed"], -1)
		seekable = trackLen >= 0 && trackPos >= 0
		w.adjPlayPosition.SetUpper(trackLen)
		w.adjPlayPosition.SetValue(trackPos)

		// Update position text
		if seekable {
			w.lblPosition.SetText(util.FormatSeconds(trackPos) + " / " + util.FormatSeconds(trackLen))
		}
	})

	// Enable the seek bar based on status
	w.scPlayPosition.SetSensitive(seekable)

	// If not seekable, remove position text
	if !seekable {
		w.lblPosition.SetText("")
	}
}
