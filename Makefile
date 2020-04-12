# Copyright (c) 2017-2020, NVIDIA CORPORATION. All rights reserved.

DOCKER   ?= docker
MKDIR    ?= mkdir
REGISTRY ?= nvidia/container-toolkit

GOLANG_VERSION := 1.14.2
VERSION        := 1.0.5
DIST_DIR       := $(CURDIR)/dist

TOOLKIT=nvidia-container-toolkit

include $(CURDIR)/docker.mk

.PHONY: all

all: ubuntu18.04 ubuntu16.04 debian10 debian9 centos7 amzn2 amzn1 opensuse-leap15.1

binary:
	go build -ldflags "-s -w" -o "$(TOOLKIT)" github.com/NVIDIA/container-toolkit/pkg

