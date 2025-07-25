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
	libnvidiaContainerCliTestTemplate = `#!/bin/bash
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive

# Add Docker's official GPG key:
apt-get update
apt-get install -y ca-certificates curl apt-utils gnupg2
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc

# Add the repository to Apt sources:
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}") stable" | \
  tee /etc/apt/sources.list.d/docker.list > /dev/null
apt-get update

apt-get install -y docker-ce docker-ce-cli containerd.io

# start dockerd in the background
dockerd &

# wait for dockerd to be ready
while ! docker info > /dev/null 2>&1; do
    echo "Waiting for dockerd to be ready..."
    sleep 1
done

# Create a temporary directory and rootfs path
TMPDIR="$(mktemp -d)"
ROOTFS="${TMPDIR}/rootfs"
mkdir -p "${ROOTFS}"

# Expose ROOTFS for the child namespace
export ROOTFS

echo "Copying package files from ${NVIDIA_TOOLKIT_IMAGE} to ${ROOTFS}" 
docker run --rm \ 
	-v $(pwd):$(pwd) \ 
	-w $(pwd) \ 
	-u $(id -u):$(id -g) \ 
	--entrypoint="sh" \ 
		${NVIDIA_TOOLKIT_IMAGE}-packaging \ 
		-c "cp -p -R /artifacts/* ${TMPDIR}" 

dpkg -i ${TMPDIR}/amd64/libnvidia-container1_*_amd64.deb \
    ${TMPDIR}/amd64/nvidia-container-toolkit-base_*_amd64.deb \
    ${TMPDIR}/amd64/libnvidia-container-tools_*_amd64.deb

nvidia-container-cli --version

curl http://cdimage.ubuntu.com/ubuntu-base/releases/22.04/release/ubuntu-base-22.04-base-amd64.tar.gz | tar -C $ROOTFS -xz

# Enter a new mount + PID namespace so we can pivot_root without touching the
# container's original filesystem.
unshare --mount --pid --fork --propagation private -- sh -eux <<'IN_NS'
  : "${ROOTFS:?}"

  # 1 Bind-mount the new root and make the mount private
  mount --bind "$ROOTFS" "$ROOTFS"
  mount --make-private "$ROOTFS"
  cd "$ROOTFS"

  # 2 Minimal virtual filesystems
  mount -t proc  proc proc
  mount -t sysfs sys  sys
  mount -t tmpfs tmp  tmp
  mount -t tmpfs run  run

  # 3 GPU setup via nvidia-container-cli
  # Add potential install locations of nvidia-container-cli to PATH.
  # /artifacts/{rpm,deb}/usr/bin are where the binary ends up after the
  # packages are extracted in the application image.  /work is included for
  # completeness since some images may copy the binary there.
  export PATH="${PATH}:/artifacts/rpm/usr/bin:/artifacts/deb/usr/bin:/work"

  nvidia-container-cli --load-kmods \
                       configure \
                       --no-cgroups --utility --device=0 "$(pwd)"

  # 4 Switch root into the prepared filesystem
  mkdir -p mnt
  pivot_root . mnt
  umount -l /mnt

  exec nvidia-smi -L
IN_NS
`

	dockerRunCmdTemplate = `docker run --name {{.ContainerName}} --privileged --runtime=nvidia \
    -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all \
    -e NVIDIA_DRIVER_CAPABILITIES=all \
	-e NVIDIA_TOOLKIT_IMAGE={{.ToolkitImage}} \
    -v {{.ScriptPath}}:/libnvidia-container-cli.sh \
	ubuntu /libnvidia-container-cli.sh`
)

var _ = Describe("nvidia-container-cli", Ordered, ContinueOnFailure, func() {
	var (
		runner         Runner
		containerName  = "nvidia-cli-e2e"
		hostOutput     string
		testScriptPath = "/tmp/libnvidia-container-cli.sh"
	)

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

		// Capture the host GPU list.
		var err error
		hostOutput, _, err = runner.Run("nvidia-smi -L")
		Expect(err).ToNot(HaveOccurred())

		// Normalize the output once
		hostOutput = strings.TrimSpace(strings.ReplaceAll(hostOutput, "\r", ""))

		// If a container with the same name exists from a previous test run, remove it first.
		_, _, err = runner.Run(fmt.Sprintf("docker rm -f %s", containerName))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterAll(func(ctx context.Context) {
		// Cleanup: remove the container and the temporary script on the host.
		runner.Run(fmt.Sprintf("docker rm -f %s", containerName))

		// Remove the script from the remote host.
		runner.Run(fmt.Sprintf("rm -f %s", testScriptPath))
	})

	It("should report the same GPUs inside the container as on the host", func(ctx context.Context) {
		// Write the script to the remote host and make it executable.
		createScriptCmd := fmt.Sprintf(
			"cat > %s <<'EOF'\n%s\nEOF\nchmod +x %s",
			testScriptPath, libnvidiaContainerCliTestTemplate, testScriptPath,
		)

		_, _, err := runner.Run(createScriptCmd)
		Expect(err).ToNot(HaveOccurred())

		// Build the docker run command (detached mode) from the template so it
		// stays readable while still resulting in a single-line invocation.
		tmpl, err := template.New("dockerRun").Parse(dockerRunCmdTemplate)
		Expect(err).ToNot(HaveOccurred())

		var dockerRunCmd strings.Builder
		err = tmpl.Execute(&dockerRunCmd, struct {
			ContainerName string
			ToolkitImage  string
			ScriptPath    string
		}{
			ContainerName: containerName,
			ToolkitImage:  imageName + ":" + imageTag,
			ScriptPath:    testScriptPath,
		})
		Expect(err).ToNot(HaveOccurred())

		// Launch the container in detached mode.
		_, _, err = runner.Run(dockerRunCmd.String())
		Expect(err).ToNot(HaveOccurred())

		// Poll the logs of the already running container until we observe
		// the GPU list matching the host or until a 5-minute timeout elapses.
		Eventually(func() string {
			logs, _, err := runner.Run(fmt.Sprintf("docker logs --tail 1 %s", containerName))
			if err != nil {
				return fmt.Sprintf("error: %v", err)
			}

			// Docker already returns only the last line due to --tail 1, so just normalize.
			return strings.TrimSpace(strings.ReplaceAll(logs, "\r", ""))
		}, "5m", "10s").Should(Equal(hostOutput))
	})
})
