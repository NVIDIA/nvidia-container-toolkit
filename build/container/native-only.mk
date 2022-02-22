# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

DOCKER_BUILD_PLATFORM_OPTIONS = --platform=linux/amd64

$(PUSH_TARGETS): push-%:
	$(DOCKER) tag "$(IMAGE)" "$(OUT_IMAGE)"
	$(DOCKER) push "$(OUT_IMAGE)"

push-short:
	$(DOCKER) tag "$(IMAGE_NAME):$(VERSION)-$(DEFAULT_PUSH_TARGET)" "$(OUT_IMAGE_NAME):$(OUT_IMAGE_VERSION)"
	$(DOCKER) push "$(OUT_IMAGE_NAME):$(OUT_IMAGE_VERSION)"
