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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/moby/sys/symlink"
	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/symlinks"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

type command struct {
	logger logger.Interface
}

type config struct {
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

// build creates the create-symlink command.
func (m command) build() *cli.Command {
	cfg := config{}

	c := cli.Command{
		Name:  "create-symlinks",
		Usage: "A hook to create symlinks in the container.",
		Action: func(c *cli.Context) error {
			return m.run(c, &cfg)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringSliceFlag{
			Name:        "link",
			Usage:       "Specify a specific link to create. The link is specified as target::link. If the link exists in the container root, it is removed.",
			Destination: &cfg.links,
		},
		// The following flags are testing-only flags.
		&cli.StringFlag{
			Name:        "container-spec",
			Usage:       "Specify the path to the OCI container spec. If empty or '-' the spec will be read from STDIN. This is only intended for testing.",
			Destination: &cfg.containerSpec,
			Hidden:      true,
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

	created := make(map[string]bool)
	for _, l := range cfg.links.Value() {
		if created[l] {
			m.logger.Debugf("Link %v already processed", l)
			continue
		}
		parts := strings.Split(l, "::")
		if len(parts) != 2 {
			return fmt.Errorf("invalid symlink specification %v", l)
		}

		err := m.createLink(containerRoot, parts[0], parts[1])
		if err != nil {
			return fmt.Errorf("failed to create link %v: %w", parts, err)
		}
		created[l] = true
	}
	return nil
}

// createLink creates a symbolic link in the specified container root.
// This is equivalent to:
//
//	chroot {{ .containerRoot }} ln -f -s {{ .target }} {{ .link }}
//
// If the specified link already exists and points to the same target, this
// operation is a no-op.
// If a file exists at the link path or the link points to a different target
// this file is removed before creating the link.
//
// Note that if the link path resolves to an absolute path oudside of the
// specified root, this is treated as an absolute path in this root.
func (m command) createLink(containerRoot string, targetPath string, link string) error {
	linkPath := filepath.Join(containerRoot, link)

	exists, err := linkExists(targetPath, linkPath)
	if err != nil {
		return fmt.Errorf("failed to check if link exists: %w", err)
	}
	if exists {
		m.logger.Debugf("Link %s already exists", linkPath)
		return nil
	}

	// We resolve the parent of the symlink that we're creating in the container root.
	// If we resolve the full link path, an existing link at the location itself
	// is also resolved here and we are unable to force create the link.
	resolvedLinkParent, err := symlink.FollowSymlinkInScope(filepath.Dir(linkPath), containerRoot)
	if err != nil {
		return fmt.Errorf("failed to follow path for link %v relative to %v: %w", link, containerRoot, err)
	}
	resolvedLinkPath := filepath.Join(resolvedLinkParent, filepath.Base(linkPath))

	m.logger.Infof("Symlinking %v to %v", resolvedLinkPath, targetPath)
	err = os.MkdirAll(filepath.Dir(resolvedLinkPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	err = symlinks.ForceCreate(targetPath, resolvedLinkPath)
	if err != nil {
		return fmt.Errorf("failed to create symlink: %v", err)
	}

	return nil
}

// linkExists checks whether the specified link exists.
// A link exists if the path exists, is a symlink, and points to the specified target.
func linkExists(target string, link string) (bool, error) {
	currentTarget, err := symlinks.Resolve(link)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to resolve existing symlink %s: %w", link, err)
	}
	if currentTarget == target {
		return true, nil
	}
	return false, nil
}
