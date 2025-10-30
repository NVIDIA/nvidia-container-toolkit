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
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
)

func TestToolkitInstaller(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	type contentCall struct {
		wrapper string
		path    string
		mode    fs.FileMode
	}
	var contentCalls []contentCall

	installer := &fileInstallerMock{
		installFileFunc: func(s1, s2 string) (os.FileMode, error) {
			return 0666, nil
		},
		installContentFunc: func(reader io.Reader, s string, fileMode fs.FileMode) error {
			var b bytes.Buffer
			if _, err := b.ReadFrom(reader); err != nil {
				return err
			}
			contents := contentCall{
				wrapper: b.String(),
				path:    s,
				mode:    fileMode,
			}

			contentCalls = append(contentCalls, contents)
			return nil
		},
		installSymlinkFunc: func(s1, s2 string) error {
			return nil
		},
	}
	installFile = installer.installFile
	installContent = installer.installContent
	installSymlink = installer.installSymlink

	root := "/artifacts/test"
	libraries := &lookup.LocatorMock{
		LocateFunc: func(s string) ([]string, error) {
			switch s {
			case "libnvidia-container.so.1":
				return []string{filepath.Join(root, "libnvidia-container.so.987.65.43")}, nil
			case "libnvidia-container-go.so.1":
				return []string{filepath.Join(root, "libnvidia-container-go.so.1.23.4")}, nil
			}
			return nil, fmt.Errorf("%v not found", s)
		},
	}
	executables := &lookup.LocatorMock{
		LocateFunc: func(s string) ([]string, error) {
			switch s {
			case "nvidia-container-runtime.cdi":
				fallthrough
			case "nvidia-container-runtime.legacy":
				fallthrough
			case "nvidia-container-runtime":
				fallthrough
			case "nvidia-ctk":
				fallthrough
			case "nvidia-container-cli":
				fallthrough
			case "nvidia-container-runtime-hook":
				fallthrough
			case "nvidia-cdi-hook":
				return []string{filepath.Join(root, "usr/bin", s)}, nil
			}
			return nil, fmt.Errorf("%v not found", s)
		},
	}

	r := &artifactRoot{
		libraries:   libraries,
		executables: executables,
	}

	createDirectory := &InstallerMock{
		InstallFunc: func(c string) error {
			return nil
		},
	}
	i := ToolkitInstaller{
		logger:                       logger,
		artifactRoot:                 r,
		ensureTargetDirectory:        createDirectory,
		defaultRuntimeExecutablePath: "runc",
	}

	err := i.Install("/foo/bar/baz")
	require.NoError(t, err)

	require.ElementsMatch(t,
		[]struct {
			S string
		}{
			{"/foo/bar/baz"},
		},
		createDirectory.InstallCalls(),
	)

	require.ElementsMatch(t,
		installer.installFileCalls(),
		[]struct {
			S1 string
			S2 string
		}{
			{"/artifacts/test/libnvidia-container-go.so.1.23.4", "/foo/bar/baz/libnvidia-container-go.so.1.23.4"},
			{"/artifacts/test/libnvidia-container.so.987.65.43", "/foo/bar/baz/libnvidia-container.so.987.65.43"},
			{"/artifacts/test/usr/bin/nvidia-container-runtime.cdi", "/foo/bar/baz/nvidia-container-runtime.cdi.real"},
			{"/artifacts/test/usr/bin/nvidia-container-runtime.legacy", "/foo/bar/baz/nvidia-container-runtime.legacy.real"},
			{"/artifacts/test/usr/bin/nvidia-container-runtime", "/foo/bar/baz/nvidia-container-runtime.real"},
			{"/artifacts/test/usr/bin/nvidia-ctk", "/foo/bar/baz/nvidia-ctk.real"},
			{"/artifacts/test/usr/bin/nvidia-cdi-hook", "/foo/bar/baz/nvidia-cdi-hook.real"},
			{"/artifacts/test/usr/bin/nvidia-container-cli", "/foo/bar/baz/nvidia-container-cli.real"},
			{"/artifacts/test/usr/bin/nvidia-container-runtime-hook", "/foo/bar/baz/nvidia-container-runtime-hook.real"},
		},
	)

	require.ElementsMatch(t,
		installer.installSymlinkCalls(),
		[]struct {
			S1 string
			S2 string
		}{
			{"libnvidia-container-go.so.1.23.4", "/foo/bar/baz/libnvidia-container-go.so.1"},
			{"libnvidia-container.so.987.65.43", "/foo/bar/baz/libnvidia-container.so.1"},
			{"nvidia-container-runtime-hook", "/foo/bar/baz/nvidia-container-toolkit"},
		},
	)

	require.ElementsMatch(t,
		contentCalls,
		[]contentCall{
			{
				path: "/foo/bar/baz/nvidia-container-runtime",
				mode: 0777,
				wrapper: `#! /bin/sh
cat /proc/modules | grep -e "^nvidia " >/dev/null 2>&1
if [ "${?}" != "0" ]; then
	echo "nvidia driver modules are not yet loaded, invoking runc directly" >&2
	exec runc "$@"
fi
NVIDIA_CTK_CONFIG_FILE_PATH=/foo/bar/baz/.config/nvidia-container-runtime/config.toml \
PATH=/foo/bar/baz:$PATH \
	/foo/bar/baz/nvidia-container-runtime.real \
		"$@"
`,
			},
			{
				path: "/foo/bar/baz/nvidia-container-runtime.cdi",
				mode: 0777,
				wrapper: `#! /bin/sh
cat /proc/modules | grep -e "^nvidia " >/dev/null 2>&1
if [ "${?}" != "0" ]; then
	echo "nvidia driver modules are not yet loaded, invoking runc directly" >&2
	exec runc "$@"
fi
NVIDIA_CTK_CONFIG_FILE_PATH=/foo/bar/baz/.config/nvidia-container-runtime/config.toml \
PATH=/foo/bar/baz:$PATH \
	/foo/bar/baz/nvidia-container-runtime.cdi.real \
		"$@"
`,
			},
			{
				path: "/foo/bar/baz/nvidia-container-runtime.legacy",
				mode: 0777,
				wrapper: `#! /bin/sh
cat /proc/modules | grep -e "^nvidia " >/dev/null 2>&1
if [ "${?}" != "0" ]; then
	echo "nvidia driver modules are not yet loaded, invoking runc directly" >&2
	exec runc "$@"
fi
NVIDIA_CTK_CONFIG_FILE_PATH=/foo/bar/baz/.config/nvidia-container-runtime/config.toml \
PATH=/foo/bar/baz:$PATH \
	/foo/bar/baz/nvidia-container-runtime.legacy.real \
		"$@"
`,
			},
			{
				path: "/foo/bar/baz/nvidia-ctk",
				mode: 0777,
				wrapper: `#! /bin/sh
PATH=/foo/bar/baz:$PATH \
	/foo/bar/baz/nvidia-ctk.real \
		"$@"
`,
			},
			{
				path: "/foo/bar/baz/nvidia-cdi-hook",
				mode: 0777,
				wrapper: `#! /bin/sh
PATH=/foo/bar/baz:$PATH \
	/foo/bar/baz/nvidia-cdi-hook.real \
		"$@"
`,
			},
			{
				path: "/foo/bar/baz/nvidia-container-cli",
				mode: 0777,
				wrapper: `#! /bin/sh
LD_LIBRARY_PATH=/foo/bar/baz:$LD_LIBRARY_PATH \
PATH=/foo/bar/baz:$PATH \
	/foo/bar/baz/nvidia-container-cli.real \
		"$@"
`,
			},
			{
				path: "/foo/bar/baz/nvidia-container-runtime-hook",
				mode: 0777,
				wrapper: `#! /bin/sh
NVIDIA_CTK_CONFIG_FILE_PATH=/foo/bar/baz/.config/nvidia-container-runtime/config.toml \
PATH=/foo/bar/baz:$PATH \
	/foo/bar/baz/nvidia-container-runtime-hook.real \
		"$@"
`,
			},
		},
	)
}
