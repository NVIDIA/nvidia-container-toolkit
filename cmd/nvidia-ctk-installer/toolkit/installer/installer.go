/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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
**/

package installer

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

//go:generate moq -rm -fmt=goimports -out installer_mock.go . Installer
type Installer interface {
	Install(string) error
}

type toolkitInstaller struct {
	logger       logger.Interface
	ignoreErrors bool
	sourceRoot   string

	artifactRoot *artifactRoot

	ensureTargetDirectory Installer
}

var _ Installer = (*toolkitInstaller)(nil)

// New creates a toolkit installer with the specified options.
func New(opts ...Option) (Installer, error) {
	t := &toolkitInstaller{
		sourceRoot: "/",
	}
	for _, opt := range opts {
		opt(t)
	}

	if t.logger == nil {
		t.logger = logger.New()
	}
	if t.artifactRoot == nil {
		artifactRoot, err := newArtifactRoot(t.logger, t.sourceRoot)
		if err != nil {
			return nil, err
		}
		t.artifactRoot = artifactRoot
	}

	if t.ensureTargetDirectory == nil {
		t.ensureTargetDirectory = t.createDirectory()
	}

	return t, nil
}

// Install ensures that the required toolkit files are installed in the specified directory.
func (t *toolkitInstaller) Install(destDir string) error {
	var installers []Installer

	installers = append(installers, t.ensureTargetDirectory)

	libraries, err := t.collectLibraries()
	if err != nil {
		return fmt.Errorf("failed to collect libraries: %w", err)
	}
	installers = append(installers, libraries...)

	executables, err := t.collectExecutables(destDir)
	if err != nil {
		return fmt.Errorf("failed to collect executables: %w", err)
	}
	installers = append(installers, executables...)

	var errs error
	for _, i := range installers {
		errs = errors.Join(errs, i.Install(destDir))
	}

	return errs
}

type symlink struct {
	linkname string
	target   string
}

func (s symlink) Install(destDir string) error {
	symlinkPath := filepath.Join(destDir, s.linkname)
	return installSymlink(s.target, symlinkPath)
}

//go:generate moq -rm -fmt=goimports -out file-installer_mock.go . fileInstaller
type fileInstaller interface {
	installContent(io.Reader, string, os.FileMode) error
	installFile(string, string) (os.FileMode, error)
	installSymlink(string, string) error
}

var installSymlink = installSymlinkStub

func installSymlinkStub(target string, link string) error {
	err := os.Symlink(target, link)
	if err != nil {
		return fmt.Errorf("error creating symlink '%v' => '%v': %v", link, target, err)
	}
	return nil
}

var installFile = installFileStub

func installFileStub(src string, dest string) (os.FileMode, error) {
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return 0, fmt.Errorf("error getting file info for '%v': %v", src, err)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, fmt.Errorf("error opening source: %w", err)
	}
	defer source.Close()

	mode := sourceInfo.Mode()
	if err := installContent(source, dest, mode); err != nil {
		return 0, err
	}
	return mode, nil
}

var installContent = installContentStub

func installContentStub(content io.Reader, dest string, mode fs.FileMode) error {
	destination, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("error creating destination: %w", err)
	}
	defer destination.Close()

	_, err = io.Copy(destination, content)
	if err != nil {
		return fmt.Errorf("error copying file: %w", err)
	}
	err = os.Chmod(dest, mode)
	if err != nil {
		return fmt.Errorf("error setting mode for '%v': %v", dest, err)
	}
	return nil
}
