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

readonly docker_dind_ctr="container-config-docker-dind-ctr-name"
readonly docker_test_ctr="container-config-docker-test-ctr-name"
readonly docker_dind_socket="/run/nvidia/docker.sock"

testing::docker::dind::setup() {
	# Docker creates /etc/docker when starting
	# by default there isn't any config in this directory (even after the daemon starts)
	docker run -d --rm --privileged \
		-v "${shared_dir}/etc/docker:/etc/docker" \
		-v "${shared_dir}/run/nvidia:/run/nvidia" \
		-v "${shared_dir}/usr/local/nvidia:/usr/local/nvidia" \
		--name "${docker_dind_ctr}" \
		docker:stable-dind -H unix://${docker_dind_socket}
}

testing::docker::dind::exec() {
	docker exec "${docker_dind_ctr}" sh -c "$*"
}

testing::docker::toolkit::run() {
	# Share the volumes so that we can edit the config file and point to the new runtime
	# Share the pid so that we can ask docker to reload its config
	docker run -d --rm --privileged \
		--volumes-from "${docker_dind_ctr}" \
		--pid "container:${docker_dind_ctr}" \
		-e "RUNTIME_ARGS=--socket ${docker_dind_socket}" \
		--name "${docker_test_ctr}" \
		"${toolkit_container_image}" "/usr/local/nvidia" "--no-daemon"

	# Ensure that we haven't broken non GPU containers
	with_retry 3 5s testing::docker::dind::exec docker run -t alpine echo foo
}

testing::docker::main() {
	testing::docker::dind::setup
	testing::docker::toolkit::run
}

testing::docker::cleanup() {
	docker kill "${docker_dind_ctr}" &> /dev/null || true
	docker kill "${docker_test_ctr}" &> /dev/null || true
}
