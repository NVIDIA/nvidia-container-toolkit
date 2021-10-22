/**
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
*/

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
)

type executableTarget struct {
	dotfileName string
	wrapperName string
}

type executable struct {
	source   string
	target   executableTarget
	env      map[string]string
	preLines []string
	argLines []string
}

// install installs an executable component of the NVIDIA container toolkit. The source executable
// is copied to a `.real` file and a wapper is created to set up the environment as required.
func (e executable) install(destFolder string) (string, error) {
	log.Infof("Installing executable '%v' to %v", e.source, destFolder)

	dotfileName := e.dotfileName()

	installedDotfileName, err := installFileToFolderWithName(destFolder, dotfileName, e.source)
	if err != nil {
		return "", fmt.Errorf("error installing file '%v' as '%v': %v", e.source, dotfileName, err)
	}
	log.Infof("Installed '%v'", installedDotfileName)

	wrapperFilename, err := e.installWrapper(destFolder, installedDotfileName)
	if err != nil {
		return "", fmt.Errorf("error wrapping '%v': %v", installedDotfileName, err)
	}
	log.Infof("Installed wrapper '%v'", wrapperFilename)

	return wrapperFilename, nil
}

func (e executable) dotfileName() string {
	return e.target.dotfileName
}

func (e executable) wrapperName() string {
	return e.target.wrapperName
}

func (e executable) installWrapper(destFolder string, dotfileName string) (string, error) {
	wrapperPath := filepath.Join(destFolder, e.wrapperName())
	wrapper, err := os.Create(wrapperPath)
	if err != nil {
		return "", fmt.Errorf("error creating executable wrapper: %v", err)
	}
	defer wrapper.Close()

	err = e.writeWrapperTo(wrapper, destFolder, dotfileName)
	if err != nil {
		return "", fmt.Errorf("error writing wrapper contents: %v", err)
	}

	err = ensureExecutable(wrapperPath)
	if err != nil {
		return "", fmt.Errorf("error making wrapper executable: %v", err)
	}
	return wrapperPath, nil
}

func (e executable) writeWrapperTo(wrapper io.Writer, destFolder string, dotfileName string) error {
	r := newReplacements(destDirPattern, destFolder)

	// Add the shebang
	fmt.Fprintln(wrapper, "#! /bin/sh")

	// Add the preceding lines if any
	for _, line := range e.preLines {
		fmt.Fprintf(wrapper, "%s\n", r.apply(line))
	}

	// Update the path to include the destination folder
	var env map[string]string
	if e.env == nil {
		env = make(map[string]string)
	} else {
		env = e.env
	}

	path, specified := env["PATH"]
	if !specified {
		path = "$PATH"
	}
	env["PATH"] = strings.Join([]string{destFolder, path}, ":")

	var sortedEnvvars []string
	for e := range env {
		sortedEnvvars = append(sortedEnvvars, e)
	}
	sort.Strings(sortedEnvvars)

	for _, e := range sortedEnvvars {
		v := env[e]
		fmt.Fprintf(wrapper, "%s=%s \\\n", e, r.apply(v))
	}
	// Add the call to the target executable
	fmt.Fprintf(wrapper, "%s \\\n", dotfileName)

	// Insert additional lines in the `arg` list
	for _, line := range e.argLines {
		fmt.Fprintf(wrapper, "\t%s \\\n", r.apply(line))
	}
	// Add the script arguments "$@"
	fmt.Fprintln(wrapper, "\t\"$@\"")

	return nil
}

// ensureExecutable is equivalent to running chmod +x on the specified file
func ensureExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("error getting file info for '%v': %v", path, err)
	}
	executableMode := info.Mode() | 0111
	err = os.Chmod(path, executableMode)
	if err != nil {
		return fmt.Errorf("error setting executable mode for '%v': %v", path, err)
	}
	return nil
}
