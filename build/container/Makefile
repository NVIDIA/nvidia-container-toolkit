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

DOCKER   ?= docker
MKDIR    ?= mkdir
DIST_DIR ?= $(CURDIR)/dist

##### Global variables #####

# TODO: These should be defined ONCE and currently duplicate the version in the
# toolkit makefile.
LIB_VERSION := 1.7.0
LIB_TAG := rc.2

VERSION ?= $(LIB_VERSION)$(if $(LIB_TAG),-$(LIB_TAG))

CUDA_VERSION ?= 11.4.2
GOLANG_VERSION ?= 1.16.4
ifeq ($(IMAGE_NAME),)
REGISTRY ?= nvidia
IMAGE_NAME := $(REGISTRY)/container-toolkit
endif

IMAGE_TAG ?= $(VERSION)-$(DIST)
IMAGE = $(IMAGE_NAME):$(IMAGE_TAG)

##### Public rules #####
DEFAULT_PUSH_TARGET := ubuntu18.04
TARGETS := ubuntu20.04 ubuntu18.04 ubi8 centos7 centos8

BUILD_TARGETS := $(patsubst %, build-%, $(TARGETS))
PUSH_TARGETS := $(patsubst %, push-%, $(TARGETS))
TEST_TARGETS := $(patsubst %, test-%, $(TARGETS))

.PHONY: $(TARGETS) $(PUSH_TARGETS) $(BUILD_TARGETS) $(TEST_TARGETS)

$(PUSH_TARGETS): push-%:
	$(DOCKER) push "$(IMAGE_NAME):$(IMAGE_TAG)"

# For the default push target we also push a short tag equal to the version.
# We skip this for the development release
DEVEL_RELEASE_IMAGE_VERSION ?= devel
ifneq ($(strip $(VERSION)),$(DEVEL_RELEASE_IMAGE_VERSION))
push-$(DEFAULT_PUSH_TARGET): push-short
endif
push-short:
	$(DOCKER) tag "$(IMAGE_NAME):$(VERSION)-$(DEFAULT_PUSH_TARGET)" "$(IMAGE_NAME):$(VERSION)"
	$(DOCKER) push "$(IMAGE_NAME):$(VERSION)"


build-%: DIST = $(*)
build-%: DOCKERFILE = $(CURDIR)/build/container/Dockerfile.$(DOCKERFILE_SUFFIX)

# Use a generic build target to build the relevant images
$(BUILD_TARGETS): build-%: $(ARTIFACTS_DIR)
	$(DOCKER) build --pull \
		--tag $(IMAGE) \
		--build-arg ARTIFACTS_DIR="$(ARTIFACTS_DIR)" \
		--build-arg BASE_DIST="$(BASE_DIST)" \
		--build-arg CUDA_VERSION="$(CUDA_VERSION)" \
		--build-arg GOLANG_VERSION="$(GOLANG_VERSION)" \
		--build-arg PACKAGE_VERSION="$(PACKAGE_VERSION)" \
		--build-arg VERSION="$(VERSION)" \
		-f $(DOCKERFILE) \
		$(CURDIR)


ARTIFACTS_ROOT ?= $(shell realpath --relative-to=$(CURDIR) $(DIST_DIR))

build-ubuntu%: DOCKERFILE_SUFFIX := ubuntu
build-ubuntu%: ARTIFACTS_DIR = $(ARTIFACTS_ROOT)/$(*)/amd64
build-ubuntu%: PACKAGE_VERSION := $(LIB_VERSION)$(if $(LIB_TAG),~$(LIB_TAG))

build-ubuntu18.04: BASE_DIST := ubuntu18.04
build-ubuntu20.04: BASE_DIST := ubuntu20.04

build-ubi8: DOCKERFILE_SUFFIX := centos
# TODO: Update this to use the centos8 packages
build-ubi8: ARTIFACTS_DIR = $(ARTIFACTS_ROOT)/centos7/x86_64
build-ubi8: PACKAGE_VERSION := $(LIB_VERSION)-$(if $(LIB_TAG),0.1.$(LIB_TAG),1)
build-ubi8: BASE_DIST := ubi8

build-centos%: DOCKERFILE_SUFFIX := centos
build-centos%: ARTIFACTS_DIR = $(ARTIFACTS_ROOT)/$(*)/x86_64
build-centos%: PACKAGE_VERSION := $(LIB_VERSION)-$(if $(LIB_TAG),0.1.$(LIB_TAG),1)

build-centos7: BASE_DIST := centos7
build-centos8: BASE_DIST := centos8

# Test targets
test-%: DIST = $(*)

TEST_CASES ?= toolkit docker crio containerd
$(TEST_TARGETS): test-%:
	TEST_CASES="$(TEST_CASES)" bash -x $(CURDIR)/test/container/main.sh run \
		$(CURDIR)/shared-$(*) \
		$(IMAGE) \
			--no-cleanup-on-error

