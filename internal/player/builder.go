package player

import (
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
	"log"
)

// Builder instance capable of finding specific types of widgets
type Builder struct {
	*gtk.Builder
}

func NewBuilder(fileName string) *Builder {
	builder, err := gtk.BuilderNewFromFile(fileName)
	if err != nil {
		log.Fatalln(errors.Errorf("Failed to instantiate a gtk.Builder"))
	}
	return &Builder{Builder: builder}
}

// getApplicationWindow() finds and returns an application window by its name
func (b *Builder) getApplicationWindow(name string) *gtk.ApplicationWindow {
	obj, err := b.GetObject(name)
	if err != nil {
		log.Fatalln(errors.Errorf("Failed to find application window %v: %v", name, err))
	}

	// Check object type
	result, ok := obj.(*gtk.ApplicationWindow)
	if !ok {
		log.Fatalln(errors.Errorf("%v is not a gtk.ApplicationWindow", name))
	}
	return result
}

// getLabel() finds and returns a label by its name
func (b *Builder) getLabel(name string) *gtk.Label {
	obj, err := b.GetObject(name)
	if err != nil {
		log.Fatalln(errors.Errorf("Failed to find label %v: %v", name, err))
	}

	// Check object type
	result, ok := obj.(*gtk.Label)
	if !ok {
		log.Fatalln(errors.Errorf("%v is not a gtk.Label", name))
	}
	return result
}

// getButton() finds and returns a button by its name
func (b *Builder) getButton(name string) *gtk.Button {
	obj, err := b.GetObject(name)
	if err != nil {
		log.Fatalln(errors.Errorf("Failed to find button %v: %v", name, err))
	}

	// Check object type
	result, ok := obj.(*gtk.Button)
	if !ok {
		log.Fatalln(errors.Errorf("%v is not a gtk.Button", name))
	}
	return result
}

// getScale() finds and returns a scale by its name
func (b *Builder) getScale(name string) *gtk.Scale {
	obj, err := b.GetObject(name)
	if err != nil {
		log.Fatalln(errors.Errorf("Failed to find scale %v: %v", name, err))
	}

	// Check object type
	result, ok := obj.(*gtk.Scale)
	if !ok {
		log.Fatalln(errors.Errorf("%v is not a gtk.Scale", name))
	}
	return result
}

// getTreeView() finds and returns a tree view by its name
func (b *Builder) getTreeView(name string) *gtk.TreeView {
	obj, err := b.GetObject(name)
	if err != nil {
		log.Fatalln(errors.Errorf("Failed to find tree view %v: %v", name, err))
	}

	// Check object type
	result, ok := obj.(*gtk.TreeView)
	if !ok {
		log.Fatalln(errors.Errorf("%v is not a gtk.TreeView", name))
	}
	return result
}

// getListStore() finds and returns a list store by its name
func (b *Builder) getListStore(name string) *gtk.ListStore {
	obj, err := b.GetObject(name)
	if err != nil {
		log.Fatalln(errors.Errorf("Failed to find list store %v: %v", name, err))
	}

	// Check object type
	result, ok := obj.(*gtk.ListStore)
	if !ok {
		log.Fatalln(errors.Errorf("%v is not a gtk.ListStore", name))
	}
	return result
}
