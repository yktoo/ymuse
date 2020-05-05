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
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

// Builder instance capable of finding specific types of widgets
type Builder struct {
	*gtk.Builder
}

func NewBuilder(fileName string) *Builder {
	builder, err := gtk.BuilderNewFromFile(fileName)
	if err != nil {
		log.Fatal(errors.Errorf("Failed to instantiate a gtk.Builder"))
	}
	return &Builder{Builder: builder}
}

// get() fetches an object with the given name or terminates the app on a failure
func (b *Builder) get(name string) glib.IObject {
	obj, err := b.GetObject(name)
	if err != nil {
		log.Fatal(err)
	}
	return obj
}

// getAdjustment() finds and returns an adjustment by its name
func (b *Builder) getAdjustment(name string) *gtk.Adjustment {
	result, ok := b.get(name).(*gtk.Adjustment)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.Adjustment", name))
	}
	return result
}

// getApplicationWindow() finds and returns an application window by its name
func (b *Builder) getApplicationWindow(name string) *gtk.ApplicationWindow {
	result, ok := b.get(name).(*gtk.ApplicationWindow)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.ApplicationWindow", name))
	}
	return result
}

// getBox() finds and returns a box by its name
func (b *Builder) getBox(name string) *gtk.Box {
	result, ok := b.get(name).(*gtk.Box)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.Box", name))
	}
	return result
}

// getButton() finds and returns a button by its name
func (b *Builder) getButton(name string) *gtk.Button {
	result, ok := b.get(name).(*gtk.Button)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.Button", name))
	}
	return result
}

// getCheckButton() finds and returns a check button by its name
func (b *Builder) getCheckButton(name string) *gtk.CheckButton {
	result, ok := b.get(name).(*gtk.CheckButton)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.CheckButton", name))
	}
	return result
}

// getComboBoxText() finds and returns a text combo box by its name
func (b *Builder) getComboBoxText(name string) *gtk.ComboBoxText {
	result, ok := b.get(name).(*gtk.ComboBoxText)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.ComboBoxText", name))
	}
	return result
}

// getDialog() finds and returns a dialog by its name
func (b *Builder) getDialog(name string) *gtk.Dialog {
	result, ok := b.get(name).(*gtk.Dialog)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.Dialog", name))
	}
	return result
}

// getEntry() finds and returns an entry by its name
func (b *Builder) getEntry(name string) *gtk.Entry {
	result, ok := b.get(name).(*gtk.Entry)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.Entry", name))
	}
	return result
}

// getLabel() finds and returns a label by its name
func (b *Builder) getLabel(name string) *gtk.Label {
	result, ok := b.get(name).(*gtk.Label)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.Label", name))
	}
	return result
}

// getListBox() finds and returns a list box by its name
func (b *Builder) getListBox(name string) *gtk.ListBox {
	result, ok := b.get(name).(*gtk.ListBox)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.ListBox", name))
	}
	return result
}

// getListStore() finds and returns a list store by its name
func (b *Builder) getListStore(name string) *gtk.ListStore {
	result, ok := b.get(name).(*gtk.ListStore)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.ListStore", name))
	}
	return result
}

// getPopoverMenu() finds and returns a popover menu by its name
func (b *Builder) getPopoverMenu(name string) *gtk.PopoverMenu {
	result, ok := b.get(name).(*gtk.PopoverMenu)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.PopoverMenu", name))
	}
	return result
}

// getScale() finds and returns a scale by its name
func (b *Builder) getScale(name string) *gtk.Scale {
	result, ok := b.get(name).(*gtk.Scale)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.Scale", name))
	}
	return result
}

// getStack() finds and returns a stack by its name
func (b *Builder) getStack(name string) *gtk.Stack {
	result, ok := b.get(name).(*gtk.Stack)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.Stack", name))
	}
	return result
}

// getSwitch() finds and returns a switch by its name
func (b *Builder) getSwitch(name string) *gtk.Switch {
	result, ok := b.get(name).(*gtk.Switch)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.Switch", name))
	}
	return result
}

// getToggleToolButton() finds and returns a toggle tool button by its name
func (b *Builder) getToggleToolButton(name string) *gtk.ToggleToolButton {
	result, ok := b.get(name).(*gtk.ToggleToolButton)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.ToggleToolButton", name))
	}
	return result
}

// getToolButton() finds and returns a tool button by its name
func (b *Builder) getToolButton(name string) *gtk.ToolButton {
	result, ok := b.get(name).(*gtk.ToolButton)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.ToolButton", name))
	}
	return result
}

// getTreeView() finds and returns a tree view by its name
func (b *Builder) getTreeView(name string) *gtk.TreeView {
	result, ok := b.get(name).(*gtk.TreeView)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.TreeView", name))
	}
	return result
}
