package player

import (
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

// whenIdle() schedules a function call on GLib's main loop thread
func whenIdle(name string, f interface{}, args ...interface{}) {
	_, err := glib.IdleAdd(f, args...)
	errCheck(err, "glib.IdleAdd() failed for "+name)
}

// clearChildren() removes all container's children
func clearChildren(container gtk.Container) {
	container.GetChildren().Foreach(func(item interface{}) {
		container.Remove(item.(gtk.IWidget))
	})
}

// newButton() creates and returns a new button
func newButton(label, tooltip, name, icon string, onClicked interface{}, onClickedData ...interface{}) *gtk.Button {
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

// newBoxToggleButton() creates, adds to a box and returns a new toggle button
func newBoxToggleButton(box *gtk.Box, label, name, icon string, active bool, onClicked interface{}, onClickedData ...interface{}) *gtk.ToggleButton {
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

// newListBoxRow() adds a new row to the list box, a horizontal box, an image and a label to it
// listBox: list box instance
// label: text for the row
// name: name of the row
// icon: optional icon name for the row
func newListBoxRow(listBox *gtk.ListBox, label, name, icon string) (*gtk.ListBoxRow, *gtk.Box, error) {
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
