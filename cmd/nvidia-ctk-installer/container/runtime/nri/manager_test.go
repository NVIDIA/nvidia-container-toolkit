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

package nri

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/sirupsen/logrus"
)

type countingRunner struct {
	starts atomic.Int32
}

func (c *countingRunner) Start(context.Context, RegistrationConfig) error {
	c.starts.Add(1)
	return nil
}

func (c *countingRunner) Stop() {}

func TestManagerStartMultipleEntries(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	first := &countingRunner{}
	second := &countingRunner{}

	manager := NewManager(logger)
	entries := []Entry{
		{Name: "first", PluginRunner: first, Config: RegistrationConfig{Index: 10}},
		{Name: "second", PluginRunner: second, Config: RegistrationConfig{Index: 11}},
	}
	if err := manager.Start(context.Background(), entries); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if first.starts.Load() != 1 || second.starts.Load() != 1 {
		t.Fatalf("expected each runner to be started once")
	}
	manager.Stop()
}
