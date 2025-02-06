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
	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-cdi-hook/chmod"
	symlinks "github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-cdi-hook/create-symlinks"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-cdi-hook/cudacompat"
	ldcache "github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-cdi-hook/update-ldcache"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

// New creates the commands associated with supported CDI hooks.
// These are shared by the nvidia-cdi-hook and nvidia-ctk hook commands.
func New(logger logger.Interface) []*cli.Command {
	return []*cli.Command{
		ldcache.NewCommand(logger),
		symlinks.NewCommand(logger),
		chmod.NewCommand(logger),
		cudacompat.NewCommand(logger),
	}
}
