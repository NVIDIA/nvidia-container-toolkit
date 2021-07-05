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

package config

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// list represents a list of config updaters that are applied in order, with later
// configs having preference.
type list struct {
	logger  *log.Logger
	configs []configUpdater
}

var _ configUpdater = (*list)(nil)

func newListWithLogger(logger *log.Logger, ci ...configUpdater) configUpdater {
	c := list{
		logger:  logger,
		configs: ci,
	}
	return &c
}

func (c list) Update(cfg *Config) error {
	for i, u := range c.configs {
		c.logger.Debugf("Applying config %v: %v", i, u)
		err := u.Update(cfg)
		if err != nil {
			return fmt.Errorf("error applying config %v: %v", i, err)
		}
	}

	return nil
}
