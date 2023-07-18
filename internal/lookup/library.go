/*
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

package lookup

import (
	"fmt"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/ldcache"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

type library struct {
	logger  logger.Interface
	symlink Locator
	cache   ldcache.LDCache
}

var _ Locator = (*library)(nil)

// NewLibraryLocator creates a library locator using the specified logger.
func NewLibraryLocator(logger logger.Interface, root string) (Locator, error) {
	cache, err := ldcache.New(logger, root)
	if err != nil {
		return nil, fmt.Errorf("error loading ldcache: %v", err)
	}

	l := library{
		logger:  logger,
		symlink: NewSymlinkLocator(WithLogger(logger), WithRoot(root)),
		cache:   cache,
	}

	return &l, nil
}

// Locate finds the specified libraryname.
// If the input is a library name, the ldcache is searched otherwise the
// provided path is resolved as a symlink.
func (l library) Locate(libname string) ([]string, error) {
	if strings.Contains(libname, "/") {
		return l.symlink.Locate(libname)
	}

	paths32, paths64 := l.cache.Lookup(libname)
	if len(paths32) > 0 {
		l.logger.Warningf("Ignoring 32-bit libraries for %v: %v", libname, paths32)
	}

	if len(paths64) == 0 {
		return nil, fmt.Errorf("64-bit library %v not found", libname)
	}

	return paths64, nil
}
