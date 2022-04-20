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

WORKFLOW ?= nvidia-docker

DISTRIBUTIONS := ubuntu18.04 centos8

IMAGE_TARGETS := $(patsubst %,image-%, $(DISTRIBUTIONS))
RUN_TARGETS := $(patsubst %,run-%, $(DISTRIBUTIONS))
RELEASE_TARGETS := $(patsubst %,release-%, $(DISTRIBUTIONS))
LOCAL_TARGETS := $(patsubst %,local-%, $(DISTRIBUTIONS))

.PHONY: $(IMAGE_TARGETS) $(RUN_TARGETS)

image-%: DOCKERFILE = docker/$(*)/Dockerfile

images: $(IMAGE_TARGETS)
$(IMAGE_TARGETS): image-%: $(DOCKERFILE)
	docker build ${PLATFORM_ARGS} \
    --build-arg WORKFLOW="$(WORKFLOW)" \
    -t nvidia-container-toolkit-repo-test:$(*) \
    -f $(DOCKERFILE) \
    $(shell dirname $(DOCKERFILE))


%-ubuntu18.04: ARCH ?= amd64
%-centos8: ARCH ?= x86_64

PLATFORM_ARGS = --platform=linux/${ARCH}

RELEASE_TEST_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
PROJECT_ROOT := $(RELEASE_TEST_DIR)/../..

LOCAL_PACKAGE_ROOT := $(PROJECT_ROOT)/dist

local-%: DIST = $(*)
local-%: LOCAL_REPO_ARGS = -v $(LOCAL_PACKAGE_ROOT)/$(DIST)/$(ARCH):/local-repository
$(LOCAL_TARGETS): local-%: run-%

run-%: DIST = $(*)
$(RUN_TARGETS): run-%: image-%
	docker run ${PLATFORM_ARGS} --rm -ti  \
    $(LOCAL_REPO_ARGS) \
    nvidia-container-toolkit-repo-test:$(*)

# Ensure that the local package root exists
$(RELEASE_TARGETS): release-%: $(LOCAL_PACKAGE_ROOT)/$(*)/$(ARCH)
	$(PROJECT_ROOT)/scripts/build-packages.sh $(*)-$(ARCH)
