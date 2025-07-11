# Copyright 2024 NVIDIA CORPORATION
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

download:
	@echo Download go.mod dependencies
	@go mod download

install-tools: download
	@echo Installing tools from tools.go
	@cat tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go install %


DOCKER ?= docker
-include $(CURDIR)/versions.mk

DOCKERFILE_DEVEL = deployments/devel/Dockerfile
DOCKERFILE_CONTEXT = deployments/devel

.PHONY: .build-image
.build-image:
	$(DOCKER) build \
		--progress=plain \
		--tag $(BUILDIMAGE) \
		-f $(DOCKERFILE_DEVEL) \
		$(DOCKERFILE_CONTEXT)

modules:
	go mod tidy
	go mod verify

check-modules: modules
	git diff --quiet HEAD -- go.mod go.sum
