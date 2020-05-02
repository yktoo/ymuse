package player

import (
	"fmt"
	"github.com/fhs/gompd/v2/mpd"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/yktoo/ymuse/internal/util"
	"path"
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
	btnPlayPause    *gtk.ToolButton
	btnRandom       *gtk.ToggleToolButton
	btnRepeat       *gtk.ToggleToolButton
	btnConsume      *gtk.ToggleToolButton
	scPlayPosition  *gtk.Scale
	adjPlayPosition *gtk.Adjustment
	// Queue widgets
	lblQueueInfo *gtk.Label
	trvQueue     *gtk.TreeView
	lstQueue     *gtk.ListStore
	pmnQueueSort *gtk.PopoverMenu
	// Library widgets
	bxLibraryPath *gtk.Box
	lbxLibrary    *gtk.ListBox
	// Playlists widgets
	lbxPlaylists *gtk.ListBox

	// Actions
	aQueueNowPlaying *glib.SimpleAction
	aQueueClear      *glib.SimpleAction
	aQueueSort       *glib.SimpleAction
	aPlayerPrevious  *glib.SimpleAction
	aPlayerStop      *glib.SimpleAction
	aPlayerPlayPause *glib.SimpleAction
	aPlayerNext      *glib.SimpleAction
	aPlayerRandom    *glib.SimpleAction
	aPlayerRepeat    *glib.SimpleAction
	aPlayerConsume   *glib.SimpleAction

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

func NewMainWindow(application *gtk.Application) (*MainWindow, error) {
	// Set up the window
	builder := NewBuilder("internal/player/player.glade")

	w := &MainWindow{
		app: application,
		// Find widgets
		window:          builder.getApplicationWindow("mainWindow"),
		lblStatus:       builder.getLabel("lblStatus"),
		lblPosition:     builder.getLabel("lblPosition"),
		btnPlayPause:    builder.getToolButton("btnPlayPause"),
		btnRandom:       builder.getToggleToolButton("btnRandom"),
		btnRepeat:       builder.getToggleToolButton("btnRepeat"),
		btnConsume:      builder.getToggleToolButton("btnConsume"),
		scPlayPosition:  builder.getScale("scPlayPosition"),
		adjPlayPosition: builder.getAdjustment("adjPlayPosition"),
		// Queue
		lblQueueInfo: builder.getLabel("lblQueueInfo"),
		trvQueue:     builder.getTreeView("trvQueue"),
		lstQueue:     builder.getListStore("lstQueue"),
		pmnQueueSort: builder.getPopoverMenu("pmnQueueSort"),
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
		"on_scPlayPosition_buttonEvent": w.onPlayPositionButtonEvent,
	})

	// Register the main window with the app
	application.AddWindow(w.window)

	// Instantiate a connector
	w.connector = NewConnector(w.onConnectorConnected, w.onConnectorHeartbeat, w.onConnectorSubsystemChange)
	return w, nil
}

// notImplemented() show a "function not implemented" message dialog
func (w *MainWindow) notImplemented() {
	dlg := gtk.MessageDialogNew(w.window, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, "Function not implemented.")
	dlg.Run()
	dlg.Destroy()
}

// addAction() add a new application action, with an optional keyboard shortcut
func (w *MainWindow) addAction(name, shortcut string, onActivate interface{}) *glib.SimpleAction {
	action := glib.SimpleActionNew(name, nil)
	if _, err := action.Connect("activate", onActivate); err != nil {
		log.Fatalf("Failed to connect activate signal of action '%v': %v", name, err)
	}
	w.app.AddAction(action)
	if shortcut != "" {
		w.app.SetAccelsForAction("app."+name, []string{shortcut})
	}
	return action
}

func (w *MainWindow) onConnectorConnected() {
	util.WhenIdle("onConnectorConnected()", w.updateAll)
}

func (w *MainWindow) onConnectorHeartbeat() {
	util.WhenIdle("onConnectorHeartbeat()", w.updateSeekBar)
}

func (w *MainWindow) onConnectorSubsystemChange(subsystem string) {
	log.Debugf("onSubsystemChange(%v)", subsystem)
	switch subsystem {
	case "database":
		util.WhenIdle("updateLibrary()", w.updateLibrary, 0)
	case "options":
		util.WhenIdle("updateOptions()", w.updateOptions)
	case "player":
		util.WhenIdle("updatePlayer()", w.updatePlayer)
	case "playlist":
		util.WhenIdle("updateQueue()", func() {
			w.updateQueue()
			w.updatePlayer()
		})
	case "stored_playlist":
		util.WhenIdle("updatePlaylists()", w.updatePlaylists)
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

	// Create actions
	// Application
	w.addAction("about", "F1", w.onAbout)
	w.addAction("prefs", "<Ctrl>comma", w.notImplemented)
	w.addAction("quit", "<Ctrl>Q", w.window.Close)
	// Queue
	w.aQueueNowPlaying = w.addAction("queue.now-playing", "<Ctrl>J", w.updateQueueNowPlaying)
	w.aQueueClear = w.addAction("queue.clear", "<Ctrl>Delete", w.connector.QueueClear)
	w.aQueueSort = w.addAction("queue.sort", "", w.pmnQueueSort.Popup)
	w.addAction("queue.sort.artist.asc", "", func() { w.connector.QueueSort("Artist", false, false) })
	w.addAction("queue.sort.artist.desc", "", func() { w.connector.QueueSort("Artist", false, true) })
	w.addAction("queue.sort.album.asc", "", func() { w.connector.QueueSort("Album", false, false) })
	w.addAction("queue.sort.album.desc", "", func() { w.connector.QueueSort("Album", false, true) })
	w.addAction("queue.sort.title.asc", "", func() { w.connector.QueueSort("Title", false, false) })
	w.addAction("queue.sort.title.desc", "", func() { w.connector.QueueSort("Title", false, true) })
	w.addAction("queue.sort.number.asc", "", func() { w.connector.QueueSort("Track", true, false) })
	w.addAction("queue.sort.number.desc", "", func() { w.connector.QueueSort("Track", true, true) })
	w.addAction("queue.sort.length.asc", "", func() { w.connector.QueueSort("duration", true, false) })
	w.addAction("queue.sort.length.desc", "", func() { w.connector.QueueSort("duration", true, true) })
	w.addAction("queue.sort.fullpath.asc", "", func() { w.connector.QueueSort("file", false, false) })
	w.addAction("queue.sort.fullpath.desc", "", func() { w.connector.QueueSort("file", false, true) })
	w.addAction("queue.sort.year.asc", "", func() { w.connector.QueueSort("Date", true, false) })
	w.addAction("queue.sort.year.desc", "", func() { w.connector.QueueSort("Date", true, true) })
	w.addAction("queue.sort.genre.asc", "", func() { w.connector.QueueSort("Genre", false, false) })
	w.addAction("queue.sort.genre.desc", "", func() { w.connector.QueueSort("Genre", false, true) })
	w.addAction("queue.sort.shuffle", "", w.connector.QueueShuffle)
	// Player
	w.aPlayerPrevious = w.addAction("player.previous", "<Ctrl>Left", w.connector.PlayerPrevious)
	w.aPlayerStop = w.addAction("player.stop", "<Ctrl>S", w.connector.PlayerStop)
	w.aPlayerPlayPause = w.addAction("player.play-pause", "<Ctrl>P", w.connector.PlayerPlayPause)
	w.aPlayerNext = w.addAction("player.next", "<Ctrl>Right", w.connector.PlayerNext)
	// TODO convert to stateful actions once Gotk3 supporting GVariant is released
	w.aPlayerRandom = w.addAction("player.toggle.random", "<Ctrl>U", func() {
		if !w.optionsUpdating {
			w.connector.PlayerToggleRandom()
		}
	})
	w.aPlayerRepeat = w.addAction("player.toggle.repeat", "<Ctrl>R", func() {
		if !w.optionsUpdating {
			w.connector.PlayerToggleRepeat()
		}
	})
	w.aPlayerConsume = w.addAction("player.toggle.consume", "<Ctrl>N", func() {
		if !w.optionsUpdating {
			w.connector.PlayerToggleConsume()
		}
	})

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
	libPath := w.currentLibPath
	if len(libPath) > 0 {
		libPath += "/"
	}
	libPath += s[2:]

	switch {
	// Directory - navigate inside it
	case strings.HasPrefix(s, "d:"):
		w.setLibraryPath(libPath)

	// File - append/replace the queue
	case strings.HasPrefix(s, "f:"):
		w.queueOne(replace, libPath)
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

// setQueueHighlight() selects or deselects an item in the Queue tree view at the given index
func (w *MainWindow) setQueueHighlight(index int, selected bool) {
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
	w.updateQueue()
	w.updateLibraryPath()
	w.updateLibrary(0)
	w.updatePlaylists()
	w.updateOptions()
	w.updatePlayer()
	w.updateSeekBar()
}

// updateLibrary() updates the current library list contents
func (w *MainWindow) updateLibrary(indexToSelect int) {
	// Clear the library list
	util.ClearChildren(w.lbxLibrary.Container)

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
			row, hbx, err := util.NewListBoxRow(w.lbxLibrary, name, prefix+name, iconName)
			if errCheck(err, "NewListBoxRow() failed") {
				return
			}
			if indexToSelect == idxRow {
				rowToSelect = row
			}

			// Add replace/append buttons
			hbx.PackEnd(util.NewButton("", "Append to the queue", "", "list-add", func() { w.queueOne(false, uri) }), false, false, 0)
			hbx.PackEnd(util.NewButton("", "Replace the queue", "", "edit-paste", func() { w.queueOne(true, uri) }), false, false, 0)

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
	util.ClearChildren(w.bxLibraryPath.Container)

	// Create buttons if there's a connection
	w.connector.IfConnected(func(client *mpd.Client) {
		// Create a button for "root"
		util.NewBoxToggleButton(w.bxLibraryPath, "Files", "", "drive-harddisk", w.currentLibPath == "", func() { w.setLibraryPath("") })

		// Create buttons for path elements
		if len(w.currentLibPath) > 0 {
			libPath := ""
			for i, s := range strings.Split(w.currentLibPath, "/") {
				// Accumulate path
				if i > 0 {
					libPath += "/"
				}
				libPath += s

				// Create a local (in-loop) copy of libPath to use in the click event closure below
				pathCopy := libPath

				// Create a button. The last button must be depressed
				util.NewBoxToggleButton(w.bxLibraryPath, s, "", "folder", libPath == w.currentLibPath, func() { w.setLibraryPath(pathCopy) })
			}
		}

		// Show all buttons
		w.bxLibraryPath.ShowAll()
	})
}

// updateOptions() updates player options widgets
func (w *MainWindow) updateOptions() {
	w.optionsUpdating = true
	status := w.connector.Status()
	w.btnRandom.SetActive(status["random"] == "1")
	w.btnRepeat.SetActive(status["repeat"] == "1")
	w.btnConsume.SetActive(status["consume"] == "1")
	w.optionsUpdating = false
}

// updatePlayer() updates player control widgets
func (w *MainWindow) updatePlayer() {
	connected := true
	w.connector.IfConnectedElse(
		// Connected
		func(client *mpd.Client) {
			// Update play/pause button's appearance
			status := w.connector.Status()
			switch status["state"] {
			case "play":
				w.btnPlayPause.SetIconName("media-playback-pause")
			default:
				w.btnPlayPause.SetIconName("media-playback-start")
			}

			// Fetch current song
			curSong, err := client.CurrentSong()
			var str string
			if errCheck(err, "CurrentSong() failed") {
				str = fmt.Sprintf("Error: %v", err)
			} else {
				// Try to determine track's display name. First check artist/album/title
				log.Debugf("Current track: %+v", curSong)
				artist, okArtist := curSong["Artist"]
				album, okAlbum := curSong["Album"]
				title, okTitle := curSong["Title"]
				if okArtist || okAlbum || okTitle {
					str = fmt.Sprintf("%v • %v • %v", artist, album, title)

				} else if name, ok := curSong["Name"]; ok {
					// Next, check name
					str = name

				} else if file, ok := curSong["file"]; ok {
					// Then use the file's base name
					str = path.Base(file)

				} else {
					// All failed
					str = "(unknown)"
				}
			}
			w.lblStatus.SetText(str)
		},
		// Disconnected
		func() {
			w.lblStatus.SetText("(not connected)")
			connected = false
		})

	// Highlight and scroll the tree to the currently played item
	w.updateQueueNowPlaying()

	// Enable or disable widgets based on the connection status
	w.aQueueNowPlaying.SetEnabled(connected)
	w.aQueueClear.SetEnabled(connected)
	w.aQueueSort.SetEnabled(connected)
	w.aPlayerPrevious.SetEnabled(connected)
	w.aPlayerStop.SetEnabled(connected)
	w.aPlayerPlayPause.SetEnabled(connected)
	w.aPlayerNext.SetEnabled(connected)
	w.aPlayerRandom.SetEnabled(connected)
	w.aPlayerRepeat.SetEnabled(connected)
	w.aPlayerConsume.SetEnabled(connected)
}

// updatePlaylists() updates the current playlists list contents
func (w *MainWindow) updatePlaylists() {
	// Clear the playlists list
	util.ClearChildren(w.lbxPlaylists.Container)

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
			_, hbx, err := util.NewListBoxRow(w.lbxPlaylists, name, name, "format-justify-left")
			if errCheck(err, "NewListBoxRow() failed") {
				return
			}

			// Add replace/append buttons
			hbx.PackEnd(util.NewButton("", "Append to the queue", "", "list-add", func() { w.queuePlaylist(false, name) }), false, false, 0)
			hbx.PackEnd(util.NewButton("", "Replace the queue", "", "edit-paste", func() { w.queuePlaylist(true, name) }), false, false, 0)
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

	// Highlight and scroll the tree to the currently played item
	w.updateQueueNowPlaying()

	// Update the queue info
	w.lblQueueInfo.SetText(status)
}

// updateQueueNowPlaying() scrolls the queue tree view to the currently played track
func (w *MainWindow) updateQueueNowPlaying() {
	// Update queue highlight
	if curIdx := util.AtoiDef(w.connector.Status()["song"], -1); w.currentIndex != curIdx {
		w.setQueueHighlight(w.currentIndex, false)
		w.setQueueHighlight(curIdx, true)
		w.currentIndex = curIdx
	}

	// Scroll to the currently playing
	if w.currentIndex >= 0 {
		if treePath, err := gtk.TreePathNewFromString(strconv.Itoa(w.currentIndex)); err == nil {
			w.trvQueue.ScrollToCell(treePath, nil, true, 0.5, 0)
		}
	}
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
		status := w.connector.Status()
		trackLen := util.ParseFloatDef(status["duration"], -1)
		trackPos := util.ParseFloatDef(status["elapsed"], -1)
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
