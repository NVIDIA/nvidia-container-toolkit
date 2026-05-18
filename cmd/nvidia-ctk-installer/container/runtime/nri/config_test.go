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
	"strings"
	"testing"
)

type stubRunner struct{}

func (stubRunner) Start(context.Context, RegistrationConfig) error { return nil }
func (stubRunner) Stop()                                           {}

func TestValidateEntries(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		entries []Entry
		wantErr string
	}{
		{
			name:    "empty",
			entries: nil,
		},
		{
			name: "valid single",
			entries: []Entry{
				{Name: "management", PluginRunner: stubRunner{}, Config: RegistrationConfig{Index: 10}},
			},
		},
		{
			name: "valid multiple",
			entries: []Entry{
				{Name: "management", PluginRunner: stubRunner{}, Config: RegistrationConfig{Index: 10}},
				{Name: "other", PluginRunner: stubRunner{}, Config: RegistrationConfig{Index: 11}},
			},
		},
		{
			name: "missing name",
			entries: []Entry{
				{PluginRunner: stubRunner{}, Config: RegistrationConfig{Index: 10}},
			},
			wantErr: "name must be specified",
		},
		{
			name: "missing runner",
			entries: []Entry{
				{Name: "management", Config: RegistrationConfig{Index: 10}},
			},
			wantErr: "implementation must be specified",
		},
		{
			name: "index out of range",
			entries: []Entry{
				{Name: "management", PluginRunner: stubRunner{}, Config: RegistrationConfig{Index: 100}},
			},
			wantErr: "index must be in the range",
		},
		{
			name: "duplicate index",
			entries: []Entry{
				{Name: "first", PluginRunner: stubRunner{}, Config: RegistrationConfig{Index: 10}},
				{Name: "second", PluginRunner: stubRunner{}, Config: RegistrationConfig{Index: 10}},
			},
			wantErr: "duplicate plugin index",
		},
		{
			name: "duplicate name",
			entries: []Entry{
				{Name: "management", PluginRunner: stubRunner{}, Config: RegistrationConfig{Index: 10}},
				{Name: "management", PluginRunner: stubRunner{}, Config: RegistrationConfig{Index: 11}},
			},
			wantErr: "duplicate plugin name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateEntries(tc.entries)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}
