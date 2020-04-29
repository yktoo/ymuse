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

// getApplicationWindow() finds and returns an application window by its name
func (b *Builder) getApplicationWindow(name string) *gtk.ApplicationWindow {
	result, ok := b.get(name).(*gtk.ApplicationWindow)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.ApplicationWindow", name))
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

// getToolButton() finds and returns a tool button by its name
func (b *Builder) getToolButton(name string) *gtk.ToolButton {
	result, ok := b.get(name).(*gtk.ToolButton)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.ToolButton", name))
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

// getScale() finds and returns a scale by its name
func (b *Builder) getScale(name string) *gtk.Scale {
	result, ok := b.get(name).(*gtk.Scale)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.Scale", name))
	}
	return result
}

// getAdjustment() finds and returns an adjustment by its name
func (b *Builder) getAdjustment(name string) *gtk.Adjustment {
	result, ok := b.get(name).(*gtk.Adjustment)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.Adjustment", name))
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

// getListStore() finds and returns a list store by its name
func (b *Builder) getListStore(name string) *gtk.ListStore {
	result, ok := b.get(name).(*gtk.ListStore)
	if !ok {
		log.Fatal(errors.Errorf("%v is not a gtk.ListStore", name))
	}
	return result
}
