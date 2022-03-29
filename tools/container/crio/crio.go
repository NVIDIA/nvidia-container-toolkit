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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	hooks "github.com/containers/podman/v4/pkg/hooks/1.0.0"
	rspec "github.com/opencontainers/runtime-spec/specs-go"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
)

const (
	defaultHooksDir     = "/usr/share/containers/oci/hooks.d"
	defaultHookFilename = "oci-nvidia-hook.json"
)

var hooksDirFlag string
var hookFilenameFlag string
var tooklitDirArg string

func main() {
	// Create the top-level CLI
	c := cli.NewApp()
	c.Name = "crio"
	c.Usage = "Update cri-o hooks to include the NVIDIA runtime hook"
	c.ArgsUsage = "<toolkit_dirname>"
	c.Version = "0.1.0"

	// Create the 'setup' subcommand
	setup := cli.Command{}
	setup.Name = "setup"
	setup.Usage = "Create the cri-o hook required to run NVIDIA GPU containers"
	setup.ArgsUsage = "<toolkit_dirname>"
	setup.Action = Setup
	setup.Before = ParseArgs

	// Create the 'cleanup' subcommand
	cleanup := cli.Command{}
	cleanup.Name = "cleanup"
	cleanup.Usage = "Remove the NVIDIA cri-o hook"
	cleanup.Action = Cleanup

	// Register the subcommands with the top-level CLI
	c.Commands = []*cli.Command{
		&setup,
		&cleanup,
	}

	// Setup common flags across both subcommands. All subcommands get the same
	// set of flags even if they don't use some of them. This is so that we
	// only require the user to specify one set of flags for both 'startup'
	// and 'cleanup' to simplify things.
	commonFlags := []cli.Flag{
		&cli.StringFlag{
			Name:        "hooks-dir",
			Aliases:     []string{"d"},
			Usage:       "path to the cri-o hooks directory",
			Value:       defaultHooksDir,
			Destination: &hooksDirFlag,
			EnvVars:     []string{"CRIO_HOOKS_DIR"},
			DefaultText: defaultHooksDir,
		},
		&cli.StringFlag{
			Name:        "hook-filename",
			Aliases:     []string{"f"},
			Usage:       "filename of the cri-o hook that will be created / removed in the hooks directory",
			Value:       defaultHookFilename,
			Destination: &hookFilenameFlag,
			EnvVars:     []string{"CRIO_HOOK_FILENAME"},
			DefaultText: defaultHookFilename,
		},
	}

	// Update the subcommand flags with the common subcommand flags
	setup.Flags = append([]cli.Flag{}, commonFlags...)
	cleanup.Flags = append([]cli.Flag{}, commonFlags...)

	// Run the top-level CLI
	if err := c.Run(os.Args); err != nil {
		log.Fatal(fmt.Errorf("error: %v", err))
	}
}

// Setup installs the prestart hook required to launch GPU-enabled containers
func Setup(c *cli.Context) error {
	log.Infof("Starting 'setup' for %v", c.App.Name)

	err := os.MkdirAll(hooksDirFlag, 0755)
	if err != nil {
		return fmt.Errorf("error creating hooks directory %v: %v", hooksDirFlag, err)
	}

	hookPath := getHookPath(hooksDirFlag, hookFilenameFlag)
	err = createHook(tooklitDirArg, hookPath)
	if err != nil {
		return fmt.Errorf("error creating hook: %v", err)
	}

	return nil
}

// Cleanup removes the specified prestart hook
func Cleanup(c *cli.Context) error {
	log.Infof("Starting 'cleanup' for %v", c.App.Name)

	hookPath := getHookPath(hooksDirFlag, hookFilenameFlag)
	err := os.Remove(hookPath)
	if err != nil {
		return fmt.Errorf("error removing hook '%v': %v", hookPath, err)
	}

	return nil
}

// ParseArgs parses the command line arguments to the CLI
func ParseArgs(c *cli.Context) error {
	args := c.Args()

	log.Infof("Parsing arguments: %v", args.Slice())
	if c.NArg() != 1 {
		return fmt.Errorf("incorrect number of arguments")
	}
	tooklitDirArg = args.Get(0)
	log.Infof("Successfully parsed arguments")

	return nil
}

func createHook(toolkitDir string, hookPath string) error {
	hook, err := os.Create(hookPath)
	if err != nil {
		return fmt.Errorf("error creating hook file '%v': %v", hookPath, err)
	}
	defer hook.Close()

	encoder := json.NewEncoder(hook)
	err = encoder.Encode(generateOciHook(tooklitDirArg))
	if err != nil {
		return fmt.Errorf("error writing hook file '%v': %v", hookPath, err)
	}
	return nil
}

func getHookPath(hooksDir string, hookFilename string) string {
	return filepath.Join(hooksDir, hookFilename)
}

func generateOciHook(toolkitDir string) hooks.Hook {
	hookPath := filepath.Join(toolkitDir, "nvidia-container-toolkit")
	envPath := "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:" + toolkitDir
	always := true

	hook := hooks.Hook{
		Version: "1.0.0",
		Stages:  []string{"prestart"},
		Hook: rspec.Hook{
			Path: hookPath,
			Args: []string{"nvidia-container-toolkit", "prestart"},
			Env:  []string{envPath},
		},
		When: hooks.When{
			Always:   &always,
			Commands: []string{".*"},
		},
	}
	return hook
}
