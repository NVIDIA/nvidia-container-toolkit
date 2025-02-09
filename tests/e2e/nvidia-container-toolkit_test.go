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
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Integration tests for Docker runtime
var _ = Describe("docker", Ordered, ContinueOnFailure, func() {
	var r Runner

	// Install the NVIDIA Container Toolkit
	BeforeAll(func(ctx context.Context) {
		r = NewRunner(
			WithHost(host),
			WithPort(sshPort),
			WithSshKey(sshKey),
			WithSshUser(sshUser),
		)
		if installCTK {
			installer, err := NewToolkitInstaller(
				WithRunner(r),
				WithImage(image),
				WithTemplate(dockerInstallTemplate),
			)
			Expect(err).ToNot(HaveOccurred())
			err = installer.Install()
			Expect(err).ToNot(HaveOccurred())
		}
	})

	// GPUs are accessible in a container: Running nvidia-smi -L inside the
	// container shows the same output inside the container as outside the
	// container. This means that the following commands must all produce
	// the same output
	When("running nvidia-smi -L", Ordered, func() {
		var hostOutput string

		BeforeAll(func(ctx context.Context) {
			_, _, err := r.Run("docker pull ubuntu")
			Expect(err).ToNot(HaveOccurred())

			hostOutput, _, err = r.Run("nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should support NVIDIA_VISIBLE_DEVICES", func(ctx context.Context) {
			containerOutput, _, err := r.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=all ubuntu nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())
			Expect(containerOutput).To(Equal(hostOutput))
		})

		It("should support automatic CDI spec generation", func(ctx context.Context) {
			containerOutput, _, err := r.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all ubuntu nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())
			Expect(containerOutput).To(Equal(hostOutput))
		})

		It("should support automatic CDI spec generation with the --gpus flag", func(ctx context.Context) {
			containerOutput, _, err := r.Run("docker run --rm -i --gpus=all --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all ubuntu nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())
			Expect(containerOutput).To(Equal(hostOutput))
		})

		It("should support the --gpus flag using the nvidia-container-runtime", func(ctx context.Context) {
			containerOutput, _, err := r.Run("docker run --rm -i --runtime=nvidia --gpus all ubuntu nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())
			Expect(containerOutput).To(Equal(hostOutput))
		})

		It("should support the --gpus flag using the nvidia-container-runtime-hook", func(ctx context.Context) {
			containerOutput, _, err := r.Run("docker run --rm -i --gpus all ubuntu nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())
			Expect(containerOutput).To(Equal(hostOutput))
		})
	})

	// A vectorAdd sample runs in a container with access to all GPUs.
	// The following should all produce the same result.
	When("Running the cuda-vectorAdd sample", Ordered, func() {
		BeforeAll(func(ctx context.Context) {
			_, _, err := r.Run("docker pull nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
		})

		var referenceOutput string

		It("should support NVIDIA_VISIBLE_DEVICES", func(ctx context.Context) {
			var err error
			referenceOutput, _, err = r.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=all nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())

			Expect(referenceOutput).To(ContainSubstring("Test PASSED"))
		})

		It("should support automatic CDI spec generation", func(ctx context.Context) {
			out2, _, err := r.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out2))
		})

		It("should support the --gpus flag using the nvidia-container-runtime", func(ctx context.Context) {
			out3, _, err := r.Run("docker run --rm -i --runtime=nvidia --gpus all nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out3))
		})

		It("should support the --gpus flag using the nvidia-container-runtime-hook", func(ctx context.Context) {
			out4, _, err := r.Run("docker run --rm -i --gpus all nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out4))
		})
	})

	// A deviceQuery sample runs in a container with access to all GPUs
	// The following should all produce the same result.
	When("Running the cuda-deviceQuery sample", Ordered, func() {
		BeforeAll(func(ctx context.Context) {
			_, _, err := r.Run("docker pull nvcr.io/nvidia/k8s/cuda-sample:devicequery-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
		})

		var referenceOutput string

		It("should support NVIDIA_VISIBLE_DEVICES", func(ctx context.Context) {
			var err error
			referenceOutput, _, err = r.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=all nvcr.io/nvidia/k8s/cuda-sample:devicequery-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())

			Expect(referenceOutput).To(ContainSubstring("Result = PASS"))
		})

		It("should support automatic CDI spec generation", func(ctx context.Context) {
			out2, _, err := r.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all nvcr.io/nvidia/k8s/cuda-sample:devicequery-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out2))
		})

		It("should support the --gpus flag using the nvidia-container-runtime", func(ctx context.Context) {
			out3, _, err := r.Run("docker run --rm -i --runtime=nvidia --gpus all nvcr.io/nvidia/k8s/cuda-sample:devicequery-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out3))
		})

		It("should support the --gpus flag using the nvidia-container-runtime-hook", func(ctx context.Context) {
			out4, _, err := r.Run("docker run --rm -i --gpus all nvcr.io/nvidia/k8s/cuda-sample:devicequery-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out4))
		})
	})

	Describe("CUDA Forward compatibility", Ordered, func() {
		BeforeAll(func(ctx context.Context) {
			_, _, err := r.Run("docker pull nvcr.io/nvidia/cuda:12.8.0-base-ubi8")
			Expect(err).ToNot(HaveOccurred())
		})

		BeforeAll(func(ctx context.Context) {
			compatOutput, _, err := r.Run("docker run --rm -i -e NVIDIA_VISIBLE_DEVICES=void nvcr.io/nvidia/cuda:12.8.0-base-ubi8 bash -c \"ls /usr/local/cuda/compat/libcuda.*.*\"")
			Expect(err).ToNot(HaveOccurred())
			Expect(compatOutput).ToNot(BeEmpty())
			compatDriverVersion := strings.TrimPrefix(filepath.Base(compatOutput), "libcuda.so.")
			compatMajor := strings.SplitN(compatDriverVersion, ".", 2)[0]

			driverOutput, _, err := r.Run("nvidia-smi -q | grep \"Driver Version\"")
			Expect(err).ToNot(HaveOccurred())
			parts := strings.SplitN(driverOutput, ":", 2)
			Expect(parts).To(HaveLen(2))

			hostDriverVersion := strings.TrimSpace(parts[1])
			Expect(hostDriverVersion).ToNot(BeEmpty())
			driverMajor := strings.SplitN(hostDriverVersion, ".", 2)[0]

			if driverMajor >= compatMajor {
				GinkgoLogr.Info("CUDA Forward Compatibility tests require an older driver version", "hostDriverVersion", hostDriverVersion, "compatDriverVersion", compatDriverVersion)
				Skip("CUDA Forward Compatibility tests require an older driver version")
			}
		})

		It("should work with the nvidia runtime in legacy mode", func(ctx context.Context) {
			ldconfigOut, _, err := r.Run("docker run --rm -i -e NVIDIA_DISABLE_REQUIRE=true --runtime=nvidia --gpus all nvcr.io/nvidia/cuda:12.8.0-base-ubi8 bash -c \"ldconfig -p | grep libcuda.so.1\"")
			Expect(err).ToNot(HaveOccurred())
			Expect(ldconfigOut).To(ContainSubstring("/usr/local/cuda/compat"))
		})

		It("should work with the nvidia runtime in CDI mode", func(ctx context.Context) {
			ldconfigOut, _, err := r.Run("docker run --rm -i -e NVIDIA_DISABLE_REQUIRE=true  --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all nvcr.io/nvidia/cuda:12.8.0-base-ubi8 bash -c \"ldconfig -p | grep libcuda.so.1\"")
			Expect(err).ToNot(HaveOccurred())
			Expect(ldconfigOut).To(ContainSubstring("/usr/local/cuda/compat"))
		})

		It("should NOT work with nvidia-container-runtime-hook", func(ctx context.Context) {
			ldconfigOut, _, err := r.Run("docker run --rm -i -e NVIDIA_DISABLE_REQUIRE=true --runtime=runc --gpus all nvcr.io/nvidia/cuda:12.8.0-base-ubi8 bash -c \"ldconfig -p | grep libcuda.so.1\"")
			Expect(err).ToNot(HaveOccurred())
			Expect(ldconfigOut).To(ContainSubstring("/usr/lib64"))
		})
	})
})
