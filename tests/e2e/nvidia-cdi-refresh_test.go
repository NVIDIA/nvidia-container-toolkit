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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	getSystemStateScript = `systemctl is-system-running 2>/dev/null`

	setSystemdDegradedScript = `#!/usr/bin/env bash
	# Start the dummy service to force systemd to enter a degraded state
	cat <<EOF > /etc/systemd/system/dummy.service
[Unit]
Description=Dummy systemd service

[Service]
Type=oneshot
ExecStart=/usr/bin/sh -c "exit 1"
EOF

	# We know the dummy service will fail, so we can ignore the error
	systemctl start --now dummy.service 2>/dev/null || true
	`

	fixSystemDegradedScript = `#!/usr/bin/env bash
	# Start the dummy service to force systemd to enter a degraded state
	cat <<EOF > /etc/systemd/system/dummy.service
[Unit]
Description=Dummy systemd service

[Service]
Type=oneshot
ExecStart=/usr/bin/sh -c "exit 0"
EOF

	systemctl daemon-reload

	systemctl start --now dummy.service 2>/dev/null || true

	rm -rf /etc/systemd/system/dummy.service
	systemctl daemon-reload
`

	nvidiaCdiRefreshPathActiveTemplate = `
	if ! systemctl status nvidia-cdi-refresh.path | grep "Active: active"; then
		echo "nvidia-cdi-refresh.path is not Active"
		exit 1
	fi
	`
	nvidiaCdiRefreshServiceLoadedTemplate = `
	if ! systemctl status nvidia-cdi-refresh.service | grep "Loaded: loaded"; then
		echo "nvidia-cdi-refresh.service is not loaded"
		exit 1
	fi
	`

	nvidiaCdiRefreshFileExistsTemplate = `
	# is /var/run/cdi/nvidia.yaml exists? and exit with 0 if it does not exist
	if [ ! -f /var/run/cdi/nvidia.yaml ]; then
		echo "nvidia.yaml file does not exist"
		exit 1
	fi

	# generate the nvidia.yaml file
	nvidia-ctk cdi generate --output=/tmp/nvidia.yaml

	# diff the generated file with the one in /var/run/cdi/nvidia.yaml and exit with 0 if they are the same
	if ! diff /var/run/cdi/nvidia.yaml /tmp/nvidia.yaml; then
		echo "nvidia.yaml file is different"
		exit 1
	fi
	`

	nvidiaCdiRefreshUpgradeTemplate = `
	# remove the generated files
	rm /var/run/cdi/nvidia.yaml /tmp/nvidia.yaml

	# Touch the nvidia-ctk binary to change the mtime
	# This will trigger the nvidia-cdi-refresh.path unit to call the
	# nvidia-cdi-refresh.service unit, simulating a change(update/downgrade) in the nvidia-ctk binary.
	touch $(which nvidia-ctk)

	# wait for 3 seconds
	sleep 3

	# Check if the file /var/run/cdi/nvidia.yaml is created
	if [ ! -f /var/run/cdi/nvidia.yaml ]; then
		echo "nvidia.yaml file is not created after updating the modules.dep file"
		exit 1
	fi

	# generate the nvidia.yaml file
	nvidia-ctk cdi generate --output=/tmp/nvidia.yaml

	# diff the generated file with the one in /var/run/cdi/nvidia.yaml and exit with 0 if they are the same
	if ! diff /var/run/cdi/nvidia.yaml /tmp/nvidia.yaml; then
		echo "nvidia.yaml file is different"
		exit 1
	fi
	`
)

var _ = Describe("nvidia-cdi-refresh", Ordered, ContinueOnFailure, Label("systemd-unit"), func() {
	var (
		containerName = "nvctk-e2e-nvidia-cdi-refresh-tests"
		systemdRunner Runner
		// TODO(@ArangoGutierrez): https://github.com/NVIDIA/nvidia-container-toolkit/pull/1235/files#r2302013660
		outerContainerImage = "docker.io/kindest/base:v20250521-31a79fd4"
	)

	BeforeAll(func(ctx context.Context) {
		var err error
		systemdRunner, err = NewNestedContainerRunner(runner, outerContainerImage, false, containerName, localCacheDir, true)
		Expect(err).ToNot(HaveOccurred())
		for range 10 {
			state, _, err := systemdRunner.Run(getSystemStateScript)
			if err == nil {
				GinkgoLogr.Info("systemd started", "state", state)
				break
			}
			GinkgoLogr.Error(err, "systemctl state")
			time.Sleep(1 * time.Second)
		}
	})

	AfterAll(func(ctx context.Context) {
		// Cleanup: remove the container and the temporary script on the host.
		// Use || true to ensure cleanup doesn't fail the test
		runner.Run(fmt.Sprintf("docker rm -f %s 2>/dev/null || true", containerName))
	})

	When("installing nvidia-container-toolkit", Ordered, func() {
		BeforeAll(func(ctx context.Context) {

			_, _, err := toolkitInstaller.Install(systemdRunner)
			Expect(err).ToNot(HaveOccurred())

			output, _, err := systemdRunner.Run("nvidia-ctk --version")
			Expect(err).ToNot(HaveOccurred())
			GinkgoLogr.Info("using nvidia-ctk", "version", strings.TrimSpace(output))
		})

		AfterAll(func(ctx context.Context) {
			_, _, err := systemdRunner.Run("apt-get purge -y libnvidia-container* nvidia-container-toolkit*")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should load the nvidia-cdi-refresh.path unit", func(ctx context.Context) {
			_, _, err := systemdRunner.Run(nvidiaCdiRefreshPathActiveTemplate)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should load the nvidia-cdi-refresh.service unit", func(ctx context.Context) {
			_, _, err := systemdRunner.Run(nvidiaCdiRefreshServiceLoadedTemplate)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should generate the nvidia.yaml file", func(ctx context.Context) {
			_, _, err := systemdRunner.Run(nvidiaCdiRefreshFileExistsTemplate)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should refresh the nvidia.yaml file after upgrading the nvidia-container-toolkit", func(ctx context.Context) {
			_, _, err := systemdRunner.Run(nvidiaCdiRefreshUpgradeTemplate)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("installing nvidia-container-toolkit on a system with a degraded systemd", Ordered, func() {
		BeforeAll(func(ctx context.Context) {
			_, _, err := systemdRunner.Run(setSystemdDegradedScript)
			Expect(err).ToNot(HaveOccurred())

			_, _, err = systemdRunner.Run(getSystemStateScript)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("degraded"))
		})

		AfterAll(func(ctx context.Context) {
			_, _, err := systemdRunner.Run(fixSystemDegradedScript)
			Expect(err).ToNot(HaveOccurred())

			state, _, err := systemdRunner.Run(getSystemStateScript)
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.TrimSpace(state)).To(Equal("running"))
		})

		It("should load the nvidia-cdi-refresh.path unit", func(ctx context.Context) {
			_, _, err := systemdRunner.Run(nvidiaCdiRefreshPathActiveTemplate)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should load the nvidia-cdi-refresh.service unit", func(ctx context.Context) {
			_, _, err := systemdRunner.Run(nvidiaCdiRefreshServiceLoadedTemplate)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should generate the nvidia.yaml file", func(ctx context.Context) {
			_, _, err := systemdRunner.Run(nvidiaCdiRefreshFileExistsTemplate)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should generate the nvidia.yaml file", func(ctx context.Context) {
			_, _, err := systemdRunner.Run(nvidiaCdiRefreshFileExistsTemplate)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
