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

import "github.com/gotk3/gotk3/gtk"

type SaveQueueDialog struct {
	dlg             *gtk.Dialog
	cbxPlaylist     *gtk.ComboBoxText
	lblPlaylistName *gtk.Label
	ePlaylistName   *gtk.Entry
	cbSelectedOnly  *gtk.CheckButton
}

const newPlaylistId = "\u0001new"

func RunSaveQueueDialog(parent gtk.IWindow, selection bool, connector *Connector) (ok, replace, selOnly, existing bool, playlistName string) {
	// Create a dialog
	builder := NewBuilder("internal/player/save-queue.glade")

	d := &SaveQueueDialog{
		dlg:             builder.getDialog("dlgSaveQueue"),
		cbxPlaylist:     builder.getComboBoxText("cbxPlaylist"),
		lblPlaylistName: builder.getLabel("lblPlaylistName"),
		ePlaylistName:   builder.getEntry("ePlaylistName"),
		cbSelectedOnly:  builder.getCheckButton("cbSelectedOnly"),
	}
	defer d.dlg.Destroy()

	// Connect signals
	builder.ConnectSignals(map[string]interface{}{
		"updateWidgets": d.updateWidgets,
	})

	// Tweak widgets
	d.cbSelectedOnly.SetVisible(selection)
	d.cbSelectedOnly.SetActive(selection)

	// Populate the playlists combo box
	d.cbxPlaylist.Append(newPlaylistId, "(new playlist)")
	for _, name := range connector.GetPlaylists() {
		d.cbxPlaylist.Append(name, name)
	}
	d.cbxPlaylist.SetActiveID(newPlaylistId)

	// Show the dialog
	d.dlg.SetTransientFor(parent)
	response := d.dlg.Run()

	// Calculate the results
	ok = response == gtk.RESPONSE_YES || response == gtk.RESPONSE_NO
	if ok {
		replace = response == gtk.RESPONSE_YES
		selOnly = selection && d.cbSelectedOnly.GetActive()
		playlistName = d.cbxPlaylist.GetActiveID()
		existing = playlistName != newPlaylistId
		if !existing {
			playlistName = d.getNewPlaylistName()
		}
	}
	return
}

func (d *SaveQueueDialog) updateWidgets() {
	// Only show new playlist widgets if (new playlist) is selected in the combo box
	selectedId := d.cbxPlaylist.GetActiveID()
	isNew := selectedId == newPlaylistId
	d.lblPlaylistName.SetVisible(isNew)
	d.ePlaylistName.SetVisible(isNew)

	// Validate the dialog
	valid := (!isNew && selectedId != "") || (isNew && d.getNewPlaylistName() != "")
	d.dlg.SetResponseSensitive(gtk.RESPONSE_YES, valid && !isNew)
	d.dlg.SetResponseSensitive(gtk.RESPONSE_NO, valid)
}

// getNewPlaylistName() returns the text entered in the New playlist name entry, or an empty string if there's an error
func (d *SaveQueueDialog) getNewPlaylistName() string {
	s, err := d.ePlaylistName.GetText()
	if errCheck(err, "ePlaylistName.GetText() failed") {
		return ""
	}
	return s
}
