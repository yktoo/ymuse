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

package player

import "C"
import (
	"bytes"
	"fmt"
	"github.com/fhs/gompd/v2/mpd"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
	"github.com/yktoo/ymuse/internal/config"
	"github.com/yktoo/ymuse/internal/generated"
	"github.com/yktoo/ymuse/internal/util"
	"html"
	"html/template"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

// MainWindow represents the main application window
type MainWindow struct {
	// Application reference
	app *gtk.Application
	// Connector instance
	connector *Connector
	// Main window
	window *gtk.ApplicationWindow
	// Control widgets
	mainStack       *gtk.Stack
	lblStatus       *gtk.Label
	lblPosition     *gtk.Label
	btnPlayPause    *gtk.ToolButton
	btnRandom       *gtk.ToggleToolButton
	btnRepeat       *gtk.ToggleToolButton
	btnConsume      *gtk.ToggleToolButton
	scPlayPosition  *gtk.Scale
	adjPlayPosition *gtk.Adjustment
	// Queue widgets
	bxQueue          *gtk.Box
	lblQueueInfo     *gtk.Label
	trvQueue         *gtk.TreeView
	pmnQueueSort     *gtk.PopoverMenu
	pmnQueueSave     *gtk.PopoverMenu
	mnQueue          *gtk.Menu
	btnQueueFilter   *gtk.ToggleToolButton
	queueSearchBar   *gtk.SearchBar
	queueSearchEntry *gtk.SearchEntry
	lblQueueFilter   *gtk.Label
	queueListStore   *gtk.ListStore
	queueListFilter  *gtk.TreeModelFilter
	// Queue sort popup
	cbxQueueSortBy *gtk.ComboBoxText
	// Queue save popup
	cbxQueueSavePlaylist     *gtk.ComboBoxText
	lblQueueSavePlaylistName *gtk.Label
	eQueueSavePlaylistName   *gtk.Entry
	cbQueueSaveSelectedOnly  *gtk.CheckButton
	// Library widgets
	pmnLibraryUpdate *gtk.PopoverMenu
	bxLibrary        *gtk.Box
	bxLibraryPath    *gtk.Box
	lbxLibrary       *gtk.ListBox
	lblLibraryInfo   *gtk.Label
	// Playlists widgets
	bxPlaylists      *gtk.Box
	lbxPlaylists     *gtk.ListBox
	lblPlaylistsInfo *gtk.Label

	// Actions
	aQueueNowPlaying  *glib.SimpleAction
	aQueueClear       *glib.SimpleAction
	aQueueSort        *glib.SimpleAction
	aQueueSortAsc     *glib.SimpleAction
	aQueueSortDesc    *glib.SimpleAction
	aQueueSortShuffle *glib.SimpleAction
	aQueueDelete      *glib.SimpleAction
	aQueueSave        *glib.SimpleAction
	aQueueSaveReplace *glib.SimpleAction
	aQueueSaveAppend  *glib.SimpleAction
	aLibraryUpdate    *glib.SimpleAction
	aLibraryUpdateAll *glib.SimpleAction
	aLibraryUpdateSel *glib.SimpleAction
	aLibraryRescanAll *glib.SimpleAction
	aLibraryRescanSel *glib.SimpleAction
	aPlaylistRename   *glib.SimpleAction
	aPlaylistDelete   *glib.SimpleAction
	aPlayerPrevious   *glib.SimpleAction
	aPlayerStop       *glib.SimpleAction
	aPlayerPlayPause  *glib.SimpleAction
	aPlayerNext       *glib.SimpleAction
	aPlayerRandom     *glib.SimpleAction
	aPlayerRepeat     *glib.SimpleAction
	aPlayerConsume    *glib.SimpleAction

	currentQueueSize  int // Number of items in the play queue
	currentQueueIndex int // Queue's track index (last) marked as current

	currentLibPath string // Current library path, separated by slashes

	playerTitleTemplate *template.Template // Compiled template for player's track title

	playPosUpdating bool // Play position manual update flag
	optionsUpdating bool // Options update flag
}

const (
	// Rendering properties for the Queue list
	fontWeightNormal = 400
	fontWeightBold   = 700
	colorBgNormal    = "#ffffff"
	colorBgActive    = "#ffffe0"

	queueSaveNewPlaylistID = "\u0001new"
)

// NewMainWindow creates and returns a new MainWindow instance
func NewMainWindow(application *gtk.Application) (*MainWindow, error) {
	// Set up the window
	builder := NewBuilder(generated.GetPlayerGlade())

	w := &MainWindow{
		app: application,
		// Find widgets
		window:          builder.getApplicationWindow("mainWindow"),
		mainStack:       builder.getStack("mainStack"),
		lblStatus:       builder.getLabel("lblStatus"),
		lblPosition:     builder.getLabel("lblPosition"),
		btnPlayPause:    builder.getToolButton("btnPlayPause"),
		btnRandom:       builder.getToggleToolButton("btnRandom"),
		btnRepeat:       builder.getToggleToolButton("btnRepeat"),
		btnConsume:      builder.getToggleToolButton("btnConsume"),
		scPlayPosition:  builder.getScale("scPlayPosition"),
		adjPlayPosition: builder.getAdjustment("adjPlayPosition"),
		// Queue
		bxQueue:          builder.getBox("bxQueue"),
		lblQueueInfo:     builder.getLabel("lblQueueInfo"),
		trvQueue:         builder.getTreeView("trvQueue"),
		pmnQueueSort:     builder.getPopoverMenu("pmnQueueSort"),
		pmnQueueSave:     builder.getPopoverMenu("pmnQueueSave"),
		mnQueue:          builder.getMenu("mnQueue"),
		btnQueueFilter:   builder.getToggleToolButton("btnQueueFilter"),
		queueSearchBar:   builder.getSearchBar("queueSearchBar"),
		queueSearchEntry: builder.getSearchEntry("queueSearchEntry"),
		lblQueueFilter:   builder.getLabel("lblQueueFilter"),
		queueListStore:   builder.getListStore("queueListStore"),
		queueListFilter:  builder.getTreeModelFilter("queueListFilter"),
		// Queue sort popup
		cbxQueueSortBy: builder.getComboBoxText("cbxQueueSortBy"),
		// Queue save popup
		cbxQueueSavePlaylist:     builder.getComboBoxText("cbxQueueSavePlaylist"),
		lblQueueSavePlaylistName: builder.getLabel("lblQueueSavePlaylistName"),
		eQueueSavePlaylistName:   builder.getEntry("eQueueSavePlaylistName"),
		cbQueueSaveSelectedOnly:  builder.getCheckButton("cbQueueSaveSelectedOnly"),
		// Library
		pmnLibraryUpdate: builder.getPopoverMenu("pmnLibraryUpdate"),
		bxLibrary:        builder.getBox("bxLibrary"),
		bxLibraryPath:    builder.getBox("bxLibraryPath"),
		lbxLibrary:       builder.getListBox("lbxLibrary"),
		lblLibraryInfo:   builder.getLabel("lblLibraryInfo"),
		// Playlists
		bxPlaylists:      builder.getBox("bxPlaylists"),
		lbxPlaylists:     builder.getListBox("lbxPlaylists"),
		lblPlaylistsInfo: builder.getLabel("lblPlaylistsInfo"),
	}

	// Initialise queue filter model
	w.queueListFilter.SetVisibleColumn(config.QueueColumnVisible)

	// Initialise player title template
	w.updatePlayerTitleTemplate()

	// Map the handlers to callback functions
	builder.ConnectSignals(map[string]interface{}{
		"on_mainWindow_delete":              w.onDelete,
		"on_mainWindow_map":                 w.onMap,
		"on_trvQueue_buttonPress":           w.onQueueTreeViewButtonPress,
		"on_trvQueue_keyPress":              w.onQueueTreeViewKeyPress,
		"on_tselQueue_changed":              w.updateQueueActions,
		"on_queueSearchBar_searchMode":      w.queueFilter,
		"on_queueSearchEntry_searchChanged": w.queueFilter,
		"on_lbxLibrary_buttonPress":         w.onLibraryListBoxButtonPress,
		"on_lbxLibrary_keyPress":            w.onLibraryListBoxKeyPress,
		"on_lbxLibrary_selectionChange":     w.updateLibraryActions,
		"on_lbxPlaylists_buttonPress":       w.onPlaylistListBoxButtonPress,
		"on_lbxPlaylists_keyPress":          w.onPlaylistListBoxKeyPress,
		"on_lbxPlaylists_selectionChange":   w.updatePlaylistsActions,
		"on_pmnQueueSave_validate":          w.onQueueSavePopoverValidate,
		"on_scPlayPosition_buttonEvent":     w.onPlayPositionButtonEvent,
		"on_scPlayPosition_valueChanged":    w.updatePlayerSeekBar,

		// For some reason binding actions to menu items keeps them grayed out, so old-school signals are used here
		"on_miQueueNowPlaying_activate": w.updateQueueNowPlaying,
		"on_miQueueClear_activate":      w.queueClear,
		"on_miQueueDelete_activate":     w.queueDelete,
	})

	// Configure search bar
	glib.BindProperty(w.queueSearchBar.Object, "search-mode-enabled", w.btnQueueFilter.Object, "active", glib.BINDING_BIDIRECTIONAL)
	glib.BindProperty(w.queueSearchBar.Object, "search-mode-enabled", w.lblQueueFilter.Object, "visible", glib.BINDING_DEFAULT)

	// Forcefully disable tree search popup on Ctrl+F
	w.trvQueue.SetSearchColumn(-1)

	// Register the main window with the app
	application.AddWindow(w.window)

	// Restore window dimensions
	dim := config.GetConfig().MainWindowDimensions
	if dim.Width > 0 && dim.Height > 0 {
		w.window.Resize(dim.Width, dim.Height)
	}
	if dim.X >= 0 && dim.Y >= 0 {
		w.window.Move(dim.X, dim.Y)
	}

	// Instantiate a connector
	w.connector = NewConnector(w.onConnectorStatusChange, w.onConnectorHeartbeat, w.onConnectorSubsystemChange)
	return w, nil
}

func (w *MainWindow) onConnectorStatusChange() {
	util.WhenIdle("onConnectorStatusChange()", w.updateAll)
}

func (w *MainWindow) onConnectorHeartbeat() {
	util.WhenIdle("onConnectorHeartbeat()", w.updatePlayerSeekBar)
}

func (w *MainWindow) onConnectorSubsystemChange(subsystem string) {
	log.Debugf("onSubsystemChange(%v)", subsystem)
	switch subsystem {
	case "database", "update":
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

func (w *MainWindow) onMap() {
	log.Debug("MainWindow.onMap()")
	cfg := config.GetConfig()

	// Create actions
	// Application
	w.addAction("mpd.connect", "<Ctrl><Shift>C", w.connect)
	w.addAction("mpd.disconnect", "<Ctrl><Shift>D", w.disconnect)
	w.addAction("prefs", "<Ctrl>comma", w.preferences)
	w.addAction("about", "F1", w.about)
	w.addAction("shortcuts", "<Ctrl><Shift>question", w.shortcutInfo)
	w.addAction("quit", "<Ctrl>Q", w.window.Close)
	w.addAction("page.queue", "<Ctrl>1", func() { w.mainStack.SetVisibleChild(w.bxQueue) })
	w.addAction("page.library", "<Ctrl>2", func() { w.mainStack.SetVisibleChild(w.bxLibrary) })
	w.addAction("page.playlists", "<Ctrl>3", func() { w.mainStack.SetVisibleChild(w.bxPlaylists) })
	// Queue
	w.aQueueNowPlaying = w.addAction("queue.now-playing", "<Ctrl>J", w.updateQueueNowPlaying)
	w.aQueueClear = w.addAction("queue.clear", "", w.queueClear)
	w.aQueueSort = w.addAction("queue.sort", "", w.pmnQueueSort.Popup)
	w.aQueueSortAsc = w.addAction("queue.sort.asc", "", func() { w.queueSortApply(false) })
	w.aQueueSortDesc = w.addAction("queue.sort.desc", "", func() { w.queueSortApply(true) })
	w.aQueueSortShuffle = w.addAction("queue.sort.shuffle", "", w.queueShuffle)
	w.aQueueDelete = w.addAction("queue.delete", "", w.queueDelete)
	w.aQueueSave = w.addAction("queue.save", "", w.queueSave)
	w.aQueueSaveReplace = w.addAction("queue.save.replace", "", func() { w.queueSaveApply(true) })
	w.aQueueSaveAppend = w.addAction("queue.save.append", "", func() { w.queueSaveApply(false) })
	// Library
	w.aLibraryUpdate = w.addAction("library.update", "", w.pmnLibraryUpdate.Popup)
	w.aLibraryUpdateAll = w.addAction("library.update.all", "", func() { w.libraryUpdate(false, false) })
	w.aLibraryUpdateSel = w.addAction("library.update.selected", "", func() { w.libraryUpdate(false, true) })
	w.aLibraryRescanAll = w.addAction("library.rescan.all", "", func() { w.libraryUpdate(true, false) })
	w.aLibraryRescanSel = w.addAction("library.rescan.selected", "", func() { w.libraryUpdate(true, true) })
	// Playlist
	w.aPlaylistRename = w.addAction("playlist.rename", "", w.onPlaylistRename)
	w.aPlaylistDelete = w.addAction("playlist.delete", "", w.onPlaylistDelete)
	// Player
	w.aPlayerPrevious = w.addAction("player.previous", "<Ctrl>Left", w.playerPrevious)
	w.aPlayerStop = w.addAction("player.stop", "<Ctrl>S", w.playerStop)
	w.aPlayerPlayPause = w.addAction("player.play-pause", "<Ctrl>P", w.playerPlayPause)
	w.aPlayerNext = w.addAction("player.next", "<Ctrl>Right", w.playerNext)
	// TODO convert to stateful actions once Gotk3 supporting GVariant is released
	w.aPlayerRandom = w.addAction("player.toggle.random", "<Ctrl>U", w.playerToggleRandom)
	w.aPlayerRepeat = w.addAction("player.toggle.repeat", "<Ctrl>R", w.playerToggleRepeat)
	w.aPlayerConsume = w.addAction("player.toggle.consume", "<Ctrl>N", w.playerToggleConsume)

	// Populate Queue sort by combo box
	for _, id := range config.MpdTrackAttributeIds {
		w.cbxQueueSortBy.Append(strconv.Itoa(id), config.MpdTrackAttributes[id].LongName)
	}
	w.cbxQueueSortBy.SetActiveID(strconv.Itoa(cfg.DefaultSortAttrID))

	// Update Queue tree view columns
	w.updateQueueColumns()

	// Start connecting if needed
	if cfg.MpdAutoConnect {
		w.connect()
	}
}

func (w *MainWindow) onDelete() {
	log.Debug("MainWindow.onDelete()")

	// Disconnect from MPD
	w.disconnect()

	// Save the current window dimensions in the config
	cfg := config.GetConfig()
	x, y := w.window.GetPosition()
	width, height := w.window.GetSize()
	cfg.MainWindowDimensions = config.Dimensions{X: x, Y: y, Width: width, Height: height}

	// Write out the config
	cfg.Save()
}

func (w *MainWindow) onPlaylistDelete() {
	var err error
	if name := w.getSelectedPlaylistName(); name != "" {
		if util.ConfirmDialog(w.window, "Delete playlist", fmt.Sprintf("Are you sure you want to delete playlist \"%s\"?", name)) {
			w.connector.IfConnected(func(client *mpd.Client) {
				err = client.PlaylistRemove(name)
			})
		}
	}

	// Check for error (outside IfConnected() because it would keep the client locked)
	w.errCheckDialog(err, "Failed to delete the playlist")
}

func (w *MainWindow) onPlaylistRename() {
	var err error
	if name := w.getSelectedPlaylistName(); name != "" {
		if newName, ok := util.EditDialog(w.window, "Rename playlist", name, "Rename"); ok {
			w.connector.IfConnected(func(client *mpd.Client) {
				err = client.PlaylistRename(name, newName)
			})
		}
	}

	// Check for error (outside IfConnected() because it would keep the client locked)
	w.errCheckDialog(err, "Failed to rename the playlist")
}

func (w *MainWindow) onQueueTreeViewColClicked(col *gtk.TreeViewColumn, index int, attr *config.MpdTrackAttribute) {
	log.Debugf("onQueueTreeViewColClicked(col, %v, %v)", index, *attr)

	// Determine the sort order: on first click on a column ascending, on next descending
	descending := col.GetSortIndicator() && col.GetSortOrder() == gtk.SORT_ASCENDING
	sortType := gtk.SORT_ASCENDING
	if descending {
		sortType = gtk.SORT_DESCENDING
	}

	// Update sort indicators on all columns
	i := 0
	for c := w.trvQueue.GetColumns(); c != nil; c = c.Next() {
		item := c.Data().(*gtk.TreeViewColumn)
		thisCol := i == index
		// Set sort direction on the clicked column
		if thisCol {
			item.SetSortOrder(sortType)
		}
		// Update every column's sort indicator
		item.SetSortIndicator(thisCol)
		i++
	}

	// Sort the queue
	w.queueSort(attr, descending)
}

func (w *MainWindow) onQueueTreeViewButtonPress(_ *gtk.TreeView, event *gdk.Event) {
	switch btn := gdk.EventButtonNewFromEvent(event); btn.Type() {
	// Mouse click
	case gdk.EVENT_BUTTON_PRESS:
		// Right click
		if btn.Button() == 3 {
			w.mnQueue.PopupAtPointer(event)
		}
	// Double click
	case gdk.EVENT_DOUBLE_BUTTON_PRESS:
		w.applyQueueSelection()
	}
}

func (w *MainWindow) onQueueTreeViewKeyPress(_ *gtk.TreeView, event *gdk.Event) {
	evt := gdk.EventKeyNewFromEvent(event)
	mask := gtk.AcceleratorGetDefaultModMask()
	state := gdk.ModifierType(evt.State())
	switch evt.KeyVal() {
	// Enter: apply current selection
	case gdk.KEY_Return:
		if state&mask == 0 {
			w.applyQueueSelection()
		}
	// Esc: exit filtering mode if it's active
	case gdk.KEY_Escape:
		if state&mask == 0 {
			w.queueSearchBar.SetSearchMode(false)
		}
	// Delete
	case gdk.KEY_Delete:
		switch state & mask {
		// Delete: delete selection
		case 0:
			w.queueDelete()
		// Ctrl+Delete: clear queue
		case gdk.GDK_CONTROL_MASK:
			w.queueClear()
		}
	// Space
	case gdk.KEY_space:
		if state&mask == 0 {
			w.playerPlayPause()
		}
	// Ctrl+F: activate search bar
	case gdk.KEY_f:
		if state&mask == gdk.GDK_CONTROL_MASK {
			w.queueSearchBar.SetSearchMode(true)
		}
	}
}

func (w *MainWindow) onLibraryListBoxButtonPress(_ *gtk.ListBox, event *gdk.Event) {
	if gdk.EventButtonNewFromEvent(event).Type() == gdk.EVENT_DOUBLE_BUTTON_PRESS {
		// Double click in the list box
		w.applyLibrarySelection(config.GetConfig().TrackDefaultReplace)
	}
}

func (w *MainWindow) onLibraryListBoxKeyPress(_ *gtk.ListBox, event *gdk.Event) {
	ek := gdk.EventKeyNewFromEvent(event)
	switch ek.KeyVal() {
	// Enter: we need to go deeper
	case gdk.KEY_Return:
		w.applyLibrarySelection(config.GetConfig().TrackDefaultReplace)

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
		w.applyPlaylistSelection(config.GetConfig().PlaylistDefaultReplace)
	}
}

func (w *MainWindow) onPlaylistListBoxKeyPress(_ *gtk.ListBox, event *gdk.Event) {
	ek := gdk.EventKeyNewFromEvent(event)
	if ek.KeyVal() == gdk.KEY_Return {
		w.applyPlaylistSelection(config.GetConfig().PlaylistDefaultReplace)
	}
}

func (w *MainWindow) onPlayPositionButtonEvent(_ interface{}, event *gdk.Event) {
	switch gdk.EventButtonNewFromEvent(event).Type() {
	case gdk.EVENT_BUTTON_PRESS:
		w.playPosUpdating = true

	case gdk.EVENT_BUTTON_RELEASE:
		w.playPosUpdating = false
		w.connector.IfConnected(func(client *mpd.Client) {
			d := time.Duration(w.adjPlayPosition.GetValue())
			errCheck(client.SeekCur(d*time.Second, false), "SeekCur() failed")
		})
	}
}

func (w *MainWindow) onQueueSavePopoverValidate() {
	// Only show new playlist widgets if (new playlist) is selected in the combo box
	selectedID := w.cbxQueueSavePlaylist.GetActiveID()
	isNew := selectedID == queueSaveNewPlaylistID
	w.lblQueueSavePlaylistName.SetVisible(isNew)
	w.eQueueSavePlaylistName.SetVisible(isNew)

	// Validate the actions
	valid := (!isNew && selectedID != "") || (isNew && w.getQueueSaveNewPlaylistName() != "")
	w.aQueueSaveReplace.SetEnabled(valid && !isNew)
	w.aQueueSaveAppend.SetEnabled(valid)
}

// about shows the application's about dialog
func (w *MainWindow) about() {
	dlg, err := gtk.AboutDialogNew()
	if errCheck(err, "AboutDialogNew() failed") {
		return
	}
	dlg.SetLogoIconName(config.AppMetadata.Icon)
	dlg.SetProgramName(config.AppMetadata.Name)
	dlg.SetComments(fmt.Sprintf("Release date: %s", config.AppMetadata.BuildDate))
	dlg.SetCopyright(config.AppMetadata.Copyright)
	dlg.SetLicense(config.AppMetadata.License)
	dlg.SetVersion(config.AppMetadata.Version)
	dlg.SetWebsite(config.AppMetadata.URL)
	dlg.SetWebsiteLabel(config.AppMetadata.URLLabel)
	dlg.SetTransientFor(w.window)
	_, err = dlg.Connect("response", dlg.Destroy)
	errCheck(err, "dlg.Connect(response) failed")
	dlg.Run()
}

// addAction add a new application action, with an optional keyboard shortcut
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

// applyLibrarySelection navigates into the folder or adds or replaces the content of the queue with the currently
// selected items in the library
func (w *MainWindow) applyLibrarySelection(replace bool) {
	// Get selected path
	libPath, isDir := w.getSelectedLibraryPath()
	if libPath == "" {
		return
	}

	// Directory - navigate inside it
	if isDir {
		w.setLibraryPath(libPath)
	} else {
		// File - append/replace the queue
		w.queueOne(replace, libPath)
	}
}

// applyPlaylistSelection adds or replaces the content of the queue with the currently selected playlist
func (w *MainWindow) applyPlaylistSelection(replace bool) {
	if name := w.getSelectedPlaylistName(); name != "" {
		w.queuePlaylist(replace, name)
	}
}

// applyQueueSelection starts playing from the currently selected track
func (w *MainWindow) applyQueueSelection() {
	// Get the tree's selection
	var err error
	if indices := w.getQueueSelectedIndices(); len(indices) > 0 {
		// Start playback from the first selected index
		w.connector.IfConnected(func(client *mpd.Client) {
			err = client.Play(indices[0])
		})
	}

	// Check for error
	w.errCheckDialog(err, "Failed to play the selected track")
}

// connect starts connecting to MPD
func (w *MainWindow) connect() {
	// First disconnect, if connected
	if w.connector.IsConnected() {
		w.disconnect()
	}

	// Start connecting
	cfg := config.GetConfig()
	w.connector.Start(cfg.MpdAddress(), cfg.MpdPassword, cfg.MpdAutoReconnect)
}

// disconnect starts disconnecting from MPD
func (w *MainWindow) disconnect() {
	w.connector.Stop()
}

// errCheckDialog checks for error, and if it isn't nil, shows an error dialog ti the given text and the error info
func (w *MainWindow) errCheckDialog(err error, message string) bool {
	if err != nil {
		formatted := fmt.Sprintf("%v: %v", message, err)
		log.Warning(formatted)
		util.ErrorDialog(w.window, formatted)
		return true
	}
	return false
}

// getQueueSaveNewPlaylistName returns the text entered in the New playlist name entry, or an empty string if there's an error
func (w *MainWindow) getQueueSaveNewPlaylistName() string {
	s, err := w.eQueueSavePlaylistName.GetText()
	if errCheck(err, "eQueueSavePlaylistName.GetText() failed") {
		return ""
	}
	return s
}

// getQueueHasSelection returns whether there's any selected rows in the queue
func (w *MainWindow) getQueueHasSelection() bool {
	if sel, err := w.trvQueue.GetSelection(); errCheck(err, "getQueueHasSelection(): trvQueue.GetSelection() failed") {
		return false
	} else {
		return sel.CountSelectedRows() > 0
	}
}

// getQueueSelectedIndices returns indices of the currently selected rows in the queue
func (w *MainWindow) getQueueSelectedIndices() []int {
	// Get the tree's selection
	sel, err := w.trvQueue.GetSelection()
	if errCheck(err, "trvQueue.GetSelection() failed") {
		return nil
	}

	// Get selected nodes' indices
	var indices []int
	err = sel.SelectedForEach(func(model *gtk.TreeModel, path *gtk.TreePath, iter *gtk.TreeIter, userData interface{}) {
		// Convert the provided tree (filtered) path into unfiltered one
		if queuePath := w.queueListFilter.ConvertPathToChildPath(path); queuePath != nil {
			if ix := queuePath.GetIndices(); len(ix) > 0 {
				indices = append(indices, ix[0])
			}
		}
	})
	if errCheck(err, "getQueueSelectedIndices(): SelectedForEach() failed") {
		return nil
	}
	return indices
}

// getSelectedLibraryPath returns the full path of the currently selected library item and whether it's a directory,
// or an empty string if there's an error
func (w *MainWindow) getSelectedLibraryPath() (string, bool) {
	// If there's selection
	row := w.lbxLibrary.GetSelectedRow()
	if row == nil {
		return "", false
	}

	// Extract path, which is stored in the row's name
	name, err := row.GetName()
	if errCheck(err, "getSelectedLibraryPath(): row.GetName() failed") {
		return "", false
	}

	// Calculate final path
	libPath := w.currentLibPath
	if len(libPath) > 0 {
		libPath += "/"
	}
	libPath += name[2:]

	// The name prefix defines whether it's a file ("f:") or a directory ("d:")
	return libPath, strings.HasPrefix(name, "d:")
}

// getSelectedPlaylistName returns the name of the currently selected playlist, or an empty string if there's an error
func (w *MainWindow) getSelectedPlaylistName() string {
	// If there's selection
	row := w.lbxPlaylists.GetSelectedRow()
	if row == nil {
		return ""
	}

	// Extract playlist's name, which is stored in the row's name
	name, err := row.GetName()
	if errCheck(err, "getSelectedPlaylistName(): row.GetName() failed") {
		return ""
	}
	return name
}

// libraryUpdate updates or rescans the library
func (w *MainWindow) libraryUpdate(rescan, selectedOnly bool) {
	// Determine the update path
	libPath := ""
	if selectedOnly {
		if libPath, _ = w.getSelectedLibraryPath(); libPath == "" {
			return
		}
	}

	// Run the update
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		if rescan {
			// TODO implement once gompd provides support for it, see https://github.com/fhs/gompd/issues/54
			err = errors.New("Rescan is not implemented yet")
		} else {
			_, err = client.Update(libPath)
		}
	})

	// Check for error
	w.errCheckDialog(err, "Failed to update the library")
}

// playerPrevious rewinds the player to the previous track
func (w *MainWindow) playerPrevious() {
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		err = client.Previous()
	})

	// Check for error
	w.errCheckDialog(err, "Failed to skip to previous track")
}

// playerStop stops the playback
func (w *MainWindow) playerStop() {
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		err = client.Stop()
	})

	// Check for error
	w.errCheckDialog(err, "Failed to stop playback")
}

// playerPlayPause pauses or resumes the playback
func (w *MainWindow) playerPlayPause() {
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		switch w.connector.Status()["state"] {
		case "pause":
			err = client.Pause(false)
		case "play":
			err = client.Pause(true)
		default:
			err = client.Play(-1)
		}
	})

	// Check for error
	w.errCheckDialog(err, "Failed to toggle playback")
}

// playerNext advances the player to the next track
func (w *MainWindow) playerNext() {
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		err = client.Next()
	})

	// Check for error
	w.errCheckDialog(err, "Failed to skip to next track")
}

// playerToggleConsume toggles player's consume mode
func (w *MainWindow) playerToggleConsume() {
	// Ignore if the state of the button is being updated programmatically
	if w.optionsUpdating {
		return
	}

	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		err = client.Consume(w.connector.Status()["consume"] == "0")
	})

	// Check for error
	w.errCheckDialog(err, "Failed to toggle consume mode")
}

// playerToggleRandom toggles player's random mode
func (w *MainWindow) playerToggleRandom() {
	// Ignore if the state of the button is being updated programmatically
	if w.optionsUpdating {
		return
	}

	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		err = client.Random(w.connector.Status()["random"] == "0")
	})

	// Check for error
	w.errCheckDialog(err, "Failed to toggle random mode")
}

// playerToggleRepeat toggles player's repeat mode
func (w *MainWindow) playerToggleRepeat() {
	// Ignore if the state of the button is being updated programmatically
	if w.optionsUpdating {
		return
	}

	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		err = client.Repeat(w.connector.Status()["repeat"] == "0")
	})

	// Check for error
	w.errCheckDialog(err, "Failed to toggle repeat mode")
}

// preferences shows the preferences dialog
func (w *MainWindow) preferences() {
	PreferencesDialog(w.window, w.connect, w.updateQueueColumns, w.updatePlayerTitleTemplate)
}

// queue adds or replaces the content of the queue with the specified URIs
func (w *MainWindow) queue(replace bool, uris []string) {
	var err error
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
		err = commands.End()
	})

	// Check for error
	w.errCheckDialog(err, "Failed to add track(s) to the queue")
}

// queueClear empties MPD's play queue
func (w *MainWindow) queueClear() {
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		err = client.Clear()
	})

	// Check for error
	w.errCheckDialog(err, "Failed to clear the queue")
}

// queueDelete deletes the selected tracks from MPD's play queue
func (w *MainWindow) queueDelete() {
	// Get selected nodes' indices
	indices := w.getQueueSelectedIndices()
	if len(indices) == 0 {
		return
	}

	// Sort indices in descending order
	sort.Slice(indices, func(i, j int) bool { return indices[j] < indices[i] })

	// Remove the tracks from the queue (also in descending order)
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		commands := client.BeginCommandList()
		for _, idx := range indices {
			errCheck(commands.Delete(idx, idx+1), "commands.Delete() failed")
		}
		err = commands.End()
	})

	// Check for error
	w.errCheckDialog(err, "Failed to delete tracks from the queue")
}

// queueFilter applies the currently entered filter substring to the queue
func (w *MainWindow) queueFilter() {
	log.Debug("queueFilter()")
	substr := ""

	// Only use filter pattern if the search bar is visible
	if w.queueSearchBar.GetSearchMode() {
		var err error
		if substr, err = w.queueSearchEntry.GetText(); errCheck(err, "queueSearchEntry.GetText() failed") {
			return
		}
	}

	// Iterate all rows in the list store
	count := 0
	err := w.queueListStore.ForEach(func(model *gtk.TreeModel, path *gtk.TreePath, iter *gtk.TreeIter, userData interface{}) bool {
		// Show all rows if no search pattern given
		visible := substr == ""
		if !visible {
			// We're going to compare case-insensitively
			substr := strings.ToLower(substr)

			// Scan all known columns in the row
			for _, id := range config.MpdTrackAttributeIds {
				// Get column's value
				v, err := model.GetValue(iter, id)
				if errCheck(err, "queueFilter(): queueListStore.GetValue() failed") {
					continue
				}

				// Convert the value into a string
				s, _ := v.GetString() // Ignoring the returned error due to https://github.com/gotk3/gotk3/issues/583

				// Check for a match and stop checking if match has already been found
				visible = s != "" && strings.Contains(strings.ToLower(s), substr)
				if visible {
					break
				}
			}
		}

		// Modify the row's visibility
		if err := w.queueListStore.SetValue(iter, config.QueueColumnVisible, visible); errCheck(err, "queueFilter(): queueListStore.SetValue() failed") {
			return true
		}
		if visible {
			count++
		}

		// Proceed to the next row
		return false
	})
	if errCheck(err, "queueListStore.ForEach() failed") {
		return
	}

	w.lblQueueFilter.SetText(fmt.Sprintf("%d tracks displayed", count))
}

// queueOne adds or replaces the content of the queue with one specified URI
func (w *MainWindow) queueOne(replace bool, uri string) {
	w.queue(replace, []string{uri})
}

// queuePlaylist adds or replaces the content of the queue with the specified playlist
func (w *MainWindow) queuePlaylist(replace bool, playlistName string) {
	log.Debugf("queuePlaylist(%v, %v)", replace, playlistName)
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		commands := client.BeginCommandList()

		// Clear the queue, if needed
		if replace {
			commands.Clear()
		}

		// Add the content of the playlist
		commands.PlaylistLoad(playlistName, -1, -1)

		// Run the commands
		err = commands.End()
	})

	// Check for error
	w.errCheckDialog(err, "Failed to add playlist to the queue")
}

// queueSave shows a dialog for saving the play queue into a playlist and performs the operation if confirmed
func (w *MainWindow) queueSave() {
	// Tweak widgets
	selection := w.getQueueHasSelection()
	w.cbQueueSaveSelectedOnly.SetVisible(selection)
	w.cbQueueSaveSelectedOnly.SetActive(selection)
	w.eQueueSavePlaylistName.SetText("")

	// Populate the playlists combo box
	w.cbxQueueSavePlaylist.RemoveAll()
	w.cbxQueueSavePlaylist.Append(queueSaveNewPlaylistID, "(new playlist)")
	for _, name := range w.connector.GetPlaylists() {
		w.cbxQueueSavePlaylist.Append(name, name)
	}
	w.cbxQueueSavePlaylist.SetActiveID(queueSaveNewPlaylistID)

	// Show the popover
	w.pmnQueueSave.Popup()
}

// queueSaveApply performs queue saving into a playlist
func (w *MainWindow) queueSaveApply(replace bool) {
	// Collect current values from the UI
	selIndices := w.getQueueSelectedIndices()
	selOnly := len(selIndices) > 0 && w.cbQueueSaveSelectedOnly.GetActive()
	name := w.cbxQueueSavePlaylist.GetActiveID()
	isNew := name == queueSaveNewPlaylistID
	if isNew {
		name = w.getQueueSaveNewPlaylistName()
	}

	err := errors.New("Not connected to MPD")
	w.connector.IfConnected(func(client *mpd.Client) {
		// Fetch the queue
		var attrs []mpd.Attrs
		attrs, err = client.PlaylistInfo(-1, -1)
		if err != nil {
			return
		}

		// Begin a command list
		commands := client.BeginCommandList()

		// If replacing an existing playlist, remove it first
		if !isNew && replace {
			commands.PlaylistRemove(name)
		}

		// If adding selection only
		if selOnly {
			for _, idx := range selIndices {
				commands.PlaylistAdd(name, attrs[idx]["file"])
			}

		} else if replace {
			// Save the entire queue
			commands.PlaylistSave(name)

		} else {
			// Append the entire queue
			for _, a := range attrs {
				commands.PlaylistAdd(name, a["file"])
			}
		}

		// Execute the command list
		err = commands.End()
	})

	// Check for error
	w.errCheckDialog(err, "Failed to create a playlist")
}

// queueShuffle randomises MPD's play queue
func (w *MainWindow) queueShuffle() {
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		err = client.Shuffle(-1, -1)
	})

	// Check for error
	w.errCheckDialog(err, "Failed to shuffle the queue")
}

// queueSort orders MPD's play queue on the provided attribute
func (w *MainWindow) queueSort(attr *config.MpdTrackAttribute, descending bool) {
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		// Fetch the current playlist
		var attrs []mpd.Attrs
		if attrs, err = client.PlaylistInfo(-1, -1); err != nil {
			return
		}

		// Sort the list
		sort.SliceStable(attrs, func(i, j int) bool {
			a, b := attrs[i][attr.AttrName], attrs[j][attr.AttrName]
			if attr.Numeric {
				an, bn := util.ParseFloatDef(a, 0), util.ParseFloatDef(b, 0)
				if descending {
					return bn < an
				}
				return an < bn
			}
			if descending {
				return b < a
			}
			return a < b
		})

		// Post the changes back to MPD
		commands := client.BeginCommandList()
		for index, a := range attrs {
			var id int
			if id, err = strconv.Atoi(a["Id"]); err != nil {
				return
			}
			commands.MoveID(id, index)
		}
		err = commands.End()
	})

	// Check for error
	w.errCheckDialog(err, "Failed to sort the queue")

}

// queueSortApply performs MPD's play queue ordering based on the currently selected in popover mode
func (w *MainWindow) queueSortApply(descending bool) {
	// Fetch the ID of the currently selected item in the Sort by combo box, and the corresponding attribute
	if attr, ok := config.MpdTrackAttributes[util.AtoiDef(w.cbxQueueSortBy.GetActiveID(), -1)]; ok {
		w.queueSort(&attr, descending)
	}
}

// shortcutInfo displays a shortcut info window
func (w *MainWindow) shortcutInfo() {
	// TODO update ShortcutsWindow usage once it's properly implemented in gotk3
	builder := NewBuilder(generated.GetShortcutsGlade())
	sw := builder.getShortcutsWindow("shortcutsWindow")
	sw.SetTransientFor(w.window)
	sw.ShowAll()
	_, err := sw.Connect("unmap", sw.Destroy)
	errCheck(err, "Failed to connect unmap signal")
}

// Show displays the window and all its child widgets
func (w *MainWindow) Show() {
	w.window.Show()
}

// setLibraryPath sets the current library path selector and updates its widget and the current library list
func (w *MainWindow) setLibraryPath(path string) {
	w.currentLibPath = path
	w.updateLibraryPath()
	w.updateLibrary(0)
	w.lbxLibrary.GrabFocus()
}

// setQueueHighlight selects or deselects an item in the Queue tree view at the given index
func (w *MainWindow) setQueueHighlight(index int, selected bool) {
	if index >= 0 {
		if iter, err := w.queueListStore.GetIterFromString(strconv.Itoa(index)); err == nil {
			weight := fontWeightNormal
			bgColor := colorBgNormal
			if selected {
				weight = fontWeightBold
				bgColor = colorBgActive
			}
			errCheck(
				w.queueListStore.SetCols(iter, map[int]interface{}{
					config.QueueColumnFontWeight: weight,
					config.QueueColumnBgColor:    bgColor,
				}),
				"lstQueue.SetValue() failed")
		}
	}
}

// updateAll updates all window's widgets and lists
func (w *MainWindow) updateAll() {
	w.updateQueue()
	w.updateLibraryPath()
	w.updateLibrary(0)
	w.updateLibraryActions()
	w.updatePlaylists()
	w.updatePlaylistsActions()
	w.updateOptions()
	w.updatePlayer()
}

// updateLibrary updates the current library list contents
func (w *MainWindow) updateLibrary(indexToSelect int) {
	// Clear the library list
	util.ClearChildren(w.lbxLibrary.Container)

	// Update the library list if there's a connection
	var attrs []mpd.Attrs
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		attrs, err = client.ListInfo(w.currentLibPath)
	})
	if errCheck(err, "ListInfo() failed") {
		return
	}

	// pathPrefix will need to be removed from element names
	pathPrefix := w.currentLibPath + "/"

	// Repopulate the library list
	var rowToSelect *gtk.ListBoxRow
	idxRow, countDirs, countFiles := 0, 0, 0
	for _, a := range attrs {
		// Pick files and directories only
		uri, iconName, prefix := "", "", ""
		if dir, ok := a["directory"]; ok {
			uri = dir
			iconName = "folder"
			prefix = "d:"
			countDirs++
		} else if file, ok := a["file"]; ok {
			uri = file
			iconName = "audio-x-generic"
			prefix = "f:"
			countFiles++
		} else {
			continue
		}

		// Add a new list box row
		name := strings.TrimPrefix(uri, pathPrefix)
		row, hbx, err := util.NewListBoxRow(
			w.lbxLibrary,
			name,
			prefix+name,
			iconName,
			// Add replace/append buttons
			util.NewButton("", "Append to the queue", "", "list-add", func() { w.queueOne(false, uri) }),
			util.NewButton("", "Replace the queue", "", "edit-paste", func() { w.queueOne(true, uri) }))

		if errCheck(err, "NewListBoxRow() failed") {
			return
		}
		if indexToSelect == idxRow {
			rowToSelect = row
		}

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

	// Compose info
	info := "No folders"
	if countDirs > 0 {
		info = fmt.Sprintf("%d folders", countDirs)
	}
	if countFiles > 0 {
		info += fmt.Sprintf(", %d files", countFiles)
	} else {
		info += ", no files"
	}
	if _, ok := w.connector.Status()["updating_db"]; ok {
		info += " — updating database…"
	}

	// Update info
	w.lblLibraryInfo.SetText(info)
}

// updateLibraryActions updates the widgets for library list
func (w *MainWindow) updateLibraryActions() {
	connected, selected := w.connector.IsConnected(), w.lbxLibrary.GetSelectedRow() != nil
	w.aLibraryUpdate.SetEnabled(connected)
	w.aLibraryUpdateAll.SetEnabled(connected)
	w.aLibraryUpdateSel.SetEnabled(connected && selected)
	w.aLibraryRescanAll.SetEnabled(connected)
	w.aLibraryRescanSel.SetEnabled(connected && selected)
}

// updateLibraryPath updates the current library path selector
func (w *MainWindow) updateLibraryPath() {
	// Remove all buttons from the box
	util.ClearChildren(w.bxLibraryPath.Container)

	// Create buttons if there's a connection
	if w.connector.IsConnected() {
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
	}
}

// updateOptions updates player options widgets
func (w *MainWindow) updateOptions() {
	w.optionsUpdating = true
	status := w.connector.Status()
	w.btnRandom.SetActive(status["random"] == "1")
	w.btnRepeat.SetActive(status["repeat"] == "1")
	w.btnConsume.SetActive(status["consume"] == "1")
	w.optionsUpdating = false
}

// updatePlayer updates player control widgets
func (w *MainWindow) updatePlayer() {
	connected := false
	statusText := "<i>(not connected)</i>"
	var curSong mpd.Attrs
	var err error

	// Fetch current song, if there's a connection
	w.connector.IfConnected(func(client *mpd.Client) {
		connected = true
		curSong, err = client.CurrentSong()
	})

	if connected {
		// Check for error
		if errCheck(err, "CurrentSong() failed") {
			statusText = fmt.Sprintf("<b>MPD error:</b> %v", err)
		} else {
			log.Debugf("Current track: %+v", curSong)

			// Apply track title template
			var buffer bytes.Buffer
			if err := w.playerTitleTemplate.Execute(&buffer, curSong); err != nil {
				statusText = html.EscapeString(fmt.Sprintf("Template error: %v", err))
			} else {
				statusText = buffer.String()
			}
		}

		// Update play/pause button's appearance
		status := w.connector.Status()
		switch status["state"] {
		case "play":
			w.btnPlayPause.SetIconName("media-playback-pause")
		default:
			w.btnPlayPause.SetIconName("media-playback-start")
		}

	} else if err := w.connector.Status()["error"]; err != "" {
		// If not connected and there's an error
		statusText = fmt.Sprintf("<span foreground=\"red\">%s</span>", err)
	}

	// Update status text
	w.lblStatus.SetMarkup(statusText)

	// Highlight and scroll the tree to the currently played item
	w.updateQueueNowPlaying()

	// Enable or disable player actions based on the connection status
	w.aPlayerPrevious.SetEnabled(connected)
	w.aPlayerStop.SetEnabled(connected)
	w.aPlayerPlayPause.SetEnabled(connected)
	w.aPlayerNext.SetEnabled(connected)
	w.aPlayerRandom.SetEnabled(connected)
	w.aPlayerRepeat.SetEnabled(connected)
	w.aPlayerConsume.SetEnabled(connected)

	// Update the seek bar
	w.updatePlayerSeekBar()
}

// updatePlayerSeekBar updates the seek bar position and status
func (w *MainWindow) updatePlayerSeekBar() {
	seekPos := ""
	var trackLen, trackPos float64

	// If the user is dragging the slider manually
	if w.playPosUpdating {
		trackLen, trackPos = w.adjPlayPosition.GetUpper(), w.adjPlayPosition.GetValue()

	} else {
		// The update comes from MPD: adjust the seek bar position if there's a connection
		trackStart := -1.0
		trackLen, trackPos = -1.0, -1.0
		if w.connector.IsConnected() {
			// Fetch current player position and track length
			status := w.connector.Status()
			trackLen = util.ParseFloatDef(status["duration"], -1)
			trackPos = util.ParseFloatDef(status["elapsed"], -1)
		}

		// If not seekable, remove the slider
		if trackPos >= 0 && trackLen >= trackPos {
			trackStart = 0
		}
		w.scPlayPosition.SetSensitive(trackStart == 0)

		// Enable the seek bar based on status and position it
		w.adjPlayPosition.SetLower(trackStart)
		w.adjPlayPosition.SetUpper(trackLen)
		w.adjPlayPosition.SetValue(trackPos)
	}

	// Update position text
	if trackPos >= 0 {
		seekPos = fmt.Sprintf("<big>%s</big>", util.FormatSeconds(trackPos))
		if trackLen >= trackPos {
			seekPos += fmt.Sprintf(" / " + util.FormatSeconds(trackLen))
		}
	}
	w.lblPosition.SetMarkup(seekPos)
}

// updatePlayerTitleTemplate compiles the player title template
func (w *MainWindow) updatePlayerTitleTemplate() {
	tmpl, err := template.New("playerTitle").
		Funcs(template.FuncMap{
			"default":  util.Default,
			"dirname":  path.Dir,
			"basename": path.Base,
		}).
		Parse(config.GetConfig().PlayerTitleTemplate)
	if errCheck(err, "Template parse error") {
		w.playerTitleTemplate = template.Must(
			template.New("error").Parse("<span foreground=\"red\">[Player title template error, check log]</span>"))
	} else {
		w.playerTitleTemplate = tmpl
	}

	// Update the displayed title if the connector is initialised
	if w.connector != nil {
		w.updatePlayer()
	}
}

// updatePlaylists updates the current playlists list contents
func (w *MainWindow) updatePlaylists() {
	// Clear the playlists list
	util.ClearChildren(w.lbxPlaylists.Container)

	// Repopulate the playlists list
	playlists := w.connector.GetPlaylists()
	for _, name := range playlists {
		name := name // Make an in-loop copy of the var
		_, _, err := util.NewListBoxRow(
			w.lbxPlaylists,
			name,
			name,
			"format-justify-left",
			// Add replace/append buttons
			util.NewButton("", "Append to the queue", "", "list-add", func() { w.queuePlaylist(false, name) }),
			util.NewButton("", "Replace the queue", "", "edit-paste", func() { w.queuePlaylist(true, name) }))
		if errCheck(err, "NewListBoxRow() failed") {
			return
		}
	}

	// Show all rows
	w.lbxPlaylists.ShowAll()

	// Compose info
	info := "No playlists"
	if cnt := len(playlists); cnt > 0 {
		info = fmt.Sprintf("%d playlists", cnt)
	}

	// Update info
	w.lblPlaylistsInfo.SetText(info)
}

// updatePlaylistsActions updates the widgets for playlists list
func (w *MainWindow) updatePlaylistsActions() {
	connected, selected := w.connector.IsConnected(), w.getSelectedPlaylistName() != ""
	w.aPlaylistRename.SetEnabled(connected && selected)
	w.aPlaylistDelete.SetEnabled(connected && selected)
}

// updateQueue updates the current play queue contents
func (w *MainWindow) updateQueue() {
	// Clear the queue list store
	w.queueListStore.Clear()
	w.currentQueueIndex = -1
	w.currentQueueSize = 0

	// Update the queue if there's a connection
	var attrs []mpd.Attrs
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		attrs, err = client.PlaylistInfo(-1, -1)
	})
	if errCheck(err, "PlaylistInfo() failed") {
		return
	}

	// Repopulate the queue list store
	totalSecs := 0.0
	for _, a := range attrs {
		rowData := make(map[int]interface{})
		for id, mpdAttr := range config.MpdTrackAttributes {
			// Fetch the raw attribute value, if any
			value, ok := a[mpdAttr.AttrName]
			if !ok {
				continue
			}

			// Format the value if needed
			if mpdAttr.Formatter != nil {
				value = mpdAttr.Formatter(value)
			}
			rowData[id] = value

			// Add the "artificial" column values
			rowData[config.QueueColumnFontWeight] = fontWeightNormal
			rowData[config.QueueColumnBgColor] = colorBgNormal
			rowData[config.QueueColumnVisible] = true
		}

		// Add a row to the list store
		errCheck(
			w.queueListStore.SetCols(w.queueListStore.Append(), rowData),
			"lstQueue.SetCols() failed")

		// Accumulate counters
		totalSecs += util.ParseFloatDef(a["duration"], 0)
		w.currentQueueSize++
	}

	// Add number of tracks
	var status string
	switch w.currentQueueSize {
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

	// Update the queue info
	w.lblQueueInfo.SetText(status)

	// Highlight and scroll the tree to the currently played item
	w.updateQueueNowPlaying()

	// Update queue actions
	w.updateQueueActions()
}

// updateQueueColumns updates the columns in the play queue tree view
func (w *MainWindow) updateQueueColumns() {
	// Remove all columns
	w.trvQueue.GetColumns().Foreach(func(item interface{}) {
		w.trvQueue.RemoveColumn(item.(*gtk.TreeViewColumn))
	})

	// Add selected columns
	queueColumns := config.GetConfig().QueueColumns
	for index, colSpec := range *queueColumns {
		index := index // Make an in-loop copy of index for the closures below

		// Fetch the attribute by its ID
		attr, ok := config.MpdTrackAttributes[colSpec.ID]
		if !ok {
			log.Errorf("Invalid column ID: %d", colSpec.ID)
			continue
		}

		// Add a text renderer
		renderer, err := gtk.CellRendererTextNew()
		if errCheck(err, "CellRendererTextNew() failed") {
			continue
		}

		// Add a new tree column
		col, err := gtk.TreeViewColumnNewWithAttribute(attr.Name, renderer, "text", colSpec.ID)
		if errCheck(err, "TreeViewColumnNewWithAttribute() failed") {
			continue
		}
		col.SetSizing(gtk.TREE_VIEW_COLUMN_FIXED)
		width := colSpec.Width
		if width == 0 {
			width = attr.Width
		}
		col.SetFixedWidth(width)
		col.SetClickable(true)
		col.SetResizable(true)
		col.AddAttribute(renderer, "weight", config.QueueColumnFontWeight)
		col.AddAttribute(renderer, "background", config.QueueColumnBgColor)

		// Bind the clicked signal
		_, err = col.Connect("clicked", func() { w.onQueueTreeViewColClicked(col, index, &attr) })
		errCheck(err, "col.Connect(clicked) failed")

		// Bind the width property change signal: update QueueColumns on each change
		_, err = col.Connect("notify::fixed-width", func() { (*queueColumns)[index].Width = col.GetFixedWidth() })
		errCheck(err, "col.Connect(notify::fixed-width) failed")

		// Add the column to the tree view
		w.trvQueue.AppendColumn(col)
	}

	// Make all columns visible
	w.trvQueue.ShowAll()
}

// updateQueueActions updates the play queue actions
func (w *MainWindow) updateQueueActions() {
	connected := w.connector.IsConnected()
	notEmpty := connected && w.currentQueueSize > 0
	selection := notEmpty && w.getQueueHasSelection()
	w.aQueueNowPlaying.SetEnabled(notEmpty)
	w.aQueueClear.SetEnabled(notEmpty)
	w.aQueueSort.SetEnabled(notEmpty)
	w.aQueueSortAsc.SetEnabled(notEmpty)
	w.aQueueSortDesc.SetEnabled(notEmpty)
	w.aQueueSortShuffle.SetEnabled(notEmpty)
	w.aQueueDelete.SetEnabled(selection)
	w.aQueueSave.SetEnabled(notEmpty)
}

// updateQueueNowPlaying scrolls the queue tree view to the currently played track
func (w *MainWindow) updateQueueNowPlaying() {
	// Update queue highlight
	if curIdx := util.AtoiDef(w.connector.Status()["song"], -1); w.currentQueueIndex != curIdx {
		w.setQueueHighlight(w.currentQueueIndex, false)
		w.setQueueHighlight(curIdx, true)
		w.currentQueueIndex = curIdx
	}

	// Scroll to the currently playing
	if w.currentQueueIndex >= 0 {
		// Obtain a path in the unfiltered list
		treePath, err := gtk.TreePathNewFromIndicesv([]int{w.currentQueueIndex})
		if errCheck(err, "updateQueueNowPlaying(): TreePathNewFromString() failed") {
			return
		}

		// Convert the path into one in the filtered list
		if treePath = w.queueListFilter.ConvertChildPathToPath(treePath); treePath != nil {
			w.trvQueue.ScrollToCell(treePath, nil, true, 0.5, 0)
		}
	}
}
