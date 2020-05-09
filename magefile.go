// +build mage

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

package main

import (
	"errors"
	"fmt"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Install the (production-) built binary of the application
func Install() error {
	mg.Deps(Deps, Resources)
	return sh.Run("go", "install", "-ldflags", "-s -w")
}

// Builds the project
func Build() error {
	mg.Deps(Deps, Resources)

	// Build the application
	if err := sh.Run("go", "build"); err != nil {
		return err
	}
	return nil
}

// Downloads all project dependencies
func Deps() error {
	if err := sh.Run("go", "mod", "download"); err != nil {
		return err
	}
	return nil
}

// Generates .go files from required resources (for now only .glade files are processed)
func Resources() error {
	// List all resource files in the project
	var files []string
	err := filepath.Walk(
		".",
		func(filename string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && path.Ext(filename) == ".glade" {
				files = append(files, filename)
			}
			return nil
		})
	if err != nil {
		return err
	}

	// Collect all functions into a string
	resourceContent := "package generated\n\n" +
		"// Generated with 'mage resources'\n"
	for _, filename := range files {
		log.Println("Processing resource file", filename)
		function, err := fileToFunction(filename)
		if err != nil {
			return err
		}
		resourceContent += function
	}

	// Store the text into the internal/generated/resources.go file
	resourceFilename := path.Join("internal", "generated", "resources.go")
	log.Println("Writing Go function file", resourceFilename)
	if err := os.MkdirAll(path.Dir(resourceFilename), 0755); err != nil {
		return err
	}
	return ioutil.WriteFile(resourceFilename, []byte(resourceContent), 0644)
}

// fileToFunction() converts a text file into a function
func fileToFunction(filename string) (string, error) {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}

	// Make a Go-compliant function name
	funcName := "Get"
	for _, s := range regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9]+`).FindAllString(path.Base(filename), -1) {
		funcName += strings.Title(s)
	}
	if funcName == "" {
		return "", errors.New("Failed to find an appropriate function name")
	}

	// Make a function out of the contents
	return fmt.Sprintf(
			"\n// %[1]s returns the contents stored in the file %[2]s\n"+
				"func %[1]s() string {\n"+
				"\treturn `%[3]s`\n"+
				"}\n",
			funcName, filename, string(contents)),
		nil
}
