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
	"github.com/gotk3/gotk3/gtk"
	"github.com/yktoo/ymuse/internal/config"
	"github.com/yktoo/ymuse/internal/generated"
	"github.com/yktoo/ymuse/internal/util"
)

type queueCol struct {
	selected bool
	id       int
	width    int
}

// PrefsDialog represents the preferences dialog
type PrefsDialog struct {
	dialog *gtk.Dialog
	// Whether the dialog is initialised
	initialised bool
	// General page widgets
	eMpdHost           *gtk.Entry
	adjMpdPort         *gtk.Adjustment
	eMpdPassword       *gtk.Entry
	cbMpdAutoConnect   *gtk.CheckButton
	cbMpdAutoReconnect *gtk.CheckButton
	// Interface page widgets
	rbLibraryDefaultReplace   *gtk.RadioButton
	rbLibraryDefaultAppend    *gtk.RadioButton
	rbPlaylistsDefaultReplace *gtk.RadioButton
	rbPlaylistsDefaultAppend  *gtk.RadioButton
	txbPlayerTitleTemplate    *gtk.TextBuffer
	// Columns page widgets
	lbxColumns   *gtk.ListBox
	queueColumns []queueCol
	// Callbacks
	onQueueColumnsChanged        func()
	onPlayerTitleTemplateChanged func()
}

// PreferencesDialog creates, shows and disposes of a Preferences dialog instance
func PreferencesDialog(parent gtk.IWindow, onMpdReconnect, onQueueColumnsChanged, onPlayerTitleTemplateChanged func()) {
	// Load the dialog layout
	builder := NewBuilder(generated.GetPrefsGlade())

	// Create the dialog and map the widgets
	d := &PrefsDialog{
		dialog: builder.getDialog("prefsDialog"),
		// General page widgets
		eMpdHost:           builder.getEntry("eMpdHost"),
		adjMpdPort:         builder.getAdjustment("adjMpdPort"),
		eMpdPassword:       builder.getEntry("eMpdPassword"),
		cbMpdAutoConnect:   builder.getCheckButton("cbMpdAutoConnect"),
		cbMpdAutoReconnect: builder.getCheckButton("cbMpdAutoReconnect"),
		// Interface page widgets
		rbLibraryDefaultReplace:   builder.getRadioButton("rbLibraryDefaultReplace"),
		rbLibraryDefaultAppend:    builder.getRadioButton("rbLibraryDefaultAppend"),
		rbPlaylistsDefaultReplace: builder.getRadioButton("rbPlaylistsDefaultReplace"),
		rbPlaylistsDefaultAppend:  builder.getRadioButton("rbPlaylistsDefaultAppend"),
		txbPlayerTitleTemplate:    builder.getTextBuffer("txbPlayerTitleTemplate"),
		// Columns page widgets
		lbxColumns: builder.getListBox("lbxColumns"),
		// Callbacks
		onQueueColumnsChanged:        onQueueColumnsChanged,
		onPlayerTitleTemplateChanged: onPlayerTitleTemplateChanged,
	}
	defer d.dialog.Destroy()

	// Set the dialog up
	d.dialog.SetTransientFor(parent)

	// Remove the 2-pixel "aura" around the notebook
	if box, err := d.dialog.GetContentArea(); err == nil {
		box.SetBorderWidth(0)
	}

	// Map the handlers to callback functions
	builder.ConnectSignals(map[string]interface{}{
		"on_prefsDialog_map":           d.onMap,
		"on_setting_change":            d.onSettingChange,
		"on_MpdReconnect":              onMpdReconnect,
		"on_btnColumnMoveUp_clicked":   d.onColumnMoveUp,
		"on_btnColumnMoveDown_clicked": d.onColumnMoveDown,
	})

	// Run the dialog
	d.dialog.Run()
}

func (d *PrefsDialog) onMap() {
	log.Debug("PrefsDialog.onMap()")

	// Initialise widgets
	cfg := config.GetConfig()
	// General page
	d.eMpdHost.SetText(cfg.MpdHost)
	d.adjMpdPort.SetValue(float64(cfg.MpdPort))
	d.eMpdPassword.SetText(cfg.MpdPassword)
	d.cbMpdAutoConnect.SetActive(cfg.MpdAutoConnect)
	d.cbMpdAutoReconnect.SetActive(cfg.MpdAutoReconnect)
	// Interface page
	d.rbLibraryDefaultReplace.SetActive(cfg.TrackDefaultReplace)
	d.rbLibraryDefaultAppend.SetActive(!cfg.TrackDefaultReplace)
	d.rbPlaylistsDefaultReplace.SetActive(cfg.PlaylistDefaultReplace)
	d.rbPlaylistsDefaultAppend.SetActive(!cfg.PlaylistDefaultReplace)
	d.txbPlayerTitleTemplate.SetText(cfg.PlayerTitleTemplate)
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
	d.lbxColumns.Add(row)

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
	lbl, err := gtk.LabelNew(config.MpdTrackAttributes[attrID].LongName)
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
		d.lbxColumns.SelectRow(row)

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
	row := d.lbxColumns.GetSelectedRow()
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
	d.lbxColumns.Remove(row)
	d.lbxColumns.Insert(row, index)

	// Re-select the row. NB: need to deselect all first, otherwise it wouldn't get selected
	d.lbxColumns.SelectRow(nil)
	d.lbxColumns.SelectRow(d.lbxColumns.GetRowAtIndex(index))

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
	config.GetConfig().QueueColumns = &colSpecs

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
	if s, err := d.eMpdHost.GetText(); !errCheck(err, "eMpdHost.GetText() failed") {
		cfg.MpdHost = s
	}
	cfg.MpdPort = int(d.adjMpdPort.GetValue())
	if s, err := d.eMpdPassword.GetText(); !errCheck(err, "eMpdPassword.GetText() failed") {
		cfg.MpdPassword = s
	}
	cfg.MpdAutoConnect = d.cbMpdAutoConnect.GetActive()
	cfg.MpdAutoReconnect = d.cbMpdAutoReconnect.GetActive()
	// Interface page
	cfg.TrackDefaultReplace = d.rbLibraryDefaultReplace.GetActive()
	cfg.PlaylistDefaultReplace = d.rbPlaylistsDefaultReplace.GetActive()
	if s, err := util.GetTextBufferText(d.txbPlayerTitleTemplate); !errCheck(err, "util.GetTextBufferText() failed") {
		if s != cfg.PlayerTitleTemplate {
			cfg.PlayerTitleTemplate = s
			d.onPlayerTitleTemplateChanged()
		}
	}
}

// populateColumns fills in the Columns list box
func (d *PrefsDialog) populateColumns() {
	// First add selected columns
	selColSpecs := config.GetConfig().QueueColumns
	for _, colSpec := range *selColSpecs {
		d.addQueueColumn(colSpec.ID, colSpec.Width, true)
	}

	// Add all unselected columns
	for _, id := range config.MpdTrackAttributeIds {
		// Check if the ID is already in the list of selected IDs
		isSelected := false
		for _, selSpec := range *selColSpecs {
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
	d.lbxColumns.ShowAll()
}
