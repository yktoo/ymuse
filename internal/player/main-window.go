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
	app       *gtk.Application // Application reference
	connector *Connector       // Connector instance
	mapped    bool             // Whether the main window is mapped (~visible)

	// Control widgets
	AppWindow              *gtk.ApplicationWindow // Main window
	MainStack              *gtk.Stack
	StatusLabel            *gtk.Label
	PositionLabel          *gtk.Label
	PlayPauseButton        *gtk.ToolButton
	RandomButton           *gtk.ToggleToolButton
	RepeatButton           *gtk.ToggleToolButton
	ConsumeButton          *gtk.ToggleToolButton
	VolumeButton           *gtk.VolumeButton
	VolumeAdjustment       *gtk.Adjustment
	PlayPositionScale      *gtk.Scale
	PlayPositionAdjustment *gtk.Adjustment
	AlbumArtworkImage      *gtk.Image
	// Queue widgets
	QueueBox                         *gtk.Box
	QueueToolbar                     *gtk.Toolbar
	QueueInfoLabel                   *gtk.Label
	QueueTreeView                    *gtk.TreeView
	QueueSortPopoverMenu             *gtk.PopoverMenu
	QueueSavePopoverMenu             *gtk.PopoverMenu
	QueueMenu                        *gtk.Menu
	QueueNowPlayingMenuItem          *gtk.MenuItem
	QueueShowAlbumInLibraryMenuItem  *gtk.MenuItem
	QueueShowArtistInLibraryMenuItem *gtk.MenuItem
	QueueShowGenreInLibraryMenuItem  *gtk.MenuItem
	QueueClearMenuItem               *gtk.MenuItem
	QueueDeleteMenuItem              *gtk.MenuItem
	QueueFilterToolButton            *gtk.ToggleToolButton
	QueueSearchBar                   *gtk.SearchBar
	QueueSearchEntry                 *gtk.SearchEntry
	QueueFilterLabel                 *gtk.Label
	QueueListStore                   *gtk.ListStore
	QueueTreeModelFilter             *gtk.TreeModelFilter
	// Queue sort popup
	QueueSortByComboBox *gtk.ComboBoxText
	// Queue save popup
	QueueSavePlaylistComboBox        *gtk.ComboBoxText
	QueueSavePlaylistNameLabel       *gtk.Label
	QueueSavePlaylistNameEntry       *gtk.Entry
	QueueSaveSelectedOnlyCheckButton *gtk.CheckButton
	// Library widgets
	LibraryUpdatePopoverMenu        *gtk.PopoverMenu
	LibraryAddToPlaylistPopoverMenu *gtk.PopoverMenu
	LibraryAddToPlaylistBox         *gtk.Box
	LibraryBox                      *gtk.Box
	LibraryPathBox                  *gtk.Box
	LibrarySearchBox                *gtk.Box
	LibrarySearchToolButton         *gtk.ToggleToolButton
	LibraryToolStack                *gtk.Stack
	LibrarySearchEntry              *gtk.SearchEntry
	LibrarySearchAttrComboBox       *gtk.ComboBoxText
	LibraryListBox                  *gtk.ListBox
	LibraryInfoLabel                *gtk.Label
	LibraryMenu                     *gtk.Menu
	LibraryAppendMenuItem           *gtk.MenuItem
	LibraryReplaceMenuItem          *gtk.MenuItem
	LibraryRenameMenuItem           *gtk.MenuItem
	LibraryDeleteMenuItem           *gtk.MenuItem
	LibraryUpdateSelMenuItem        *gtk.MenuItem
	LibraryAddToPlaylistMenuItem    *gtk.MenuItem
	// Streams widgets
	StreamsBox             *gtk.Box
	StreamsAddToolButton   *gtk.ToolButton
	StreamsEditToolButton  *gtk.ToolButton
	StreamsListBox         *gtk.ListBox
	StreamsInfoLabel       *gtk.Label
	StreamsMenu            *gtk.Menu
	StreamsAppendMenuItem  *gtk.MenuItem
	StreamsReplaceMenuItem *gtk.MenuItem
	StreamsEditMenuItem    *gtk.MenuItem
	StreamsDeleteMenuItem  *gtk.MenuItem
	// Streams props popup
	StreamPropsPopoverMenu *gtk.PopoverMenu
	StreamPropsNameEntry   *gtk.Entry
	StreamPropsUriEntry    *gtk.Entry

	// Actions
	aMPDDisconnect        *glib.SimpleAction
	aMPDInfo              *glib.SimpleAction
	aMPDOutputs           *glib.SimpleAction
	aQueueNowPlaying      *glib.SimpleAction
	aQueueClear           *glib.SimpleAction
	aQueueSort            *glib.SimpleAction
	aQueueSortAsc         *glib.SimpleAction
	aQueueSortDesc        *glib.SimpleAction
	aQueueSortShuffle     *glib.SimpleAction
	aQueueDelete          *glib.SimpleAction
	aQueueSave            *glib.SimpleAction
	aQueueSaveReplace     *glib.SimpleAction
	aQueueSaveAppend      *glib.SimpleAction
	aLibraryUpdate        *glib.SimpleAction
	aLibraryUpdateAll     *glib.SimpleAction
	aLibraryUpdateSel     *glib.SimpleAction
	aLibraryRescanAll     *glib.SimpleAction
	aLibraryRescanSel     *glib.SimpleAction
	aLibraryRename        *glib.SimpleAction
	aLibraryDelete        *glib.SimpleAction
	aLibraryAddToPlaylist *glib.SimpleAction
	aStreamAdd            *glib.SimpleAction
	aStreamEdit           *glib.SimpleAction
	aStreamDelete         *glib.SimpleAction
	aStreamPropsApply     *glib.SimpleAction
	aPlayerPrevious       *glib.SimpleAction
	aPlayerStop           *glib.SimpleAction
	aPlayerPlayPause      *glib.SimpleAction
	aPlayerNext           *glib.SimpleAction
	aPlayerRandom         *glib.SimpleAction
	aPlayerRepeat         *glib.SimpleAction
	aPlayerConsume        *glib.SimpleAction

	// Colours
	colourBgNormal string // Normal background colour
	colourBgActive string // Active background colour

	currentQueueSize  int // Number of items in the play queue
	currentQueueIndex int // Queue's track index (last) marked as current

	libPath                *LibraryPath // Current library path
	libPathElementToSelect string       // Library path element to select after list load (serialised)

	playerTitleTemplate      *template.Template // Compiled template for player's track title
	playerCurrentAlbumArtUri string             // URI of the current player's album art

	volumeUpdating  bool // Volume button update (initiated by an MPD event) flag
	playPosUpdating bool // Play position manual update flag
	optionsUpdating bool // Options update flag
	addingStream    bool // Whether the property popover is open to add a stream (rather than edit an existing one)
}

const (
	// Rendering properties for the Queue list
	fontWeightNormal = 400
	fontWeightBold   = 700

	queueSaveNewPlaylistID = "\u0001new"
	librarySearchAllAttrID = "\u0001any"
)

type triBool int

const (
	tbNone triBool = iota - 1
	tbFalse
	tbTrue
)

// NewMainWindow creates and returns a new MainWindow instance
func NewMainWindow(application *gtk.Application) (*MainWindow, error) {
	// Set up the window
	builder, err := NewBuilder(playerGlade)
	if err != nil {
		log.Fatalf("NewBuilder() failed: %v", err)
	}

	// Instantiate a window and bind widgets
	w := &MainWindow{app: application}
	if err := builder.BindWidgets(w); err != nil {
		log.Fatalf("BindWidgets() failed: %v", err)
	}

	// Initialise queue filter model
	w.QueueTreeModelFilter.SetVisibleColumn(config.QueueColumnVisible)

	// Initialise player settings
	w.applyPlayerSettings()

	// Initialise widgets and actions
	w.initWidgets()

	// Map the handlers to callback functions
	builder.ConnectSignals(map[string]interface{}{
		"on_MainWindow_delete":                         w.onDelete,
		"on_MainWindow_map":                            w.onMap,
		"on_MainWindow_styleUpdated":                   w.updateStyle,
		"on_MainStack_switched":                        w.focusMainList,
		"on_QueueTreeView_buttonPress":                 w.onQueueTreeViewButtonPress,
		"on_QueueTreeView_keyPress":                    w.onQueueTreeViewKeyPress,
		"on_QueueTreeSelection_changed":                w.updateQueueActions,
		"on_QueueSearchBar_searchMode":                 w.onQueueSearchMode,
		"on_QueueSearchEntry_searchChanged":            w.queueFilter,
		"on_LibraryListBox_buttonPress":                w.onLibraryListBoxButtonPress,
		"on_LibraryListBox_keyPress":                   w.onLibraryListBoxKeyPress,
		"on_LibraryListBox_selectionChange":            w.updateLibraryActions,
		"on_LibrarySearchChanged":                      w.updateLibrary,
		"on_LibrarySearchStop":                         w.onLibraryStopSearch,
		"on_StreamsListBox_buttonPress":                w.onStreamListBoxButtonPress,
		"on_StreamsListBox_keyPress":                   w.onStreamListBoxKeyPress,
		"on_StreamsListBox_selectionChange":            w.updateStreamsActions,
		"on_StreamPropsChanged":                        w.onStreamPropsChanged,
		"on_QueueSavePopoverMenu_validate":             w.onQueueSavePopoverValidate,
		"on_VolumeButton_valueChanged":                 w.onVolumeValueChanged,
		"on_PlayPositionScale_buttonEvent":             w.onPlayPositionButtonEvent,
		"on_PlayPositionScale_valueChanged":            w.updatePlayerSeekBar,
		"on_QueueNowPlayingMenuItem_activate":          w.updateQueueNowPlaying,
		"on_QueueShowAlbumInLibraryMenuItem_activate":  w.libraryShowAlbumFromQueue,
		"on_QueueShowArtistInLibraryMenuItem_activate": w.libraryShowArtistFromQueue,
		"on_QueueShowGenreInLibraryMenuItem_activate":  w.libraryShowGenreFromQueue,
		"on_QueueClearMenuItem_activate":               w.queueClear,
		"on_QueueDeleteMenuItem_activate":              w.queueDelete,
		"on_LibraryAddToPlaylistMenuItem_activate":     w.libraryAddToPlaylist,
		"on_LibraryAppendMenuItem_activate":            func() { w.applyLibrarySelection(tbFalse) },
		"on_LibraryReplaceMenuItem_activate":           func() { w.applyLibrarySelection(tbTrue) },
		"on_LibraryRenameMenuItem_activate":            w.libraryRename,
		"on_LibraryDeleteMenuItem_activate":            w.libraryDelete,
		"on_LibraryUpdateSelMenuItem_activate":         func() { w.libraryUpdate(false, true) },
		"on_StreamsAppendMenuItem_activate":            func() { w.applyStreamSelection(tbFalse) },
		"on_StreamsReplaceMenuItem_activate":           func() { w.applyStreamSelection(tbTrue) },
		"on_StreamsEditMenuItem_activate":              w.onStreamEdit,
		"on_StreamsDeleteMenuItem_activate":            w.onStreamDelete,
	})

	// Register the main window with the app
	application.AddWindow(w.AppWindow)

	// Restore library path
	cfg := config.GetConfig()
	errCheck(w.libPath.Unmarshal(cfg.LibraryPath), "Failed to restore library path")

	// Restore window dimensions
	dim := cfg.MainWindowDimensions
	if dim.Width > 0 && dim.Height > 0 {
		w.AppWindow.Resize(dim.Width, dim.Height)
	}
	if dim.X >= 0 && dim.Y >= 0 {
		w.AppWindow.Move(dim.X, dim.Y)
	}

	// Instantiate a connector
	w.connector = NewConnector(w.onConnectorStatusChange, w.onConnectorHeartbeat, w.onConnectorSubsystemChange)
	return w, nil
}

func (w *MainWindow) onConnectorStatusChange() {
	// Ignore when not mapped
	if w.mapped {
		glib.IdleAdd(w.updateAll)
	}
}

func (w *MainWindow) onConnectorHeartbeat() {
	// Ignore when not mapped
	if w.mapped {
		glib.IdleAdd(w.updatePlayerSeekBar)
	}
}

func (w *MainWindow) onConnectorSubsystemChange(subsystem string) {
	log.Debugf("onSubsystemChange(%v)", subsystem)
	// Ignore when not mapped
	if !w.mapped {
		return
	}

	switch subsystem {
	case "database", "update":
		glib.IdleAdd(w.updateLibrary)
	case "mixer":
		glib.IdleAdd(w.updateVolume)
	case "options":
		glib.IdleAdd(w.updateOptions)
	case "player":
		glib.IdleAdd(w.updatePlayer)
	case "playlist":
		glib.IdleAdd(func() {
			w.updateQueue()
			w.updatePlayer()
		})
	case "stored_playlist":
		if _, ok := w.libPath.Last().(*PlaylistsLibElement); ok {
			glib.IdleAdd(w.updateLibrary)
		}
	}
}

func (w *MainWindow) onMap() {
	log.Debug("MainWindow.onMap()")

	// Update all lists
	w.updateAll()
	w.updateStreams()

	// Activate the Queue tree view
	w.focusMainList()

	// Start connecting if needed
	if config.GetConfig().MpdAutoConnect {
		w.connect()
	}
	w.mapped = true
}

func (w *MainWindow) onDelete() {
	log.Debug("MainWindow.onDelete()")
	w.mapped = false
	cfg := config.GetConfig()

	// Save the current library path
	cfg.LibraryPath = w.libPath.Marshal()

	// Save the current window dimensions in the config
	x, y := w.AppWindow.GetPosition()
	width, height := w.AppWindow.GetSize()
	cfg.MainWindowDimensions = config.Dimensions{X: x, Y: y, Width: width, Height: height}

	// Write out the config
	cfg.Save()

	// Disconnect from MPD
	w.disconnect()
}

func (w *MainWindow) onLibraryAddToPlaylist(playlist string) {
	log.Debugf("MainWindow.onLibraryAddToPlaylist(%s)", playlist)

	// Fetch the selected element, which must be playable
	element := w.getSelectedLibraryElement()
	if element == nil || !element.IsPlayable() {
		return
	}

	// If it's a URI-enabled element
	if uh, ok := element.(URIHolder); ok {
		w.libraryAppendPlaylist(playlist, uh.URI())
		return
	}

	// Playlist-enabled element
	if ph, ok := element.(PlaylistHolder); ok {
		var attrs []mpd.Attrs
		var err error
		w.connector.IfConnected(func(client *mpd.Client) {
			attrs, err = client.PlaylistContents(ph.PlaylistName())
		})
		if w.errCheckDialog(err, glib.Local("Failed to add item to the playlist")) {
			return
		}

		// Extract the URIs and append them to the playlist
		w.libraryAppendPlaylist(playlist, util.MapAttrsToSlice(attrs, "file")...)
		return
	}

	// Attribute-enabled path: extend the current path filter with the element and query the tracks
	if filter := w.libPath.AsFilter(element); len(filter) > 0 {
		var attrs []mpd.Attrs
		var err error
		w.connector.IfConnected(func(client *mpd.Client) {
			attrs, err = client.Find(filter...)
		})
		if w.errCheckDialog(err, glib.Local("Failed to add item to the playlist")) {
			return
		}

		// Extract the URIs and append them to the playlist
		w.libraryAppendPlaylist(playlist, util.MapAttrsToSlice(attrs, "file")...)
		return
	}

	// Oops
	log.Errorf("element %T cannot be added to a playlist", element)
}

func (w *MainWindow) onLibraryListBoxButtonPress(_ *gtk.ListBox, event *gdk.Event) {
	switch btn := gdk.EventButtonNewFromEvent(event); btn.Type() {
	// Mouse click
	case gdk.EVENT_BUTTON_PRESS:
		// Right click
		if btn.Button() == 3 {
			w.LibraryListBox.SelectRow(w.LibraryListBox.GetRowAtY(int(btn.Y())))
			w.LibraryMenu.PopupAtPointer(event)
		}
	// Double click
	case gdk.EVENT_DOUBLE_BUTTON_PRESS:
		w.applyLibrarySelection(tbNone)
	}
}

func (w *MainWindow) onLibraryListBoxKeyPress(_ *gtk.ListBox, event *gdk.Event) {
	evt := gdk.EventKeyNewFromEvent(event)
	state := gdk.ModifierType(evt.State()) & gtk.AcceleratorGetDefaultModMask()
	switch evt.KeyVal() {
	// Enter: we need to go deeper
	case gdk.KEY_Return:
		switch state {
		// Enter: use default mode
		case 0:
			w.applyLibrarySelection(tbNone)
		// Ctrl+Enter: replace
		case gdk.CONTROL_MASK:
			w.applyLibrarySelection(tbTrue)
		// Shift+Enter: append
		case gdk.SHIFT_MASK:
			w.applyLibrarySelection(tbFalse)
		}

	// Backspace: go level up (not in search mode)
	case gdk.KEY_BackSpace:
		if state == 0 && !w.LibrarySearchToolButton.GetActive() {
			w.libraryLevelUp()
		}

	// Escape: deactivate search mode
	case gdk.KEY_Escape:
		if state == 0 {
			w.onLibraryStopSearch()
		}

	// Ctrl+F: activate search mode
	case gdk.KEY_f:
		if state == gdk.CONTROL_MASK {
			w.LibrarySearchToolButton.SetActive(true)
		}
	}
}

// onLibrarySearchToggle activates or deactivates library search mode
func (w *MainWindow) onLibrarySearchToggle() {
	searchMode := w.LibrarySearchToolButton.GetActive()

	// Show the appropriate tool stack's page
	if searchMode {
		w.LibraryToolStack.SetVisibleChild(w.LibrarySearchBox)
		// Clear and shift focus to the search entry
		w.LibrarySearchEntry.SetText("")
		w.LibrarySearchEntry.GrabFocus()
	} else {
		w.LibraryToolStack.SetVisibleChild(w.LibraryPathBox)
	}

	// Run search or load library
	w.updateLibrary()

	// If search mode finished, move focus to the library list
	if !searchMode {
		w.focusMainList()
	}
}

// onLibraryStopSearch deactivates library search mode
func (w *MainWindow) onLibraryStopSearch() {
	w.LibrarySearchToolButton.SetActive(false)
}

func (w *MainWindow) onLibraryPathChanged() {
	// Ignore when not mapped
	if w.mapped {
		w.updateLibraryPath()
		w.updateLibrary()
		w.focusMainList()
	}
}

func (w *MainWindow) onPlayPositionButtonEvent(_ interface{}, event *gdk.Event) {
	switch gdk.EventButtonNewFromEvent(event).Type() {
	case gdk.EVENT_BUTTON_PRESS:
		w.playPosUpdating = true

	case gdk.EVENT_BUTTON_RELEASE:
		w.playPosUpdating = false
		w.connector.IfConnected(func(client *mpd.Client) {
			d := time.Duration(w.PlayPositionAdjustment.GetValue())
			errCheck(client.SeekCur(d*time.Second, false), "SeekCur() failed")
		})
	}
}

func (w *MainWindow) onQueueSavePopoverValidate() {
	// Only show new playlist widgets if (new playlist) is selected in the combo box
	selectedID := w.QueueSavePlaylistComboBox.GetActiveID()
	isNew := selectedID == queueSaveNewPlaylistID
	w.QueueSavePlaylistNameLabel.SetVisible(isNew)
	w.QueueSavePlaylistNameEntry.SetVisible(isNew)

	// Validate the actions
	valid := (!isNew && selectedID != "") || (isNew && util.EntryText(w.QueueSavePlaylistNameEntry, "") != "")
	w.aQueueSaveReplace.SetEnabled(valid && !isNew)
	w.aQueueSaveAppend.SetEnabled(valid)
}

func (w *MainWindow) onQueueSearchMode() {
	w.queueFilter()

	// Return focus to the queue on deactivating search
	if !w.QueueSearchBar.GetSearchMode() {
		w.focusMainList()
	}
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
	for c := w.QueueTreeView.GetColumns(); c != nil; c = c.Next() {
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

func (w *MainWindow) onQueueTreeViewButtonPress(_ *gtk.TreeView, event *gdk.Event) bool {
	switch btn := gdk.EventButtonNewFromEvent(event); btn.Type() {
	// Mouse click
	case gdk.EVENT_BUTTON_PRESS:
		// Right click
		if btn.Button() == 3 {
			w.QueueMenu.PopupAtPointer(event)
			// Stop event propagation
			return true
		}
	// Double click
	case gdk.EVENT_DOUBLE_BUTTON_PRESS:
		w.applyQueueSelection()
	}
	return false
}

func (w *MainWindow) onQueueTreeViewKeyPress(_ *gtk.TreeView, event *gdk.Event) {
	evt := gdk.EventKeyNewFromEvent(event)
	state := gdk.ModifierType(evt.State()) & gtk.AcceleratorGetDefaultModMask()
	switch evt.KeyVal() {
	// Enter: apply current selection
	case gdk.KEY_Return:
		if state == 0 {
			w.applyQueueSelection()
		}
	// Esc: exit filtering mode if it's active
	case gdk.KEY_Escape:
		if state == 0 {
			w.QueueSearchBar.SetSearchMode(false)
		}
	// Delete
	case gdk.KEY_Delete:
		switch state {
		// Delete: delete selection
		case 0:
			w.queueDelete()
		// Ctrl+Delete: clear queue
		case gdk.CONTROL_MASK:
			w.queueClear()
		}
	// Space
	case gdk.KEY_space:
		if state == 0 {
			w.playerPlayPause()
		}
	// Ctrl+F: activate search bar
	case gdk.KEY_f:
		if state == gdk.CONTROL_MASK {
			w.QueueSearchBar.SetSearchMode(true)
		}
	}
}

func (w *MainWindow) onStreamAdd() {
	// Reset property values
	w.StreamPropsNameEntry.SetText("")
	w.StreamPropsUriEntry.SetText("")

	// Disable the Apply action initially
	w.aStreamPropsApply.SetEnabled(false)

	// Show the popover
	w.addingStream = true
	w.StreamPropsPopoverMenu.SetRelativeTo(w.StreamsAddToolButton)
	w.StreamPropsPopoverMenu.Popup()
}

func (w *MainWindow) onStreamDelete() {
	// Fetch the selected stream
	idx := w.getSelectedStreamIndex()
	if idx < 0 {
		return
	}

	// Ask for a confirmation
	streams := &config.GetConfig().Streams
	if util.ConfirmDialog(w.AppWindow, glib.Local("Delete stream"), fmt.Sprintf(glib.Local("Are you sure you want to delete stream \"%s\"?"), (*streams)[idx].Name)) {
		// Delete the selected stream from the slice
		*streams = append((*streams)[:idx], (*streams)[idx+1:]...)

		// Update stream list
		w.updateStreams()
		w.focusMainList()
	}
}

func (w *MainWindow) onStreamEdit() {
	// Fetch the selected stream
	idx := w.getSelectedStreamIndex()
	if idx < 0 {
		return
	}
	stream := config.GetConfig().Streams[idx]

	// Reset property values
	w.StreamPropsNameEntry.SetText(stream.Name)
	w.StreamPropsUriEntry.SetText(stream.URI)

	// Disable the Apply action initially
	w.aStreamPropsApply.SetEnabled(false)

	// Show the popover
	w.addingStream = false
	w.StreamPropsPopoverMenu.SetRelativeTo(w.StreamsEditToolButton)
	w.StreamPropsPopoverMenu.Popup()
}

func (w *MainWindow) onStreamListBoxButtonPress(_ *gtk.ListBox, event *gdk.Event) {
	switch btn := gdk.EventButtonNewFromEvent(event); btn.Type() {
	// Mouse click
	case gdk.EVENT_BUTTON_PRESS:
		// Right click
		if btn.Button() == 3 {
			w.StreamsListBox.SelectRow(w.StreamsListBox.GetRowAtY(int(btn.Y())))
			w.StreamsMenu.PopupAtPointer(event)
		}
	// Double click
	case gdk.EVENT_DOUBLE_BUTTON_PRESS:
		w.applyStreamSelection(tbNone)
	}
}

func (w *MainWindow) onStreamListBoxKeyPress(_ *gtk.ListBox, event *gdk.Event) {
	evt := gdk.EventKeyNewFromEvent(event)
	state := gdk.ModifierType(evt.State()) & gtk.AcceleratorGetDefaultModMask()
	switch evt.KeyVal() {
	// Enter: apply selection
	case gdk.KEY_Return:
		switch state {
		// Enter: use default mode
		case 0:
			w.applyStreamSelection(tbNone)
		// Ctrl+Enter: replace
		case gdk.CONTROL_MASK:
			w.applyStreamSelection(tbTrue)
		// Shift+Enter: append
		case gdk.SHIFT_MASK:
			w.applyStreamSelection(tbFalse)
		}
	}
}

func (w *MainWindow) onStreamPropsApply() {
	// Fetch entered data
	name, uri := util.EntryText(w.StreamPropsNameEntry, ""), util.EntryText(w.StreamPropsUriEntry, "")
	if name == "" || uri == "" {
		return
	}

	// Make a stream spec instance
	stream := config.StreamSpec{
		Name: name,
		URI:  uri,
	}

	// Adding a stream
	cfg := config.GetConfig()
	if w.addingStream {
		cfg.Streams = append(cfg.Streams, stream)

	} else if idx := w.getSelectedStreamIndex(); idx >= 0 {
		// Editing an existing stream
		cfg.Streams[idx] = stream
	}

	// Update stream list
	w.updateStreams()
	w.focusMainList()
}

func (w *MainWindow) onStreamPropsChanged() {
	// Validate the popover
	w.aStreamPropsApply.SetEnabled(
		util.EntryText(w.StreamPropsNameEntry, "") != "" &&
			util.EntryText(w.StreamPropsUriEntry, "") != "")
}

func (w *MainWindow) onVolumeValueChanged() {
	if !w.volumeUpdating {
		vol := int(w.VolumeAdjustment.GetValue())
		log.Debugf("Adjusting volume to %d", vol)
		w.connector.IfConnected(func(client *mpd.Client) {
			errCheck(client.SetVolume(vol), "SetVolume() failed")
		})
	}
}

// addAction add a new application action, with an optional keyboard shortcut
func (w *MainWindow) addAction(name, shortcut string, onActivate func()) *glib.SimpleAction {
	action := glib.SimpleActionNew(name, nil)
	action.Connect("activate", onActivate)
	w.app.AddAction(action)
	if shortcut != "" {
		w.app.SetAccelsForAction("app."+name, []string{shortcut})
	}
	return action
}

// applyLibrarySelection navigates into the folder or adds or replaces the content of the queue with the currently
// selected items in the library
func (w *MainWindow) applyLibrarySelection(replace triBool) {
	// Get selected element
	e := w.getSelectedLibraryElement()
	if e == nil {
		return
	}

	// Level-up element
	if _, ok := e.(*LevelUpLibElement); ok {
		w.libraryLevelUp()

	} else if replace == tbNone && e.IsFolder() {
		// Default for folders is entering into
		w.libPath.Append(e)

	} else {
		// Queue the element up otherwise
		w.queueLibraryElement(replace, e)
	}
}

// applyPlayerSettings compiles the player title template and updates the player
func (w *MainWindow) applyPlayerSettings() {
	// Apply toolbar setting
	cfg := config.GetConfig()
	w.QueueToolbar.SetVisible(cfg.QueueToolbar)

	// Compile and apply the track title template
	tmpl, err := template.New("playerTitle").
		Funcs(template.FuncMap{
			"default":  util.Default,
			"dirname":  path.Dir,
			"basename": path.Base,
		}).
		Parse(cfg.PlayerTitleTemplate)
	if errCheck(err, "Template parse error") {
		w.playerTitleTemplate = template.Must(
			template.New("error").Parse("<span foreground=\"red\">[" + glib.Local("Player title template error, check log") + "]</span>"))
	} else {
		w.playerTitleTemplate = tmpl
	}

	// Update the displayed title/artwork if the connector is initialised
	if w.connector != nil {
		w.updatePlayer()
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
	w.errCheckDialog(err, glib.Local("Failed to play the selected track"))
}

// applyStreamSelection adds or replaces the content of the queue with the currently selected stream
func (w *MainWindow) applyStreamSelection(replace triBool) {
	if idx := w.getSelectedStreamIndex(); idx >= 0 {
		w.queueStream(replace, config.GetConfig().Streams[idx].URI)
	}
}

// connect starts connecting to MPD
func (w *MainWindow) connect() {
	// First disconnect, if connected
	w.disconnect()

	// Start connecting
	cfg := config.GetConfig()
	network, addr := cfg.MpdNetworkAddress()
	w.connector.Start(network, addr, cfg.MpdPassword, cfg.MpdAutoReconnect)
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
		util.ErrorDialog(w.AppWindow, formatted)
		return true
	}
	return false
}

// focusMainList transfers the focus to the main list on the currently visible page
func (w *MainWindow) focusMainList() {
	var widget *gtk.Widget
	switch w.MainStack.GetVisibleChildName() {
	case "queue":
		widget = &w.QueueTreeView.Widget

	// Library: move focus to the selected row, if any
	case "library":
		if row := w.LibraryListBox.GetSelectedRow(); row != nil {
			widget = &row.Widget
		} else {
			widget = &w.LibraryListBox.Widget
		}

	// Streams: move focus to the selected row, if any
	case "streams":
		if row := w.StreamsListBox.GetSelectedRow(); row != nil {
			widget = &row.Widget
		} else {
			widget = &w.StreamsListBox.Widget
		}
	}

	// Move focus
	if widget != nil {
		widget.GrabFocus()
	}
}

// getQueueHasSelection returns whether there's any selected rows in the queue
func (w *MainWindow) getQueueSelectedCount() int {
	if sel, err := w.QueueTreeView.GetSelection(); !errCheck(err, "getQueueHasSelection(): QueueTreeView.GetSelection() failed") {
		return sel.CountSelectedRows()
	}
	return 0
}

// getQueueSelectedIndices returns indices of the currently selected rows in the queue
func (w *MainWindow) getQueueSelectedIndices() []int {
	// Get the tree's selection
	sel, err := w.QueueTreeView.GetSelection()
	if errCheck(err, "QueueTreeView.GetSelection() failed") {
		return nil
	}

	// Get selected nodes' indices
	var indices []int
	sel.SelectedForEach(func(model *gtk.TreeModel, path *gtk.TreePath, iter *gtk.TreeIter) {
		// Convert the provided tree (filtered) path into unfiltered one
		if queuePath := w.QueueTreeModelFilter.ConvertPathToChildPath(path); queuePath != nil {
			if ix := queuePath.GetIndices(); len(ix) > 0 {
				indices = append(indices, ix[0])
			}
		}
	})
	return indices
}

// getQueueSelectedTrackAttrs returns attributes of the first currently selected row in the queue
func (w *MainWindow) getQueueSelectedTrackAttrs() (mpd.Attrs, error) {
	// Get the tree's selection
	if indices := w.getQueueSelectedIndices(); len(indices) > 0 {
		// Fetch the attrs of the first selected index
		var err error
		var attrs []mpd.Attrs
		w.connector.IfConnected(func(client *mpd.Client) {
			attrs, err = client.PlaylistInfo(indices[0], -1)
		})

		// If there's an error
		if err != nil {
			return nil, err
		}

		// If no data returned
		if len(attrs) == 0 {
			return nil, errors.New("No data returned by MPD for the current selection")
		}

		// All OK
		return attrs[0], nil
	}
	return nil, errors.New("No selection in the queue")
}

// getSelectedLibraryElement returns the path element of the currently selected library item or nil if there's an error
func (w *MainWindow) getSelectedLibraryElement() LibraryPathElement {
	// If there's selection
	row := w.LibraryListBox.GetSelectedRow()
	if row == nil {
		return nil
	}

	// Extract path, which is stored in the row's name
	name, err := row.GetName()
	if errCheck(err, "getSelectedLibraryPath(): row.GetName() failed") {
		return nil
	}

	// Unmarshal the element from name
	if element, err := UnmarshalLibPathElement(name); !errCheck(err, "Unmarshalling failed") {
		return element
	}
	return nil
}

// getSelectedStreamIndex returns the index of the currently selected stream, or -1 if there's an error
func (w *MainWindow) getSelectedStreamIndex() int {
	// If there's selection
	row := w.StreamsListBox.GetSelectedRow()
	if row == nil {
		return -1
	}
	return row.GetIndex()
}

// initLibraryWidgets initialises library widgets and actions
func (w *MainWindow) initLibraryWidgets() {
	// Create actions
	w.aLibraryUpdate = w.addAction("library.update", "", w.LibraryUpdatePopoverMenu.Popup)
	w.aLibraryUpdateAll = w.addAction("library.update.all", "", func() { w.libraryUpdate(false, false) })
	w.aLibraryUpdateSel = w.addAction("library.update.selected", "", func() { w.libraryUpdate(false, true) })
	w.aLibraryRescanAll = w.addAction("library.rescan.all", "", func() { w.libraryUpdate(true, false) })
	w.aLibraryRescanSel = w.addAction("library.rescan.selected", "", func() { w.libraryUpdate(true, true) })
	w.aLibraryRename = w.addAction("library.rename", "", w.libraryRename)
	w.aLibraryDelete = w.addAction("library.delete", "", w.libraryDelete)
	w.aLibraryAddToPlaylist = w.addAction("library.add-to-playlist", "", w.libraryAddToPlaylist)
	w.addAction("library.search.toggle", "", w.onLibrarySearchToggle)

	// Create a library path instance
	w.libPath = NewLibraryPath(w.onLibraryPathChanged)

	// Populate search attribute combo box
	w.LibrarySearchAttrComboBox.Append(librarySearchAllAttrID, glib.Local("Everywhere"))
	for _, id := range config.MpdTrackAttributeIds {
		if config.MpdTrackAttributes[id].Searchable {
			w.LibrarySearchAttrComboBox.Append(strconv.Itoa(id), glib.Local(config.MpdTrackAttributes[id].LongName))
		}
	}
	w.LibrarySearchAttrComboBox.SetActiveID(librarySearchAllAttrID)
}

// initPlayerWidgets initialises player widgets and actions
func (w *MainWindow) initPlayerWidgets() {
	// Create actions
	w.aPlayerPrevious = w.addAction("player.previous", "<Ctrl>Left", w.playerPrevious)
	w.aPlayerStop = w.addAction("player.stop", "<Ctrl>S", w.playerStop)
	w.aPlayerPlayPause = w.addAction("player.play-pause", "<Ctrl>P", w.playerPlayPause)
	w.aPlayerNext = w.addAction("player.next", "<Ctrl>Right", w.playerNext)
	// NB convert to stateful actions once Gotk3 supporting GVariant is released
	w.aPlayerRandom = w.addAction("player.toggle.random", "<Ctrl>U", w.playerToggleRandom)
	w.aPlayerRepeat = w.addAction("player.toggle.repeat", "<Ctrl>R", w.playerToggleRepeat)
	w.aPlayerConsume = w.addAction("player.toggle.consume", "<Ctrl>N", w.playerToggleConsume)
}

// initQueueWidgets initialises queue widgets and actions
func (w *MainWindow) initQueueWidgets() {
	// Configure the search bar
	glib.BindProperty(w.QueueSearchBar.Object, "search-mode-enabled", w.QueueFilterToolButton.Object, "active", glib.BINDING_BIDIRECTIONAL)
	glib.BindProperty(w.QueueSearchBar.Object, "search-mode-enabled", w.QueueFilterLabel.Object, "visible", glib.BINDING_DEFAULT)

	// Forcefully disable tree search popup on Ctrl+F
	w.QueueTreeView.SetSearchColumn(-1)

	// Create actions
	w.aQueueNowPlaying = w.addAction("queue.now-playing", "<Ctrl>J", w.updateQueueNowPlaying)
	w.aQueueClear = w.addAction("queue.clear", "", w.queueClear)
	w.aQueueSort = w.addAction("queue.sort", "", w.QueueSortPopoverMenu.Popup)
	w.aQueueSortAsc = w.addAction("queue.sort.asc", "", func() { w.queueSortApply(false) })
	w.aQueueSortDesc = w.addAction("queue.sort.desc", "", func() { w.queueSortApply(true) })
	w.aQueueSortShuffle = w.addAction("queue.sort.shuffle", "<Ctrl><Shift>R", w.queueShuffle)
	w.aQueueDelete = w.addAction("queue.delete", "", w.queueDelete)
	w.aQueueSave = w.addAction("queue.save", "", w.queueSave)
	w.aQueueSaveReplace = w.addAction("queue.save.replace", "", func() { w.queueSaveApply(true) })
	w.aQueueSaveAppend = w.addAction("queue.save.append", "", func() { w.queueSaveApply(false) })

	// Populate "Queue sort by" combo box
	for _, id := range config.MpdTrackAttributeIds {
		w.QueueSortByComboBox.Append(strconv.Itoa(id), glib.Local(config.MpdTrackAttributes[id].LongName))
	}
	w.QueueSortByComboBox.SetActiveID(strconv.Itoa(config.GetConfig().DefaultSortAttrID))

	// Update Queue tree view columns
	w.updateQueueColumns()
}

// initStreamsWidgets initialises streams widgets and actions
func (w *MainWindow) initStreamsWidgets() {
	// Create actions
	w.aStreamAdd = w.addAction("stream.add", "", w.onStreamAdd)
	w.aStreamEdit = w.addAction("stream.edit", "", w.onStreamEdit)
	w.aStreamDelete = w.addAction("stream.delete", "", w.onStreamDelete)
	w.aStreamPropsApply = w.addAction("stream.props.apply", "", w.onStreamPropsApply)
}

// initWidgets initialises all widgets and actions
func (w *MainWindow) initWidgets() {
	// Determine base colours
	w.updateStyle()

	// Create global actions
	w.addAction("mpd.connect", "<Ctrl><Shift>C", w.connect)
	w.aMPDDisconnect = w.addAction("mpd.disconnect", "<Ctrl><Shift>D", w.disconnect)
	w.aMPDInfo = w.addAction("mpd.info", "<Ctrl><Shift>I", w.showMPDInfo)
	w.addAction("prefs", "<Ctrl>comma", w.showPreferences)
	w.aMPDOutputs = w.addAction("outputs", "<Ctrl>O", w.showOutputs)
	w.addAction("about", "F1", w.showAbout)
	w.addAction("shortcuts", "<Ctrl><Shift>question", w.showShortcuts)
	w.addAction("quit", "<Ctrl>Q", w.AppWindow.Close)
	w.addAction("page.queue", "<Ctrl>1", func() { w.MainStack.SetVisibleChild(w.QueueBox) })
	w.addAction("page.library", "<Ctrl>2", func() { w.MainStack.SetVisibleChild(w.LibraryBox) })
	w.addAction("page.streams", "<Ctrl>3", func() { w.MainStack.SetVisibleChild(w.StreamsBox) })

	// Init other widgets and actions
	w.initQueueWidgets()
	w.initLibraryWidgets()
	w.initStreamsWidgets()
	w.initPlayerWidgets()
}

// libraryAddToPlaylist shows a popover menu that allows to add the selected library element to a playlist
func (w *MainWindow) libraryAddToPlaylist() {
	// Clean up and repopulate the menu with playlists
	util.ClearChildren(w.LibraryAddToPlaylistBox.Container)
	for _, name := range w.connector.GetPlaylists() {
		name := name // Make an in-loop copy

		// Make a new button
		btn, err := gtk.ModelButtonNew()
		if errCheck(err, "ModelButtonNew() failed") {
			return
		}

		// Set the text using a generic setter (due to https://github.com/gotk3/gotk3/issues/742)
		errCheck(btn.Set("text", name), "Set(text) failed")

		// Cannot bind to "activate" here as it's not triggered for Actionable widgets
		btn.Connect("clicked", func() { w.onLibraryAddToPlaylist(name) })

		// Add the button to the popover
		w.LibraryAddToPlaylistBox.PackStart(btn, false, true, 0)
	}

	// Show the popover
	w.LibraryAddToPlaylistBox.ShowAll()
	w.LibraryAddToPlaylistPopoverMenu.Popup()
}

// libraryAppendPlaylist appends the provided URIs to a playlist with the given name
func (w *MainWindow) libraryAppendPlaylist(name string, uris ...string) {
	err := errors.New(glib.Local("Not connected to MPD"))
	w.connector.IfConnected(func(client *mpd.Client) {
		commands := client.BeginCommandList()
		for _, uri := range uris {
			commands.PlaylistAdd(name, uri)
		}
		err = commands.End()
	})

	// Check for error
	w.errCheckDialog(err, glib.Local("Failed to add item to the playlist"))
}

// libraryDelete allows to delete the selected library element
func (w *MainWindow) libraryDelete() {
	element := w.getSelectedLibraryElement()
	if ph, ok := element.(PlaylistHolder); ok {
		if util.ConfirmDialog(w.AppWindow, glib.Local("Delete playlist"), fmt.Sprintf(glib.Local("Are you sure you want to delete playlist \"%s\"?"), ph.PlaylistName())) {
			var err error
			w.connector.IfConnected(func(client *mpd.Client) {
				err = client.PlaylistRemove(ph.PlaylistName())
			})
			// Check for error (outside IfConnected() because it would keep the client locked)
			w.errCheckDialog(err, glib.Local("Failed to delete the playlist"))
		}
	}
}

// libraryLevelUp navigates to the library element at the upper level
func (w *MainWindow) libraryLevelUp() {
	if e := w.libPath.Last(); e != nil {
		// Save the currently active path element for subsequent selection
		w.libPathElementToSelect = e.Marshal()
		// Move up a level
		w.libPath.LevelUp()
	}
}

// libraryRename allows to rename the selected library element
func (w *MainWindow) libraryRename() {
	element := w.getSelectedLibraryElement()
	if ph, ok := element.(PlaylistHolder); ok {
		if newName, ok := util.EditDialog(w.AppWindow, glib.Local("Rename playlist"), ph.PlaylistName(), glib.Local("Rename")); ok {
			var err error
			w.connector.IfConnected(func(client *mpd.Client) {
				err = client.PlaylistRename(ph.PlaylistName(), newName)
			})
			// Check for error (outside IfConnected() because it would keep the client locked)
			w.errCheckDialog(err, glib.Local("Failed to rename the playlist"))
		}
	}
}

// libraryShowAlbumFromQueue opens the currently selected queue album in the library
func (w *MainWindow) libraryShowAlbumFromQueue() {
	if attrs, err := w.getQueueSelectedTrackAttrs(); !w.errCheckDialog(err, glib.Local("Failed to get album information")) {
		// Update the current library path
		w.libPath.SetElements([]LibraryPathElement{
			NewArtistsLibElement(),
			NewArtistLibElementVal(attrs[config.MpdTrackAttributes[config.MTAttrArtist].AttrName]),
			NewAlbumLibElementVal(attrs[config.MpdTrackAttributes[config.MTAttrAlbum].AttrName]),
		})

		// Switch to the library tab
		w.MainStack.SetVisibleChild(w.LibraryBox)
	}
}

// libraryShowArtistFromQueue opens the currently selected queue artist in the library
func (w *MainWindow) libraryShowArtistFromQueue() {
	if attrs, err := w.getQueueSelectedTrackAttrs(); !w.errCheckDialog(err, glib.Local("Failed to get artist information")) {
		// Update the current library path
		w.libPath.SetElements([]LibraryPathElement{
			NewArtistsLibElement(),
			NewArtistLibElementVal(attrs[config.MpdTrackAttributes[config.MTAttrArtist].AttrName]),
		})

		// Switch to the library tab
		w.MainStack.SetVisibleChild(w.LibraryBox)
	}
}

// libraryShowGenreFromQueue opens the currently selected queue genre in the library
func (w *MainWindow) libraryShowGenreFromQueue() {
	if attrs, err := w.getQueueSelectedTrackAttrs(); !w.errCheckDialog(err, glib.Local("Failed to get genre information")) {
		// Update the current library path
		w.libPath.SetElements([]LibraryPathElement{
			NewGenresLibElement(),
			NewGenreLibElementVal(attrs[config.MpdTrackAttributes[config.MTAttrGenre].AttrName]),
		})

		// Switch to the library tab
		w.MainStack.SetVisibleChild(w.LibraryBox)
	}
}

// libraryUpdate updates or rescans the library
func (w *MainWindow) libraryUpdate(rescan, selectedOnly bool) {
	// Determine the update path
	libPath := ""
	if selectedOnly {
		// We only support updating file-based items
		uh, ok := w.getSelectedLibraryElement().(URIHolder)
		if !ok {
			return
		}
		libPath = uh.URI()
	}

	// Run the update
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		if rescan {
			_, err = client.Rescan(libPath)
		} else {
			_, err = client.Update(libPath)
		}
	})

	// Check for error
	w.errCheckDialog(err, glib.Local("Failed to update the library"))
}

// playerPrevious rewinds the player to the previous track
func (w *MainWindow) playerPrevious() {
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		err = client.Previous()
	})

	// Check for error
	w.errCheckDialog(err, glib.Local("Failed to skip to previous track"))
}

// playerStop stops the playback
func (w *MainWindow) playerStop() {
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		err = client.Stop()
	})

	// Check for error
	w.errCheckDialog(err, glib.Local("Failed to stop playback"))
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
	w.errCheckDialog(err, glib.Local("Failed to toggle playback"))
}

// playerNext advances the player to the next track
func (w *MainWindow) playerNext() {
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		err = client.Next()
	})

	// Check for error
	w.errCheckDialog(err, glib.Local("Failed to skip to next track"))
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
	w.errCheckDialog(err, glib.Local("Failed to toggle consume mode"))
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
	w.errCheckDialog(err, glib.Local("Failed to toggle random mode"))
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
	w.errCheckDialog(err, glib.Local("Failed to toggle repeat mode"))
}

// queueClear empties MPD's play queue
func (w *MainWindow) queueClear() {
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		err = client.Clear()
	})

	// Check for error
	w.errCheckDialog(err, glib.Local("Failed to clear the queue"))
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
	w.errCheckDialog(err, glib.Local("Failed to delete tracks from the queue"))
}

// queueFilter applies the currently entered filter substring to the queue
func (w *MainWindow) queueFilter() {
	substr := ""

	// Only use filter pattern if the search bar is visible
	if w.QueueSearchBar.GetSearchMode() {
		substr = util.EntryText(&w.QueueSearchEntry.Entry, "")
	}

	// Iterate all rows in the list store
	count := 0
	w.QueueListStore.ForEach(func(model *gtk.TreeModel, path *gtk.TreePath, iter *gtk.TreeIter) bool {
		// Show all rows if no search pattern given
		visible := substr == ""
		if !visible {
			// We're going to compare case-insensitively
			substr := strings.ToLower(substr)

			// Scan all known columns in the row
			for _, id := range config.MpdTrackAttributeIds {
				// Get column's value
				v, err := model.GetValue(iter, id)
				if errCheck(err, "queueFilter(): QueueListStore.GetValue() failed") {
					continue
				}

				// Convert the value into a string (ignore any error caused by a missing value as we don't store them)
				s, _ := v.GetString()

				// Check for a match and stop checking if match has already been found
				visible = s != "" && strings.Contains(strings.ToLower(s), substr)
				if visible {
					break
				}
			}
		}

		// Modify the row's visibility
		if err := w.QueueListStore.SetValue(iter, config.QueueColumnVisible, visible); errCheck(err, "queueFilter(): QueueListStore.SetValue() failed") {
			return true
		}
		if visible {
			count++
		}

		// Proceed to the next row
		return false
	})
	w.QueueFilterLabel.SetText(fmt.Sprintf(glib.Local("%d track(s) displayed"), count))
}

// queueLibraryElement adds or replaces the content of the queue with the specified library path element
func (w *MainWindow) queueLibraryElement(replace triBool, element LibraryPathElement) {
	// Element must be playable
	if !element.IsPlayable() {
		return
	}

	// If it's a URI-enabled element
	if uh, ok := element.(URIHolder); ok {
		w.queueURIs(replace, uh.URI())
		return
	}

	// Playlist-enabled element
	if ph, ok := element.(PlaylistHolder); ok {
		w.queuePlaylist(replace, ph.PlaylistName())
		return
	}

	// Attribute-enabled path: extend the current path filter with the element
	if filter := w.libPath.AsFilter(element); len(filter) > 0 {
		var attrs []mpd.Attrs
		var err error
		w.connector.IfConnected(func(client *mpd.Client) {
			// For the lack of FindAdd() command in gompd, we need to query tracks first
			attrs, err = client.Find(filter...)
		})

		// Check for error
		if w.errCheckDialog(err, glib.Local("Failed to add item to the queue")) {
			return
		}

		// Convert attrs to list of URIs and queue them
		w.queueURIs(replace, util.MapAttrsToSlice(attrs, "file")...)
		return
	}

	// Oops
	log.Errorf("Element %T cannot be queued", element)
}

// queuePlaylist adds or replaces the content of the queue with the specified playlist
func (w *MainWindow) queuePlaylist(replace triBool, uri string) {
	log.Debugf("queuePlaylist(%v, %v)", replace, uri)
	var err error
	replaced := replace == tbTrue || replace == tbNone && config.GetConfig().PlaylistDefaultReplace
	w.connector.IfConnected(func(client *mpd.Client) {
		commands := client.BeginCommandList()

		// Clear the queue, if needed
		if replaced {
			commands.Clear()
		}

		// Add the content of the playlist
		// NB: extract only playlist name from the URI for now
		commands.PlaylistLoad(strings.TrimSuffix(path.Base(uri), ".m3u"), -1, -1)

		// Run the commands
		err = commands.End()
	})

	// Check for error
	if w.errCheckDialog(err, glib.Local("Failed to add playlist to the queue")) {
		return
	}

	// Initiate post-replace actions, if necessary
	if replaced {
		w.queueReplaced()
	}
}

// queueReplaced runs necessary post-queue-replace actions
func (w *MainWindow) queueReplaced() {
	// Switch to the queue tab
	if config.GetConfig().SwitchToOnQueueReplace {
		w.MainStack.SetVisibleChild(w.QueueBox)
	}

	// Initiate playback
	if config.GetConfig().PlayOnQueueReplace {
		var err error
		w.connector.IfConnected(func(client *mpd.Client) {
			err = client.Play(0)
		})

		// Check for error
		if w.errCheckDialog(err, glib.Local("Failed to start playback")) {
			return
		}
	}
}

// queueSave shows a dialog for saving the play queue into a playlist and performs the operation if confirmed
func (w *MainWindow) queueSave() {
	// Tweak widgets
	selection := w.getQueueSelectedCount() > 0
	w.QueueSaveSelectedOnlyCheckButton.SetVisible(selection)
	w.QueueSaveSelectedOnlyCheckButton.SetActive(selection)
	w.QueueSavePlaylistNameEntry.SetText("")

	// Populate the playlists combo box
	w.QueueSavePlaylistComboBox.RemoveAll()
	w.QueueSavePlaylistComboBox.Append(queueSaveNewPlaylistID, glib.Local("(new playlist)"))
	for _, name := range w.connector.GetPlaylists() {
		w.QueueSavePlaylistComboBox.Append(name, name)
	}
	w.QueueSavePlaylistComboBox.SetActiveID(queueSaveNewPlaylistID)

	// Show the popover
	w.QueueSavePopoverMenu.Popup()
}

// queueSaveApply performs queue saving into a playlist
func (w *MainWindow) queueSaveApply(replace bool) {
	// Collect current values from the UI
	selIndices := w.getQueueSelectedIndices()
	selOnly := len(selIndices) > 0 && w.QueueSaveSelectedOnlyCheckButton.GetActive()
	name := w.QueueSavePlaylistComboBox.GetActiveID()
	isNew := name == queueSaveNewPlaylistID
	if isNew {
		name = util.EntryText(w.QueueSavePlaylistNameEntry, glib.Local("Unnamed"))
	}

	err := errors.New(glib.Local("Not connected to MPD"))
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
	w.errCheckDialog(err, glib.Local("Failed to create a playlist"))
}

// queueShuffle randomises MPD's play queue
func (w *MainWindow) queueShuffle() {
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		err = client.Shuffle(-1, -1)
	})

	// Check for error
	w.errCheckDialog(err, glib.Local("Failed to shuffle the queue"))
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
	w.errCheckDialog(err, glib.Local("Failed to sort the queue"))

}

// queueSortApply performs MPD's play queue ordering based on the currently selected in popover mode
func (w *MainWindow) queueSortApply(descending bool) {
	// Fetch the ID of the currently selected item in the Sort by combo box, and the corresponding attribute
	if attr, ok := config.MpdTrackAttributes[util.AtoiDef(w.QueueSortByComboBox.GetActiveID(), -1)]; ok {
		w.queueSort(&attr, descending)
	}
}

// queueStream adds or replaces the content of the queue with the specified stream
func (w *MainWindow) queueStream(replace triBool, uri string) {
	log.Debugf("queueStream(%v, %v)", replace, uri)
	var err error
	replaced := replace == tbTrue || replace == tbNone && config.GetConfig().StreamDefaultReplace
	w.connector.IfConnected(func(client *mpd.Client) {
		commands := client.BeginCommandList()

		// Clear the queue, if needed
		if replaced {
			commands.Clear()
		}

		// Add the URI of the stream
		commands.Add(uri)

		// Run the commands
		err = commands.End()
	})

	// Check for error
	if w.errCheckDialog(err, glib.Local("Failed to add stream to the queue")) {
		return
	}

	// Initiate post-replace actions, if necessary
	if replaced {
		w.queueReplaced()
	}
}

// queueURIs adds or replaces the content of the queue with the specified URIs
func (w *MainWindow) queueURIs(replace triBool, uris ...string) {
	var err error
	replaced := replace == tbTrue || replace == tbNone && config.GetConfig().TrackDefaultReplace
	w.connector.IfConnected(func(client *mpd.Client) {
		commands := client.BeginCommandList()

		// Clear the queue, if needed
		if replaced {
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
	if w.errCheckDialog(err, glib.Local("Failed to add track(s) to the queue")) {
		return
	}

	// Initiate post-replace actions, if necessary
	if replaced {
		w.queueReplaced()
	}
}

// Show displays the window and all its child widgets
func (w *MainWindow) Show() {
	w.AppWindow.Show()
}

// showAbout shows the application's about dialog
func (w *MainWindow) showAbout() {
	dlg, err := gtk.AboutDialogNew()
	if errCheck(err, "AboutDialogNew() failed") {
		return
	}
	dlg.SetLogoIconName(config.AppMetadata.Icon)
	dlg.SetProgramName(config.AppMetadata.Name)
	dlg.SetComments(fmt.Sprintf(glib.Local("Release date: %s"), config.AppMetadata.BuildDate))
	dlg.SetCopyright(glib.Local(config.AppMetadata.Copyright))
	dlg.SetLicense(config.AppMetadata.License)
	dlg.SetVersion(config.AppMetadata.Version)
	dlg.SetWebsite(config.AppMetadata.URL)
	dlg.SetWebsiteLabel(config.AppMetadata.URLLabel)
	dlg.SetTransientFor(w.AppWindow)
	dlg.Connect("response", dlg.Destroy)
	dlg.Run()
}

// showMPDInfo displays a dialog with MPD information
func (w *MainWindow) showMPDInfo() {
	// Fetch information
	var version string
	var stats mpd.Attrs
	var decoders []mpd.Attrs
	var err error
	w.connector.IfConnected(func(client *mpd.Client) {
		// Fetch client version
		version = client.Version()
		// Fetch stats
		stats, err = client.Stats()
		if errCheck(err, "Stats() failed") {
			return
		}
		// Fetch decoder configuration
		decoders, err = client.Command("decoders").AttrsList("plugin")
		if errCheck(err, "Command(decoders) failed") {
			return
		}
	})
	if w.errCheckDialog(err, glib.Local("Failed to retrieve information from MPD")) || stats == nil {
		return
	}

	// Parse DB update time
	updateTime := glib.Local("(unknown)")
	if i, err := strconv.ParseInt(stats["db_update"], 10, 64); err == nil {
		updateTime = time.Unix(i, 0).Format("2006-01-02 15:04:05")
	}

	// Load widgets from Glade file
	var dlg struct {
		MPDInfoDialog           *gtk.MessageDialog
		PropertyGrid            *gtk.Grid
		DaemonVersionLabel      *gtk.Label
		NumberOfArtistsLabel    *gtk.Label
		NumberOfAlbumsLabel     *gtk.Label
		NumberOfTracksLabel     *gtk.Label
		TotalPlayingTimeLabel   *gtk.Label
		LastDatabaseUpdateLabel *gtk.Label
		DaemonUptimeLabel       *gtk.Label
		ListeningTimeLabel      *gtk.Label
		DecoderPluginsExpander  *gtk.Expander
		DecoderPluginsGrid      *gtk.Grid
	}
	builder, err := NewBuilder(mpdInfoGlade)
	if err == nil {
		err = builder.BindWidgets(&dlg)
	}
	if w.errCheckDialog(err, glib.Local("Failed to load UI widgets")) {
		return
	}
	defer dlg.MPDInfoDialog.Destroy()

	// Set info properties
	dlg.DaemonVersionLabel.SetLabel(version)
	dlg.NumberOfArtistsLabel.SetLabel(stats["artists"])
	dlg.NumberOfAlbumsLabel.SetLabel(stats["albums"])
	dlg.NumberOfTracksLabel.SetLabel(stats["songs"])
	dlg.TotalPlayingTimeLabel.SetLabel(util.FormatSecondsStr(stats["db_playtime"]))
	dlg.LastDatabaseUpdateLabel.SetLabel(updateTime)
	dlg.DaemonUptimeLabel.SetLabel(util.FormatSecondsStr(stats["uptime"]))
	dlg.ListeningTimeLabel.SetLabel(util.FormatSecondsStr(stats["playtime"]))

	// Add decoder plugins
	for i, decoder := range decoders {
		dlg.DecoderPluginsGrid.Attach(util.NewLabel(decoder["plugin"]), 0, i, 1, 1)
		if s, ok := decoder["suffix"]; ok {
			dlg.DecoderPluginsGrid.Attach(util.NewLabel("."+s), 1, i, 1, 1)
		}
		if s, ok := decoder["mime_type"]; ok {
			dlg.DecoderPluginsGrid.Attach(util.NewLabel(s), 2, i, 1, 1)
		}
	}

	// Set up and show the dialog
	dlg.MPDInfoDialog.SetTransientFor(w.AppWindow)
	dlg.MPDInfoDialog.ShowAll()
	dlg.MPDInfoDialog.Run()
}

// showOutputs shows the Outputs dialog
func (w *MainWindow) showOutputs() {
	ShowOutputsDialog(w.AppWindow, w.connector)
}

// showPreferences shows the Preferences dialog
func (w *MainWindow) showPreferences() {
	ShowPreferencesDialog(w.AppWindow, w.connect, w.updateQueueColumns, w.applyPlayerSettings)
}

// showShortcuts displays a shortcut info window
func (w *MainWindow) showShortcuts() {
	// Construct a window from the Glade resource
	builder, err := NewBuilder(shortcutsGlade)

	// Map the window's widgets
	win := struct {
		ShortcutsWindow *gtk.ShortcutsWindow
	}{}
	if err == nil {
		err = builder.BindWidgets(&win)
	}

	// Check for errors
	if w.errCheckDialog(err, "Failed to open the Shortcuts Window") {
		return
	}

	// Set up the window
	sw := win.ShortcutsWindow
	sw.SetTransientFor(w.AppWindow)
	sw.Connect("unmap", sw.Destroy)

	// Show the window
	sw.ShowAll()

	// For some reason, setting the active section name only works if the window is shown
	errCheck(sw.SetProperty("section-name", "shortcuts"), "Failed to set shortcut window's section name")
}

// setQueueHighlight selects or deselects an item in the Queue tree view at the given index
func (w *MainWindow) setQueueHighlight(index int, selected bool) {
	if index >= 0 {
		if iter, err := w.QueueListStore.GetIterFromString(strconv.Itoa(index)); err == nil {
			weight := fontWeightNormal
			bgColor := w.colourBgNormal
			if selected {
				weight = fontWeightBold
				bgColor = w.colourBgActive
			}
			errCheck(
				w.QueueListStore.SetCols(iter, map[int]interface{}{
					config.QueueColumnFontWeight: weight,
					config.QueueColumnBgColor:    bgColor,
				}),
				"setQueueHighlight(): SetCols() failed")
		}
	}
}

// updateAll updates all window's widgets and lists
func (w *MainWindow) updateAll() {
	// Update global actions
	connected, connecting := w.connector.ConnectStatus()
	w.aMPDDisconnect.SetEnabled(connected || connecting)
	w.aMPDInfo.SetEnabled(connected)
	w.aMPDOutputs.SetEnabled(connected)

	// Update other widgets
	w.updateQueue()
	w.updateLibraryPath()
	w.updateLibrary()
	w.updateLibraryActions()
	w.updateOptions()
	w.updatePlayer()
	w.updateVolume()
}

// updateLibrary updates the current library list contents
func (w *MainWindow) updateLibrary() {
	// Clear the library list
	util.ClearChildren(w.LibraryListBox.Container)

	var (
		elements []LibraryPathElement
		err      error
		pattern  string
	)
	maxResultRows := -1
	lastElement := w.libPath.Last()

	// If search mode activated
	if w.LibrarySearchToolButton.GetActive() {
		pattern = util.EntryText(&w.LibrarySearchEntry.Entry, "")
	}

	// Search mode: fetch selected attribute
	if pattern != "" {
		attrName := "any"
		if attr, ok := config.MpdTrackAttributes[util.AtoiDef(w.LibrarySearchAttrComboBox.GetActiveID(), -1)]; ok {
			attrName = attr.AttrName
		}

		// Run search
		var attrs []mpd.Attrs
		w.connector.IfConnected(func(client *mpd.Client) {
			attrs, err = client.Search(fmt.Sprintf("(%s contains \"%s\")", attrName, pattern))
		})
		if errCheck(err, "updateLibrary(): Search() failed") {
			return
		}
		maxResultRows = config.GetConfig().MaxSearchResults

		// Convert the list into elements
		elements = AttrsToElements(attrs, "")

	} else if lastElement == nil {
		// Root
		elements = []LibraryPathElement{
			NewFilesystemLibElement(),
			NewGenresLibElement(),
			NewArtistsLibElement(),
			NewAlbumsLibElement(),
			NewPlaylistsLibElement(),
		}

	} else if uh, ok := lastElement.(URIHolder); ok {
		// URI-enabled element: load list of directories/files at the current path
		var attrs []mpd.Attrs
		w.connector.IfConnected(func(client *mpd.Client) {
			attrs, err = client.ListInfo(uh.URI())
		})
		if errCheck(err, "updateLibrary(): ListInfo() failed") {
			return
		}

		// Convert the list into elements
		elements = AttrsToElements(attrs, uh.URI()+"/")

	} else if browseBy, ok := lastElement.(AttributeHolderParent); ok {
		// Attribute-enabled path: determine the attribute we're browsing by
		args := append(
			// First element is the attribute we're browsing by
			[]string{config.MpdTrackAttributes[browseBy.ChildAttributeID()].AttrName},
			// Then the filter arguments follow
			w.libPath.AsFilter()...)

		// Load the list of tags
		var list []string
		w.connector.IfConnected(func(client *mpd.Client) {
			list, err = client.List(args...)
		})
		if errCheck(err, "updateLibrary(): List() failed") {
			return
		}

		// Convert the string list into a list of elements
		elements = make([]LibraryPathElement, 0, len(list))
		for _, s := range list {
			if c := browseBy.NewChild(s); c != nil {
				elements = append(elements, c)
			}
		}

	} else if pl, ok := lastElement.(*PlaylistsLibElement); ok {
		// Playlists list element: load list of playlists
		for _, name := range w.connector.GetPlaylists() {
			elements = append(elements, pl.NewChild(name))
		}

	} else {
		log.Errorf("Unknown library path kind (last element is %T)", lastElement)
		return
	}

	// If no search mode and not root, insert a "level up" element
	if pattern == "" && lastElement != nil {
		elements = append([]LibraryPathElement{NewLevelUpLibElement()}, elements...)
	}

	// Repopulate the library list
	var rowToSelect *gtk.ListBoxRow
	countItems, limited := 0, false
	for _, element := range elements {
		element := element // Make an in-loop copy for closures
		label := element.Label()
		markup := false

		// Add replace/append buttons if needed
		var buttons []gtk.IWidget
		if element.IsPlayable() {
			buttons = []gtk.IWidget{
				util.NewButton("", glib.Local("Append to the queue"), "", "ymuse-add-symbolic", func() { w.queueLibraryElement(tbFalse, element) }),
				util.NewButton("", glib.Local("Replace the queue"), "", "ymuse-replace-queue-symbolic", func() { w.queueLibraryElement(tbTrue, element) }),
			}
		} else {
			// Make non-playable (root) elements bold
			label = "<b>" + label + "</b>"
			markup = true
		}

		// Add a new list box row
		row, hbx, err := util.NewListBoxRow(w.LibraryListBox, markup, label, MarshalLibPathElement(element), element.Icon(), buttons...)
		if errCheck(err, "NewListBoxRow() failed") {
			return
		}

		// If no specific row to select, pick the first one. Otherwise check for a matching marshalled form
		if rowToSelect == nil && (w.libPathElementToSelect == "" || w.libPathElementToSelect == element.Marshal()) {
			rowToSelect = row
		}

		// Add a label with details [track length], if any
		if dh, ok := element.(DetailsHolder); ok {
			if details := dh.Details(); details != "" {
				lbl, err := gtk.LabelNew(details)
				// Just ignore the error and proceed
				if !errCheck(err, "LabelNew() failed") {
					hbx.PackEnd(lbl, false, false, 0)
				}
			}
		}
		countItems++

		if maxResultRows >= 0 && countItems >= maxResultRows {
			limited = true
			break
		}
	}

	// Show all rows
	w.LibraryListBox.ShowAll()

	// Select the required row and scroll to it (later)
	w.LibraryListBox.SelectRow(rowToSelect)
	glib.IdleAdd(func() { util.ListBoxScrollToSelected(w.LibraryListBox) })
	w.libPathElementToSelect = ""

	// Compose info
	info := ""
	if countItems == 0 {
		info = glib.Local("No items")
	} else {
		// Compose info
		info += fmt.Sprintf(glib.Local("%d items"), countItems)

		// Add note about limited set, if applicable
		if limited {
			info += " " + fmt.Sprintf(glib.Local("(limited selection of %d items)"), len(elements))
		}
	}

	if _, ok := w.connector.Status()["updating_db"]; ok {
		info += " — " + glib.Local("updating database…")
	}

	// Update info
	w.LibraryInfoLabel.SetText(info)
}

// updateLibraryActions updates the widgets for library list
func (w *MainWindow) updateLibraryActions() {
	element := w.getSelectedLibraryElement()
	connected, _ := w.connector.ConnectStatus()
	selected := element != nil
	_, playlist := element.(PlaylistHolder)
	_, filesystem := element.(URIHolder)
	editable := playlist && connected && selected
	updatable := connected && selected && filesystem
	playable := connected && selected && element.IsPlayable()
	// Actions
	w.aLibraryUpdate.SetEnabled(connected)
	w.aLibraryUpdateAll.SetEnabled(connected)
	w.aLibraryUpdateSel.SetEnabled(updatable)
	w.aLibraryRescanAll.SetEnabled(connected)
	w.aLibraryRescanSel.SetEnabled(updatable)
	w.aLibraryRename.SetEnabled(editable)
	w.aLibraryDelete.SetEnabled(editable)
	w.aLibraryAddToPlaylist.SetEnabled(playable)
	// Menu items
	w.LibraryAppendMenuItem.SetSensitive(playable)
	w.LibraryReplaceMenuItem.SetSensitive(playable)
	w.LibraryRenameMenuItem.SetSensitive(editable)
	w.LibraryDeleteMenuItem.SetSensitive(editable)
	w.LibraryUpdateSelMenuItem.SetSensitive(updatable)
	w.LibraryAddToPlaylistMenuItem.SetSensitive(playable)
}

// updateLibraryPath updates the current library path selector
func (w *MainWindow) updateLibraryPath() {
	// Remove all buttons from the box
	util.ClearChildren(w.LibraryPathBox.Container)

	// Create a button for "root"
	util.NewBoxToggleButton(
		w.LibraryPathBox,
		"",
		"",
		"ymuse-home-symbolic",
		w.libPath.IsRoot(),
		func() { w.libPath.SetLength(0) })

	// Create buttons for path elements
	for i, element := range w.libPath.Elements() {
		// Create a button. The last button must be depressed
		i := i // Make an in-loop copy of i
		util.NewBoxToggleButton(
			w.LibraryPathBox,
			element.Label(),
			"",
			element.Icon(),
			element == w.libPath.Last(),
			func() {
				// Save the first path element from the chopped-off tail for subsequent selection
				if e := w.libPath.ElementAt(i + 1); e != nil {
					w.libPathElementToSelect = e.Marshal()
				}

				// Move to the selected level
				w.libPath.SetLength(i + 1)
			})
	}

	// Show all buttons
	w.LibraryPathBox.ShowAll()
}

// updateOptions updates player options widgets
func (w *MainWindow) updateOptions() {
	w.optionsUpdating = true
	status := w.connector.Status()
	w.RandomButton.SetActive(status["random"] == "1")
	w.RepeatButton.SetActive(status["repeat"] == "1")
	w.ConsumeButton.SetActive(status["consume"] == "1")
	w.optionsUpdating = false
}

// updatePlayer updates player control widgets
func (w *MainWindow) updatePlayer() {
	connected, connecting := w.connector.ConnectStatus()
	status := w.connector.Status()
	var statusHTML string
	var err error
	curURI := ""

	switch {
	// Still connecting
	case connecting:
		statusHTML = fmt.Sprintf("<i>%s</i>", html.EscapeString(glib.Local("Connecting to MPD…")))

	// Already connected
	case connected:
		// Fetch the current track
		var curSong mpd.Attrs
		w.connector.IfConnected(func(client *mpd.Client) {
			curSong, err = client.CurrentSong()
			errCheck(err, "CurrentSong() failed")
		})

		if err == nil {
			// Enrich the current track with the status info
			curSong["Bitrate"] = status["bitrate"]
			curSong["Format"] = status["audio"]

			// Dump the current track for debug purposes
			log.Debugf("Current track: %#v", curSong)

			// Apply track title template
			var buffer bytes.Buffer
			if err := w.playerTitleTemplate.Execute(&buffer, curSong); err != nil {
				statusHTML = html.EscapeString(fmt.Sprintf("%s: %v", glib.Local("Template error"), err))
			} else {
				statusHTML = buffer.String()
			}

			// Get the current URI
			curURI = curSong["file"]
		}

		// Update play/pause button's appearance
		switch status["state"] {
		case "play":
			w.PlayPauseButton.SetIconName("ymuse-pause-symbolic")
		default:
			w.PlayPauseButton.SetIconName("ymuse-play-symbolic")
		}

	// Not connected
	default:
		statusHTML = fmt.Sprintf("<i>%s</i>", html.EscapeString(glib.Local("Not connected to MPD")))
	}

	// If there's an error
	if errMsg, ok := status["error"]; ok {
		statusHTML += fmt.Sprintf(" — <span foreground=\"red\">%s</span>", html.EscapeString(errMsg))
	}

	// Update the album art
	w.updatePlayerAlbumArt(curURI)

	// Update status text
	w.StatusLabel.SetMarkup(statusHTML)

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

// updatePlayerAlbumArt updates player's album art image appearance and visibility
func (w *MainWindow) updatePlayerAlbumArt(uri string) {
	// Check if the album art is to be shown
	show := false
	if uri != "" {
		isStream := util.IsStreamURI(uri)
		cfg := config.GetConfig()
		size := cfg.PlayerAlbumArtSize
		if (isStream && cfg.PlayerAlbumArtStreams || !isStream && cfg.PlayerAlbumArtTracks) && size > 0 {
			// Avoid updating album art if there's no change in the URI or size
			if curPx := w.AlbumArtworkImage.GetPixbuf(); curPx != nil && curPx.GetWidth() == size && w.playerCurrentAlbumArtUri == uri {
				show = true
			} else {
				// Try to fetch the album art
				var albumArt []byte
				log.Debugf("Fetching album art for %s", uri)
				w.connector.IfConnected(func(client *mpd.Client) {
					var err error
					if albumArt, err = client.AlbumArt(uri); err != nil {
						log.Debugf("Failed to obtain album art: %v", err)
						albumArt = nil
					}
				})

				// If succeeded
				if len(albumArt) > 0 {
					log.Debugf("Fetched album art: %d bytes", len(albumArt))
					// Make a pixbuf from the data bytes
					if px, err := gdk.PixbufNewFromBytesOnly(albumArt); !errCheck(err, "PixbufNewFromBytesOnly() failed") {
						// Downscale the image if needed
						if px, err = px.ScaleSimple(size, size, gdk.INTERP_BILINEAR); !errCheck(err, "ScaleSimple() failed") {
							w.AlbumArtworkImage.SetFromPixbuf(px)
							show = true
							// Save the last used URI
							w.playerCurrentAlbumArtUri = uri
						}
					}
				}
			}
		}
	}

	// Show or hide the album art
	if !show {
		w.AlbumArtworkImage.Clear()
		w.playerCurrentAlbumArtUri = ""
	}
	w.AlbumArtworkImage.SetVisible(show)

	// If the image isn't visible, center-justify the title. Otherwise use left justification
	justification := gtk.JUSTIFY_CENTER
	if show {
		justification = gtk.JUSTIFY_LEFT
	}
	w.StatusLabel.SetJustify(justification)
}

// updatePlayerSeekBar updates the seek bar position and status
func (w *MainWindow) updatePlayerSeekBar() {
	seekPos := ""
	var trackLen, trackPos float64

	// If the user is dragging the slider manually
	if w.playPosUpdating {
		trackLen, trackPos = w.PlayPositionAdjustment.GetUpper(), w.PlayPositionAdjustment.GetValue()

	} else {
		// The update comes from MPD: adjust the seek bar position if there's a connection
		trackStart := -1.0
		trackLen, trackPos = -1.0, -1.0
		if connected, _ := w.connector.ConnectStatus(); connected {
			// Fetch current player position and track length
			status := w.connector.Status()
			trackLen = util.ParseFloatDef(status["duration"], -1)
			trackPos = util.ParseFloatDef(status["elapsed"], -1)
		}

		// If not seekable, remove the slider
		if trackPos >= 0 && trackLen >= trackPos {
			trackStart = 0
		}
		w.PlayPositionScale.SetSensitive(trackStart == 0)

		// Enable the seek bar based on status and position it
		w.PlayPositionAdjustment.SetLower(trackStart)
		w.PlayPositionAdjustment.SetUpper(trackLen)
		w.PlayPositionAdjustment.SetValue(trackPos)
	}

	// Update position text
	if trackPos >= 0 {
		seekPos = fmt.Sprintf("<big>%s</big>", util.FormatSeconds(trackPos))
		if trackLen >= trackPos {
			seekPos += fmt.Sprintf(" / " + util.FormatSeconds(trackLen))
		}
	}
	w.PositionLabel.SetMarkup(seekPos)
}

// updateQueue updates the current play queue contents
func (w *MainWindow) updateQueue() {
	// Lock tree updates
	w.QueueTreeView.FreezeChildNotify()
	defer w.QueueTreeView.ThawChildNotify()

	// Detach the tree view from the list model to speed up processing
	w.QueueTreeView.SetModel(nil)

	// Clear the queue list store
	w.QueueListStore.Clear()
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
		// Iterate attributes
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

			// Only store non-empty values
			if value != "" {
				rowData[id] = value
			}
		}

		// Check for possible fallbacks once all values are known
		for id, mpdAttr := range config.MpdTrackAttributes {
			// If no value for attribute and there are fallback attributes
			if _, ok := rowData[id]; !ok && mpdAttr.FallbackAttrIDs != nil {
				// Pick the first available value from fallback list
				for _, fbId := range mpdAttr.FallbackAttrIDs {
					if value, ok := rowData[fbId]; ok {
						rowData[id] = value
						break
					}
				}
			}
		}

		// Add the "artificial" column values
		iconName := "ymuse-audio-file"
		if uri, ok := a["file"]; ok && util.IsStreamURI(uri) {
			iconName = "ymuse-stream"
		}
		rowData[config.QueueColumnIcon] = iconName
		rowData[config.QueueColumnFontWeight] = fontWeightNormal
		rowData[config.QueueColumnBgColor] = w.colourBgNormal
		rowData[config.QueueColumnVisible] = true

		// Create arrays (indices and values)
		rowIndices, rowValues := make([]int, len(rowData)), make([]interface{}, len(rowData))
		colIdx := 0
		for key, value := range rowData {
			rowIndices[colIdx] = key
			rowValues[colIdx] = value
			colIdx++
		}

		// Add a row to the list store
		errCheck(
			w.QueueListStore.InsertWithValues(nil, -1, rowIndices, rowValues),
			"QueueListStore.SetCols() failed")

		// Accumulate counters
		totalSecs += util.ParseFloatDef(a["duration"], 0)
		w.currentQueueSize++
	}

	// Add number of tracks
	var status string
	switch w.currentQueueSize {
	case 0:
		status = glib.Local("Queue is empty")
	case 1:
		status = glib.Local("One track")
	default:
		status = fmt.Sprintf(glib.Local("%d tracks"), len(attrs))
	}

	// Add playing time, if any
	if totalSecs > 0 {
		status += ", " + fmt.Sprintf(glib.Local("playing time %s"), util.FormatSeconds(totalSecs))
	}

	// Update the queue info
	w.QueueInfoLabel.SetText(status)

	// Update queue actions
	w.updateQueueActions()

	// Restore the tree view model
	w.QueueTreeView.SetModel(w.QueueTreeModelFilter)

	// Highlight and scroll the tree to the currently played item
	w.updateQueueNowPlaying()
}

// updateQueueColumns updates the columns in the play queue tree view
func (w *MainWindow) updateQueueColumns() {
	// Remove all columns
	w.QueueTreeView.GetColumns().Foreach(func(item interface{}) {
		w.QueueTreeView.RemoveColumn(item.(*gtk.TreeViewColumn))
	})

	// Add an icon renderer
	if renderer, err := gtk.CellRendererPixbufNew(); !errCheck(err, "CellRendererPixbufNew() failed") {
		// Add an icon column
		if col, err := gtk.TreeViewColumnNewWithAttribute("", renderer, "icon-name", config.QueueColumnIcon); !errCheck(err, "TreeViewColumnNewWithAttribute() failed") {
			col.SetSizing(gtk.TREE_VIEW_COLUMN_FIXED)
			col.SetFixedWidth(-1)
			col.AddAttribute(renderer, "cell-background", config.QueueColumnBgColor)
			w.QueueTreeView.AppendColumn(col)
		}
	}

	// Add selected columns
	for index, colSpec := range config.GetConfig().QueueColumns {
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
		errCheck(renderer.SetProperty("xalign", attr.XAlign), "renderer.SetProperty(xalign) failed")

		// Add a new tree column
		col, err := gtk.TreeViewColumnNewWithAttribute(glib.Local(attr.Name), renderer, "text", colSpec.ID)
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
		col.AddAttribute(renderer, "cell-background", config.QueueColumnBgColor)

		// Bind the clicked signal
		col.Connect("clicked", func(c *gtk.TreeViewColumn) {
			w.onQueueTreeViewColClicked(c, index, &attr)
		})

		// Bind the width property change signal: update QueueColumns on each change
		col.Connect("notify::fixed-width", func(c *gtk.TreeViewColumn) {
			config.GetConfig().QueueColumns[index].Width = c.GetFixedWidth()
		})

		// Add the column to the tree view
		w.QueueTreeView.AppendColumn(col)
	}

	// Make all columns visible
	w.QueueTreeView.ShowAll()
}

// updateQueueActions updates the play queue actions
func (w *MainWindow) updateQueueActions() {
	connected, _ := w.connector.ConnectStatus()
	notEmpty := connected && w.currentQueueSize > 0
	selCount := w.getQueueSelectedCount()
	selection := notEmpty && selCount > 0
	selOne := notEmpty && selCount == 1
	// Actions
	w.aQueueNowPlaying.SetEnabled(notEmpty)
	w.aQueueClear.SetEnabled(notEmpty)
	w.aQueueSort.SetEnabled(notEmpty)
	w.aQueueSortAsc.SetEnabled(notEmpty)
	w.aQueueSortDesc.SetEnabled(notEmpty)
	w.aQueueSortShuffle.SetEnabled(notEmpty)
	w.aQueueDelete.SetEnabled(selection)
	w.aQueueSave.SetEnabled(notEmpty)
	// Menu items
	w.QueueNowPlayingMenuItem.SetSensitive(notEmpty)
	w.QueueShowAlbumInLibraryMenuItem.SetSensitive(selOne)
	w.QueueShowArtistInLibraryMenuItem.SetSensitive(selOne)
	w.QueueShowGenreInLibraryMenuItem.SetSensitive(selOne)
	w.QueueClearMenuItem.SetSensitive(notEmpty)
	w.QueueDeleteMenuItem.SetSensitive(selection)
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
		if treePath = w.QueueTreeModelFilter.ConvertChildPathToPath(treePath); treePath != nil {
			w.QueueTreeView.ScrollToCell(treePath, nil, true, 0.5, 0)
		}
	}
}

// updateStreams updates the current streams list contents
func (w *MainWindow) updateStreams() {
	// Clear the streams list
	util.ClearChildren(w.StreamsListBox.Container)

	// Make sure the streams are sorted by name
	cfg := config.GetConfig()
	sort.Slice(cfg.Streams, func(i, j int) bool {
		return strings.ToUpper(cfg.Streams[i].Name) < strings.ToUpper(cfg.Streams[j].Name)
	})

	// Repopulate the streams list
	var rowToSelect *gtk.ListBoxRow
	for _, stream := range config.GetConfig().Streams {
		stream := stream // Make an in-loop copy of the var
		row, _, err := util.NewListBoxRow(
			w.StreamsListBox,
			false,
			stream.Name,
			"",
			"ymuse-stream",
			// Add replace/append buttons
			util.NewButton("", glib.Local("Append to the queue"), "", "ymuse-add-symbolic", func() { w.queueStream(tbFalse, stream.URI) }),
			util.NewButton("", glib.Local("Replace the queue"), "", "ymuse-replace-queue-symbolic", func() { w.queueStream(tbTrue, stream.URI) }))
		if errCheck(err, "NewListBoxRow() failed") {
			return
		}

		// Select the first row in the list
		if rowToSelect == nil {
			rowToSelect = row
		}
	}

	// Show all rows
	w.StreamsListBox.ShowAll()

	// Select the required row
	w.StreamsListBox.SelectRow(rowToSelect)

	// Compose info
	var info string
	if cnt := len(config.GetConfig().Streams); cnt > 0 {
		info = fmt.Sprintf(glib.Local("%d streams"), cnt)
	} else {
		info = glib.Local("No streams")
	}

	// Update info
	w.StreamsInfoLabel.SetText(info)
}

// updateStreamsActions updates the widgets for streams list
func (w *MainWindow) updateStreamsActions() {
	connected, _ := w.connector.ConnectStatus()
	selected := w.getSelectedStreamIndex() >= 0
	// Actions
	w.aStreamAdd.SetEnabled(true) // Adding a stream is always possible
	w.aStreamEdit.SetEnabled(selected)
	w.aStreamDelete.SetEnabled(selected)
	// Menu items
	w.StreamsAppendMenuItem.SetSensitive(connected && selected)
	w.StreamsReplaceMenuItem.SetSensitive(connected && selected)
	w.StreamsEditMenuItem.SetSensitive(selected)
	w.StreamsDeleteMenuItem.SetSensitive(selected)
}

// updateStyle updates custom colours based on the current theme
func (w *MainWindow) updateStyle() {
	// Fetch window's style context
	ctx, err := w.AppWindow.GetStyleContext()
	if errCheck(err, "updateStyle(): GetStyleContext() failed") {
		return
	}

	// Determine normal background colour
	var bgNormal, bgActive string
	if rgba, ok := ctx.LookupColor("theme_base_color"); ok {
		bgNormal = rgba.String()
	} else {
		log.Warning("Unknown colour: theme_base_color")
		bgNormal = "#ffffff"
	}

	// Determine active background colour: same as selected colour, but at 30% opacity
	if rgba, ok := ctx.LookupColor("theme_selected_bg_color"); ok {
		newRGBA := rgba.Floats()
		rgba.SetColors(newRGBA[0], newRGBA[1], newRGBA[2], newRGBA[3]*0.3)
		bgActive = rgba.String()
	} else {
		log.Warning("Unknown colour: theme_selected_bg_color")
		bgActive = "#ffffe0"
	}

	// If the colours changed, we need to update the queue list store
	if w.colourBgNormal != bgNormal || w.colourBgActive != bgActive {
		w.colourBgNormal = bgNormal
		w.colourBgActive = bgActive
		w.currentQueueIndex = -1

		w.QueueListStore.ForEach(func(model *gtk.TreeModel, path *gtk.TreePath, iter *gtk.TreeIter) bool {
			// Update item's background color
			if err := w.QueueListStore.SetValue(iter, config.QueueColumnBgColor, w.colourBgNormal); errCheck(err, "updateStyle(): SetValue() failed") {
				return true
			}

			// Proceed to the next row
			return false
		})

		// Update the active row, if the app has been initialised
		if w.connector != nil {
			w.updateQueueNowPlaying()
		}
	}
}

// updateVolume synchronises the volume scale position to the current MPD volume
func (w *MainWindow) updateVolume() {
	// Update the volume button's state
	connected, _ := w.connector.ConnectStatus()
	w.VolumeButton.SetSensitive(connected)

	// The update comes from MPD: adjust the volume bar position if there's a connection
	if vol := util.AtoiDef(w.connector.Status()["volume"], -1); vol >= 0 && vol <= 100 {
		w.volumeUpdating = true
		w.VolumeAdjustment.SetValue(float64(vol))
		w.volumeUpdating = false
	}
}
