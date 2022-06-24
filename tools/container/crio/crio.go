/**
# Copyright (c) 2021-2022, NVIDIA CORPORATION.  All rights reserved.
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

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
)

const (
	defaultHooksDir     = "/usr/share/containers/oci/hooks.d"
	defaultHookFilename = "oci-nvidia-hook.json"
)

// options stores the configuration from the command line or environment variables
type options struct {
	hooksDir     string
	hookFilename string
	runtimeDir   string
}

func main() {
	options := options{}

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
	setup.Action = func(c *cli.Context) error {
		return Setup(c, &options)
	}
	setup.Before = func(c *cli.Context) error {
		return ParseArgs(c, &options)
	}

	// Create the 'cleanup' subcommand
	cleanup := cli.Command{}
	cleanup.Name = "cleanup"
	cleanup.Usage = "Remove the NVIDIA cri-o hook"
	cleanup.Action = func(c *cli.Context) error {
		return Cleanup(c, &options)
	}
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
			Destination: &options.hooksDir,
			EnvVars:     []string{"CRIO_HOOKS_DIR"},
			DefaultText: defaultHooksDir,
		},
		&cli.StringFlag{
			Name:        "hook-filename",
			Aliases:     []string{"f"},
			Usage:       "filename of the cri-o hook that will be created / removed in the hooks directory",
			Value:       defaultHookFilename,
			Destination: &options.hookFilename,
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
func Setup(c *cli.Context, o *options) error {
	log.Infof("Starting 'setup' for %v", c.App.Name)

	err := os.MkdirAll(o.hooksDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating hooks directory %v: %v", o.hooksDir, err)
	}

	hookPath := getHookPath(o.hooksDir, o.hookFilename)
	err = createHook(o.runtimeDir, hookPath)
	if err != nil {
		return fmt.Errorf("error creating hook: %v", err)
	}

	return nil
}

// Cleanup removes the specified prestart hook
func Cleanup(c *cli.Context, o *options) error {
	log.Infof("Starting 'cleanup' for %v", c.App.Name)

	hookPath := getHookPath(o.hooksDir, o.hookFilename)
	err := os.Remove(hookPath)
	if err != nil {
		return fmt.Errorf("error removing hook '%v': %v", hookPath, err)
	}

	return nil
}

// ParseArgs parses the command line arguments to the CLI
func ParseArgs(c *cli.Context, o *options) error {
	args := c.Args()

	log.Infof("Parsing arguments: %v", args.Slice())
	if c.NArg() != 1 {
		return fmt.Errorf("incorrect number of arguments")
	}
	o.runtimeDir = args.Get(0)
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
	err = encoder.Encode(generateOciHook(toolkitDir))
	if err != nil {
		return fmt.Errorf("error writing hook file '%v': %v", hookPath, err)
	}
	return nil
}

func getHookPath(hooksDir string, hookFilename string) string {
	return filepath.Join(hooksDir, hookFilename)
}

func generateOciHook(toolkitDir string) podmanHook {
	hookPath := filepath.Join(toolkitDir, config.NVIDIAContainerRuntimeHookExecutable)
	envPath := "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:" + toolkitDir
	always := true

	hook := podmanHook{
		Version: "1.0.0",
		Stages:  []string{"prestart"},
		Hook: specHook{
			Path: hookPath,
			Args: []string{filepath.Base(config.NVIDIAContainerRuntimeHookExecutable), "prestart"},
			Env:  []string{envPath},
		},
		When: When{
			Always:   &always,
			Commands: []string{".*"},
		},
	}
	return hook
}
