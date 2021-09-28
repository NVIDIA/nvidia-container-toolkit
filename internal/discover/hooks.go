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

package discover

import (
	log "github.com/sirupsen/logrus"
)

type hooks struct {
	None
	logger *log.Logger
}

var _ Discover = (*hooks)(nil)

// NewHooks creates a discoverer for linux containers
func NewHooks() Discover {
	return NewHooksWithLogger(log.StandardLogger())
}

// NewHooksWithLogger creates a discoverer as with NewHooks with the specified logger
func NewHooksWithLogger(logger *log.Logger) Discover {
	h := hooks{
		logger: logger,
	}

	return &h
}

func (h hooks) Hooks() ([]Hook, error) {
	var hooks []Hook

	hooks = append(hooks, newLdconfigHook())

	return hooks, nil
}

func newLdconfigHook() Hook {
	const rootPattern = "@Root.Path@"

	h := Hook{
		Path: "/sbin/ldconfig",
		Args: []string{
			// TODO: Testing seems to indicate that this is -v flag is required
			"-v",
			"-r", rootPattern,
		},
		// TODO: CreateContainer hooks were only added to a later OCI spec version
		// We will have to find a way to deal with OCI versions before 1.0.2
		HookName: "create-container",
		Labels: map[string]string{
			"min-oci-version": "1.0.2",
		},
	}

	return h
}
