/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

package image

import (
	"strings"
)

// VisibleDevices represents the devices selected in a container image
// through the NVIDIA_VISIBLE_DEVICES or other environment variables
type VisibleDevices interface {
	List() []string
	Has(string) bool
}

var _ VisibleDevices = (*all)(nil)
var _ VisibleDevices = (*none)(nil)
var _ VisibleDevices = (*void)(nil)
var _ VisibleDevices = (*devices)(nil)

// NewVisibleDevices creates a VisibleDevices based on the value of the
// specified envvar.
// If any of the envvars are the special values 'void', 'none', or 'all', these
// are treated as special cases and all other requests are ignored.
// The order of precedence for the special values is void > none > all.
func NewVisibleDevices(envvars ...string) VisibleDevices {
	hasSpecialDevice := make(map[string]bool)
	for _, envvar := range envvars {
		switch {
		case envvar == "" || envvar == "void":
			hasSpecialDevice["void"] = true
		case envvar == "none":
			hasSpecialDevice["none"] = true
		case envvar == "all":
			hasSpecialDevice["all"] = true
		}
	}

	switch {
	case hasSpecialDevice["void"]:
		return void{}
	case hasSpecialDevice["none"]:
		return none{}
	case hasSpecialDevice["all"]:
		return all{}
	}

	return newDevices(envvars...)
}

type all struct{}

// List returns ["all"] for all devices
func (a all) List() []string {
	return []string{"all"}
}

// Has for all devices is true for any id except the empty ID
func (a all) Has(id string) bool {
	return id != ""
}

type none struct{}

// List returns [""] for the none devices
func (n none) List() []string {
	return []string{""}
}

// Has for none devices is false for any id
func (n none) Has(id string) bool {
	return false
}

type void struct {
	none
}

// List returns nil for the void devices
func (v void) List() []string {
	return nil
}

type devices struct {
	list   []string
	lookup map[string]bool
}

func newDevices(idOrCommaSeparated ...string) devices {
	var list []string
	lookup := make(map[string]bool)
	for _, commaSeparated := range idOrCommaSeparated {
		for _, id := range strings.Split(commaSeparated, ",") {
			lookup[id] = true
			list = append(list, id)
		}
	}

	d := devices{
		list:   list,
		lookup: lookup,
	}
	return d
}

// List returns the list of requested devices
func (d devices) List() []string {
	return d.list
}

// Has checks whether the specified ID is in the set of requested devices
func (d devices) Has(id string) bool {
	if id == "" {
		return false
	}

	_, exist := d.lookup[id]
	return exist
}
