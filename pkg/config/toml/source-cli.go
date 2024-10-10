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

package toml

import (
	"bytes"
	"fmt"
	"os/exec"
)

type tomlCliSource struct {
	command string
	args    []string
}

func (c tomlCliSource) Load() (*Tree, error) {
	//nolint:gosec  // Subprocess launched with a potential tainted input or cmd arguments
	cmd := exec.Command(c.command, c.args...)

	var outb bytes.Buffer
	var errb bytes.Buffer

	cmd.Stdout = &outb
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		// TODO: Log to stderr in case of failure
		return nil, fmt.Errorf("failed to run command %v %v: %w", c.command, c.args, err)
	}

	return LoadBytes(outb.Bytes())
}
