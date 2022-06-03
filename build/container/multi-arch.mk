# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

PUSH_ON_BUILD ?= false
DOCKER_BUILD_OPTIONS = --output=type=image,push=$(PUSH_ON_BUILD)
DOCKER_BUILD_PLATFORM_OPTIONS = --platform=linux/amd64,linux/arm64

REGCTL ?= regctl
$(PUSH_TARGETS): push-%:
	$(REGCTL) \
	        image copy \
	        $(IMAGE) $(OUT_IMAGE)

push-short:
	$(REGCTL) \
	        image copy \
	        $(IMAGE) $(OUT_IMAGE_NAME):$(OUT_IMAGE_VERSION)

# We only have x86_64 packages for centos7
build-centos7: DOCKER_BUILD_PLATFORM_OPTIONS = --platform=linux/amd64

# We only generate amd64 image for ubuntu18.04
build-ubuntu18.04: DOCKER_BUILD_PLATFORM_OPTIONS = --platform=linux/amd64

# We only generate a single image for packaging targets
build-packaging: DOCKER_BUILD_PLATFORM_OPTIONS = --platform=linux/amd64
