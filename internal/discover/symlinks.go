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
	"fmt"
	"path/filepath"
)

type additionalSymlinks struct {
	Discover
	version           string
	nvidiaCDIHookPath string
}

// WithDriverDotSoSymlinks decorates the provided discoverer.
// A hook is added that checks for specific driver symlinks that need to be created.
func WithDriverDotSoSymlinks(mounts Discover, version string, nvidiaCDIHookPath string) Discover {
	if version == "" {
		version = "*.*"
	}
	return &additionalSymlinks{
		Discover:          mounts,
		nvidiaCDIHookPath: nvidiaCDIHookPath,
		version:           version,
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

		for _, link := range d.getLinksForMount(mount.Path) {
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

	hook := CreateCreateSymlinkHook(d.nvidiaCDIHookPath, links).(Hook)
	return append(hooks, hook), nil
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
