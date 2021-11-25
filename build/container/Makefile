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
include $(CURDIR)/versions.mk

ifeq ($(IMAGE_NAME),)
REGISTRY ?= nvidia
IMAGE_NAME := $(REGISTRY)/container-toolkit
endif

VERSION ?= $(LIB_VERSION)$(if $(LIB_TAG),-$(LIB_TAG))

IMAGE_TAG ?= $(VERSION)-$(DIST)
IMAGE = $(IMAGE_NAME):$(IMAGE_TAG)

##### Public rules #####
DEFAULT_PUSH_TARGET := ubuntu18.04
TARGETS := ubuntu20.04 ubuntu18.04 ubi8 centos7 centos8 packaging

BUILD_TARGETS := $(patsubst %, build-%, $(TARGETS))
PUSH_TARGETS := $(patsubst %, push-%, $(TARGETS))
TEST_TARGETS := $(patsubst %, test-%, $(filter-out packaging,$(TARGETS)))

.PHONY: $(TARGETS) $(PUSH_TARGETS) $(BUILD_TARGETS) $(TEST_TARGETS)

push-%: DIST = $(*)
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
	DOCKER_BUILDKIT=1 \
		$(DOCKER) build --platform=linux/amd64 --pull \
		--tag $(IMAGE) \
		--build-arg ARTIFACTS_DIR="$(ARTIFACTS_DIR)" \
		--build-arg BASE_DIST="$(BASE_DIST)" \
		--build-arg CUDA_VERSION="$(CUDA_VERSION)" \
		--build-arg GOLANG_VERSION="$(GOLANG_VERSION)" \
		--build-arg PACKAGE_VERSION="$(PACKAGE_VERSION)" \
		--build-arg PACKAGE_ARCH="$(PACKAGE_ARCH)" \
		--build-arg VERSION="$(VERSION)" \
		-f $(DOCKERFILE) \
		$(CURDIR)


ARTIFACTS_ROOT ?= $(shell realpath --relative-to=$(CURDIR) $(DIST_DIR))
ARTIFACTS_DIR = $(ARTIFACTS_ROOT)/$(*)

PACKAGE_ARCH = amd64

build-ubuntu%: DOCKERFILE_SUFFIX := ubuntu
build-ubuntu%: PACKAGE_VERSION := $(LIB_VERSION)$(if $(LIB_TAG),~$(LIB_TAG))

build-ubuntu18.04: BASE_DIST := ubuntu18.04
build-ubuntu20.04: BASE_DIST := ubuntu20.04

build-ubi8: DOCKERFILE_SUFFIX := centos
# TODO: Update this to use the centos8 packages
build-ubi8: ARTIFACTS_DIR = $(ARTIFACTS_ROOT)/centos7
build-ubi8: PACKAGE_VERSION := $(LIB_VERSION)-$(if $(LIB_TAG),0.1.$(LIB_TAG),1)
build-ubi8: BASE_DIST := ubi8
build-ubi8: PACKAGE_ARCH := x86_64

build-centos%: DOCKERFILE_SUFFIX := centos
build-centos%: PACKAGE_VERSION := $(LIB_VERSION)-$(if $(LIB_TAG),0.1.$(LIB_TAG),1)
build-centos%: PACKAGE_ARCH := x86_64

build-centos7: BASE_DIST := centos7
build-centos8: BASE_DIST := centos8

build-packaging: BASE_DIST := ubuntu20.04
build-packaging: DOCKERFILE_SUFFIX := packaging
build-packaging: ARTIFACTS_DIR = $(ARTIFACTS_ROOT)

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
	@$(DOCKER) run --rm $(IMAGE) test -d "/artifacts/packages/amazonlinux1/x86_64" || echo "Missing amazonlinux1/x86_64"
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
