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

package ldcache

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/moby/sys/reexec"
	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

const (
	// ldsoconfdFilenamePattern specifies the pattern for the filename
	// in ld.so.conf.d that includes references to the specified directories.
	// The 00-nvcr prefix is chosen to ensure that these libraries have a
	// higher precedence than other libraries on the system, but lower than
	// the 00-cuda-compat that is included in some containers.
	ldsoconfdFilenamePattern = "00-nvcr-*.conf"

	reexecUpdateLdCacheCommandName = "reexec-update-ldcache"
)

type command struct {
	logger logger.Interface
}

type options struct {
	folders       cli.StringSlice
	ldconfigPath  string
	containerSpec string
}

func init() {
	reexec.Register(reexecUpdateLdCacheCommandName, updateLdCacheHandler)
	if reexec.Init() {
		os.Exit(0)
	}
}

// NewCommand constructs an update-ldcache command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build the update-ldcache command
func (m command) build() *cli.Command {
	cfg := options{}

	// Create the 'update-ldcache' command
	c := cli.Command{
		Name:  "update-ldcache",
		Usage: "Update ldcache in a container by running ldconfig",
		Before: func(c *cli.Context) error {
			return m.validateFlags(c, &cfg)
		},
		Action: func(c *cli.Context) error {
			return m.run(c, &cfg)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringSliceFlag{
			Name:        "folder",
			Usage:       "Specify a folder to add to /etc/ld.so.conf before updating the ld cache",
			Destination: &cfg.folders,
		},
		&cli.StringFlag{
			Name:        "ldconfig-path",
			Usage:       "Specify the path to the ldconfig program",
			Destination: &cfg.ldconfigPath,
			Value:       "/sbin/ldconfig",
		},
		&cli.StringFlag{
			Name:        "container-spec",
			Usage:       "Specify the path to the OCI container spec. If empty or '-' the spec will be read from STDIN",
			Destination: &cfg.containerSpec,
		},
	}

	return &c
}

func (m command) validateFlags(c *cli.Context, cfg *options) error {
	if cfg.ldconfigPath == "" {
		return errors.New("ldconfig-path must be specified")
	}
	return nil
}

func (m command) run(c *cli.Context, cfg *options) error {
	s, err := oci.LoadContainerState(cfg.containerSpec)
	if err != nil {
		return fmt.Errorf("failed to load container state: %v", err)
	}

	containerRootDir, err := s.GetContainerRoot()
	if err != nil || containerRootDir == "" || containerRootDir == "/" {
		return fmt.Errorf("failed to determined container root: %v", err)
	}

	args := []string{
		reexecUpdateLdCacheCommandName,
		strings.TrimPrefix(config.NormalizeLDConfigPath("@"+cfg.ldconfigPath), "@"),
		containerRootDir,
	}
	args = append(args, cfg.folders.Value()...)

	cmd := createReexecCommand(args)

	return cmd.Run()
}

// updateLdCacheHandler wraps updateLdCache with error handling.
func updateLdCacheHandler() {
	if err := updateLdCache(os.Args); err != nil {
		log.Printf("Error updating ldcache: %v", err)
		os.Exit(1)
	}
}

// updateLdCache is invoked from a reexec'd handler and provides namespace
// isolation for the operations performed by this hook.
// At the point where this is invoked, we are in a new mount namespace that is
// cloned from the parent.
//
// args[0] is the reexec initializer function name
// args[1] is the path of the ldconfig binary on the host
// args[2] is the container root directory
// The remaining args are folders that need to be added to the ldcache.
func updateLdCache(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("incorrect arguments: %v", args)
	}
	hostLdconfigPath := args[1]
	containerRootDirPath := args[2]

	// To prevent leaking the parent proc filesystem, we create a new proc mount
	// in the container root.
	if err := mountProc(containerRootDirPath); err != nil {
		return fmt.Errorf("error mounting /proc: %w", err)
	}

	// We mount the host ldconfig before we pivot root since host paths are not
	// visible after the pivot root operation.
	ldconfigPath, err := mountLdConfig(hostLdconfigPath, containerRootDirPath)
	if err != nil {
		return fmt.Errorf("error mounting host ldconfig: %w", err)
	}

	// We pivot to the container root for the new process, this further limits
	// access to the host.
	if err := pivotRoot(containerRootDirPath); err != nil {
		return fmt.Errorf("error running pivot_root: %w", err)
	}

	return runLdconfig(ldconfigPath, args[3:]...)
}

// runLdconfig runs the ldconfig binary and ensures that the specified directories
// are processed for the ldcache.
func runLdconfig(ldconfigPath string, directories ...string) error {
	args := []string{
		"ldconfig",
		// Explicitly specify using /etc/ld.so.conf since the host's ldconfig may
		// be configured to use a different config file by default.
		// Note that since we apply the `-r {{ .containerRootDir }}` argument, /etc/ld.so.conf is
		// in the container.
		"-f", "/etc/ld.so.conf",
	}

	containerRoot := containerRoot("/")

	if containerRoot.hasPath("/etc/ld.so.cache") {
		args = append(args, "-C", "/etc/ld.so.cache")
	} else {
		args = append(args, "-N")
	}

	if containerRoot.hasPath("/etc/ld.so.conf.d") {
		err := createLdsoconfdFile(ldsoconfdFilenamePattern, directories...)
		if err != nil {
			return fmt.Errorf("failed to update ld.so.conf.d: %w", err)
		}
	} else {
		args = append(args, directories...)
	}

	return SafeExec(ldconfigPath, args, nil)
}

// createLdsoconfdFile creates a file at /etc/ld.so.conf.d/.
// The file is created at /etc/ld.so.conf.d/{{ .pattern }} using `CreateTemp` and
// contains the specified directories on each line.
func createLdsoconfdFile(pattern string, dirs ...string) error {
	if len(dirs) == 0 {
		return nil
	}

	ldsoconfdDir := "/etc/ld.so.conf.d"
	if err := os.MkdirAll(ldsoconfdDir, 0755); err != nil {
		return fmt.Errorf("failed to create ld.so.conf.d: %w", err)
	}

	configFile, err := os.CreateTemp(ldsoconfdDir, pattern)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer func() {
		_ = configFile.Close()
	}()

	added := make(map[string]bool)
	for _, dir := range dirs {
		if added[dir] {
			continue
		}
		_, err = fmt.Fprintf(configFile, "%s\n", dir)
		if err != nil {
			return fmt.Errorf("failed to update config file: %w", err)
		}
		added[dir] = true
	}

	// The created file needs to be world readable for the cases where the container is run as a non-root user.
	if err := configFile.Chmod(0644); err != nil {
		return fmt.Errorf("failed to chmod config file: %w", err)
	}

	return nil
}
