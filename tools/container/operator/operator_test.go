/**
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

package operator

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptions(t *testing.T) {
	testCases := []struct {
		setAsDefault           bool
		nvidiaRuntimeName      string
		root                   string
		expectedDefaultRuntime string
		expectedRuntimes       Runtimes
	}{
		{
			expectedRuntimes: Runtimes{
				"nvidia": Runtime{
					name: "nvidia",
					Path: "/usr/bin/nvidia-container-runtime",
				},
				"nvidia-cdi": Runtime{
					name: "nvidia-cdi",
					Path: "/usr/bin/nvidia-container-runtime.cdi",
				},
				"nvidia-legacy": Runtime{
					name: "nvidia-legacy",
					Path: "/usr/bin/nvidia-container-runtime.legacy",
				},
			},
		},
		{
			setAsDefault:           true,
			expectedDefaultRuntime: "nvidia",
			expectedRuntimes: Runtimes{
				"nvidia": Runtime{
					name:         "nvidia",
					Path:         "/usr/bin/nvidia-container-runtime",
					SetAsDefault: true,
				},
				"nvidia-cdi": Runtime{
					name: "nvidia-cdi",
					Path: "/usr/bin/nvidia-container-runtime.cdi",
				},
				"nvidia-legacy": Runtime{
					name: "nvidia-legacy",
					Path: "/usr/bin/nvidia-container-runtime.legacy",
				},
			},
		},
		{
			setAsDefault:           true,
			nvidiaRuntimeName:      "nvidia",
			expectedDefaultRuntime: "nvidia",
			expectedRuntimes: Runtimes{
				"nvidia": Runtime{
					name:         "nvidia",
					Path:         "/usr/bin/nvidia-container-runtime",
					SetAsDefault: true,
				},
				"nvidia-cdi": Runtime{
					name: "nvidia-cdi",
					Path: "/usr/bin/nvidia-container-runtime.cdi",
				},
				"nvidia-legacy": Runtime{
					name: "nvidia-legacy",
					Path: "/usr/bin/nvidia-container-runtime.legacy",
				},
			},
		},
		{
			setAsDefault:           true,
			nvidiaRuntimeName:      "NAME",
			expectedDefaultRuntime: "NAME",
			expectedRuntimes: Runtimes{
				"NAME": Runtime{
					name:         "NAME",
					Path:         "/usr/bin/nvidia-container-runtime",
					SetAsDefault: true,
				},
				"nvidia-cdi": Runtime{
					name: "nvidia-cdi",
					Path: "/usr/bin/nvidia-container-runtime.cdi",
				},
				"nvidia-legacy": Runtime{
					name: "nvidia-legacy",
					Path: "/usr/bin/nvidia-container-runtime.legacy",
				},
			},
		},
		{
			setAsDefault:      false,
			nvidiaRuntimeName: "NAME",
			expectedRuntimes: Runtimes{
				"NAME": Runtime{
					name: "NAME",
					Path: "/usr/bin/nvidia-container-runtime",
				},
				"nvidia-cdi": Runtime{
					name: "nvidia-cdi",
					Path: "/usr/bin/nvidia-container-runtime.cdi",
				},
				"nvidia-legacy": Runtime{
					name: "nvidia-legacy",
					Path: "/usr/bin/nvidia-container-runtime.legacy",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			runtimes := GetRuntimes(
				WithNvidiaRuntimeName(tc.nvidiaRuntimeName),
				WithSetAsDefault(tc.setAsDefault),
				WithRoot(tc.root),
			)

			require.EqualValues(t, tc.expectedRuntimes, runtimes)
			require.Equal(t, tc.expectedDefaultRuntime, runtimes.DefaultRuntimeName())
		})
	}
}
