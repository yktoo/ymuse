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
	"strconv"
)

type PrefsDialog struct {
	dialog *gtk.Dialog
	// General page widgets
	eMpdHost     *gtk.Entry
	adjMpdPort   *gtk.Adjustment
	eMpdPassword *gtk.Entry
	// Interface page widgets
	rbLibraryDefaultReplace   *gtk.RadioButton
	rbLibraryDefaultAppend    *gtk.RadioButton
	rbPlaylistsDefaultReplace *gtk.RadioButton
	rbPlaylistsDefaultAppend  *gtk.RadioButton
	txbPlayerTitleTemplate    *gtk.TextBuffer
	// Columns page widgets
	lbxColumns *gtk.ListBox
	// Queue column checkboxes
	queueColumnCheckboxes []*gtk.CheckButton
	// Callbacks
	onQueueColumnsChanged func()
}

func PreferencesDialog(parent gtk.IWindow, onQueueColumnsChanged func()) {
	// Load the dialog layout
	builder := NewBuilder("internal/player/prefs.glade")

	// Create the dialog and map the widgets
	d := &PrefsDialog{
		dialog: builder.getDialog("prefsDialog"),
		// General page widgets
		eMpdHost:     builder.getEntry("eMpdHost"),
		adjMpdPort:   builder.getAdjustment("adjMpdPort"),
		eMpdPassword: builder.getEntry("eMpdPassword"),
		// Interface page widgets
		rbLibraryDefaultReplace:   builder.getRadioButton("rbLibraryDefaultReplace"),
		rbLibraryDefaultAppend:    builder.getRadioButton("rbLibraryDefaultAppend"),
		rbPlaylistsDefaultReplace: builder.getRadioButton("rbPlaylistsDefaultReplace"),
		rbPlaylistsDefaultAppend:  builder.getRadioButton("rbPlaylistsDefaultAppend"),
		txbPlayerTitleTemplate:    builder.getTextBuffer("txbPlayerTitleTemplate"),
		// Columns page widgets
		lbxColumns: builder.getListBox("lbxColumns"),
		// Callbacks
		onQueueColumnsChanged: onQueueColumnsChanged,
	}
	defer d.dialog.Destroy()

	// Set the dialog up
	d.dialog.SetTransientFor(parent)

	// Map the handlers to callback functions
	builder.ConnectSignals(map[string]interface{}{
		"on_prefsDialog_map": d.onMap,
	})

	// Run the dialog
	d.dialog.Run()
}

func (d *PrefsDialog) onMap() {
	log.Debug("PrefsDialog.onMap()")

	// Initialise widgets
	cfg := GetConfig()
	// General page
	d.eMpdHost.SetText(cfg.MpdHost)
	d.adjMpdPort.SetValue(float64(cfg.MpdPort))
	d.eMpdPassword.SetText(cfg.MpdPassword)
	// Interface page
	d.rbLibraryDefaultReplace.SetActive(cfg.TrackDefaultReplace)
	d.rbLibraryDefaultAppend.SetActive(!cfg.TrackDefaultReplace)
	d.rbPlaylistsDefaultReplace.SetActive(cfg.PlaylistDefaultReplace)
	d.rbPlaylistsDefaultAppend.SetActive(!cfg.PlaylistDefaultReplace)
	d.txbPlayerTitleTemplate.SetText(cfg.PlayerTitleTemplate)
	// Columns page
	d.populateColumns()
}

// addColumn() adds a row with a check box to the Columns list box
func (d *PrefsDialog) addColumn(attrId int, checked bool) {
	attr := MpdTrackAttributes[attrId]

	// Add a new list box row
	row, err := gtk.ListBoxRowNew()
	if errCheck(err, "ListBoxRowNew() failed") {
		return
	}
	d.lbxColumns.Add(row)

	// Add a checkbox
	cb, err := gtk.CheckButtonNewWithLabel(attr.longName)
	if errCheck(err, "CheckButtonNewWithLabel() failed") {
		return
	}
	cb.SetActive(checked)
	_, err = cb.Connect("toggled", d.updateColumnsFromListBox)
	if errCheck(err, "cb.Connect(toggled) failed") {
		return
	}
	row.Add(cb)

	// Save the ID into the checkbox's name
	cb.SetName(strconv.Itoa(attrId))

	// Save the checkbox in the dialog for future column updates
	d.queueColumnCheckboxes = append(d.queueColumnCheckboxes, cb)
}

// populateColumns() fills in the Columns list box
func (d *PrefsDialog) populateColumns() {
	// First add selected columns
	selIds := GetConfig().QueueColumnIds
	for _, id := range selIds {
		d.addColumn(id, true)
	}

	// Add all unselected columns
	for _, id := range MpdTrackAttributeIds {
		// Check if the ID is already in the list of selected IDs
		isSelected := false
		for _, selId := range selIds {
			if id == selId {
				isSelected = true
				break
			}
		}

		// If not, add it
		if !isSelected {
			d.addColumn(id, false)
		}
	}
	d.lbxColumns.ShowAll()
}

// updateColumnsFromListBox() updates queue tree view columns from the currently selected ones in the Columns list box
func (d *PrefsDialog) updateColumnsFromListBox() {
	// Collect IDs of checked attributes
	var ids []int
	for _, cb := range d.queueColumnCheckboxes {
		if cb.GetActive() {
			// Extract check box name
			name, err := cb.GetName()
			if errCheck(err, "cb.GetName() failed") {
				return
			}

			// Extract attribute ID from the checkbox's name
			if id, err := strconv.Atoi(name); err == nil {
				ids = append(ids, id)
			}
		}
	}

	// Save the IDs in the config
	GetConfig().QueueColumnIds = ids

	// Notify the callback
	d.onQueueColumnsChanged()
}
