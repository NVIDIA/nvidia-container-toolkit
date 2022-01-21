#! /bin/bash
# Copyright (c) 2019-2021, NVIDIA CORPORATION.  All rights reserved.
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

testing::toolkit::install() {
	local -r uid=$(id -u)
	local -r gid=$(id -g)

	local READLINK="readlink"
	local -r platform=$(uname)
	if [[ "${platform}" == "Darwin" ]]; then
		READLINK="greadlink"
	fi

	testing::docker_run::toolkit::shell 'toolkit install /usr/local/nvidia/toolkit'
	docker run --rm -v "${shared_dir}:/work" alpine sh -c "chown -R ${uid}:${gid} /work/"

	# Ensure toolkit dir is correctly setup
	test ! -z "$(ls -A "${shared_dir}/usr/local/nvidia/toolkit")"

	test -L "${shared_dir}/usr/local/nvidia/toolkit/libnvidia-container.so.1"
	test -e "$(${READLINK} -f "${shared_dir}/usr/local/nvidia/toolkit/libnvidia-container.so.1")"
	test -L "${shared_dir}/usr/local/nvidia/toolkit/libnvidia-container-go.so.1"
	test -e "$(${READLINK} -f "${shared_dir}/usr/local/nvidia/toolkit/libnvidia-container-go.so.1")"

	test -e "${shared_dir}/usr/local/nvidia/toolkit/nvidia-container-cli"
	test -e "${shared_dir}/usr/local/nvidia/toolkit/nvidia-container-toolkit"
	test -e "${shared_dir}/usr/local/nvidia/toolkit/nvidia-container-runtime"

	grep -q -E "nvidia driver modules are not yet loaded, invoking runc directly" "${shared_dir}/usr/local/nvidia/toolkit/nvidia-container-runtime"
	grep -q -E "exec runc \".@\"" "${shared_dir}/usr/local/nvidia/toolkit/nvidia-container-runtime"

	test -e "${shared_dir}/usr/local/nvidia/toolkit/nvidia-container-cli.real"
	test -e "${shared_dir}/usr/local/nvidia/toolkit/nvidia-container-toolkit.real"
	test -e "${shared_dir}/usr/local/nvidia/toolkit/nvidia-container-runtime.real"

	test -e "${shared_dir}/usr/local/nvidia/toolkit/nvidia-container-runtime.experimental"
	test -e "${shared_dir}/usr/local/nvidia/toolkit/nvidia-container-runtime-experimental"

	grep -q -E "nvidia driver modules are not yet loaded, invoking runc directly" "${shared_dir}/usr/local/nvidia/toolkit/nvidia-container-runtime-experimental"
	grep -q -E "exec runc \".@\"" "${shared_dir}/usr/local/nvidia/toolkit/nvidia-container-runtime-experimental"
	grep -q -E "LD_LIBRARY_PATH=/run/nvidia/driver/usr/lib64:\\\$LD_LIBRARY_PATH " "${shared_dir}/usr/local/nvidia/toolkit/nvidia-container-runtime-experimental"

	test -e "${shared_dir}/usr/local/nvidia/toolkit/.config/nvidia-container-runtime/config.toml"

	# Ensure that the config file has the required contents.
	# NOTE: This assumes that RUN_DIR is '/run/nvidia'
	local -r nvidia_run_dir="/run/nvidia"
	grep -q -E "^\s*ldconfig = \"@${nvidia_run_dir}/driver/sbin/ldconfig(.real)?\"" "${shared_dir}/usr/local/nvidia/toolkit/.config/nvidia-container-runtime/config.toml"
	grep -q -E "^\s*root = \"${nvidia_run_dir}/driver\"" "${shared_dir}/usr/local/nvidia/toolkit/.config/nvidia-container-runtime/config.toml"
	grep -q -E "^\s*path = \"/usr/local/nvidia/toolkit/nvidia-container-cli\"" "${shared_dir}/usr/local/nvidia/toolkit/.config/nvidia-container-runtime/config.toml"
}

testing::toolkit::delete() {
	testing::docker_run::toolkit::shell 'mkdir -p /usr/local/nvidia/delete-toolkit'
	testing::docker_run::toolkit::shell 'touch /usr/local/nvidia/delete-toolkit/test.file'
	testing::docker_run::toolkit::shell 'toolkit delete /usr/local/nvidia/delete-toolkit'

	test ! -z "$(ls -A "${shared_dir}/usr/local/nvidia")"
	test ! -e "${shared_dir}/usr/local/nvidia/delete-toolkit"
}

testing::toolkit::main() {
	testing::toolkit::install
	testing::toolkit::delete
}

testing::toolkit::cleanup() {
	:
}
