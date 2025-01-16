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
LIB_VERSION := 1.17.4
LIB_TAG :=

# The package version is the combination of the library version and tag.
# If the tag is specified the two components are joined with a tilde (~).
PACKAGE_VERSION := $(LIB_VERSION)$(if $(LIB_TAG),~$(LIB_TAG))
PACKAGE_REVISION := 1

GOLANG_VERSION := $(shell ./hack/golang-version.sh)

BUILDIMAGE_TAG ?= devel-go$(GOLANG_VERSION)
BUILDIMAGE ?=  $(LIB_NAME):$(BUILDIMAGE_TAG)

GIT_COMMIT ?= $(shell git describe --match="" --dirty --long --always --abbrev=40 2> /dev/null || echo "")
GIT_COMMIT_SHORT ?= $(shell git rev-parse --short HEAD 2> /dev/null || echo "")
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD 2> /dev/null || echo "${GIT_COMMIT}")
SOURCE_DATE_EPOCH ?= $(shell git log -1 --format=%ct  2> /dev/null || echo "")

ifeq ($(IMAGE_NAME),)
REGISTRY ?= nvidia
IMAGE_NAME := $(REGISTRY)/container-toolkit
endif

VERSION ?= $(LIB_VERSION)$(if $(LIB_TAG),-$(LIB_TAG))
