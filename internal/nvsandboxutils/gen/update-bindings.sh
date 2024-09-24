#!/bin/bash
# Copyright 2024 NVIDIA CORPORATION
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

# This file allows for the nvsandboxutils bindings to be updated using the tooling
# implemented in https://github.com/NVIDIA/go-nvml.
# To run this:
#   cd internal/nvsandboxutils
#   ./update-bindings.sh

set -e

BUILDIMAGE=bindings

docker build \
    --build-arg GOLANG_VERSION=1.22.1 \
    --build-arg C_FOR_GO_TAG=8eeee8c3b71f9c3c90c4a73db54ed08b0bba971d \
    -t ${BUILDIMAGE} \
    -f docker/Dockerfile.devel \
    https://github.com/NVIDIA/go-nvml.git


docker run --rm -ti \
		-e GOCACHE=/tmp/.cache/go \
		-e GOMODCACHE=/tmp/.cache/gomod \
    -v $(pwd):/nvsandboxutils \
    -w /nvsandboxutils \
    -u $(id -u):$(id -g) \
    ${BUILDIMAGE} \
    ./gen/generate-bindings.sh
