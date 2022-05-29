/*
 *   Copyright 2022 Dmitry Kann
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
	"github.com/fhs/gompd/v2/mpd"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/yktoo/ymuse/internal/util"
	"strconv"
)

// OutputsDialog represents the output selection dialog
type OutputsDialog struct {
	OutputsDialog  *gtk.Dialog
	OutputsListBox *gtk.ListBox

	// Connector instance
	connector *Connector
}

// ShowOutputsDialog creates, shows and disposes of an Outputs dialog instance
func ShowOutputsDialog(parent gtk.IWindow, c *Connector) {
	// Create the dialog
	d := &OutputsDialog{
		connector: c,
	}

	// Load the dialog layout and map the widgets
	builder, err := NewBuilder(outputsGlade)
	if err == nil {
		err = builder.BindWidgets(d)
	}

	// Check for errors
	if errCheck(err, "OutputsDialog(): failed to initialise dialog") {
		util.ErrorDialog(parent, fmt.Sprint(glib.Local("Failed to load UI widgets"), err))
		return
	}
	defer d.OutputsDialog.Destroy()

	// Set the dialog up
	d.OutputsDialog.SetTransientFor(parent)

	// Map the handlers to callback functions
	builder.ConnectSignals(map[string]interface{}{
		"on_OutputsDialog_map": d.populateOutputs,
	})

	// Run the dialog
	d.OutputsDialog.Run()
}

func (d *OutputsDialog) switchStateSet(id int, active bool) {
	log.Debugf("switchStateSet(%v, %v)", id, active)
	d.connector.IfConnected(func(client *mpd.Client) {
		if active {
			errCheck(client.EnableOutput(id), "EnableOutput() failed")
		} else {
			errCheck(client.DisableOutput(id), "DisableOutput() failed")
		}
	})
}

// populateOutputs fills in the Outputs list box
func (d *OutputsDialog) populateOutputs() {
	// Fetch the outputs
	var attrs []mpd.Attrs
	var err error
	d.connector.IfConnected(func(client *mpd.Client) {
		attrs, err = client.ListOutputs()
	})
	if errCheck(err, "populateOutputs(): ListOutputs() failed") {
		return
	}

	// Add output rows to the list
	for _, a := range attrs {
		// Parse the output ID
		var id int
		if id, err = strconv.Atoi(a["outputid"]); errCheck(err, "Invalid output ID") {
			return
		}

		// Add a switch
		sw, err := gtk.SwitchNew()
		if errCheck(err, "SwitchNew() failed") {
			return
		}
		sw.SetActive(a["outputenabled"] != "0")
		sw.Connect("state-set", func(_ *gtk.Switch, state bool) {
			d.switchStateSet(id, state)
		})

		// Add a new list box row
		text := fmt.Sprintf("%s <i>(%s)</i>", a["outputname"], a["plugin"])
		if _, _, err := util.NewListBoxRow(d.OutputsListBox, true, text, "", "", sw); errCheck(err, "NewListBoxRow() failed") {
			return
		}
	}
	d.OutputsListBox.ShowAll()
}
