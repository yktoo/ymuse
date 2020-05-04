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
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

// WhenIdle() schedules a function call on GLib's main loop thread
func WhenIdle(name string, f interface{}, args ...interface{}) {
	_, err := glib.IdleAdd(f, args...)
	errCheck(err, "glib.IdleAdd() failed for "+name)
}

// ClearChildren() removes all container's children
func ClearChildren(container gtk.Container) {
	container.GetChildren().Foreach(func(item interface{}) {
		container.Remove(item.(gtk.IWidget))
	})
}

// NewButton() creates and returns a new button
func NewButton(label, tooltip, name, icon string, onClicked interface{}, onClickedData ...interface{}) *gtk.Button {
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
	_, err = btn.Connect("clicked", onClicked, onClickedData)
	if errCheck(err, "Connect() failed") {
		return nil
	}
	return btn
}

// NewBoxToggleButton() creates, adds to a box and returns a new toggle button
func NewBoxToggleButton(box *gtk.Box, label, name, icon string, active bool, onClicked interface{}, onClickedData ...interface{}) *gtk.ToggleButton {
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
	_, err = btn.Connect("clicked", onClicked, onClickedData)
	if errCheck(err, "Connect() failed") {
		return nil
	}

	// Add the button to the box
	box.PackStart(btn, false, false, 0)
	return btn
}

// NewListBoxRow() adds a new row to the list box, a horizontal box, an image and a label to it
// listBox: list box instance
// label: text for the row
// name: name of the row
// icon: optional icon name for the row
func NewListBoxRow(listBox *gtk.ListBox, label, name, icon string) (*gtk.ListBoxRow, *gtk.Box, error) {
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

	// Insert icon, if needed
	if icon != "" {
		// Icon is optional, do not fail entirely on an error
		if img, err := gtk.ImageNewFromIconName(icon, gtk.ICON_SIZE_LARGE_TOOLBAR); !errCheck(err, "ImageNewFromIconName() failed") {
			hbx.PackStart(img, false, false, 0)
		}
	}

	// Insert label with directory/file name
	lbl, err := gtk.LabelNew(label)
	if err != nil {
		return nil, nil, err
	}
	lbl.SetXAlign(0)
	lbl.SetEllipsize(pango.ELLIPSIZE_END)
	hbx.PackStart(lbl, true, true, 0)

	// Add the row to the list box
	listBox.Add(row)
	return row, hbx, nil
}

// NotImplemented() shows a "function not implemented" message dialog
func NotImplemented(parent gtk.IWindow) {
	dlg := gtk.MessageDialogNew(parent, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, "Function not implemented.")
	defer dlg.Destroy()
	dlg.Run()
}

// ConfirmDialog() shows a confirmation message dialog
func ConfirmDialog(parent gtk.IWindow, text string) bool {
	dlg := gtk.MessageDialogNew(parent, gtk.DIALOG_MODAL, gtk.MESSAGE_QUESTION, gtk.BUTTONS_OK_CANCEL, text)
	defer dlg.Destroy()
	return dlg.Run() == gtk.RESPONSE_OK
}

// ErrorDialog() shows an error message dialog
func ErrorDialog(parent gtk.IWindow, text string) {
	dlg := gtk.MessageDialogNew(parent, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, text)
	defer dlg.Destroy()
	dlg.Run()
}

// EditDialog() show a dialog with a single text entry
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
			w.SetSensitive(err == nil && text != "")
		}
	}
	_, _ = entry.Connect("changed", validate)
	_, _ = dlg.Connect("map", validate)
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
