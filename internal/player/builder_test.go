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
	"testing"
)

func TestBuilder_BindWidgets(t *testing.T) {
	vInt := 1
	tests := []struct {
		name    string
		content string
		target  interface{}
		wantErr bool
	}{
		{
			name:    "int instead of pointer",
			target:  1,
			wantErr: true,
		},
		{
			name:    "*int instead of pointer",
			target:  &vInt,
			wantErr: true,
		},
		{
			name:    "struct with non-pointer field",
			target:  &struct{ Value string }{},
			wantErr: true,
		},
		{
			name:    "struct field of wrong name",
			content: `<interface><object class="GtkButton" id="Value1"/></interface>`,
			target:  &struct{ Value *gtk.Button }{},
			wantErr: true,
		},
		{
			name:    "struct field of wrong type",
			content: `<interface><object class="GtkToolButton" id="MyButton"/></interface>`,
			target:  &struct{ MyButton *gtk.Button }{},
			wantErr: true,
		},
		{
			name:   "empty struct",
			target: &struct{}{},
		},
		{
			name: "struct with no exported fields",
			target: &struct {
				x int
				y string
			}{},
		},
		{
			name:    "happy flow for MainWindow",
			content: playerGlade,
			target:  &MainWindow{},
		},
		{
			name:    "happy flow for Preferences",
			content: prefsGlade,
			target:  &PrefsDialog{},
		},
		{
			name:    "happy flow for Shortcuts",
			content: shortcutsGlade,
			target:  &struct{ ShortcutsWindow *gtk.ShortcutsWindow }{},
		},
	}

	// Need to init GTK first
	gtk.Init(nil)

	// Run the tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Instantiate a builder
			if b, err := NewBuilder(tt.content); err != nil {
				t.Errorf("BindWidgets() error in NewBuilder() = %v, wantErr %v", err, tt.wantErr)
			} else if err := b.BindWidgets(tt.target); (err != nil) != tt.wantErr {
				t.Errorf("BindWidgets() error = %v, wantErr %v", err, tt.wantErr)
			} else if err != nil {
				fmt.Println("Got error:", err)
			}
		})
	}
}

func TestNewBuilder(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantErr  bool
		id       string
		wantType string
	}{
		{name: "empty file"},
		{
			name:    "bad XML",
			content: "<unclosed-tag><some-tag/>",
			wantErr: true,
		},
		{
			name: "bad GTK version",
			content: `<?xml version="1.0" encoding="UTF-8"?>
				<interface>
				  <requires lib="gtk+" version="30.22"/>
				  <object class="GtkShortcutsWindow" id="x"/>
				</interface>`,
			wantErr: true,
		},
		{
			name:    "unknown widget class",
			content: `<interface><object class="FooBarWindow" id="foo"/></interface>`,
			wantErr: true,
		},
		{
			name:     "happy flow for GtkButton",
			content:  `<interface><object class="GtkButton" id="btn"/></interface>`,
			id:       "btn",
			wantType: "*gtk.Button",
		},
		{
			name:     "happy flow for GtkApplicationWindow",
			content:  `<interface><object class="GtkApplicationWindow" id="win"/></interface>`,
			id:       "win",
			wantType: "*gtk.ApplicationWindow",
		},
		{
			name:     "happy flow for GtkDialog",
			content:  `<interface><object class="GtkDialog" id="dlg"/></interface>`,
			id:       "dlg",
			wantType: "*gtk.Dialog",
		},
		{
			name:     "happy flow for GtkEntry",
			content:  `<interface><object class="GtkEntry" id="ENTRY"/></interface>`,
			id:       "ENTRY",
			wantType: "*gtk.Entry",
		},
		{
			name:     "happy flow for GtkMenu",
			content:  `<interface><object class="GtkMenu" id="mnu"/></interface>`,
			id:       "mnu",
			wantType: "*gtk.Menu",
		},
		{
			name:     "happy flow for GtkToolbar",
			content:  `<interface><object class="GtkToolbar" id="TB"/></interface>`,
			id:       "TB",
			wantType: "*gtk.Toolbar",
		},
		{
			name: "happy flow for nested GtkButton",
			content: `<interface>
					<object class="GtkBox">
						<child>
							<object class="GtkButton" id="btn"/>
						</child>
					</object>
				</interface>`,
			id:       "btn",
			wantType: "*gtk.Button",
		},
	}

	// Need to init GTK first
	gtk.Init(nil)

	// Run the tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewBuilder(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBuilder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// On an expected error or if there's nothing more to check: stop here
			if err != nil || tt.id == "" {
				if err != nil {
					fmt.Println("Got error:", err)
				}
				return
			}

			// Fetch and check the type of the object
			if obj, err := got.Builder.GetObject(tt.id); err != nil {
				t.Errorf("NewBuilder() failed to get object with ID=%v: %v", tt.id, err)
			} else if gotType := fmt.Sprintf("%T", obj); gotType != tt.wantType {
				t.Errorf("NewBuilder() object with ID=%v: got = %v, want %v", tt.id, gotType, tt.wantType)
			}
		})
	}
}
