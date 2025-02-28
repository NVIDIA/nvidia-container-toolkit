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
ARCH ?= $(shell uname -m)
DOCKER_BUILD_PLATFORM_OPTIONS = --platform=linux/$(ARCH)

ifeq ($(PUSH_ON_BUILD),true)
DOCKER_BUILD_OPTIONS = --output=type=image,push=$(PUSH_ON_BUILD)
$(BUILD_TARGETS): build-%: image-%
	$(DOCKER) push "$(IMAGE)"
else
$(BUILD_TARGETS): build-%: image-%
endif

# For the default distribution we also retag the image.
# Note: This needs to be updated for multi-arch images.
ifeq ($(IMAGE_TAG),$(VERSION)-$(DIST))
$(DEFAULT_PUSH_TARGET):
	$(DOCKER) image inspect $(IMAGE) > /dev/null || $(DOCKER) pull $(IMAGE)
	$(DOCKER) tag $(IMAGE) $(subst :$(IMAGE_TAG),:$(VERSION),$(IMAGE))
endif
