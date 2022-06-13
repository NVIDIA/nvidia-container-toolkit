# Copyright (c) 2017-2021, NVIDIA CORPORATION.  All rights reserved.
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

# Supported OSs by architecture
AMD64_TARGETS := ubuntu20.04 ubuntu18.04 ubuntu16.04 debian10 debian9
X86_64_TARGETS := centos7 centos8 rhel7 rhel8 amazonlinux2 opensuse-leap15.1
PPC64LE_TARGETS := ubuntu18.04 ubuntu16.04 centos7 centos8 rhel7 rhel8
ARM64_TARGETS := ubuntu20.04 ubuntu18.04
AARCH64_TARGETS := centos8 rhel8 amazonlinux2

# Define top-level build targets
docker%: SHELL:=/bin/bash

# Native targets
PLATFORM ?= $(shell uname -m)
ifeq ($(PLATFORM),x86_64)
NATIVE_TARGETS := $(AMD64_TARGETS) $(X86_64_TARGETS)
$(AMD64_TARGETS): %: %-amd64
$(X86_64_TARGETS): %: %-x86_64
else ifeq ($(PLATFORM),ppc64le)
NATIVE_TARGETS := $(PPC64LE_TARGETS)
$(PPC64LE_TARGETS): %: %-ppc64le
else ifeq ($(PLATFORM),aarch64)
NATIVE_TARGETS := $(ARM64_TARGETS) $(AARCH64_TARGETS)
$(ARM64_TARGETS): %: %-arm64
$(AARCH64_TARGETS): %: %-aarch64
endif
docker-native: $(NATIVE_TARGETS)

# amd64 targets
AMD64_TARGETS := $(patsubst %, %-amd64, $(AMD64_TARGETS))
$(AMD64_TARGETS): ARCH := amd64
$(AMD64_TARGETS): %: --%
docker-amd64: $(AMD64_TARGETS)

# x86_64 targets
X86_64_TARGETS := $(patsubst %, %-x86_64, $(X86_64_TARGETS))
$(X86_64_TARGETS): ARCH := x86_64
$(X86_64_TARGETS): %: --%
docker-x86_64: $(X86_64_TARGETS)

# arm64 targets
ARM64_TARGETS := $(patsubst %, %-arm64, $(ARM64_TARGETS))
$(ARM64_TARGETS): ARCH := arm64
$(ARM64_TARGETS): %: --%
docker-arm64: $(ARM64_TARGETS)

# aarch64 targets
AARCH64_TARGETS := $(patsubst %, %-aarch64, $(AARCH64_TARGETS))
$(AARCH64_TARGETS): ARCH := aarch64
$(AARCH64_TARGETS): %: --%
docker-aarch64: $(AARCH64_TARGETS)

# ppc64le targets
PPC64LE_TARGETS := $(patsubst %, %-ppc64le, $(PPC64LE_TARGETS))
$(PPC64LE_TARGETS): ARCH := ppc64le
$(PPC64LE_TARGETS): WITH_LIBELF := yes
$(PPC64LE_TARGETS): %: --%
docker-ppc64le: $(PPC64LE_TARGETS)

# docker target to build for all os/arch combinations
docker-all: $(AMD64_TARGETS) $(X86_64_TARGETS) \
            $(ARM64_TARGETS) $(AARCH64_TARGETS) \
            $(PPC64LE_TARGETS)

# Default variables for all private '--' targets below.
# One private target is defined for each OS we support.
--%: TARGET_PLATFORM = $(*)
--%: VERSION = $(patsubst $(OS)%-$(ARCH),%,$(TARGET_PLATFORM))
--%: BASEIMAGE = $(OS):$(VERSION)
--%: BUILDIMAGE = nvidia/$(LIB_NAME)/$(OS)$(VERSION)-$(ARCH)
--%: DOCKERFILE = $(CURDIR)/docker/Dockerfile.$(OS)
--%: ARTIFACTS_DIR = $(DIST_DIR)/$(OS)$(VERSION)/$(ARCH)
--%: docker-build-%
	@

LIBNVIDIA_CONTAINER_VERSION ?= $(LIB_VERSION)
LIBNVIDIA_CONTAINER_TAG ?= $(LIB_TAG)

# private ubuntu target
--ubuntu%: OS := ubuntu
--ubuntu%: LIB_VERSION := $(LIB_VERSION)$(if $(LIB_TAG),~$(LIB_TAG))
--ubuntu%: LIBNVIDIA_CONTAINER_TOOLS_VERSION := $(LIBNVIDIA_CONTAINER_VERSION)$(if $(LIBNVIDIA_CONTAINER_TAG),~$(LIBNVIDIA_CONTAINER_TAG))-1
--ubuntu%: PKG_REV := 1

# private debian target
--debian%: OS := debian
--debian%: LIB_VERSION := $(LIB_VERSION)$(if $(LIB_TAG),~$(LIB_TAG))
--debian%: LIBNVIDIA_CONTAINER_TOOLS_VERSION := $(LIBNVIDIA_CONTAINER_VERSION)$(if $(LIBNVIDIA_CONTAINER_TAG),~$(LIBNVIDIA_CONTAINER_TAG))-1
--debian%: PKG_REV := 1

# private centos target
--centos%: OS := centos
--centos%: PKG_REV := $(if $(LIB_TAG),0.1.$(LIB_TAG),1)
--centos%: LIBNVIDIA_CONTAINER_TOOLS_VERSION := $(LIBNVIDIA_CONTAINER_VERSION)-$(if $(LIBNVIDIA_CONTAINER_TAG),0.1.$(LIBNVIDIA_CONTAINER_TAG),1)
--centos8%: BASEIMAGE = quay.io/centos/centos:stream8

# private amazonlinux target
--amazonlinux%: OS := amazonlinux
--amazonlinux%: LIBNVIDIA_CONTAINER_TOOLS_VERSION := $(LIBNVIDIA_CONTAINER_VERSION)-$(if $(LIBNVIDIA_CONTAINER_TAG),0.1.$(LIBNVIDIA_CONTAINER_TAG),1)
--amazonlinux%: PKG_REV := $(if $(LIB_TAG),0.1.$(LIB_TAG),1)

# private opensuse-leap target
--opensuse-leap%: OS = opensuse-leap
--opensuse-leap%: BASEIMAGE = opensuse/leap:$(VERSION)
--opensuse-leap%: LIBNVIDIA_CONTAINER_TOOLS_VERSION := $(LIBNVIDIA_CONTAINER_VERSION)-$(if $(LIBNVIDIA_CONTAINER_TAG),0.1.$(LIBNVIDIA_CONTAINER_TAG),1)
--opensuse-leap%: PKG_REV := $(if $(LIB_TAG),0.1.$(LIB_TAG),1)

# private rhel target (actually built on centos)
--rhel%: OS := centos
--rhel%: LIBNVIDIA_CONTAINER_TOOLS_VERSION := $(LIBNVIDIA_CONTAINER_VERSION)-$(if $(LIBNVIDIA_CONTAINER_TAG),0.1.$(LIBNVIDIA_CONTAINER_TAG),1)
--rhel%: PKG_REV := $(if $(LIB_TAG),0.1.$(LIB_TAG),1)
--rhel%: VERSION = $(patsubst rhel%-$(ARCH),%,$(TARGET_PLATFORM))
--rhel%: ARTIFACTS_DIR = $(DIST_DIR)/rhel$(VERSION)/$(ARCH)
--rhel8%: BASEIMAGE = quay.io/centos/centos:stream8

# We allow the CONFIG_TOML_SUFFIX to be overridden.
CONFIG_TOML_SUFFIX ?= $(OS)

docker-build-%:
	@echo "Building for $(TARGET_PLATFORM)"
	docker pull --platform=linux/$(ARCH) $(BASEIMAGE)
	DOCKER_BUILDKIT=1 \
	$(DOCKER) build \
	    --platform=linux/$(ARCH) \
	    --progress=plain \
	    --build-arg BASEIMAGE="$(BASEIMAGE)" \
	    --build-arg GOLANG_VERSION="$(GOLANG_VERSION)" \
	    --build-arg PKG_NAME="$(LIB_NAME)" \
	    --build-arg PKG_VERS="$(LIB_VERSION)" \
	    --build-arg PKG_REV="$(PKG_REV)" \
		--build-arg LIBNVIDIA_CONTAINER_TOOLS_VERSION="$(LIBNVIDIA_CONTAINER_TOOLS_VERSION)" \
	    --build-arg CONFIG_TOML_SUFFIX="$(CONFIG_TOML_SUFFIX)" \
		--build-arg GIT_COMMIT="$(GIT_COMMIT)" \
	    --tag $(BUILDIMAGE) \
	    --file $(DOCKERFILE) .
	$(DOCKER) run \
	    --platform=linux/$(ARCH) \
	    -e DISTRIB \
	    -e SECTION \
	    -v $(ARTIFACTS_DIR):/dist \
	    $(BUILDIMAGE)

docker-clean:
	IMAGES=$$(docker images "nvidia/$(LIB_NAME)/*" --format="{{.ID}}"); \
	if [ "$${IMAGES}" != "" ]; then \
	    docker rmi -f $${IMAGES}; \
	fi

distclean:
	rm -rf $(DIST_DIR)
