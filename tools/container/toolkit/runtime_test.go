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

package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNvidiaContainerRuntimeInstallerWrapper(t *testing.T) {
	r := newNvidiaContainerRuntimeInstaller()

	const shebang = "#! /bin/sh"
	const destFolder = "/dest/folder"
	const dotfileName = "source.real"

	buf := &bytes.Buffer{}

	err := r.writeWrapperTo(buf, destFolder, dotfileName)
	require.NoError(t, err)

	expectedLines := []string{
		shebang,
		"",
		"cat /proc/modules | grep -e \"^nvidia \" >/dev/null 2>&1",
		"if [ \"${?}\" != \"0\" ]; then",
		"	echo \"nvidia driver modules are not yet loaded, invoking runc directly\"",
		"	exec runc \"$@\"",
		"fi",
		"",
		"PATH=/dest/folder:$PATH \\",
		"XDG_CONFIG_HOME=/dest/folder/.config \\",
		"source.real \\",
		"\t\"$@\"",
		"",
	}

	exepectedContents := strings.Join(expectedLines, "\n")
	require.Equal(t, exepectedContents, buf.String())
}

func TestExperimentalContainerRuntimeInstallerWrapper(t *testing.T) {
	r := newNvidiaContainerRuntimeExperimentalInstaller("/some/root/usr/lib64")

	const shebang = "#! /bin/sh"
	const destFolder = "/dest/folder"
	const dotfileName = "source.real"

	buf := &bytes.Buffer{}

	err := r.writeWrapperTo(buf, destFolder, dotfileName)
	require.NoError(t, err)

	expectedLines := []string{
		shebang,
		"",
		"cat /proc/modules | grep -e \"^nvidia \" >/dev/null 2>&1",
		"if [ \"${?}\" != \"0\" ]; then",
		"	echo \"nvidia driver modules are not yet loaded, invoking runc directly\"",
		"	exec runc \"$@\"",
		"fi",
		"",
		"LD_LIBRARY_PATH=/some/root/usr/lib64:$LD_LIBRARY_PATH \\",
		"PATH=/dest/folder:$PATH \\",
		"XDG_CONFIG_HOME=/dest/folder/.config \\",
		"source.real \\",
		"\t\"$@\"",
		"",
	}

	exepectedContents := strings.Join(expectedLines, "\n")
	require.Equal(t, exepectedContents, buf.String())
}
