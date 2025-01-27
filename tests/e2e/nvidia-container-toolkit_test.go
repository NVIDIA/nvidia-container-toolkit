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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Integration tests for Docker runtime
var _ = Describe("docker", Ordered, func() {
	var s Script

	// Install the NVIDIA Container Toolkit
	BeforeAll(func(ctx context.Context) {
		switch {
		case sshKey == "":
			s = localRun{}
		default:
			s = remoteRun{sshKey, sshUser, remoteHost}
		}

		if installCTK {
			installer, err := newInstaller(dockerInstallTemplate, imageRepo+":"+imageTag, sshKey, sshUser, remoteHost)
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
			_, err := s.Run("docker pull ubuntu")
			Expect(err).ToNot(HaveOccurred())

			hostOutput, err = s.Run("nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should support NVIDIA_VISIBLE_DEVICES", func(ctx context.Context) {
			containerOutput, err := s.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=all ubuntu nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())
			Expect(containerOutput).To(Equal(hostOutput))
		})

		It("should support automatic CDI spec generation", func(ctx context.Context) {
			containerOutput, err := s.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all ubuntu nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())
			Expect(containerOutput).To(Equal(hostOutput))
		})

		It("should support the --gpus flag using the nvidia-container-runtime", func(ctx context.Context) {
			containerOutput, err := s.Run("docker run --rm -i --runtime=nvidia --gpus all ubuntu nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())
			Expect(containerOutput).To(Equal(hostOutput))
		})

		It("should support the --gpus flag using the nvidia-container-runtime-hook", func(ctx context.Context) {
			containerOutput, err := s.Run("docker run --rm -i --gpus all ubuntu nvidia-smi -L")
			Expect(err).ToNot(HaveOccurred())
			Expect(containerOutput).To(Equal(hostOutput))
		})
	})

	// A vectorAdd sample runs in a container with access to all GPUs.
	// The following should all produce the same result.
	When("Running the cuda-vectorAdd sample", Ordered, func() {
		BeforeAll(func(ctx context.Context) {
			_, err := s.Run("docker pull nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
		})

		var referenceOutput string

		It("should support NVIDIA_VISIBLE_DEVICES", func(ctx context.Context) {
			var err error
			referenceOutput, err = s.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=all nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())

			Expect(referenceOutput).To(ContainSubstring("Test PASSED"))
		})

		It("should support automatic CDI spec generation", func(ctx context.Context) {
			out2, err := s.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out2))
		})

		It("should support the --gpus flag using the nvidia-container-runtime", func(ctx context.Context) {
			out3, err := s.Run("docker run --rm -i --runtime=nvidia --gpus all nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out3))
		})

		It("should support the --gpus flag using the nvidia-container-runtime-hook", func(ctx context.Context) {
			out4, err := s.Run("docker run --rm -i --gpus all nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out4))
		})
	})

	// A deviceQuery sample runs in a container with access to all GPUs
	// The following should all produce the same result.
	When("Running the cuda-deviceQuery sample", Ordered, func() {
		BeforeAll(func(ctx context.Context) {
			_, err := s.Run("docker pull nvcr.io/nvidia/k8s/cuda-sample:devicequery-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
		})

		var referenceOutput string

		It("should support NVIDIA_VISIBLE_DEVICES", func(ctx context.Context) {
			var err error
			referenceOutput, err = s.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=all nvcr.io/nvidia/k8s/cuda-sample:devicequery-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())

			Expect(referenceOutput).To(ContainSubstring("Result = PASS"))
		})

		It("should support automatic CDI spec generation", func(ctx context.Context) {
			out2, err := s.Run("docker run --rm -i --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all nvcr.io/nvidia/k8s/cuda-sample:devicequery-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out2))
		})

		It("should support the --gpus flag using the nvidia-container-runtime", func(ctx context.Context) {
			out3, err := s.Run("docker run --rm -i --runtime=nvidia --gpus all nvcr.io/nvidia/k8s/cuda-sample:devicequery-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out3))
		})

		It("should support the --gpus flag using the nvidia-container-runtime-hook", func(ctx context.Context) {
			out4, err := s.Run("docker run --rm -i --gpus all nvcr.io/nvidia/k8s/cuda-sample:devicequery-cuda12.5.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(referenceOutput).To(Equal(out4))
		})
	})
})
