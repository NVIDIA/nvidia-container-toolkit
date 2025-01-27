# Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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

GO_CMD ?= go

include $(CURDIR)/versions.mk

E2E_RUNTIME ?= docker

E2E_INSTALL_CTK ?= true

E2E_IMAGE_NAME ?= ghcr.io/nvidia/container-toolkit
E2E_IMAGE_TAG ?=
ifeq ($(E2E_IMAGE_TAG),)
$(error E2E_IMAGE_TAG is not set)
endif

E2E_SSH_KEY ?=
E2E_SSH_USER ?=
E2E_SSH_HOST ?=

.PHONY: test
test:
	cd $(CURDIR)/tests/e2e && $(GO_CMD) test -v . -args \
		-ginkgo.focus="$(E2E_RUNTIME)" \
		-test.timeout=1h \
		-ginkgo.v \
		-install-ctk=$(E2E_INSTALL_CTK) \
		-image-repo=$(E2E_IMAGE_NAME) \
		-image-tag=$(E2E_IMAGE_TAG) \
		-ssh-key=$(E2E_SSH_KEY) \
		-ssh-user=$(E2E_SSH_USER) \
		-remote-host=$(E2E_SSH_HOST)
