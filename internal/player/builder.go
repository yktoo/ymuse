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
	"fmt"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
	"reflect"
)

// Builder instance capable of finding specific types of widgets
type Builder struct {
	*gtk.Builder
}

// NewBuilder creates and returns a new Builder instance
func NewBuilder(content string) *Builder {
	builder, err := gtk.BuilderNew()
	if err != nil {
		panic(errors.Errorf("gtk.BuilderNew() failed"))
	}
	if err := builder.AddFromString(content); err != nil {
		panic(errors.Errorf("builder.AddFromString() failed"))
	}
	return &Builder{Builder: builder}
}

// BindWidgets binds the builder's widgets to same-named fields in the provided struct. Only exported fields are taken
// into account
func (b *Builder) BindWidgets(obj interface{}) error {
	// We're only dealing with structs
	vPtr := reflect.ValueOf(obj)
	if vPtr.Kind() != reflect.Ptr || vPtr.IsNil() || vPtr.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("*struct expected, %T was given", obj)
	}

	// Fetch a value for the struct vPtr points to
	v := vPtr.Elem()

	// Iterate over struct's fields
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		valField := v.Field(i)
		if valField.CanSet() {
			// Verify it's a pointer
			typeField := t.Field(i)
			if valField.Kind() != reflect.Ptr {
				return fmt.Errorf("struct'd field %s is %v, but only pointers are supported", typeField.Name, valField.Kind())
			}

			// Try to find a widget with the field's name
			widget, err := b.GetObject(typeField.Name)
			if err != nil {
				return err
			}

			// Try to cast the value to the target type
			var targetVal reflect.Value
			func() {
				err = nil
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("failed to cast IObject (ID=%s) to %s: %v", typeField.Name, typeField.Type, r)
					}
				}()
				targetVal = reflect.ValueOf(widget).Convert(typeField.Type)
			}()
			if err != nil {
				return err
			}

			// Set the value. Any possible panic won't be recovered
			valField.Set(targetVal)
		}
	}
	return nil
}
