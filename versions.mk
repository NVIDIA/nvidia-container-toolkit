# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

LIB_NAME := nvidia-container-toolkit
LIB_VERSION := 1.11.0
LIB_TAG := rc.1

# Specify the nvidia-docker2 and nvidia-container-runtime package versions.
# Note: The tag is automatically specified to match LIB_TAG.
NVIDIA_DOCKER_VERSION := 2.11.0
NVIDIA_CONTAINER_RUNTIME_VERSION := 3.11.0

# Specify the expected libnvidia-container0 version for arm64-based ubuntu builds.
LIBNVIDIA_CONTAINER0_VERSION := 0.10.0+jetpack

CUDA_VERSION := 11.7.0
GOLANG_VERSION := 1.17.8

GIT_COMMIT ?= $(shell git describe --dirty --long --always 2> /dev/null || echo "")
