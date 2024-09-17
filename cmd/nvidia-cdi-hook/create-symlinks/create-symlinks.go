/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

package symlinks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/symlinks"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra/csv"
)

type command struct {
	logger logger.Interface
}

type config struct {
	hostRoot      string
	filenames     cli.StringSlice
	links         cli.StringSlice
	containerSpec string
}

// NewCommand constructs a hook command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build
func (m command) build() *cli.Command {
	cfg := config{}

	// Create the '' command
	c := cli.Command{
		Name:  "create-symlinks",
		Usage: "A hook to create symlinks in the container. This can be used to process CSV mount specs",
		Action: func(c *cli.Context) error {
			return m.run(c, &cfg)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "host-root",
			Usage:       "The root on the host filesystem to use to resolve symlinks",
			Destination: &cfg.hostRoot,
		},
		&cli.StringSliceFlag{
			Name:        "csv-filename",
			Usage:       "Specify a (CSV) filename to process",
			Destination: &cfg.filenames,
		},
		&cli.StringSliceFlag{
			Name:        "link",
			Usage:       "Specify a specific link to create. The link is specified as target::link",
			Destination: &cfg.links,
		},
		&cli.StringFlag{
			Name:        "container-spec",
			Usage:       "Specify the path to the OCI container spec. If empty or '-' the spec will be read from STDIN",
			Destination: &cfg.containerSpec,
		},
	}

	return &c
}

func (m command) run(c *cli.Context, cfg *config) error {
	s, err := oci.LoadContainerState(cfg.containerSpec)
	if err != nil {
		return fmt.Errorf("failed to load container state: %v", err)
	}

	containerRoot, err := s.GetContainerRoot()
	if err != nil {
		return fmt.Errorf("failed to determined container root: %v", err)
	}

	csvFiles := cfg.filenames.Value()

	chainLocator := lookup.NewSymlinkChainLocator(
		lookup.WithLogger(m.logger),
		lookup.WithRoot(cfg.hostRoot),
	)

	var candidates []string
	for _, file := range csvFiles {
		mountSpecs, err := csv.NewCSVFileParser(m.logger, file).Parse()
		if err != nil {
			m.logger.Debugf("Skipping CSV file %v: %v", file, err)
			continue
		}

		for _, ms := range mountSpecs {
			if ms.Type != csv.MountSpecSym {
				continue
			}
			targets, err := chainLocator.Locate(ms.Path)
			if err != nil {
				m.logger.Warningf("Failed to locate symlink %v", ms.Path)
			}
			candidates = append(candidates, targets...)
		}
	}

	created := make(map[string]bool)
	// candidates is a list of absolute paths to symlinks in a chain, or the final target of the chain.
	for _, candidate := range candidates {
		target, err := symlinks.Resolve(candidate)
		if err != nil {
			m.logger.Debugf("Skipping invalid link: %v", err)
			continue
		} else if target == candidate {
			m.logger.Debugf("%v is not a symlink", candidate)
			continue
		}

		err = m.createLink(created, cfg.hostRoot, containerRoot, target, candidate)
		if err != nil {
			m.logger.Warningf("Failed to create link %v: %v", []string{target, candidate}, err)
		}
	}

	links := cfg.links.Value()
	for _, l := range links {
		parts := strings.Split(l, "::")
		if len(parts) != 2 {
			return fmt.Errorf("invalid symlink specification %v", l)
		}

		err := m.createLink(created, cfg.hostRoot, containerRoot, parts[0], parts[1])
		if err != nil {
			return fmt.Errorf("failed to create link %v: %w", parts, err)
		}
	}

	return nil

}

func (m command) createLink(created map[string]bool, hostRoot string, containerRoot string, target string, link string) error {
	linkPath, err := changeRoot(hostRoot, containerRoot, link)
	if err != nil {
		return fmt.Errorf("failed to resolve path for link %v relative to %v: %w", link, containerRoot, err)
	}
	if created[linkPath] {
		m.logger.Debugf("Link %v already created", linkPath)
		return nil
	}
	if err := assertPathInRoot(containerRoot, linkPath); err != nil {
		return err
	}

	targetPath, err := changeRoot(hostRoot, "/", target)
	if err != nil {
		return fmt.Errorf("failed to resolve path for target %v relative to %v: %w", target, "/", err)
	}

	parent := containerRoot
	if !filepath.IsAbs(targetPath) {
		parent = filepath.Dir(linkPath)
	}
	if err := assertPathInRoot(containerRoot, filepath.Join(parent, targetPath)); err != nil {
		return err
	}

	m.logger.Infof("Symlinking %v to %v", linkPath, targetPath)
	err = os.MkdirAll(filepath.Dir(linkPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	err = os.Symlink(target, linkPath)
	if err != nil {
		return fmt.Errorf("failed to create symlink: %v", err)
	}

	return nil
}

// assertPathInRoot ensures that the specified path is a subpath of root.
// This includes the resolution of relative paths and symlinks.
func assertPathInRoot(root string, path string) error {
	resolved, err := tryResolveLink(filepath.Clean(path))
	if err != nil {
		return err
	}
	if !strings.HasPrefix(resolved, root) {
		return fmt.Errorf("path %v resolves to %v which is outside the root %v", path, resolved, root)
	}
	return nil
}

// tryResolveLink resolves the specified path following symlinks.
// If the path does not exist, recursively resolve the parent of the path until
// one resolves successfully.
//
// For example, assuming that we call this with a path
//
//	/container-root/foo/bar/baz.so
//
// If this path exists, then the symlink evaluation will succeed and the target
// will be returned.
//
// Assuming that /container-root/foo exists, but /container-root/foo/bar does
// not, the first call to EvalSymlinks with argument
// /container-root/foo/bar/baz.so will fail resulting in a recursive call to
// this function with an argument /container-root/foo/bar. Since this path also
// doesn't exist this will result in an additional recursive call with an
// argument /container-root/foo. Here EvalSymlinks will succeed and the strings
// /bar and /baz.so will be appended to the result as we return along the
// recursive call stack.
//
// If none of the parents of the path (with the exception of /) exist the path
// will be returned as is with no error.
func tryResolveLink(path string) (string, error) {
	if path == "" || path == "/" {
		return path, nil
	}

	resolved, err := filepath.EvalSymlinks(path)
	if os.IsNotExist(err) {
		resolvedParent, err := tryResolveLink(filepath.Dir(path))
		if err != nil {
			return "", err
		}
		resolved := filepath.Join(resolvedParent, filepath.Base(path))
		return resolved, nil
	}

	return resolved, err
}

func changeRoot(current string, new string, path string) (string, error) {
	if !filepath.IsAbs(path) {
		return path, nil
	}

	relative := path
	if current != "" {
		r, err := filepath.Rel(current, path)
		if err != nil {
			return "", err
		}
		relative = r
	}

	return filepath.Join(new, relative), nil
}

// Locate returns the link target of the specified filename or an empty slice if the
// specified filename is not a symlink.
func (m command) Locate(filename string) ([]string, error) {
	info, err := os.Lstat(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", info)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		m.logger.Debugf("%v is not a symlink", filename)
		return nil, nil
	}

	target, err := os.Readlink(filename)
	if err != nil {
		return nil, fmt.Errorf("error checking symlink: %v", err)
	}

	m.logger.Debugf("Resolved link: '%v' => '%v'", filename, target)

	return []string{target}, nil
}
