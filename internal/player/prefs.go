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

import (
	"fmt"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/yktoo/ymuse/internal/config"
	"github.com/yktoo/ymuse/internal/generated"
	"github.com/yktoo/ymuse/internal/util"
	"sync"
	"time"
)

type queueCol struct {
	selected bool
	id       int
	width    int
}

// PrefsDialog represents the preferences dialog
type PrefsDialog struct {
	PreferencesDialog *gtk.Dialog
	// General page widgets
	MpdNetworkComboBox          *gtk.ComboBoxText
	MpdPathEntry                *gtk.Entry
	MpdPathLabel                *gtk.Label
	MpdHostEntry                *gtk.Entry
	MpdHostLabel                *gtk.Label
	MpdHostLabelRemark          *gtk.Label
	MpdPortSpinButton           *gtk.SpinButton
	MpdPortLabel                *gtk.Label
	MpdPortAdjustment           *gtk.Adjustment
	MpdPasswordEntry            *gtk.Entry
	MpdAutoConnectCheckButton   *gtk.CheckButton
	MpdAutoReconnectCheckButton *gtk.CheckButton
	// Interface page widgets
	LibraryDefaultReplaceRadioButton   *gtk.RadioButton
	LibraryDefaultAppendRadioButton    *gtk.RadioButton
	PlaylistsDefaultReplaceRadioButton *gtk.RadioButton
	PlaylistsDefaultAppendRadioButton  *gtk.RadioButton
	StreamsDefaultReplaceRadioButton   *gtk.RadioButton
	StreamsDefaultAppendRadioButton    *gtk.RadioButton
	PlayerShowAlbumArtCheckButton      *gtk.CheckButton
	PlayerTitleTemplateTextBuffer      *gtk.TextBuffer
	// Columns page widgets
	ColumnsListBox *gtk.ListBox

	// Whether the dialog is initialised
	initialised bool
	// Columns, in the same order as in the ColumnsListBox
	queueColumns []queueCol
	// Timer for delayed player setting change callback invocation
	playerSettingChangeTimer *time.Timer
	playerSettingChangeMutex sync.Mutex
	// Callbacks
	onQueueColumnsChanged  func()
	onPlayerSettingChanged func()
}

// PreferencesDialog creates, shows and disposes of a Preferences dialog instance
func PreferencesDialog(parent gtk.IWindow, onMpdReconnect, onQueueColumnsChanged, onPlayerSettingChanged func()) {
	// Create the dialog
	d := &PrefsDialog{
		onQueueColumnsChanged:  onQueueColumnsChanged,
		onPlayerSettingChanged: onPlayerSettingChanged,
	}

	// Load the dialog layout and map the widgets
	builder, err := NewBuilder(generated.GetPrefsGlade())
	if err == nil {
		err = builder.BindWidgets(d)
	}

	// Check for errors
	if errCheck(err, "PreferencesDialog(): failed to initialise dialog") {
		util.ErrorDialog(parent, fmt.Sprint(glib.Local("Failed to load UI widgets"), err))
		return
	}
	defer d.PreferencesDialog.Destroy()

	// Set the dialog up
	d.PreferencesDialog.SetTransientFor(parent)

	// Remove the 2-pixel "aura" around the notebook
	if box, err := d.PreferencesDialog.GetContentArea(); err == nil {
		box.SetBorderWidth(0)
	}

	// Map the handlers to callback functions
	builder.ConnectSignals(map[string]interface{}{
		"on_PreferencesDialog_map":            d.onMap,
		"on_Setting_change":                   d.onSettingChange,
		"on_MpdReconnect":                     onMpdReconnect,
		"on_ColumnMoveUpToolButton_clicked":   d.onColumnMoveUp,
		"on_ColumnMoveDownToolButton_clicked": d.onColumnMoveDown,
	})

	// Run the dialog
	d.PreferencesDialog.Run()
}

func (d *PrefsDialog) onMap() {
	log.Debug("PrefsDialog.onMap()")

	// Initialise widgets
	cfg := config.GetConfig()
	// General page
	d.MpdNetworkComboBox.SetActiveID(cfg.MpdNetwork)
	d.MpdPathEntry.SetText(cfg.MpdSocketPath)
	d.MpdHostEntry.SetText(cfg.MpdHost)
	d.MpdPortAdjustment.SetValue(float64(cfg.MpdPort))
	d.MpdPasswordEntry.SetText(cfg.MpdPassword)
	d.MpdAutoConnectCheckButton.SetActive(cfg.MpdAutoConnect)
	d.MpdAutoReconnectCheckButton.SetActive(cfg.MpdAutoReconnect)
	d.updateGeneralWidgets()
	// Interface page
	d.LibraryDefaultReplaceRadioButton.SetActive(cfg.TrackDefaultReplace)
	d.LibraryDefaultAppendRadioButton.SetActive(!cfg.TrackDefaultReplace)
	d.PlaylistsDefaultReplaceRadioButton.SetActive(cfg.PlaylistDefaultReplace)
	d.PlaylistsDefaultAppendRadioButton.SetActive(!cfg.PlaylistDefaultReplace)
	d.StreamsDefaultReplaceRadioButton.SetActive(cfg.StreamDefaultReplace)
	d.StreamsDefaultAppendRadioButton.SetActive(!cfg.StreamDefaultReplace)
	d.PlayerShowAlbumArtCheckButton.SetActive(cfg.PlayerAlbumArt)
	d.PlayerTitleTemplateTextBuffer.SetText(cfg.PlayerTitleTemplate)
	// Columns page
	d.populateColumns()
	d.initialised = true
}

// addQueueColumn adds a row with a check box to the Columns list box, and also registers a new item in d.queueColumns
func (d *PrefsDialog) addQueueColumn(attrID, width int, selected bool) {
	// Add an entry to queue columns slice
	d.queueColumns = append(d.queueColumns, queueCol{selected: selected, id: attrID, width: width})

	// Add a new list box row
	row, err := gtk.ListBoxRowNew()
	if errCheck(err, "ListBoxRowNew() failed") {
		return
	}
	d.ColumnsListBox.Add(row)

	// Add a container box
	hbx, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 6)
	if errCheck(err, "BoxNew() failed") {
		return
	}
	row.Add(hbx)

	// Add a checkbox
	cb, err := gtk.CheckButtonNew()
	if errCheck(err, "CheckButtonNew() failed") {
		return
	}
	cb.SetActive(selected)
	_, err = cb.Connect("toggled", func() { d.columnCheckboxToggled(attrID, cb.GetActive(), row) })
	if errCheck(err, "cb.Connect(toggled) failed") {
		return
	}
	hbx.PackStart(cb, false, false, 0)

	// Add a label
	lbl, err := gtk.LabelNew(glib.Local(config.MpdTrackAttributes[attrID].LongName))
	if errCheck(err, "LabelNew() failed") {
		return
	}
	lbl.SetXAlign(0)
	hbx.PackStart(lbl, true, true, 0)
}

// columnCheckboxToggled is a handler of the toggled signal for queue column checkboxes
func (d *PrefsDialog) columnCheckboxToggled(id int, selected bool, row *gtk.ListBoxRow) {
	// Find and toggle the column for the attribute
	if i := d.indexOfColumnWithAttrID(id); i >= 0 {
		d.queueColumns[i].selected = selected

		// Select the row
		d.ColumnsListBox.SelectRow(row)

		// Update the queue columns
		d.notifyColumnsChanged()
	}
}

// indexOfColumnWithAttrID returns the index of the queue column with given attribute ID, or -1 if not found
func (d *PrefsDialog) indexOfColumnWithAttrID(id int) int {
	for i := range d.queueColumns {
		if id == d.queueColumns[i].id {
			return i
		}
	}
	return -1
}

// moveSelectedColumnRow moves the row selected in the Columns listbox up or down
func (d *PrefsDialog) moveSelectedColumnRow(up bool) {
	// Get and check the selection
	row := d.ColumnsListBox.GetSelectedRow()
	if row == nil {
		return
	}

	// Get the row's index in the list
	index := row.GetIndex()
	if index < 0 || (up && index == 0) || (!up && index >= len(d.queueColumns)-1) {
		return
	}

	// Reorder the elements in the queue columns slice
	prevIndex := index
	if up {
		index--
	} else {
		index++
	}
	d.queueColumns[index], d.queueColumns[prevIndex] = d.queueColumns[prevIndex], d.queueColumns[index]

	// Remove and re-insert the row
	d.ColumnsListBox.Remove(row)
	d.ColumnsListBox.Insert(row, index)

	// Re-select the row. NB: need to deselect all first, otherwise it wouldn't get selected
	d.ColumnsListBox.SelectRow(nil)
	d.ColumnsListBox.SelectRow(d.ColumnsListBox.GetRowAtIndex(index))

	// Scroll the listbox to center the row
	util.WhenIdle("ListBoxScrollToSelected()", util.ListBoxScrollToSelected, d.ColumnsListBox)

	// Update the queue's columns
	d.notifyColumnsChanged()
}

// notifyColumnsChanged updates queue tree view columns from the currently selected ones in the Columns list box
func (d *PrefsDialog) notifyColumnsChanged() {
	// Collect IDs of selected attributes
	var colSpecs []config.ColumnSpec
	for _, col := range d.queueColumns {
		if col.selected {
			colSpecs = append(colSpecs, config.ColumnSpec{ID: col.id, Width: col.width})
		}
	}

	// Save the IDs in the config
	config.GetConfig().QueueColumns = colSpecs

	// Notify the callback
	d.onQueueColumnsChanged()
}

// onColumnMoveUp is a signal handler for the Move up button click
func (d *PrefsDialog) onColumnMoveUp() {
	d.moveSelectedColumnRow(true)
}

// onColumnMoveDown is a signal handler for the Move down button click
func (d *PrefsDialog) onColumnMoveDown() {
	d.moveSelectedColumnRow(false)
}

// onSettingChange is a signal handler for a change of a simple setting widget
func (d *PrefsDialog) onSettingChange() {
	// Ignore if the dialog is not initialised yet
	if !d.initialised {
		return
	}
	log.Debug("onSettingChange()")

	// Collect settings
	cfg := config.GetConfig()
	// General page
	cfg.MpdNetwork = d.MpdNetworkComboBox.GetActiveID()
	cfg.MpdSocketPath = util.EntryText(d.MpdPathEntry, "")
	cfg.MpdHost = util.EntryText(d.MpdHostEntry, "")
	cfg.MpdPort = int(d.MpdPortAdjustment.GetValue())
	if s, err := d.MpdPasswordEntry.GetText(); !errCheck(err, "MpdPasswordEntry.GetText() failed") {
		cfg.MpdPassword = s
	}
	cfg.MpdAutoConnect = d.MpdAutoConnectCheckButton.GetActive()
	cfg.MpdAutoReconnect = d.MpdAutoReconnectCheckButton.GetActive()
	d.updateGeneralWidgets()
	// Interface page
	cfg.TrackDefaultReplace = d.LibraryDefaultReplaceRadioButton.GetActive()
	cfg.PlaylistDefaultReplace = d.PlaylistsDefaultReplaceRadioButton.GetActive()
	cfg.StreamDefaultReplace = d.StreamsDefaultReplaceRadioButton.GetActive()

	b := d.PlayerShowAlbumArtCheckButton.GetActive()
	if b != cfg.PlayerAlbumArt {
		cfg.PlayerAlbumArt = b
		d.schedulePlayerSettingChange()
	}
	if s, err := util.GetTextBufferText(d.PlayerTitleTemplateTextBuffer); !errCheck(err, "util.GetTextBufferText() failed") {
		if s != cfg.PlayerTitleTemplate {
			cfg.PlayerTitleTemplate = s
			d.schedulePlayerSettingChange()
		}
	}
}

// populateColumns fills in the Columns list box
func (d *PrefsDialog) populateColumns() {
	// First add selected columns
	selColSpecs := config.GetConfig().QueueColumns
	for _, colSpec := range selColSpecs {
		d.addQueueColumn(colSpec.ID, colSpec.Width, true)
	}

	// Add all unselected columns
	for _, id := range config.MpdTrackAttributeIds {
		// Check if the ID is already in the list of selected IDs
		isSelected := false
		for _, selSpec := range selColSpecs {
			if id == selSpec.ID {
				isSelected = true
				break
			}
		}

		// If not, add it
		if !isSelected {
			d.addQueueColumn(id, 0, false)
		}
	}
	d.ColumnsListBox.ShowAll()
}

func (d *PrefsDialog) schedulePlayerSettingChange() {
	// Cancel the currently scheduled callback, if any
	d.playerSettingChangeMutex.Lock()
	defer d.playerSettingChangeMutex.Unlock()
	if d.playerSettingChangeTimer != nil {
		d.playerSettingChangeTimer.Stop()
	}

	// Schedule a new callback
	d.playerSettingChangeTimer = time.AfterFunc(time.Second, func() {
		d.playerSettingChangeMutex.Lock()
		d.playerSettingChangeTimer = nil
		d.playerSettingChangeMutex.Unlock()
		util.WhenIdle("onPlayerSettingChanged()", d.onPlayerSettingChanged)
	})
}

// updateGeneralWidgets updates widget states on the General tab
func (d *PrefsDialog) updateGeneralWidgets() {
	network := d.MpdNetworkComboBox.GetActiveID()
	unix, tcp := network == "unix", network == "tcp"
	d.MpdPathEntry.SetVisible(unix)
	d.MpdPathLabel.SetVisible(unix)
	d.MpdHostEntry.SetVisible(tcp)
	d.MpdHostLabel.SetVisible(tcp)
	d.MpdHostLabelRemark.SetVisible(tcp)
	d.MpdPortSpinButton.SetVisible(tcp)
	d.MpdPortLabel.SetVisible(tcp)
}
