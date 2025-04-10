//go:build !linux

/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

package ldcache

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/moby/sys/reexec"
)

func pivotRoot(newroot string) error {
	return fmt.Errorf("not supported")
}

func mountLdConfig(hostLdconfigPath string, containerRootDirPath string) (string, error) {
	return "", fmt.Errorf("not supported")
}

func mountProc(newroot string) error {
	return fmt.Errorf("not supported")
}

// createReexecCommand creates a command that can be used ot trigger the reexec
// initializer.
func createReexecCommand(args []string) *exec.Cmd {
	cmd := reexec.Command(args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd
}
