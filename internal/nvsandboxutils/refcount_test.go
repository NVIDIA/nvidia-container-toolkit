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

package nvsandboxutils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRefcount(t *testing.T) {
	testCases := []struct {
		description      string
		workload         func(r *refcount)
		expectedRefcount refcount
	}{
		{
			description:      "No inc or dec",
			workload:         func(r *refcount) {},
			expectedRefcount: refcount(0),
		},
		{
			description: "Single inc, no error",
			workload: func(r *refcount) {
				r.IncOnNoError(nil)
			},
			expectedRefcount: refcount(1),
		},
		{
			description: "Single inc, with error",
			workload: func(r *refcount) {
				r.IncOnNoError(errors.New(""))
			},
			expectedRefcount: refcount(0),
		},
		{
			description: "Double inc, no error",
			workload: func(r *refcount) {
				r.IncOnNoError(nil)
				r.IncOnNoError(nil)
			},
			expectedRefcount: refcount(2),
		},
		{
			description: "Double inc, one with error",
			workload: func(r *refcount) {
				r.IncOnNoError(nil)
				r.IncOnNoError(errors.New(""))
			},
			expectedRefcount: refcount(1),
		},
		{
			description: "Single dec, no error",
			workload: func(r *refcount) {
				r.DecOnNoError(nil)
			},
			expectedRefcount: refcount(0),
		},
		{
			description: "Single dec, with error",
			workload: func(r *refcount) {
				r.DecOnNoError(errors.New(""))
			},
			expectedRefcount: refcount(0),
		},
		{
			description: "Single inc, single dec, no errors",
			workload: func(r *refcount) {
				r.IncOnNoError(nil)
				r.DecOnNoError(nil)
			},
			expectedRefcount: refcount(0),
		},
		{
			description: "Double inc, Double dec, no errors",
			workload: func(r *refcount) {
				r.IncOnNoError(nil)
				r.IncOnNoError(nil)
				r.DecOnNoError(nil)
				r.DecOnNoError(nil)
			},
			expectedRefcount: refcount(0),
		},
		{
			description: "Double inc, Double dec, one inc error",
			workload: func(r *refcount) {
				r.IncOnNoError(nil)
				r.IncOnNoError(errors.New(""))
				r.DecOnNoError(nil)
				r.DecOnNoError(nil)
			},
			expectedRefcount: refcount(0),
		},
		{
			description: "Double inc, Double dec, one dec error",
			workload: func(r *refcount) {
				r.IncOnNoError(nil)
				r.IncOnNoError(nil)
				r.DecOnNoError(nil)
				r.DecOnNoError(errors.New(""))
			},
			expectedRefcount: refcount(1),
		},
		{
			description: "Double inc, Tripple dec, one dec error early on",
			workload: func(r *refcount) {
				r.IncOnNoError(nil)
				r.IncOnNoError(nil)
				r.DecOnNoError(errors.New(""))
				r.DecOnNoError(nil)
				r.DecOnNoError(nil)
			},
			expectedRefcount: refcount(0),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			var r refcount
			tc.workload(&r)
			require.Equal(t, tc.expectedRefcount, r)
		})
	}
}
