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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Integration tests for containerd drop-in config functionality
var _ = Describe("containerd", Ordered, ContinueOnFailure, func() {
	var (
		nestedContainerRunner Runner
		// docker.io/kindest/base:v20250521-31a79fd4 comes with
		// containerd v2.1.1
		outerContainerImage = "docker.io/kindest/base:v20250521-31a79fd4"
	)

	BeforeAll(func(ctx context.Context) {
		var err error
		nestedContainerRunner, err = NewNestedContainerRunner(runner, outerContainerImage, installCTK, imageName+":"+imageTag, testContainerName)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterAll(func(ctx context.Context) {
		runner.Run(fmt.Sprintf("docker rm -f %s 2>/dev/null || true", testContainerName)) //nolint:errcheck
	})

	BeforeEach(func(ctx context.Context) {
		_, _, err := nestedContainerRunner.Run(`
			# Clean up any previous drop-in configs
			rm -rf /etc/containerd/conf.d
			rm -rf /etc/containerd/custom.d
			mkdir -p /etc/containerd/conf.d
			
			# Remove any imports line from the config (reset to original state)
			if [ -f /etc/containerd/config.toml ]; then
				grep -v "^imports = " /etc/containerd/config.toml > /tmp/config.toml.tmp && mv /tmp/config.toml.tmp /etc/containerd/config.toml || true
			fi

			# Restart containerd to pick up the clean config
			systemctl restart containerd
			sleep 2
		`)
		Expect(err).ToNot(HaveOccurred(), "Failed to reset containerd configuration")
	})

	When("configuring containerd on a Kubernetes node", func() {
		It("should add NVIDIA runtime using drop-in config without modifying the main config", func(ctx context.Context) {
			// Use the installer to configure containerd with drop-in support
			installer, err := NewToolkitInstaller(
				WithRunner(nestedContainerRunner),
				WithImage(imageName+":"+imageTag),
				WithTemplate(containerdInstallTemplate),
				WithAdditionalFlags("--enable-cdi-in-runtime --set-as-default"),
			)
			Expect(err).ToNot(HaveOccurred())

			err = installer.Install()
			Expect(err).ToNot(HaveOccurred(), "Failed to install toolkit for containerd")

			// Verify the drop-in config was created
			output, _, err := nestedContainerRunner.Run(`cat /etc/containerd/conf.d/99-nvidia.toml`)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(ContainSubstring(`nvidia`))
			Expect(output).To(ContainSubstring(`nvidia-cdi`))
			Expect(output).To(ContainSubstring(`nvidia-legacy`))

			// Restart containerd to apply config
			_, _, err = nestedContainerRunner.Run(`systemctl restart containerd`)
			Expect(err).ToNot(HaveOccurred())

			// Verify containerd loaded the config correctly
			output, _, err = nestedContainerRunner.Run(`containerd config dump`)
			Expect(err).ToNot(HaveOccurred())
			// Verify imports section is in the merged config
			Expect(output).To(ContainSubstring(`imports = ['/etc/containerd/conf.d/*.toml']`))
			// Check for either single or double quotes as containerd v3 uses single quotes
			Expect(output).To(SatisfyAny(
				ContainSubstring(`default_runtime_name = "nvidia"`),
				ContainSubstring(`default_runtime_name = 'nvidia'`),
			))
			Expect(output).To(ContainSubstring(`enable_cdi = true`))
			// Verify Kubernetes settings are still present
			Expect(output).To(ContainSubstring(`SystemdCgroup = true`))
			// Verify NVIDIA runtime was added
			Expect(output).To(SatisfyAny(
				ContainSubstring(`[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia]`),
				ContainSubstring(`[plugins.'io.containerd.cri.v1.runtime'.containerd.runtimes.nvidia]`),
			))
			Expect(output).To(ContainSubstring(`BinaryName = `)) // The NVIDIA container runtime binary
		})
	})

	When("containerd already has a custom default runtime configured", func() {
		It("should preserve the existing default runtime when --set-as-default=false is specified", func(ctx context.Context) {
			// Create a containerd config with a custom runtime
			_, _, err := nestedContainerRunner.Run(`
				rm -f /etc/containerd/config.toml
				rm -rf /etc/containerd/conf.d
				mkdir -p /etc/containerd/conf.d
				
				cat > /etc/containerd/config.toml <<'EOF'
version = 2

[plugins]
  [plugins."io.containerd.cri.v1.runtime"]
    [plugins."io.containerd.cri.v1.runtime".containerd]
      default_runtime_name = "custom-runtime"
      
      [plugins."io.containerd.cri.v1.runtime".containerd.runtimes]
        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.custom-runtime]
          runtime_type = "io.containerd.runc.v2"
          
          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.custom-runtime.options]
            BinaryName = "/usr/bin/custom-runtime"
            SystemdCgroup = true
            CustomOption = "custom-value"
            
        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"
          
          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
EOF
			`)
			Expect(err).ToNot(HaveOccurred())

			// Configure containerd with drop-in config (explicitly not setting as default)
			installer, err := NewToolkitInstaller(
				WithRunner(nestedContainerRunner),
				WithImage(imageName+":"+imageTag),
				WithTemplate(containerdInstallTemplate),
				WithAdditionalFlags("--enable-cdi-in-runtime --set-as-default=false"),
			)
			Expect(err).ToNot(HaveOccurred())

			err = installer.Install()
			Expect(err).ToNot(HaveOccurred(), "Failed to install toolkit for containerd")

			// Verify the drop-in config was created
			output, _, err := nestedContainerRunner.Run(`cat /etc/containerd/conf.d/99-nvidia.toml`)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(ContainSubstring(`nvidia`))

			// Restart containerd
			_, _, err = nestedContainerRunner.Run(`systemctl restart containerd`)
			Expect(err).ToNot(HaveOccurred())

			// Verify containerd merged configs correctly
			output, _, err = nestedContainerRunner.Run(`containerd config dump`)
			Expect(err).ToNot(HaveOccurred())
			// Default runtime should still be custom-runtime (handle both quote styles)
			Expect(output).To(SatisfyAny(
				ContainSubstring(`default_runtime_name = "custom-runtime"`),
				ContainSubstring(`default_runtime_name = 'custom-runtime'`),
			))
			// Custom runtime should be preserved with all its options
			Expect(output).To(MatchRegexp(`(?s)custom-runtime.*CustomOption.*custom-value`))
			// NVIDIA runtimes should be added
			Expect(output).To(ContainSubstring(`nvidia`))
			Expect(output).To(ContainSubstring(`nvidia-cdi`))
			Expect(output).To(ContainSubstring(`nvidia-legacy`))
			// CDI should be enabled
			Expect(output).To(ContainSubstring(`enable_cdi = true`))
		})
	})

	When("containerd has multiple custom runtimes and plugins configured", func() {
		It("should add NVIDIA runtime alongside existing runtimes like kata", func(ctx context.Context) {
			// Add some custom settings to the existing kindest/base config
			_, _, err := nestedContainerRunner.Run(`
				rm -rf /etc/containerd/conf.d
				mkdir -p /etc/containerd/conf.d
				
				# Add a kata runtime to the existing config
				cat >> /etc/containerd/config.toml <<'EOF'

# Custom kata runtime
[plugins."io.containerd.cri.v1.runtime".containerd.runtimes.kata]
  runtime_type = "io.containerd.kata.v2"
  
  [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.kata.options]
    ConfigPath = "/etc/kata-containers/configuration.toml"
EOF
			`)
			Expect(err).ToNot(HaveOccurred())

			// Verify kata runtime was added
			output, _, err := nestedContainerRunner.Run(`systemctl restart containerd && sleep 2 && containerd config dump | grep -A5 kata`)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(ContainSubstring(`kata`))

			// Configure containerd with drop-in config and set NVIDIA as default
			installer, err := NewToolkitInstaller(
				WithRunner(nestedContainerRunner),
				WithImage(imageName+":"+imageTag),
				WithTemplate(containerdInstallTemplate),
				WithAdditionalFlags("--enable-cdi-in-runtime --set-as-default"),
			)
			Expect(err).ToNot(HaveOccurred())

			err = installer.Install()
			Expect(err).ToNot(HaveOccurred(), "Failed to install toolkit for containerd")

			// Restart containerd
			_, _, err = nestedContainerRunner.Run(`systemctl restart containerd`)
			Expect(err).ToNot(HaveOccurred())

			// Verify the final configuration
			output, _, err = nestedContainerRunner.Run(`containerd config dump`)
			Expect(err).ToNot(HaveOccurred())

			// Verify NVIDIA is now the default runtime
			Expect(output).To(SatisfyAny(
				ContainSubstring(`default_runtime_name = "nvidia"`),
				ContainSubstring(`default_runtime_name = 'nvidia'`),
			))

			// Verify kata runtime is still present (our custom runtime was preserved)
			Expect(output).To(ContainSubstring(`kata`))
			Expect(output).To(ContainSubstring(`io.containerd.kata.v2`))

			// Verify NVIDIA runtimes were added with correct settings
			Expect(output).To(ContainSubstring(`nvidia`))
			Expect(output).To(ContainSubstring(`nvidia-cdi`))
			Expect(output).To(ContainSubstring(`nvidia-legacy`))
			Expect(output).To(ContainSubstring(`enable_cdi = true`))
		})
	})

	When("using containerd with version 3 configuration format", func() {
		It("should correctly add NVIDIA runtime to v3 config structure", func(ctx context.Context) {
			// Create a v3 containerd config
			_, _, err := nestedContainerRunner.Run(`
				rm -f /etc/containerd/config.toml
				rm -rf /etc/containerd/conf.d
				mkdir -p /etc/containerd/conf.d
				
				cat > /etc/containerd/config.toml <<'EOF'
version = 3

[plugins]
  [plugins."io.containerd.cri.v1.runtime"]
    [plugins."io.containerd.cri.v1.runtime".containerd]
      default_runtime_name = "runc"
      
      [plugins."io.containerd.cri.v1.runtime".containerd.runtimes]
        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"
          
          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
            SystemdCgroup = true
EOF
			`)
			Expect(err).ToNot(HaveOccurred())

			// Configure containerd with drop-in config
			installer, err := NewToolkitInstaller(
				WithRunner(nestedContainerRunner),
				WithImage(imageName+":"+imageTag),
				WithTemplate(containerdInstallTemplate),
				WithAdditionalFlags("--enable-cdi-in-runtime"),
			)
			Expect(err).ToNot(HaveOccurred())

			err = installer.Install()
			Expect(err).ToNot(HaveOccurred(), "Failed to install toolkit for containerd")

			// Verify the drop-in config uses v3 format
			output, _, err := nestedContainerRunner.Run(`cat /etc/containerd/conf.d/99-nvidia.toml`)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(ContainSubstring(`version = 3`))
			Expect(output).To(ContainSubstring(`plugins."io.containerd.cri.v1.runtime"`))

			// Restart containerd
			_, _, err = nestedContainerRunner.Run(`systemctl restart containerd`)
			Expect(err).ToNot(HaveOccurred())

			// Verify containerd loaded v3 config correctly
			output, _, err = nestedContainerRunner.Run(`containerd config dump`)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(ContainSubstring(`version = 3`))
			Expect(output).To(ContainSubstring(`nvidia`))
			Expect(output).To(ContainSubstring(`enable_cdi = true`))
		})
	})

	When("containerd already uses import directives for modular configuration", func() {
		It("should preserve existing imports when adding NVIDIA drop-in config", func(ctx context.Context) {
			// Create a containerd config with existing imports
			_, _, err := nestedContainerRunner.Run(`
				rm -f /etc/containerd/config.toml
				rm -rf /etc/containerd/conf.d
				rm -rf /etc/containerd/custom.d
				mkdir -p /etc/containerd/conf.d
				mkdir -p /etc/containerd/custom.d
				
			# Create a custom config that will be imported
			# Use the correct plugin path for containerd v2/v3
			cat > /etc/containerd/custom.d/10-custom.toml <<'EOF'
version = 2

[plugins]
  [plugins."io.containerd.cri.v1.images"]
    [plugins."io.containerd.cri.v1.images".registry]
      [plugins."io.containerd.cri.v1.images".registry.mirrors]
        [plugins."io.containerd.cri.v1.images".registry.mirrors."myregistry.io"]
          endpoint = ["https://myregistry.io"]
EOF
				
			# Create main config with existing imports
			# Use the correct plugin path for containerd v2/v3
			cat > /etc/containerd/config.toml <<'EOF'
imports = ["/etc/containerd/custom.d/*.toml"]
version = 2

[plugins]
  [plugins."io.containerd.cri.v1.runtime"]
    [plugins."io.containerd.cri.v1.runtime".containerd]
      default_runtime_name = "runc"
      
      [plugins."io.containerd.cri.v1.runtime".containerd.runtimes]
        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"
          
          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc.options]
            BinaryName = "/usr/bin/runc"
EOF
			`)
			Expect(err).ToNot(HaveOccurred())

			// Verify containerd can load the custom import before installer
			_, _, err = nestedContainerRunner.Run(`systemctl restart containerd && sleep 2 && containerd config dump | grep -i myregistry`)
			Expect(err).ToNot(HaveOccurred())

			// Configure containerd with drop-in config
			installer, err := NewToolkitInstaller(
				WithRunner(nestedContainerRunner),
				WithImage(imageName+":"+imageTag),
				WithTemplate(containerdInstallTemplate),
			)
			Expect(err).ToNot(HaveOccurred())

			err = installer.Install()
			Expect(err).ToNot(HaveOccurred(), "Failed to install toolkit for containerd")

			// Restart containerd to load the new configuration
			_, _, err = nestedContainerRunner.Run(`systemctl restart containerd`)
			Expect(err).ToNot(HaveOccurred())

			// Verify all configs are merged correctly using containerd config dump
			var output string
			output, _, err = nestedContainerRunner.Run(`containerd config dump`)
			Expect(err).ToNot(HaveOccurred())

			// From custom import
			Expect(output).To(ContainSubstring(`myregistry.io`))
			// From NVIDIA drop-in
			Expect(output).To(ContainSubstring(`nvidia`))
			// From main config
			Expect(output).To(ContainSubstring(`runc`))
		})
	})

	When("running a sample workload after configuring NVIDIA runtime", func() {
		It("should successfully run containers using the NVIDIA runtime", func(ctx context.Context) {
			// Configure containerd with NVIDIA runtime using drop-in config
			installer, err := NewToolkitInstaller(
				WithRunner(nestedContainerRunner),
				WithImage(imageName+":"+imageTag),
				WithTemplate(containerdInstallTemplate),
				WithAdditionalFlags("--enable-cdi-in-runtime --set-as-default"),
			)
			Expect(err).ToNot(HaveOccurred())

			err = installer.Install()
			Expect(err).ToNot(HaveOccurred(), "Failed to install toolkit for containerd")

			// Restart containerd to apply the new configuration
			_, _, err = nestedContainerRunner.Run(`systemctl restart containerd`)
			Expect(err).ToNot(HaveOccurred())

			// Pull a test image
			output, stderr, err := nestedContainerRunner.Run(`
				ctr image pull docker.io/library/busybox:latest
			`)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull image: %s\nSTDERR: %s", output, stderr)

			// Run a container with NVIDIA runtime
			// Note: We use native snapshotter because overlay doesn't work in nested containers
			output, stderr, err = nestedContainerRunner.Run(`
				ctr run --rm --snapshotter native docker.io/library/busybox:latest test-nvidia echo "Hello from NVIDIA runtime"
			`)
			Expect(err).ToNot(HaveOccurred(), "Failed to run container with NVIDIA runtime: %s\nSTDERR: %s", output, stderr)
			Expect(output).To(ContainSubstring("Hello from NVIDIA runtime"))
		})
	})
})
