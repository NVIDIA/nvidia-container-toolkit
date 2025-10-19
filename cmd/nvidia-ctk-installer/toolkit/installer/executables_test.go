/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

package installer

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrapperRender(t *testing.T) {
	testCases := []struct {
		description string
		w           *wrapper
		expected    string
	}{
		{
			description: "executable is added",
			w: &wrapper{
				WrappedExecutable:            "some-runtime",
				DefaultRuntimeExecutablePath: "runc",
			},
			expected: `#! /bin/sh
	/dest-dir/some-runtime \
		"$@"
`,
		},
		{
			description: "module check is added",
			w: &wrapper{
				WrappedExecutable:            "some-runtime",
				CheckModules:                 true,
				DefaultRuntimeExecutablePath: "runc",
			},
			expected: `#! /bin/sh
cat /proc/modules | grep -e "^nvidia " >/dev/null 2>&1
if [ "${?}" != "0" ]; then
	echo "nvidia driver modules are not yet loaded, invoking runc directly"
	exec runc "$@"
fi
	/dest-dir/some-runtime \
		"$@"
`,
		},
		{
			description: "environment is added",
			w: &wrapper{
				WrappedExecutable: "some-runtime",
				Envvars: map[string]string{
					"PATH": "/foo/bar/baz",
				},
				DefaultRuntimeExecutablePath: "runc",
			},
			expected: `#! /bin/sh
PATH=/foo/bar/baz \
	/dest-dir/some-runtime \
		"$@"
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			r := render{
				wrapper: tc.w,
				DestDir: "/dest-dir",
			}
			reader, err := r.render()
			require.NoError(t, err)

			var content bytes.Buffer
			_, err = content.ReadFrom(reader)
			require.NoError(t, err)

			require.Equal(t, tc.expected, content.String())
		})
	}
}
