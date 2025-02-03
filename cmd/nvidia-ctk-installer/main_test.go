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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestParseArgs(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	testCases := []struct {
		args              []string
		expectedRemaining []string
		expectedRoot      string
		expectedError     error
	}{
		{
			args:              []string{},
			expectedRemaining: []string{},
			expectedRoot:      "",
			expectedError:     nil,
		},
		{
			args:              []string{"app"},
			expectedRemaining: []string{"app"},
		},
		{
			args:              []string{"app", "root"},
			expectedRemaining: []string{"app"},
			expectedRoot:      "root",
		},
		{
			args:              []string{"app", "--flag"},
			expectedRemaining: []string{"app", "--flag"},
		},
		{
			args:              []string{"app", "root", "--flag"},
			expectedRemaining: []string{"app", "--flag"},
			expectedRoot:      "root",
		},
		{
			args:          []string{"app", "root", "not-root", "--flag"},
			expectedError: fmt.Errorf("unexpected positional argument(s) [not-root]"),
		},
		{
			args:          []string{"app", "root", "not-root"},
			expectedError: fmt.Errorf("unexpected positional argument(s) [not-root]"),
		},
		{
			args:          []string{"app", "root", "not-root", "also"},
			expectedError: fmt.Errorf("unexpected positional argument(s) [not-root also]"),
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			remaining, root, err := ParseArgs(logger, tc.args)
			if tc.expectedError != nil {
				require.EqualError(t, err, tc.expectedError.Error())
			} else {
				require.NoError(t, err)
			}

			require.ElementsMatch(t, tc.expectedRemaining, remaining)
			require.Equal(t, tc.expectedRoot, root)
		})
	}
}

func TestApp(t *testing.T) {
	t.Setenv("__NVCT_TESTING_DEVICES_ARE_FILES", "true")
	logger, _ := testlog.NewNullLogger()

	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)

	artifactRoot := filepath.Join(moduleRoot, "testdata", "installer", "artifacts")
	hostRoot := filepath.Join(moduleRoot, "testdata", "lookup", "rootfs-1")

	testCases := []struct {
		description           string
		args                  []string
		expectedToolkitConfig string
		expectedRuntimeConfig string
	}{
		{
			description: "no args",
			expectedToolkitConfig: `accept-nvidia-visible-devices-as-volume-mounts = false
accept-nvidia-visible-devices-envvar-when-unprivileged = true
disable-require = false
supported-driver-capabilities = "compat32,compute,display,graphics,ngx,utility,video"
swarm-resource = ""

[nvidia-container-cli]
  debug = ""
  environment = []
  ldcache = ""
  ldconfig = "@/run/nvidia/driver/sbin/ldconfig"
  load-kmods = true
  no-cgroups = false
  path = "{{ .toolkitRoot }}/toolkit/nvidia-container-cli"
  root = "/run/nvidia/driver"
  user = ""

[nvidia-container-runtime]
  debug = "/dev/null"
  log-level = "info"
  mode = "auto"
  runtimes = ["docker-runc", "runc", "crun"]

  [nvidia-container-runtime.modes]

    [nvidia-container-runtime.modes.cdi]
      annotation-prefixes = ["cdi.k8s.io/"]
      default-kind = "nvidia.com/gpu"
      spec-dirs = ["/etc/cdi", "/var/run/cdi"]

    [nvidia-container-runtime.modes.csv]
      mount-spec-path = "/etc/nvidia-container-runtime/host-files-for-container.d"

[nvidia-container-runtime-hook]
  path = "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime-hook"
  skip-mode-detection = true

[nvidia-ctk]
  path = "{{ .toolkitRoot }}/toolkit/nvidia-ctk"
`,
			expectedRuntimeConfig: `{
    "default-runtime": "nvidia",
    "runtimes": {
        "nvidia": {
            "args": [],
            "path": "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime"
        },
        "nvidia-cdi": {
            "args": [],
            "path": "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime.cdi"
        },
        "nvidia-legacy": {
            "args": [],
            "path": "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime.legacy"
        }
    }
}`,
		},
		{
			description: "CDI enabled enables CDI in docker",
			args:        []string{"--cdi-enabled", "--create-device-nodes=none"},
			expectedToolkitConfig: `accept-nvidia-visible-devices-as-volume-mounts = false
accept-nvidia-visible-devices-envvar-when-unprivileged = true
disable-require = false
supported-driver-capabilities = "compat32,compute,display,graphics,ngx,utility,video"
swarm-resource = ""

[nvidia-container-cli]
  debug = ""
  environment = []
  ldcache = ""
  ldconfig = "@/run/nvidia/driver/sbin/ldconfig"
  load-kmods = true
  no-cgroups = false
  path = "{{ .toolkitRoot }}/toolkit/nvidia-container-cli"
  root = "/run/nvidia/driver"
  user = ""

[nvidia-container-runtime]
  debug = "/dev/null"
  log-level = "info"
  mode = "auto"
  runtimes = ["docker-runc", "runc", "crun"]

  [nvidia-container-runtime.modes]

    [nvidia-container-runtime.modes.cdi]
      annotation-prefixes = ["cdi.k8s.io/"]
      default-kind = "nvidia.com/gpu"
      spec-dirs = ["/etc/cdi", "/var/run/cdi"]

    [nvidia-container-runtime.modes.csv]
      mount-spec-path = "/etc/nvidia-container-runtime/host-files-for-container.d"

[nvidia-container-runtime-hook]
  path = "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime-hook"
  skip-mode-detection = true

[nvidia-ctk]
  path = "{{ .toolkitRoot }}/toolkit/nvidia-ctk"
`,
			expectedRuntimeConfig: `{
    "default-runtime": "nvidia",
    "features": {
        "cdi": true
    },
    "runtimes": {
        "nvidia": {
            "args": [],
            "path": "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime"
        },
        "nvidia-cdi": {
            "args": [],
            "path": "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime.cdi"
        },
        "nvidia-legacy": {
            "args": [],
            "path": "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime.legacy"
        }
    }
}`,
		},
		{
			description: "--enable-cdi-in-runtime=false overrides --cdi-enabled in Docker",
			args:        []string{"--cdi-enabled", "--create-device-nodes=none", "--enable-cdi-in-runtime=false"},
			expectedToolkitConfig: `accept-nvidia-visible-devices-as-volume-mounts = false
accept-nvidia-visible-devices-envvar-when-unprivileged = true
disable-require = false
supported-driver-capabilities = "compat32,compute,display,graphics,ngx,utility,video"
swarm-resource = ""

[nvidia-container-cli]
  debug = ""
  environment = []
  ldcache = ""
  ldconfig = "@/run/nvidia/driver/sbin/ldconfig"
  load-kmods = true
  no-cgroups = false
  path = "{{ .toolkitRoot }}/toolkit/nvidia-container-cli"
  root = "/run/nvidia/driver"
  user = ""

[nvidia-container-runtime]
  debug = "/dev/null"
  log-level = "info"
  mode = "auto"
  runtimes = ["docker-runc", "runc", "crun"]

  [nvidia-container-runtime.modes]

    [nvidia-container-runtime.modes.cdi]
      annotation-prefixes = ["cdi.k8s.io/"]
      default-kind = "nvidia.com/gpu"
      spec-dirs = ["/etc/cdi", "/var/run/cdi"]

    [nvidia-container-runtime.modes.csv]
      mount-spec-path = "/etc/nvidia-container-runtime/host-files-for-container.d"

[nvidia-container-runtime-hook]
  path = "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime-hook"
  skip-mode-detection = true

[nvidia-ctk]
  path = "{{ .toolkitRoot }}/toolkit/nvidia-ctk"
`,
			expectedRuntimeConfig: `{
    "default-runtime": "nvidia",
    "runtimes": {
        "nvidia": {
            "args": [],
            "path": "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime"
        },
        "nvidia-cdi": {
            "args": [],
            "path": "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime.cdi"
        },
        "nvidia-legacy": {
            "args": [],
            "path": "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime.legacy"
        }
    }
}`,
		},
		{
			description: "CDI enabled enables CDI in containerd",
			args:        []string{"--cdi-enabled", "--runtime=containerd"},
			expectedToolkitConfig: `accept-nvidia-visible-devices-as-volume-mounts = false
accept-nvidia-visible-devices-envvar-when-unprivileged = true
disable-require = false
supported-driver-capabilities = "compat32,compute,display,graphics,ngx,utility,video"
swarm-resource = ""

[nvidia-container-cli]
  debug = ""
  environment = []
  ldcache = ""
  ldconfig = "@/run/nvidia/driver/sbin/ldconfig"
  load-kmods = true
  no-cgroups = false
  path = "{{ .toolkitRoot }}/toolkit/nvidia-container-cli"
  root = "/run/nvidia/driver"
  user = ""

[nvidia-container-runtime]
  debug = "/dev/null"
  log-level = "info"
  mode = "auto"
  runtimes = ["docker-runc", "runc", "crun"]

  [nvidia-container-runtime.modes]

    [nvidia-container-runtime.modes.cdi]
      annotation-prefixes = ["cdi.k8s.io/"]
      default-kind = "nvidia.com/gpu"
      spec-dirs = ["/etc/cdi", "/var/run/cdi"]

    [nvidia-container-runtime.modes.csv]
      mount-spec-path = "/etc/nvidia-container-runtime/host-files-for-container.d"

[nvidia-container-runtime-hook]
  path = "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime-hook"
  skip-mode-detection = true

[nvidia-ctk]
  path = "{{ .toolkitRoot }}/toolkit/nvidia-ctk"
`,
			expectedRuntimeConfig: `version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]
    enable_cdi = true

    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "nvidia"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia]
          privileged_without_host_devices = false
          runtime_engine = ""
          runtime_root = ""
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia.options]
            BinaryName = "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi]
          privileged_without_host_devices = false
          runtime_engine = ""
          runtime_root = ""
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi.options]
            BinaryName = "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime.cdi"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy]
          privileged_without_host_devices = false
          runtime_engine = ""
          runtime_root = ""
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy.options]
            BinaryName = "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime.legacy"
`,
		},
		{
			description: "--enable-cdi-in-runtime=false overrides --cdi-enabled in containerd",
			args:        []string{"--cdi-enabled", "--create-device-nodes=none", "--enable-cdi-in-runtime=false", "--runtime=containerd"},
			expectedToolkitConfig: `accept-nvidia-visible-devices-as-volume-mounts = false
accept-nvidia-visible-devices-envvar-when-unprivileged = true
disable-require = false
supported-driver-capabilities = "compat32,compute,display,graphics,ngx,utility,video"
swarm-resource = ""

[nvidia-container-cli]
  debug = ""
  environment = []
  ldcache = ""
  ldconfig = "@/run/nvidia/driver/sbin/ldconfig"
  load-kmods = true
  no-cgroups = false
  path = "{{ .toolkitRoot }}/toolkit/nvidia-container-cli"
  root = "/run/nvidia/driver"
  user = ""

[nvidia-container-runtime]
  debug = "/dev/null"
  log-level = "info"
  mode = "auto"
  runtimes = ["docker-runc", "runc", "crun"]

  [nvidia-container-runtime.modes]

    [nvidia-container-runtime.modes.cdi]
      annotation-prefixes = ["cdi.k8s.io/"]
      default-kind = "nvidia.com/gpu"
      spec-dirs = ["/etc/cdi", "/var/run/cdi"]

    [nvidia-container-runtime.modes.csv]
      mount-spec-path = "/etc/nvidia-container-runtime/host-files-for-container.d"

[nvidia-container-runtime-hook]
  path = "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime-hook"
  skip-mode-detection = true

[nvidia-ctk]
  path = "{{ .toolkitRoot }}/toolkit/nvidia-ctk"
`,
			expectedRuntimeConfig: `version = 2

[plugins]

  [plugins."io.containerd.grpc.v1.cri"]

    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "nvidia"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia]
          privileged_without_host_devices = false
          runtime_engine = ""
          runtime_root = ""
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia.options]
            BinaryName = "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi]
          privileged_without_host_devices = false
          runtime_engine = ""
          runtime_root = ""
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-cdi.options]
            BinaryName = "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime.cdi"

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy]
          privileged_without_host_devices = false
          runtime_engine = ""
          runtime_root = ""
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia-legacy.options]
            BinaryName = "{{ .toolkitRoot }}/toolkit/nvidia-container-runtime.legacy"
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			testRoot := t.TempDir()

			cdiOutputDir := filepath.Join(testRoot, "/var/run/cdi")
			runtimeConfigFile := filepath.Join(testRoot, "config.file")

			toolkitRoot := filepath.Join(testRoot, "toolkit-test")
			toolkitConfigFile := filepath.Join(toolkitRoot, "toolkit/.config/nvidia-container-runtime/config.toml")

			app := NewApp(logger, toolkitRoot)

			testArgs := []string{
				"nvidia-ctk-installer",
				"--no-daemon",
				"--cdi-output-dir=" + cdiOutputDir,
				"--config=" + runtimeConfigFile,
				"--create-device-nodes=none",
				"--driver-root-ctr-path=" + hostRoot,
				"--pid-file=" + filepath.Join(testRoot, "toolkit.pid"),
				"--restart-mode=none",
				"--source-root=" + filepath.Join(artifactRoot, "deb"),
			}

			err := app.Run(append(testArgs, tc.args...))

			require.NoError(t, err)

			require.FileExists(t, toolkitConfigFile)
			toolkitConfigFileContents, err := os.ReadFile(toolkitConfigFile)
			require.NoError(t, err)
			require.EqualValues(t, strings.ReplaceAll(tc.expectedToolkitConfig, "{{ .toolkitRoot }}", toolkitRoot), string(toolkitConfigFileContents))

			require.FileExists(t, runtimeConfigFile)
			runtimeConfigFileContents, err := os.ReadFile(runtimeConfigFile)
			require.NoError(t, err)
			require.EqualValues(t, strings.ReplaceAll(tc.expectedRuntimeConfig, "{{ .toolkitRoot }}", toolkitRoot), string(runtimeConfigFileContents))
		})
	}

}
