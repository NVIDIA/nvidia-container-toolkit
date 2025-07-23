/**
# Copyright 2024 NVIDIA CORPORATION
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

package discover

import (
	"debug/elf"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

type Symlink struct {
	target string
	link   string
}

func (s *Symlink) String() string {
	return fmt.Sprintf("%s::%s", s.target, s.link)
}

type additionalSymlinks struct {
	logger logger.Interface
	Discover
	version     string
	hookCreator HookCreator
}

// WithDriverDotSoSymlinks decorates the provided discoverer.
// A hook is added that checks for specific driver symlinks that need to be created.
func WithDriverDotSoSymlinks(logger logger.Interface, mounts Discover, version string, hookCreator HookCreator) Discover {
	if version == "" {
		version = "*.*"
	}
	return &additionalSymlinks{
		logger:      logger,
		Discover:    mounts,
		hookCreator: hookCreator,
		version:     version,
	}
}

// Hooks returns a hook to create the additional symlinks based on the mounts.
func (d *additionalSymlinks) Hooks() ([]Hook, error) {
	mounts, err := d.Mounts()
	if err != nil {
		return nil, fmt.Errorf("failed to get library mounts: %v", err)
	}
	hooks, err := d.Discover.Hooks()
	if err != nil {
		return nil, fmt.Errorf("failed to get hooks: %v", err)
	}

	var links []string
	processedPaths := make(map[string]bool)
	processedLinks := make(map[string]bool)
	for _, mount := range mounts {
		if processedPaths[mount.Path] {
			continue
		}
		processedPaths[mount.Path] = true

		linksForMount := d.getLinksForMount(mount.Path)
		soSymlinks, err := d.getDotSoSymlinks(mount.HostPath, mount.Path)
		if err != nil {
			d.logger.Warningf("Failed to get soname symlinks for %+v: %v", mount, err)
		}
		linksForMount = append(linksForMount, soSymlinks...)

		for _, link := range linksForMount {
			if processedLinks[link] {
				continue
			}
			processedLinks[link] = true
			links = append(links, link)
		}
	}

	if len(links) == 0 {
		return hooks, nil
	}

	createSymlinkHooks, err := d.hookCreator.Create("create-symlinks", links...).Hooks()
	if err != nil {
		return nil, fmt.Errorf("failed to create symlink hook: %v", err)
	}

	return append(hooks, createSymlinkHooks...), nil
}

// getLinksForMount maps the path to created links if any.
func (d additionalSymlinks) getLinksForMount(path string) []string {
	dir, filename := filepath.Split(path)
	switch {
	case d.isDriverLibrary("libcuda.so", filename):
		// XXX Many applications wrongly assume that libcuda.so exists (e.g. with dlopen).
		// create libcuda.so -> libcuda.so.1 symlink
		link := fmt.Sprintf("%s::%s", "libcuda.so.1", filepath.Join(dir, "libcuda.so"))
		return []string{link}
	case d.isDriverLibrary("libGLX_nvidia.so", filename):
		// XXX GLVND requires this symlink for indirect GLX support.
		// create libGLX_indirect.so.0 -> libGLX_nvidia.so.VERSION symlink
		link := fmt.Sprintf("%s::%s", filename, filepath.Join(dir, "libGLX_indirect.so.0"))
		return []string{link}
	case d.isDriverLibrary("libnvidia-opticalflow.so", filename):
		// XXX Fix missing symlink for libnvidia-opticalflow.so.
		// create libnvidia-opticalflow.so -> libnvidia-opticalflow.so.1 symlink
		link := fmt.Sprintf("%s::%s", "libnvidia-opticalflow.so.1", filepath.Join(dir, "libnvidia-opticalflow.so"))
		return []string{link}
	}
	return nil
}

// isDriverLibrary checks whether the specified filename is a specific driver library.
func (d additionalSymlinks) isDriverLibrary(libraryName string, filename string) bool {
	pattern := libraryName + "." + d.version
	match, _ := filepath.Match(pattern, filename)
	return match
}

func (d *additionalSymlinks) getDotSoSymlinks(hostLibraryPath string, libraryContainerPath string) ([]string, error) {
	hostLibraryDir := filepath.Dir(hostLibraryPath)
	containerLibraryDir, libraryName := filepath.Split(libraryContainerPath)
	if !d.isDriverLibrary("*", libraryName) {
		return nil, nil
	}

	soname, err := getSoname(hostLibraryPath)
	if err != nil {
		return nil, err
	}

	var soSymlinks []string
	// Create the SONAME -> libraryName symlink.
	// If the soname matches the library path, or the expected SONAME link does
	// not exist on the host, we do not create it in the container.
	if soname != libraryName && d.linkExistsInDir(hostLibraryDir, soname) {
		s := Symlink{
			target: libraryName,
			link:   filepath.Join(containerLibraryDir, soname),
		}
		soSymlinks = append(soSymlinks, s.String())
	}

	soTarget := soname
	if soTarget == "" {
		soTarget = libraryName
	}
	// Create the .so -> SONAME symlink.
	// If the .so link name matches the SONAME link, or the expected .so link
	// does not exist on the host, we do not create it in the container.
	if soLink := getSoLink(soTarget); soLink != soTarget && d.linkExistsInDir(hostLibraryDir, soLink) {
		s := Symlink{
			target: soTarget,
			link:   filepath.Join(containerLibraryDir, soLink),
		}
		soSymlinks = append(soSymlinks, s.String())
	}
	return soSymlinks, nil
}

func (d *additionalSymlinks) linkExistsInDir(dir string, link string) bool {
	if link == "" {
		return false
	}
	linkPath := filepath.Join(dir, link)
	exists, err := linkExists(linkPath)
	if err != nil {
		d.logger.Warningf("Failed to check symlink %q: %v", linkPath, err)
		return false
	}
	return exists
}

// linkExists returns true if the specified symlink exists.
// We use a function variable here to allow this to be overridden for testing.
var linkExists = func(linkPath string) (bool, error) {
	info, err := os.Lstat(linkPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// The linkPath is a symlink.
	if info.Mode()&os.ModeSymlink != 0 {
		return true, nil
	}

	return false, nil
}

// getSoname returns the soname for the specified library path.
// We use a function variable here to allow this to be overridden for testing.
var getSoname = func(libraryPath string) (string, error) {
	lib, err := elf.Open(libraryPath)
	if err != nil {
		return "", err
	}
	defer lib.Close()

	sonames, err := lib.DynString(elf.DT_SONAME)
	if err != nil {
		return "", err
	}
	if len(sonames) > 1 {
		return "", fmt.Errorf("multiple SONAMEs detected for %v: %v", libraryPath, sonames)
	}
	if len(sonames) == 0 {
		return filepath.Base(libraryPath), nil
	}
	return sonames[0], nil
}

// getSoLink returns the filename for the .so symlink that should point to the
// soname symlink for the specified library.
// If the soname / library name does not end in a `.so[.*]` then an empty string
// is returned.
func getSoLink(soname string) string {
	ext := filepath.Ext(soname)
	if ext == "" {
		return ""
	}
	if ext == ".so" {
		return soname
	}
	return getSoLink(strings.TrimSuffix(soname, ext))
}
