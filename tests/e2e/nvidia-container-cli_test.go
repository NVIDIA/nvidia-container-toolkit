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
	libnvidiaContainerCliDockerRunTemplate = `
docker run -d --name test-nvidia-container-cli \
  --privileged \
  --runtime=nvidia \
  -e NVIDIA_VISIBLE_DEVICES=all \
  -e NVIDIA_DRIVER_CAPABILITIES=all \
  -v $HOME/libnvidia-container-cli.sh:/usr/local/bin/libnvidia-container-cli.sh \
  -v /usr/bin/nvidia-container-cli:/usr/bin/nvidia-container-cli \
  -v /usr/bin/nvidia-ctk:/usr/bin/nvidia-ctk \
  -v /usr/bin/nvidia-container-runtime:/usr/bin/nvidia-container-runtime \
  -v /usr/bin/nvidia-container-runtime-hook:/usr/bin/nvidia-container-runtime-hook \
  -v /usr/bin/nvidia-container-toolkit:/usr/bin/nvidia-container-toolkit \
  -v /usr/bin/nvidia-cdi-hook:/usr/bin/nvidia-cdi-hook \
  -v /usr/bin/nvidia-container-runtime.cdi:/usr/bin/nvidia-container-runtime.cdi \
  -v /usr/bin/nvidia-container-runtime.legacy:/usr/bin/nvidia-container-runtime.legacy \
  -v /usr/local/nvidia/toolkit:/usr/local/nvidia/toolkit \
  -v /etc/nvidia-container-runtime:/etc/nvidia-container-runtime \
  -v /usr/lib/x86_64-linux-gnu/libnvidia-container.so.1:/usr/lib/x86_64-linux-gnu/libnvidia-container.so.1 \
  -v /usr/lib/x86_64-linux-gnu/{{.LibNvidiaContainerTarget}}:/usr/lib/x86_64-linux-gnu/{{.LibNvidiaContainerTarget}} \
  -v /usr/lib/x86_64-linux-gnu/libnvidia-container-go.so.1:/usr/lib/x86_64-linux-gnu/libnvidia-container-go.so.1 \
  -v /usr/lib/x86_64-linux-gnu/{{.LibNvidiaContainerGoTarget}}:/usr/lib/x86_64-linux-gnu/{{.LibNvidiaContainerGoTarget}} \
  -e LD_LIBRARY_PATH=/usr/lib64:/usr/lib/x86_64-linux-gnu:/usr/lib/aarch64-linux-gnu:/lib64:/lib/x86_64-linux-gnu:/lib/aarch64-linux-gnu \
  --entrypoint /usr/local/bin/libnvidia-container-cli.sh \
  ubuntu
`

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

	// Install the NVIDIA Container Toolkit
	BeforeAll(func(ctx context.Context) {
		runner = NewRunner(
			WithHost(sshHost),
			WithPort(sshPort),
			WithSshKey(sshKey),
			WithSshUser(sshUser),
		)

		if installCTK {
			installer, err := NewToolkitInstaller(
				WithRunner(runner),
				WithImage(imageName+":"+imageTag),
				WithTemplate(dockerInstallTemplate),
			)
			Expect(err).ToNot(HaveOccurred())

			err = installer.Install()
			Expect(err).ToNot(HaveOccurred())
		}
	})

	When("running nvidia-smi -L", Ordered, func() {
		var hostOutput string
		var err error

		BeforeAll(func(ctx context.Context) {
			hostOutput, _, err = runner.Run("nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())

			_, _, err := runner.Run("docker pull ubuntu")
			Expect(err).ToNot(HaveOccurred())
		})

		AfterAll(func(ctx context.Context) {
			_, _, err := runner.Run("docker rm -f test-nvidia-container-cli")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should support NVIDIA_VISIBLE_DEVICES and NVIDIA_DRIVER_CAPABILITIES", func(ctx context.Context) {
			// 1. Create the test script on the remote host at $HOME/test.sh using a here-document
			testScriptPath := "$HOME/libnvidia-container-cli.sh"
			testScript := libnvidiaContainerCliTestTemplate
			createScriptCmd := fmt.Sprintf("cat > %s <<'EOF'\n%s\nEOF\nchmod +x %s", testScriptPath, testScript, testScriptPath)
			_, _, err := runner.Run(createScriptCmd)
			Expect(err).ToNot(HaveOccurred())

			// 2. Discover the symlink targets for the libraries on the remote host
			getTargetCmd := func(lib string) string {
				return fmt.Sprintf("readlink -f /usr/lib/x86_64-linux-gnu/%s.1", lib)
			}
			libNvidiaContainerTarget, _, err := runner.Run(getTargetCmd("libnvidia-container.so"))
			Expect(err).ToNot(HaveOccurred())

			libNvidiaContainerTarget = strings.TrimSpace(libNvidiaContainerTarget)
			libNvidiaContainerTarget = strings.TrimPrefix(libNvidiaContainerTarget, "/usr/lib/x86_64-linux-gnu/")

			libNvidiaContainerGoTarget, _, err := runner.Run(getTargetCmd("libnvidia-container-go.so"))
			Expect(err).ToNot(HaveOccurred())

			libNvidiaContainerGoTarget = strings.TrimSpace(libNvidiaContainerGoTarget)
			libNvidiaContainerGoTarget = strings.TrimPrefix(libNvidiaContainerGoTarget, "/usr/lib/x86_64-linux-gnu/")

			// 3. Render the docker run template with the discovered targets
			tmpl, err := template.New("dockerRun").Parse(libnvidiaContainerCliDockerRunTemplate)
			Expect(err).ToNot(HaveOccurred())
			var dockerRunCmdBuilder strings.Builder
			err = tmpl.Execute(&dockerRunCmdBuilder, map[string]string{
				"LibNvidiaContainerTarget":   libNvidiaContainerTarget,
				"LibNvidiaContainerGoTarget": libNvidiaContainerGoTarget,
			})
			Expect(err).ToNot(HaveOccurred())
			dockerRunCmd := dockerRunCmdBuilder.String()

			// 4. Start the container using the rendered docker run command
			_, _, err = runner.Run(dockerRunCmd)
			Expect(err).ToNot(HaveOccurred())

			// 5. Use Eventually to check the container logs contain hostOutput
			Eventually(func() string {
				logs, _, err := runner.Run("docker logs test-nvidia-container-cli")
				if err != nil {
					return ""
				}
				return logs
			}, "5m", "5s").Should(ContainSubstring(hostOutput))
		})
	})
})
