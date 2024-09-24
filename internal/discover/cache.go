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

package discover

import "sync"

type cache struct {
	d Discover

	sync.Mutex
	devices []Device
	hooks   []Hook
	mounts  []Mount
}

var _ Discover = (*cache)(nil)

// WithCache decorates the specified disoverer with a cache.
func WithCache(d Discover) Discover {
	if d == nil {
		return None{}
	}
	return &cache{d: d}
}

func (c *cache) Devices() ([]Device, error) {
	c.Lock()
	defer c.Unlock()

	if c.devices == nil {
		devices, err := c.d.Devices()
		if err != nil {
			return nil, err
		}
		c.devices = devices
	}
	return c.devices, nil
}

func (c *cache) Hooks() ([]Hook, error) {
	c.Lock()
	defer c.Unlock()

	if c.hooks == nil {
		hooks, err := c.d.Hooks()
		if err != nil {
			return nil, err
		}
		c.hooks = hooks
	}
	return c.hooks, nil
}

func (c *cache) Mounts() ([]Mount, error) {
	c.Lock()
	defer c.Unlock()

	if c.mounts == nil {
		mounts, err := c.d.Mounts()
		if err != nil {
			return nil, err
		}
		c.mounts = mounts
	}
	return c.mounts, nil
}
