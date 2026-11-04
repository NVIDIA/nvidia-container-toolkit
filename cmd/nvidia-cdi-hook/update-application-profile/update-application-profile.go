/**
# SPDX-FileCopyrightText: Copyright (c) NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

package updateapplicationprofile

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"syscall"

	"github.com/google/uuid"
	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

const (
	applicationProfileDir  = "etc/nvidia/nvidia-application-profiles-rc.d"
	applicationProfileFile = applicationProfileDir + "/10-container.conf"
)

// nvidiaGPUDeviceNode matches the device node names for physical NVIDIA GPUs,
// e.g. nvidia0, nvidia12.
var nvidiaGPUDeviceNode = regexp.MustCompile(`^nvidia[0-9]+$`)

// A deviceNode describes a single entry discovered under the container's /dev
// directory. minor is only meaningful when isCharDevice is true.
type deviceNode struct {
	name         string
	isCharDevice bool
	minor        int
}

// selectGPUMinors returns the minor numbers of the entries that are physical
// NVIDIA GPU device nodes (/dev/nvidiaN char devices).
func selectGPUMinors(nodes []deviceNode) []int {
	var minors []int
	for _, n := range nodes {
		if n.isCharDevice && nvidiaGPUDeviceNode.MatchString(n.name) {
			minors = append(minors, n.minor)
		}
	}
	return minors
}

type options struct {
	containerSpec string
}

// NewCommand constructs an update-application-profile subcommand with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	cfg := options{}

	c := cli.Command{
		Name:  "update-application-profile",
		Usage: `Updates driver settings through "application profiles". Currently, this hook sets EGLVisibleDGPUDevices to restrict EGL/Vulkan GPU visibility inside the container`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return run(ctx, cmd, &cfg, logger)
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "container-spec",
				Hidden:      true,
				Usage:       "Specify the path to the OCI container spec. If empty or '-' the spec will be read from STDIN",
				Destination: &cfg.containerSpec,
			},
		},
	}

	return &c
}

func run(_ context.Context, _ *cli.Command, cfg *options, logger logger.Interface) error {
	s, err := oci.LoadContainerState(cfg.containerSpec)
	if err != nil {
		return fmt.Errorf("failed to load container state: %w", err)
	}

	containerRootDirPath, err := s.GetContainerRoot()
	if err != nil {
		return fmt.Errorf("failed to determine container root: %w", err)
	}

	containerRoot, err := os.OpenRoot(containerRootDirPath)
	if err != nil {
		return fmt.Errorf("failed to open root: %w", err)
	}
	defer containerRoot.Close()

	nodes, err := listDeviceNodes(containerRoot)
	if err != nil {
		return fmt.Errorf("failed to enumerate device nodes in container: %w", err)
	}

	minors := selectGPUMinors(nodes)
	if len(minors) == 0 {
		return nil
	}

	var mask uint64
	for _, minor := range minors {
		if minor < 0 || minor >= 64 {
			logger.Warningf("dropping invalid minor number %d: must be in range [0, 63]", minor)
			continue
		}
		mask |= uint64(1) << uint(minor)
	}

	if mask == 0 {
		return nil
	}

	if err = func() error {
		if err := containerRoot.MkdirAll(applicationProfileDir, 0555); err != nil {
			return fmt.Errorf("failed to create application profiles directory: %w", err)
		}
		if err := writeApplicationProfile(containerRoot, buildApplicationProfileConfig(mask)); err != nil {
			return fmt.Errorf("failed to write application profile: %w", err)
		}
		return nil
	}(); err != nil {
		if !isIgnorableWriteError(err) {
			return err
		}
		logger.Warningf("Ignoring error writing application profile: %v", err)
	}

	return nil
}

func isIgnorableWriteError(err error) bool {
	return errors.Is(err, syscall.EROFS) || errors.Is(err, syscall.EBUSY)
}

func writeApplicationProfile(containerRoot *os.Root, contents []byte) (rerr error) {
	profileDir, err := containerRoot.OpenRoot(applicationProfileDir)
	if err != nil {
		return err
	}
	defer profileDir.Close()

	profileFileName := filepath.Base(applicationProfileFile)
	tmpFileName := profileFileName + "-" + uuid.NewString()

	tmpFile, err := profileDir.OpenFile(tmpFileName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0444)
	if err != nil {
		return err
	}
	defer func() {
		if rerr != nil {
			_ = profileDir.Remove(tmpFileName)
		}
	}()

	if _, err := tmpFile.Write(contents); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	return profileDir.Rename(tmpFileName, profileFileName)
}

// buildApplicationProfileConfig returns the contents of the NVIDIA driver application profile.
func buildApplicationProfileConfig(mask uint64) []byte {
	return fmt.Appendf([]byte{},
		`{"profiles":[{"name":"_container_","settings":["EGLVisibleDGPUDevices",0x%x]}],"rules":[{"pattern":[],"profile":"_container_"}]}`,
		mask,
	)
}
