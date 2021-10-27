#! /bin/bash
# Copyright (c) 2019, NVIDIA CORPORATION.  All rights reserved.
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

testing::crio::hook_created() {
	testing::docker_run::toolkit::shell 'crio setup /run/nvidia/toolkit'

	test ! -z "$(ls -A "${shared_dir}${CRIO_HOOKS_DIR}")"

	cat "${shared_dir}${CRIO_HOOKS_DIR}/${CRIO_HOOK_FILENAME}" | \
		jq -r '.hook.path' | grep -q "/run/nvidia/toolkit/"
	test $? -eq 0
	cat "${shared_dir}${CRIO_HOOKS_DIR}/${CRIO_HOOK_FILENAME}" | \
		jq -r '.hook.env[0]' | grep -q ":/run/nvidia/toolkit"
	test $? -eq 0
}

testing::crio::hook_cleanup() {
	testing::docker_run::toolkit::shell 'crio cleanup'

	test -z "$(ls -A "${shared_dir}${CRIO_HOOKS_DIR}")"
}

testing::crio::main() {
	testing::crio::hook_created
	testing::crio::hook_cleanup
}

testing::crio::cleanup() {
	:
}
