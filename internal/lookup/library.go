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

	log "github.com/sirupsen/logrus"
	"gitlab.com/nvidia/cloud-native/container-toolkit/internal/ldcache"
)

type library struct {
	logger *log.Logger
	cache  *ldcache.LDCache
}

var _ Locator = (*library)(nil)

// NewLibraryLocatorWithLogger creates a library locator using the standard logger
func NewLibraryLocator(root string) (Locator, error) {
	return NewLibraryLocatorWithLogger(log.StandardLogger(), root)
}

// NewLibraryLocatorWithLogger creates a library locator using the specified logger.
func NewLibraryLocatorWithLogger(logger *log.Logger, root string) (Locator, error) {
	logger.Infof("Reading ldcache at %v", root)
	cache, err := ldcache.NewLDCacheWithLogger(logger, root)
	if err != nil {
		return nil, fmt.Errorf("error loading ldcache: %v", err)
	}

	l := library{
		logger: logger,
		cache:  cache,
	}

	return &l, nil
}

func (l library) Locate(libname string) ([]string, error) {
	paths32, paths64 := l.cache.Lookup(libname)
	if len(paths32) > 0 {
		l.logger.Warnf("Ignoring 32-bit libraries for %v: %v", libname, paths32)
	}

	if len(paths64) == 0 {
		return nil, fmt.Errorf("64-bit library %v not found", libname)
	}

	return paths64, nil
}
