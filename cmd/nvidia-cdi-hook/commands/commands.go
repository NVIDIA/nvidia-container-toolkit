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

package commands

import (
	"context"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-cdi-hook/chmod"
	symlinks "github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-cdi-hook/create-symlinks"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-cdi-hook/cudacompat"
	disabledevicenodemodification "github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-cdi-hook/disable-device-node-modification"
	ldcache "github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-cdi-hook/update-ldcache"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

// ConfigureCDIHookCommand configures a base command with supported CDI hooks
// and error handling for unsupported hooks.
// This allows the same command to be used for the nvidia-cdi-hook and
// nvidia-ctk hook commands.
func ConfigureCDIHookCommand(logger logger.Interface, base *cli.Command) *cli.Command {
	// We set the default action for the command to issue a warning and exit
	// with no error.
	// This means that if an unsupported hook is run, a container will not fail
	// to launch. An unsupported hook could be the result of a CDI specification
	// referring to a new hook that is not yet supported by an older NVIDIA
	// Container Toolkit version or a hook that has been removed in newer
	// version.
	base.Action = func(ctx context.Context, cmd *cli.Command) error {
		return issueUnsupportedHookWarning(logger, cmd)
	}
	// CommandNotFound is triggered when an unrecognised (sub)command is detected.
	// We assume that an unrecognised (sub)command represents an unsupported hook
	// (usually a hook that was added or removed)
	base.CommandNotFound = func(ctx context.Context, cmd *cli.Command, commandName string) {
		_ = issueUnsupportedHookWarning(logger, cmd)
	}
	// OnUsageError is triggered when an unexpected flag is detected.
	// We check the invoked command to determine whether it is an expected
	// hook, and assume that this is an unsupported hook otherwise.
	base.OnUsageError = func(ctx context.Context, cmd *cli.Command, err error, isSubcommand bool) error {
		// If this is not an error that comes from parsing an unrecognised flag,
		// return it as is.
		if !strings.HasPrefix(err.Error(), "flag provided but not defined: -") {
			return err
		}

		// If the first argument is a recognised command, we return the error as
		// is since it represents an incorrect argument to the specific hook.
		var subcommandName string
		for _, arg := range cmd.Args().Slice() {
			if strings.HasPrefix(arg, "-") {
				continue
			}
			subcommandName = arg
			break
		}
		// If a subcommand is detected and is a recognised subcommand, we return
		// the error as is.
		if subcommandName != "" && cmd.Command(subcommandName) != nil {
			return err
		}

		// At this point either no args have been supplied or the (sub)command
		// (first arg) is not regognised.
		// We issue a warning and returun nil.
		return issueUnsupportedHookWarning(logger, cmd)
	}

	// Define the supported hooks.
	base.Commands = []*cli.Command{
		ldcache.NewCommand(logger),
		symlinks.NewCommand(logger),
		chmod.NewCommand(logger),
		cudacompat.NewCommand(logger),
		disabledevicenodemodification.NewCommand(logger),
		{
			Name:   "noop",
			Usage:  "The noop hook performs no actions and is only added to facilitate basic testing of the CLI",
			Hidden: true,
			Action: func(_ context.Context, _ *cli.Command) error {
				return nil
			},
		},
	}

	return base
}

// issueUnsupportedHookWarning logs a warning that no hook or an unsupported
// hook has been specified.
// This happens if a subcommand is provided that does not match one of the
// subcommands that has been explicitly specified.
func issueUnsupportedHookWarning(logger logger.Interface, c *cli.Command) error {
	args := c.Args().Slice()
	if len(args) == 0 {
		logger.Warningf("No CDI hook specified")
	} else {
		logger.Warningf("Unsupported CDI hook: %v", args[0])
	}
	return nil
}
