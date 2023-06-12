/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

package nvmodules

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

//go:generate moq -stub -out cmd_mock.go . cmder
type cmder interface {
	// Run executes the command and returns the stdout, stderr, and an error if any
	Run(string, ...string) error
}

type cmderLogger struct {
	logger.Interface
}

func (c *cmderLogger) Run(cmd string, args ...string) error {
	c.Infof("Running: %v %v", cmd, strings.Join(args, " "))
	return nil
}

type cmderExec struct{}

func (c *cmderExec) Run(cmd string, args ...string) error {
	if output, err := exec.Command(cmd, args...).CombinedOutput(); err != nil {
		return fmt.Errorf("%w; output=%v", err, string(output))
	}
	return nil
}
