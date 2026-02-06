/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

package devices

import (
	"fmt"
	"os"
)

func SetAllForTest() func() {
	funcs := []func(){
		SetDeviceFromPathForTest(nil),
		SetAssertCharDeviceForTest(nil),
	}
	return func() {
		for _, f := range funcs {
			f()
		}
	}
}

func SetDeviceFromPathForTest(testFunc func(string, string) (*Device, error)) func() {
	current := deviceFromPathStub
	if testFunc != nil {
		deviceFromPathStub = testFunc
	} else {
		deviceFromPathStub = deviceFromPathTestDefault
	}
	return func() {
		deviceFromPathStub = current
	}
}

func SetAssertCharDeviceForTest(testFunc func(string) error) func() {
	current := assertCharDeviceStub
	if testFunc != nil {
		assertCharDeviceStub = testFunc
	} else {
		assertCharDeviceStub = assertCharDeviceTestDefault
	}
	return func() {
		assertCharDeviceStub = current
	}
}

func deviceFromPathTestDefault(path string, permissions string) (*Device, error) {
	return nil, fmt.Errorf("not implemented")
}

func assertCharDeviceTestDefault(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("error getting info for %v: %v", path, err)
	}

	if info.IsDir() {
		return fmt.Errorf("specified path '%v' is a directory", path)
	}

	return nil
}
