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
	"html/template"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	nvidiaCdiRefreshDegradedSystemdTemplate = `
	# Read the TMPDIR
	TMPDIR=$(cat /tmp/ctk_e2e_temp_dir.txt)
	export TMPDIR

	# uninstall the nvidia-container-toolkit
	apt-get remove -y nvidia-container-toolkit nvidia-container-toolkit-base libnvidia-container-tools libnvidia-container1
	apt-get autoremove -y

	# Remove the cdi file if it exists
	if [ -f /var/run/cdi/nvidia.yaml ]; then
		rm -f /var/run/cdi/nvidia.yaml
	fi

	# Stop the nvidia-cdi-refresh.path and nvidia-cdi-refresh.service units
	systemctl stop nvidia-cdi-refresh.path
	systemctl stop nvidia-cdi-refresh.service
	
	# Reload the systemd daemon
	systemctl daemon-reload
	
	# Start the dummy service to force systemd to enter a degraded state
	cat <<EOF > /etc/systemd/system/dummy.service
[Unit]
Description=Dummy systemd service

[Service]
Type=oneshot
ExecStart=/usr/bin/sh -c "exit 0"
EOF

	# We know the dummy service will fail, so we can ignore the error
	systemctl start dummy.service 2>/dev/null || true

	# Install the nvidia-container-toolkit
	dpkg -i ${TMPDIR}/packages/ubuntu18.04/amd64/libnvidia-container1_*_amd64.deb ${TMPDIR}/packages/ubuntu18.04/amd64/nvidia-container-toolkit-base_*_amd64.deb ${TMPDIR}/packages/ubuntu18.04/amd64/libnvidia-container-tools_*_amd64.deb
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
		nestedContainerRunner Runner
		// TODO(@ArangoGutierrez): https://github.com/NVIDIA/nvidia-container-toolkit/pull/1235/files#r2302013660
		outerContainerImage = "docker.io/kindest/base:v20250521-31a79fd4"
	)

	BeforeAll(func(ctx context.Context) {
		var err error
		nestedContainerRunner, err = NewNestedContainerRunner(runner, outerContainerImage, installCTK, imageName+":"+imageTag, testContainerName)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterAll(func(ctx context.Context) {
		// Cleanup: remove the container and the temporary script on the host.
		// Use || true to ensure cleanup doesn't fail the test
		runner.Run(fmt.Sprintf("docker rm -f %s 2>/dev/null || true", testContainerName)) //nolint:errcheck
	})

	When("installing nvidia-container-toolkit", Ordered, func() {
		It("should load the nvidia-cdi-refresh.path unit", func(ctx context.Context) {
			_, _, err := nestedContainerRunner.Run(nvidiaCdiRefreshPathActiveTemplate)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should load the nvidia-cdi-refresh.service unit", func(ctx context.Context) {
			_, _, err := nestedContainerRunner.Run(nvidiaCdiRefreshServiceLoadedTemplate)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should generate the nvidia.yaml file", func(ctx context.Context) {
			_, _, err := nestedContainerRunner.Run(nvidiaCdiRefreshFileExistsTemplate)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should refresh the nvidia.yaml file after upgrading the nvidia-container-toolkit", func(ctx context.Context) {
			_, _, err := nestedContainerRunner.Run(nvidiaCdiRefreshUpgradeTemplate)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("installing nvidia-container-toolkit on a system with a degraded systemd", Ordered, func() {
		BeforeAll(func(ctx context.Context) {
			tmpl, err := template.New("nvidiaCdiRefreshDegradedSystemd").Parse(nvidiaCdiRefreshDegradedSystemdTemplate)
			Expect(err).ToNot(HaveOccurred())

			var nvidiaCdiRefreshDegradedSystemd strings.Builder
			err = tmpl.Execute(&nvidiaCdiRefreshDegradedSystemd, struct {
				ToolkitImage string
			}{
				ToolkitImage: imageName + ":" + imageTag,
			})
			Expect(err).ToNot(HaveOccurred())

			_, _, err = nestedContainerRunner.Run(nvidiaCdiRefreshDegradedSystemd.String())
			Expect(err).ToNot(HaveOccurred())
		})

		It("should load the nvidia-cdi-refresh.path unit", func(ctx context.Context) {
			_, _, err := nestedContainerRunner.Run(nvidiaCdiRefreshPathActiveTemplate)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should load the nvidia-cdi-refresh.service unit", func(ctx context.Context) {
			_, _, err := nestedContainerRunner.Run(nvidiaCdiRefreshServiceLoadedTemplate)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should generate the nvidia.yaml file", func(ctx context.Context) {
			_, _, err := nestedContainerRunner.Run(nvidiaCdiRefreshFileExistsTemplate)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should generate the nvidia.yaml file", func(ctx context.Context) {
			_, _, err := nestedContainerRunner.Run(nvidiaCdiRefreshFileExistsTemplate)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
