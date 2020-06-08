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
	"errors"
	"github.com/op/go-logging"
	"testing"
)

type TestLogBackend struct {
	logging.Leveled
	level  logging.Level
	record *logging.Record
}

func (t *TestLogBackend) Log(level logging.Level, _ int, record *logging.Record) error {
	t.level = level
	t.record = record
	return nil
}

func (t *TestLogBackend) reset() {
	t.level = 0
	t.record = nil
}

func Test_errCheck(t *testing.T) {
	type args struct {
		err     error
		message string
	}
	tests := []struct {
		name        string
		args        args
		want        bool
		wantLevel   logging.Level
		wantMessage string
	}{
		{"no error", args{err: nil, message: "foo"}, false, 0, ""},
		{"error", args{err: errors.New("boom"), message: "foo failed"}, true, logging.WARNING, "foo failed: boom"},
	}

	// Use a fake log backend for test
	backend := &TestLogBackend{}
	log.SetBackend(backend)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check the function
			backend.reset()
			if got := errCheck(tt.args.err, tt.args.message); got != tt.want {
				t.Errorf("errCheck() = %v, want %v", got, tt.want)
			}

			// Check the logging
			if backend.level != tt.wantLevel {
				t.Errorf("errCheck() log level = %v, want %v", backend.level, tt.wantLevel)
			}
			if tt.wantMessage == "" {
				if backend.record != nil {
					t.Errorf("errCheck() log message must be nil but it's %v", backend.record.Message())
				}

			} else if backend.record.Message() != tt.wantMessage {
				t.Errorf("errCheck() log message = %v, want %v", backend.record.Message(), tt.wantMessage)
			}
		})
	}
}
