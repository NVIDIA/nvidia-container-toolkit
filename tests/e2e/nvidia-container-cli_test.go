/*
 * Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package e2e

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	dockerDindTemplate = `docker run -d --rm --privileged \
  -v {{.SharedDir}}/etc/docker:/etc/docker \
  -v {{.SharedDir}}/run/nvidia:/run/nvidia \
  -v {{.SharedDir}}/usr/local/nvidia:/usr/local/nvidia \
  --name {{.ContainerName}} \
  docker:dind -H unix://{{.DockerSocket}}`

	dockerToolkitTemplate = `docker run -d --rm --privileged \
  --volumes-from {{.DindContainerName}} \
  --pid "container:{{.DindContainerName}}" \
  -e RUNTIME_ARGS="--socket {{.DockerSocket}}" \
  -v {{.TestScriptPath}}:/usr/local/bin/libnvidia-container-cli.sh \
  --name {{.ContainerName}} \
  {{.ToolkitImage}} /usr/local/bin/libnvidia-container-cli.sh`

	dockerDefaultConfigTemplate = `
{
  "registry-mirrors": ["https://mirror.gcr.io"]
}`

	libnvidiaContainerCliTestTemplate = `#!/usr/bin/env bash
set -euo pipefail

apt-get update -y && apt-get install -y curl gnupg2

WORKDIR="$(mktemp -d)"
ROOTFS="${WORKDIR}/rootfs"
mkdir -p "${ROOTFS}"

export WORKDIR ROOTFS          # make them visible in the child shell

unshare --mount --pid --fork --propagation private -- bash -eux <<'IN_NS'
  : "${ROOTFS:?}" "${WORKDIR:?}"      # abort if either is empty

  # 1 Populate minimal Ubuntu base
  curl -L http://cdimage.ubuntu.com/ubuntu-base/releases/22.04/release/ubuntu-base-22.04-base-amd64.tar.gz \
       | tar -C "$ROOTFS" -xz

  # 2 Add non-root user
  useradd -R "$ROOTFS" -U -u 1000 -s /bin/bash nvidia

  # 3 Bind-mount new root and unshare mounts
  mount --bind "$ROOTFS" "$ROOTFS"
  mount --make-private "$ROOTFS"
  cd "$ROOTFS"

  # 4 Minimal virtual filesystems
  mount -t proc  proc proc
  mount -t sysfs sys  sys
  mount -t tmpfs tmp  tmp
  mount -t tmpfs run  run

  # 5 GPU setup
  nvidia-container-cli --load-kmods --debug=container-cli.log \
                       configure --ldconfig=@/sbin/ldconfig.real \
                       --no-cgroups --utility --device=0 "$(pwd)"

  # 6 Switch root
  mkdir -p mnt
  pivot_root . mnt
  umount -l /mnt

  exec nvidia-smi -L
IN_NS
`
)

// Integration tests for Docker runtime
var _ = Describe("nvidia-container-cli", Ordered, ContinueOnFailure, func() {
	var runner Runner
	var sharedDir string
	var dindContainerName string
	var toolkitContainerName string
	var dockerSocket string
	var hostOutput string

	// Install the NVIDIA Container Toolkit
	BeforeAll(func(ctx context.Context) {
		runner = NewRunner(
			WithHost(sshHost),
			WithPort(sshPort),
			WithSshKey(sshKey),
			WithSshUser(sshUser),
		)

		// Setup shared directory and container names
		sharedDir = "/tmp/nvidia-container-toolkit-test"
		dindContainerName = "nvidia-container-toolkit-dind"
		toolkitContainerName = "nvidia-container-toolkit-test"
		dockerSocket = "/run/nvidia/docker.sock"

		// Get host nvidia-smi output
		var err error
		hostOutput, _, err = runner.Run("nvidia-smi -L")
		Expect(err).ToNot(HaveOccurred())

		// Pull ubuntu image
		_, _, err = runner.Run("docker pull ubuntu")
		Expect(err).ToNot(HaveOccurred())

		// Create shared directory structure
		_, _, err = runner.Run(fmt.Sprintf("mkdir -p %s/{etc/docker,run/nvidia,usr/local/nvidia}", sharedDir))
		Expect(err).ToNot(HaveOccurred())

		// Copy docker default config
		createDockerConfigCmd := fmt.Sprintf("cat > %s/etc/docker/daemon.json <<'EOF'\n%s\nEOF",
			sharedDir, dockerDefaultConfigTemplate)
		_, _, err = runner.Run(createDockerConfigCmd)
		Expect(err).ToNot(HaveOccurred())

		// Start Docker-in-Docker container
		tmpl, err := template.New("dockerDind").Parse(dockerDindTemplate)
		Expect(err).ToNot(HaveOccurred())

		var dindCmdBuilder strings.Builder
		err = tmpl.Execute(&dindCmdBuilder, map[string]string{
			"SharedDir":     sharedDir,
			"ContainerName": dindContainerName,
			"DockerSocket":  dockerSocket,
		})
		Expect(err).ToNot(HaveOccurred())

		_, _, err = runner.Run(dindCmdBuilder.String())
		Expect(err).ToNot(HaveOccurred())
	})

	AfterAll(func(ctx context.Context) {
		// Cleanup containers
		runner.Run(fmt.Sprintf("docker rm -f %s", toolkitContainerName))
		runner.Run(fmt.Sprintf("docker rm -f %s", dindContainerName))

		// Cleanup shared directory
		_, _, err := runner.Run(fmt.Sprintf("rm -rf %s", sharedDir))
		Expect(err).ToNot(HaveOccurred())
	})

	When("running nvidia-smi -L", Ordered, func() {
		It("should support NVIDIA_VISIBLE_DEVICES and NVIDIA_DRIVER_CAPABILITIES", func(ctx context.Context) {
			// 1. Create the test script
			testScriptPath := fmt.Sprintf("%s/libnvidia-container-cli.sh", sharedDir)
			createScriptCmd := fmt.Sprintf("cat > %s <<'EOF'\n%s\nEOF\nchmod +x %s",
				testScriptPath, libnvidiaContainerCliTestTemplate, testScriptPath)
			_, _, err := runner.Run(createScriptCmd)
			Expect(err).ToNot(HaveOccurred())

			// 2. Start the toolkit container
			tmpl, err := template.New("dockerToolkit").Parse(dockerToolkitTemplate)
			Expect(err).ToNot(HaveOccurred())

			var toolkitCmdBuilder strings.Builder
			err = tmpl.Execute(&toolkitCmdBuilder, map[string]string{
				"DindContainerName": dindContainerName,
				"ContainerName":     toolkitContainerName,
				"DockerSocket":      dockerSocket,
				"TestScriptPath":    testScriptPath,
				"ToolkitImage":      imageName + ":" + imageTag,
			})
			Expect(err).ToNot(HaveOccurred())

			_, _, err = runner.Run(toolkitCmdBuilder.String())
			Expect(err).ToNot(HaveOccurred())

			// 3. Wait for and verify the output
			expected := strings.TrimSpace(strings.ReplaceAll(hostOutput, "\r", ""))
			Eventually(func() string {
				logs, _, err := runner.Run(fmt.Sprintf("docker logs %s | tail -n 20", toolkitContainerName))
				if err != nil {
					return ""
				}

				logLines := strings.Split(strings.TrimSpace(logs), "\n")
				if len(logLines) == 0 {
					return ""
				}
				return strings.TrimSpace(strings.ReplaceAll(logLines[len(logLines)-1], "\r", ""))
			}, "5m", "5s").Should(Equal(expected))
		})
	})
})
