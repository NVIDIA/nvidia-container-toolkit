# Copyright (c) 2017-2020, NVIDIA CORPORATION. All rights reserved.

DOCKER   ?= docker
MKDIR    ?= mkdir
DIST_DIR ?= $(CURDIR)/dist

LIB_NAME := nvidia-container-toolkit
LIB_VERSION := 1.4.0
LIB_TAG ?=

GOLANG_VERSION := 1.14.2
GOLANG_PKG_PATH := github.com/NVIDIA/nvidia-container-toolkit/pkg

# By default run all native docker-based targets
docker-native:
include $(CURDIR)/docker.mk

binary:
	go build -ldflags "-s -w" -o "$(LIB_NAME)" $(GOLANG_PKG_PATH)
