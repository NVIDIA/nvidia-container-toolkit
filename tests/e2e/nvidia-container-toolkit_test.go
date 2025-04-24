/*
 * SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
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
	// GPUs are accessible in a container: Running nvidia-smi -L inside the
	// container shows the same output inside the container as outside the
	// container. This means that the following commands must all produce
	// the same output
	When("running nvidia-smi -L", Ordered, func() {
		var hostOutput string
		var err error

		BeforeAll(func(ctx context.Context) {
			hostOutput, _, err = runner.Run("nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should support NVIDIA_VISIBLE_DEVICES", func(ctx context.Context) {
			By("Running docker run with --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=all")
			containerOutput, _, err := runner.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=all ubuntu nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())
			Expect(containerOutput).To(Equal(hostOutput))
		})

		It("should support automatic CDI spec generation", func(ctx context.Context) {
			By("Running docker run with --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all")
			containerOutput, _, err := runner.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all ubuntu nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())
			Expect(containerOutput).To(Equal(hostOutput))
		})

		It("should support automatic CDI spec generation with the --gpus flag", func(ctx context.Context) {
			By("Running docker run with --gpus=all --runtime=nvidia --gpus all")
			containerOutput, _, err := runner.Run("docker run --rm -i --gpus=all --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all ubuntu nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())
			Expect(containerOutput).To(Equal(hostOutput))
		})

		It("should support the --gpus flag using the nvidia-container-runtime", func(ctx context.Context) {
			By("Running docker run with --runtime=nvidia --gpus all")
			containerOutput, _, err := runner.Run("docker run --rm -i --runtime=nvidia --gpus all ubuntu nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())
			Expect(containerOutput).To(Equal(hostOutput))
		})

		It("should support the --gpus flag using the nvidia-container-runtime-hook", func(ctx context.Context) {
			By("Running docker run with --gpus all")
			containerOutput, _, err := runner.Run("docker run --rm -i --gpus all ubuntu nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())
			Expect(containerOutput).To(Equal(hostOutput))
		})
	})

	// A vectorAdd sample runs in a container with access to all GPUs.
	// The following should all produce the same result.
	When("Running the cuda-vectorAdd sample", Ordered, func() {
		var referenceOutput string

		It("should support NVIDIA_VISIBLE_DEVICES", func(ctx context.Context) {
			By("Running docker run with --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=all")
			var err error
			referenceOutput, _, err = runner.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=all nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())

			Expect(referenceOutput).To(ContainSubstring("Test PASSED"))
		})

		It("should support automatic CDI spec generation", func(ctx context.Context) {
			By("Running docker run with --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all")
			out2, _, err := runner.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out2))
		})

		It("should support the --gpus flag using the nvidia-container-runtime", func(ctx context.Context) {
			By("Running docker run with --runtime=nvidia --gpus all")
			out3, _, err := runner.Run("docker run --rm -i --runtime=nvidia --gpus all nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out3))
		})

		It("should support the --gpus flag using the nvidia-container-runtime-hook", func(ctx context.Context) {
			By("Running docker run with --gpus all")
			out4, _, err := runner.Run("docker run --rm -i --gpus all nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out4))
		})
	})

	// A deviceQuery sample runs in a container with access to all GPUs
	// The following should all produce the same result.
	When("Running the cuda-deviceQuery sample", Ordered, func() {
		BeforeAll(func(ctx context.Context) {
			_, _, err := runner.Run("docker pull nvcr.io/nvidia/k8s/cuda-sample:devicequery-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
		})

		var referenceOutput string

		It("should support NVIDIA_VISIBLE_DEVICES", func(ctx context.Context) {
			By("Running docker run with --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=all")
			var err error
			referenceOutput, _, err = runner.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=all nvcr.io/nvidia/k8s/cuda-sample:devicequery-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(ContainSubstring("Result = PASS"))
		})

		It("should support automatic CDI spec generation", func(ctx context.Context) {
			By("Running docker run with --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all")
			out2, _, err := runner.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all nvcr.io/nvidia/k8s/cuda-sample:devicequery-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out2))
		})

		It("should support the --gpus flag using the nvidia-container-runtime", func(ctx context.Context) {
			By("Running docker run with --runtime=nvidia --gpus all")
			out3, _, err := runner.Run("docker run --rm -i --runtime=nvidia --gpus all nvcr.io/nvidia/k8s/cuda-sample:devicequery-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out3))
		})

		It("should support the --gpus flag using the nvidia-container-runtime-hook", func(ctx context.Context) {
			By("Running docker run with --gpus all")
			out4, _, err := runner.Run("docker run --rm -i --gpus all nvcr.io/nvidia/k8s/cuda-sample:devicequery-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out4))
		})
	})

	When("Testing CUDA Forward compatibility", Ordered, func() {
		BeforeAll(func(ctx context.Context) {
			compatOutput, _, err := runner.Run("docker run --rm -i -e NVIDIA_VISIBLE_DEVICES=void nvcr.io/nvidia/cuda:12.8.0-base-ubi8 bash -c \"ls /usr/local/cuda/compat/libcuda.*.*\"")
			Expect(err).ToNot(HaveOccurred())
			Expect(compatOutput).ToNot(BeEmpty())

			compatDriverVersion := strings.TrimPrefix(filepath.Base(compatOutput), "libcuda.so.")
			compatMajor := strings.SplitN(compatDriverVersion, ".", 2)[0]

			driverOutput, _, err := runner.Run("nvidia-smi -q | grep \"Driver Version\"")
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
			By("Running docker run with -e NVIDIA_DISABLE_REQUIRE=true --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=all")
			ldconfigOut, _, err := runner.Run("docker run --rm -i -e NVIDIA_DISABLE_REQUIRE=true --runtime=nvidia --gpus all nvcr.io/nvidia/cuda:12.8.0-base-ubi8 bash -c \"ldconfig -p | grep libcuda.so.1\"")
			Expect(err).ToNot(HaveOccurred())
			Expect(ldconfigOut).To(ContainSubstring("/usr/local/cuda/compat"))
		})

		It("should work with the nvidia runtime in CDI mode", func(ctx context.Context) {
			By("Running docker run with -e NVIDIA_DISABLE_REQUIRE=true --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all")
			ldconfigOut, _, err := runner.Run("docker run --rm -i -e NVIDIA_DISABLE_REQUIRE=true  --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all nvcr.io/nvidia/cuda:12.8.0-base-ubi8 bash -c \"ldconfig -p | grep libcuda.so.1\"")
			Expect(err).ToNot(HaveOccurred())
			Expect(ldconfigOut).To(ContainSubstring("/usr/local/cuda/compat"))
		})

		It("should NOT work with nvidia-container-runtime-hook", func(ctx context.Context) {
			By("Running docker run with -e NVIDIA_DISABLE_REQUIRE=true --gpus all")
			ldconfigOut, _, err := runner.Run("docker run --rm -i -e NVIDIA_DISABLE_REQUIRE=true --runtime=runc --gpus all nvcr.io/nvidia/cuda:12.8.0-base-ubi8 bash -c \"ldconfig -p | grep libcuda.so.1\"")
			Expect(err).ToNot(HaveOccurred())
			Expect(ldconfigOut).To(ContainSubstring("/usr/lib64"))
		})
	})
})
