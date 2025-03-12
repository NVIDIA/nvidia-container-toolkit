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

package nvdevices

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

//go:generate moq -rm -fmt=goimports -stub -out mknod_mock.go . mknoder
type mknoder interface {
	Mknode(string, int, int) error
}

type mknodLogger struct {
	logger.Interface
}

func (m *mknodLogger) Mknode(path string, major, minor int) error {
	m.Infof("Running: mknod --mode=0666 %s c %d %d", path, major, minor)
	return nil
}

type mknodUnix struct {
	logger logger.Interface
}

func (m *mknodUnix) Mknode(path string, major, minor int) error {
	// TODO: Ensure that the existing device node has the correct properties.
	if _, err := os.Stat(path); err == nil {
		m.logger.Infof("Skipping: %s already exists", path)
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat %s: %v", path, err)
	}

	err := unix.Mknod(path, unix.S_IFCHR, int(unix.Mkdev(uint32(major), uint32(minor))))
	if err != nil {
		return err
	}
	return unix.Chmod(path, 0666)
}
