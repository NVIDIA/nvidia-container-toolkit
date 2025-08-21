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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	libnvidiaContainerCliTestTemplate = `
# Create a temporary directory and rootfs path
TMPDIR="$(mktemp -d)"
ROOTFS="${TMPDIR}/rootfs"
mkdir -p "${ROOTFS}"

# Expose ROOTFS for the child namespace
export ROOTFS TMPDIR

# Download Ubuntu base image with error handling
curl -fsSL http://cdimage.ubuntu.com/ubuntu-base/releases/22.04/release/ubuntu-base-22.04-base-amd64.tar.gz | tar -C $ROOTFS -xz || {
    echo "Failed to download or extract Ubuntu base image"
    exit 1
}

# Enter a new mount + PID namespace so we can pivot_root without touching the
# container'\''s original filesystem.
unshare --mount --pid --fork --propagation private -- sh -eux <<'\''IN_NS'\''
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

  # 3 Configure NVIDIA devices
  nvidia-container-cli --load-kmods configure --ldconfig=@/sbin/ldconfig.real --no-cgroups --utility --device 0 $(pwd)

  # 4 Switch root into the prepared filesystem
  pivot_root . mnt
  umount -l mnt
  nvidia-smi -L

IN_NS
`
)

var _ = Describe("nvidia-container-cli", Ordered, ContinueOnFailure, Label("libnvidia-container"), func() {
	var (
		nestedContainerRunner Runner
		hostOutput            string
	)

	BeforeAll(func(ctx context.Context) {
		var err error
		nestedContainerRunner, err = NewNestedContainerRunner(runner, "ubuntu", installCTK, imageName+":"+imageTag, testContainerName)
		Expect(err).ToNot(HaveOccurred())

		// Capture the host GPU list.
		hostOutput, _, err = runner.Run("nvidia-smi -L")
		Expect(err).ToNot(HaveOccurred())

		// Normalize the output once
		hostOutput = strings.TrimSpace(strings.ReplaceAll(hostOutput, "\r", ""))
	})

	AfterAll(func(ctx context.Context) {
		// Cleanup: remove the container and the temporary script on the host.
		// Use || true to ensure cleanup doesn't fail the test
		runner.Run(fmt.Sprintf("docker rm -f %s 2>/dev/null || true", testContainerName)) //nolint:errcheck
	})

	It("should report the same GPUs inside the container as on the host", func(ctx context.Context) {
		// Run the test script in the container.
		output, _, err := nestedContainerRunner.Run(libnvidiaContainerCliTestTemplate)
		Expect(err).ToNot(HaveOccurred())
		Expect(strings.TrimSpace(output)).ToNot(BeEmpty())
		Expect(hostOutput).To(ContainSubstring(strings.TrimSpace(output)))
	})
})
