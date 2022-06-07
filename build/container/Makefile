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

BUILD_MULTI_ARCH_IMAGES ?= false
DOCKER ?= docker

BUILDX =
ifeq ($(BUILD_MULTI_ARCH_IMAGES),true)
BUILDX = buildx
endif

MKDIR    ?= mkdir
DIST_DIR ?= $(CURDIR)/dist

##### Global variables #####
include $(CURDIR)/versions.mk

ifeq ($(IMAGE_NAME),)
REGISTRY ?= nvidia
IMAGE_NAME := $(REGISTRY)/container-toolkit
endif

VERSION ?= $(LIB_VERSION)$(if $(LIB_TAG),-$(LIB_TAG))
IMAGE_VERSION := $(VERSION)

IMAGE_TAG ?= $(VERSION)-$(DIST)
IMAGE = $(IMAGE_NAME):$(IMAGE_TAG)

OUT_IMAGE_NAME ?= $(IMAGE_NAME)
OUT_IMAGE_VERSION ?= $(IMAGE_VERSION)
OUT_IMAGE_TAG = $(OUT_IMAGE_VERSION)-$(DIST)
OUT_IMAGE = $(OUT_IMAGE_NAME):$(OUT_IMAGE_TAG)

##### Public rules #####
DEFAULT_PUSH_TARGET := ubuntu20.04
DISTRIBUTIONS := ubuntu20.04 ubuntu18.04 ubi8 centos7

META_TARGETS := packaging

BUILD_TARGETS := $(patsubst %,build-%,$(DISTRIBUTIONS) $(META_TARGETS))
PUSH_TARGETS := $(patsubst %,push-%,$(DISTRIBUTIONS) $(META_TARGETS))
TEST_TARGETS := $(patsubst %,test-%, $(DISTRIBUTIONS))

.PHONY: $(DISTRIBUTIONS) $(PUSH_TARGETS) $(BUILD_TARGETS) $(TEST_TARGETS)

ifneq ($(BUILD_MULTI_ARCH_IMAGES),true)
include $(CURDIR)/build/container/native-only.mk
else
include $(CURDIR)/build/container/multi-arch.mk
endif

# For the default push target we also push a short tag equal to the version.
# We skip this for the development release
DEVEL_RELEASE_IMAGE_VERSION ?= devel
PUSH_MULTIPLE_TAGS ?= true
ifeq ($(strip $(OUT_IMAGE_VERSION)),$(DEVEL_RELEASE_IMAGE_VERSION))
PUSH_MULTIPLE_TAGS = false
endif
ifeq ($(PUSH_MULTIPLE_TAGS),true)
push-$(DEFAULT_PUSH_TARGET): push-short
endif

push-%: DIST = $(*)
push-short: DIST = $(DEFAULT_PUSH_TARGET)

build-%: DIST = $(*)
build-%: DOCKERFILE = $(CURDIR)/build/container/Dockerfile.$(DOCKERFILE_SUFFIX)

ARTIFACTS_ROOT ?= $(shell realpath --relative-to=$(CURDIR) $(DIST_DIR))

# Use a generic build target to build the relevant images
$(BUILD_TARGETS): build-%: $(ARTIFACTS_ROOT)
	DOCKER_BUILDKIT=1 \
		$(DOCKER) $(BUILDX) build --pull \
		$(DOCKER_BUILD_OPTIONS) \
		$(DOCKER_BUILD_PLATFORM_OPTIONS) \
		--tag $(IMAGE) \
		--build-arg ARTIFACTS_ROOT="$(ARTIFACTS_ROOT)" \
		--build-arg BASE_DIST="$(BASE_DIST)" \
		--build-arg CUDA_VERSION="$(CUDA_VERSION)" \
		--build-arg GOLANG_VERSION="$(GOLANG_VERSION)" \
		--build-arg LIBNVIDIA_CONTAINER0_VERSION="$(LIBNVIDIA_CONTAINER0_DEPENDENCY)" \
		--build-arg PACKAGE_DIST="$(PACKAGE_DIST)" \
		--build-arg PACKAGE_VERSION="$(PACKAGE_VERSION)" \
		--build-arg VERSION="$(VERSION)" \
		--build-arg CVE_UPDATES="$(CVE_UPDATES)" \
		-f $(DOCKERFILE) \
		$(CURDIR)


build-ubuntu%: BASE_DIST = $(*)
build-ubuntu%: DOCKERFILE_SUFFIX := ubuntu
build-ubuntu%: PACKAGE_DIST = ubuntu18.04
build-ubuntu%: PACKAGE_VERSION := $(LIB_VERSION)$(if $(LIB_TAG),~$(LIB_TAG))
build-ubuntu%: LIBNVIDIA_CONTAINER0_DEPENDENCY=$(LIBNVIDIA_CONTAINER0_VERSION)

build-ubi8: BASE_DIST := ubi8
build-ubi8: DOCKERFILE_SUFFIX := centos
build-ubi8: PACKAGE_DIST = centos8
build-ubi8: PACKAGE_VERSION := $(LIB_VERSION)-$(if $(LIB_TAG),0.1.$(LIB_TAG),1)

build-centos7: BASE_DIST = $(*)
build-centos7: DOCKERFILE_SUFFIX := centos
build-centos7: PACKAGE_DIST = $(BASE_DIST)
build-centos7: PACKAGE_VERSION := $(LIB_VERSION)-$(if $(LIB_TAG),0.1.$(LIB_TAG),1)

build-packaging: BASE_DIST := ubuntu20.04
build-packaging: DOCKERFILE_SUFFIX := packaging
build-packaging: PACKAGE_ARCH := amd64
build-packaging: PACKAGE_DIST = all
build-packaging: PACKAGE_VERSION := $(LIB_VERSION)$(if $(LIB_TAG),-$(LIB_TAG))

# Test targets
test-%: DIST = $(*)

TEST_CASES ?= toolkit docker crio containerd
$(TEST_TARGETS): test-%:
	TEST_CASES="$(TEST_CASES)" bash -x $(CURDIR)/test/container/main.sh run \
		$(CURDIR)/shared-$(*) \
		$(IMAGE) \
			--no-cleanup-on-error

.PHONY: test-packaging
test-packaging: DIST = packaging
test-packaging:
	@echo "Testing package image contents"
	@$(DOCKER) run --rm $(IMAGE) test -d "/artifacts/packages/amazonlinux2/aarch64" || echo "Missing amazonlinux2/aarch64"
	@$(DOCKER) run --rm $(IMAGE) test -d "/artifacts/packages/amazonlinux2/x86_64" || echo "Missing amazonlinux2/x86_64"
	@$(DOCKER) run --rm $(IMAGE) test -d "/artifacts/packages/centos7/ppc64le" || echo "Missing centos7/ppc64le"
	@$(DOCKER) run --rm $(IMAGE) test -d "/artifacts/packages/centos7/x86_64" || echo "Missing centos7/x86_64"
	@$(DOCKER) run --rm $(IMAGE) test -d "/artifacts/packages/centos8/aarch64" || echo "Missing centos8/aarch64"
	@$(DOCKER) run --rm $(IMAGE) test -d "/artifacts/packages/centos8/ppc64le" || echo "Missing centos8/ppc64le"
	@$(DOCKER) run --rm $(IMAGE) test -d "/artifacts/packages/centos8/x86_64" || echo "Missing centos8/x86_64"
	@$(DOCKER) run --rm $(IMAGE) test -d "/artifacts/packages/debian10/amd64" || echo "Missing debian10/amd64"
	@$(DOCKER) run --rm $(IMAGE) test -d "/artifacts/packages/debian9/amd64" || echo "Missing debian9/amd64"
	@$(DOCKER) run --rm $(IMAGE) test -d "/artifacts/packages/opensuse-leap15.1/x86_64" || echo "Missing opensuse-leap15.1/x86_64"
	@$(DOCKER) run --rm $(IMAGE) test -d "/artifacts/packages/ubuntu16.04/amd64" || echo "Missing ubuntu16.04/amd64"
	@$(DOCKER) run --rm $(IMAGE) test -d "/artifacts/packages/ubuntu16.04/ppc64le" || echo "Missing ubuntu16.04/ppc64le"
	@$(DOCKER) run --rm $(IMAGE) test -d "/artifacts/packages/ubuntu18.04/amd64" || echo "Missing ubuntu18.04/amd64"
	@$(DOCKER) run --rm $(IMAGE) test -d "/artifacts/packages/ubuntu18.04/arm64" || echo "Missing ubuntu18.04/arm64"
	@$(DOCKER) run --rm $(IMAGE) test -d "/artifacts/packages/ubuntu18.04/ppc64le" || echo "Missing ubuntu18.04/ppc64le"
