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

package toolkit

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	log "github.com/sirupsen/logrus"
)

type executableTarget struct {
	wrapperName string
}

type executable struct {
	source string
	target executableTarget
	argv   []string
	envm   map[string]string
}

// install installs an executable component of the NVIDIA container toolkit. The source executable
// is copied to a `.real` file and a wapper is created to set up the environment as required.
func (e executable) install(destFolder string) (string, error) {
	if destFolder == "" {
		return "", fmt.Errorf("destination folder must be specified")
	}
	if e.source == "" {
		return "", fmt.Errorf("source executable must be specified")
	}
	log.Infof("Installing executable '%v' to %v", e.source, destFolder)
	dotRealFilename := e.dotRealFilename()
	dotRealPath, err := installFileToFolderWithName(destFolder, dotRealFilename, e.source)
	if err != nil {
		return "", fmt.Errorf("error installing file '%v' as '%v': %v", e.source, dotRealFilename, err)
	}
	log.Infof("Installed '%v'", dotRealPath)

	wrapperPath, err := e.installWrapper(destFolder)
	if err != nil {
		return "", fmt.Errorf("error installing wrapper: %v", err)
	}
	log.Infof("Installed wrapper '%v'", wrapperPath)
	return wrapperPath, nil
}

func (e executable) dotRealFilename() string {
	return e.wrapperName() + ".real"
}

func (e executable) wrapperName() string {
	if e.target.wrapperName == "" {
		return filepath.Base(e.source)
	}
	return e.target.wrapperName
}

func (e executable) installWrapper(destFolder string) (string, error) {
	currentExe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("error getting current executable: %v", err)
	}
	src := filepath.Join(filepath.Dir(currentExe), "wrapper")
	wrapperPath, err := installFileToFolderWithName(destFolder, e.wrapperName(), src)
	if err != nil {
		return "", fmt.Errorf("error installing wrapper program: %v", err)
	}
	err = e.writeWrapperArgv(wrapperPath, destFolder)
	if err != nil {
		return "", fmt.Errorf("error writing wrapper argv: %v", err)
	}
	err = e.writeWrapperEnvv(wrapperPath, destFolder)
	if err != nil {
		return "", fmt.Errorf("error writing wrapper envv: %v", err)
	}
	err = ensureExecutable(wrapperPath)
	if err != nil {
		return "", fmt.Errorf("error making wrapper executable: %v", err)
	}
	return wrapperPath, nil
}

func (e executable) writeWrapperArgv(wrapperPath string, destFolder string) error {
	if e.argv == nil {
		return nil
	}
	r := newReplacements(destDirPattern, destFolder)
	f, err := os.OpenFile(wrapperPath+".argv", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0440)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, arg := range e.argv {
		fmt.Fprintf(f, "%s\n", r.apply(arg))
	}
	return nil
}

func (e executable) writeWrapperEnvv(wrapperPath string, destFolder string) error {
	r := newReplacements(destDirPattern, destFolder)
	f, err := os.OpenFile(wrapperPath+".envv", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0440)
	if err != nil {
		return err
	}
	defer f.Close()

	// Update PATH to insert the destination folder at the head.
	var envm map[string]string
	if e.envm == nil {
		envm = make(map[string]string)
	} else {
		envm = e.envm
	}
	if path, ok := envm["PATH"]; ok {
		envm["PATH"] = destFolder + ":" + path
	} else {
		// Replace PATH with <PATH, which instructs wrapper to insert the value at the head of a
		// colon-separated environment variable list.
		delete(envm, "PATH")
		envm["<PATH"] = destFolder
	}

	var envv []string
	for k, v := range envm {
		envv = append(envv, k+"="+r.apply(v))
	}
	sort.Strings(envv)
	for _, e := range envv {
		fmt.Fprintf(f, "%s\n", e)
	}
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
