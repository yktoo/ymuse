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

package util

import (
	"fmt"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"html"
)

// ClearChildren removes all container's children
func ClearChildren(container gtk.Container) {
	container.GetChildren().Foreach(func(item interface{}) {
		container.Remove(item.(gtk.IWidget))
	})
}

// NewButton creates and returns a new button
func NewButton(label, tooltip, name, icon string, onClicked func()) *gtk.Button {
	btn, err := gtk.ButtonNewWithLabel(label)
	if errCheck(err, "ButtonNewWithLabel() failed") {
		return nil
	}
	btn.SetName(name)
	btn.SetTooltipText(tooltip)

	// Create an icon, if needed
	if icon != "" {
		// Icon is optional, do not fail entirely on an error
		if img, err := gtk.ImageNewFromIconName(icon, gtk.ICON_SIZE_BUTTON); !errCheck(err, "ImageNewFromIconName() failed") {
			btn.SetImage(img)
			btn.SetAlwaysShowImage(true)
		}
	}

	// Bind the clicked signal
	btn.Connect("clicked", onClicked)
	return btn
}

// NewBoxToggleButton creates, adds to a box and returns a new toggle button
func NewBoxToggleButton(box *gtk.Box, label, name, icon string, active bool, onClicked func()) *gtk.ToggleButton {
	btn, err := gtk.ToggleButtonNewWithLabel(label)
	if errCheck(err, "ToggleButtonNewWithLabel() failed") {
		return nil
	}
	btn.SetName(name)
	btn.SetActive(active)

	// Create an icon, if needed
	if icon != "" {
		// Icon is optional, do not fail entirely on an error
		if img, err := gtk.ImageNewFromIconName(icon, gtk.ICON_SIZE_BUTTON); !errCheck(err, "ImageNewFromIconName() failed") {
			btn.SetImage(img)
			btn.SetAlwaysShowImage(true)
		}
	}

	// Bind the clicked signal
	btn.Connect("clicked", onClicked)

	// Add the button to the box
	box.PackStart(btn, false, false, 0)
	return btn
}

// NewLabel instantiates and returns a new label
func NewLabel(label string) *gtk.Label {
	lbl, err := gtk.LabelNew(label)
	if errCheck(err, "LabelNew() failed") {
		return nil
	}
	lbl.SetXAlign(0)
	return lbl
}

// NewListBoxRow adds a new row to the list box, a horizontal box, an image and a label to it
// listBox: list box instance
// useMarkup: whether label is markup
// label: text for the row
// name: name of the row
// icon: optional icon name for the row
// widgets: extra widgets to insert into the beginning of the row
func NewListBoxRow(listBox *gtk.ListBox, useMarkup bool, label, name, icon string, widgets ...gtk.IWidget) (*gtk.ListBoxRow, *gtk.Box, error) {
	// Add a new list box row
	row, err := gtk.ListBoxRowNew()
	if err != nil {
		return nil, nil, err
	}
	row.SetName(name)

	// Add horizontal box
	hbx, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 6)
	if err != nil {
		return nil, nil, err
	}
	hbx.SetMarginStart(6)
	hbx.SetMarginEnd(6)
	row.Add(hbx)

	// Add extra widgets, if any
	for _, w := range widgets {
		hbx.PackStart(w, false, false, 0)
	}

	// Insert icon, if needed
	if icon != "" {
		// Icon is optional, do not fail entirely on an error
		if img, err := gtk.ImageNewFromIconName(icon, gtk.ICON_SIZE_LARGE_TOOLBAR); !errCheck(err, "ImageNewFromIconName() failed") {
			hbx.PackStart(img, false, false, 0)
		}
	}

	// Insert label with directory/file name
	lbl, err := gtk.LabelNew("")
	if err != nil {
		return nil, nil, err
	}
	lbl.SetXAlign(0)
	lbl.SetEllipsize(pango.ELLIPSIZE_END)
	if useMarkup {
		lbl.SetMarkup(label)
	} else {
		lbl.SetText(label)
	}
	hbx.PackStart(lbl, true, true, 0)

	// Add the row to the list box
	listBox.Add(row)
	return row, hbx, nil
}

// ListBoxScrollToSelected scrolls the provided list box so that the selected row is centered in the window
func ListBoxScrollToSelected(listBox *gtk.ListBox) {
	// If there's selection
	if row := listBox.GetSelectedRow(); row != nil {
		// Convert the row's Y coordinate into the list box's coordinate
		if _, y, _ := row.TranslateCoordinates(listBox, 0, 0); y >= 0 {
			// Scroll the vertical adjustment to center the row in the viewport
			if adj := listBox.GetAdjustment(); adj != nil {
				_, rowHeight := row.GetPreferredHeight()
				adj.SetValue(float64(y) - (adj.GetPageSize()-float64(rowHeight))/2)
			}
		}
	}
}

// ConfirmDialog shows a confirmation message dialog
func ConfirmDialog(parent gtk.IWindow, title, text string) bool {
	dlg := gtk.MessageDialogNew(parent, gtk.DIALOG_MODAL, gtk.MESSAGE_QUESTION, gtk.BUTTONS_OK_CANCEL, "")
	dlg.SetMarkup(fmt.Sprintf("<big><b>%v</b></big>\n\n%v", html.EscapeString(title), html.EscapeString(text)))
	defer dlg.Destroy()
	return dlg.Run() == gtk.RESPONSE_OK
}

// GetTextBufferText returns the entire text stored in a text buffer
func GetTextBufferText(buf *gtk.TextBuffer) (string, error) {
	start, end := buf.GetBounds()
	return buf.GetText(start, end, true)
}

// EditDialog show a dialog with a single text entry
func EditDialog(parent gtk.IWindow, title, value, okButton string) (string, bool) {
	// Create a dialog
	dlg, err := gtk.DialogNewWithButtons(
		title,
		parent,
		gtk.DIALOG_MODAL,
		[]interface{}{okButton, gtk.RESPONSE_OK},
		[]interface{}{"Cancel", gtk.RESPONSE_CANCEL})
	if errCheck(err, "DialogNewWithButtons() failed") {
		return "", false
	}
	defer dlg.Destroy()

	// Obtain the dialog's content area
	bx, err := dlg.GetContentArea()
	if errCheck(err, "GetContentArea() failed") {
		return "", false
	}

	// Add a text entry to the dialog
	entry, err := gtk.EntryNew()
	if errCheck(err, "EntryNew() failed") {
		return "", false
	}
	entry.SetSizeRequest(400, -1)
	entry.SetText(value)
	entry.SetMarginStart(12)
	entry.SetMarginEnd(12)
	entry.SetMarginTop(12)
	entry.SetMarginBottom(12)
	entry.GrabFocus()
	bx.Add(entry)

	bx.ShowAll()

	// Enable or disable the OK button based on text presence
	validate := func() {
		if w, err := dlg.GetWidgetForResponse(gtk.RESPONSE_OK); err == nil {
			text, err := entry.GetText()
			w.ToWidget().SetSensitive(err == nil && text != "")
		}
	}
	entry.Connect("changed", validate)
	dlg.Connect("map", validate)
	dlg.SetDefaultResponse(gtk.RESPONSE_OK)

	// Run the dialog
	response := dlg.Run()
	value, err = entry.GetText()
	if errCheck(err, "entry.GetText() failed") {
		return "", false
	}

	// Check the response
	if response == gtk.RESPONSE_OK {
		return value, true
	}
	return "", false
}

// EntryText returns the text in an entry, or the default string if an error occurred
func EntryText(entry *gtk.Entry, def string) string {
	s, err := entry.GetText()
	if errCheck(err, "EntryText(): GetText() failed") {
		return def
	}
	return s
}

// ErrorDialog shows an error message dialog
func ErrorDialog(parent gtk.IWindow, text string) {
	dlg := gtk.MessageDialogNew(parent, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, text)
	defer dlg.Destroy()
	dlg.Run()
}
