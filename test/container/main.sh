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

set -eEuo pipefail
shopt -s lastpipe

readonly basedir="$(dirname "$(realpath "$0")")"
source "${basedir}/common.sh"

source "${basedir}/toolkit_test.sh"
source "${basedir}/docker_test.sh"
source "${basedir}/crio_test.sh"
source "${basedir}/containerd_test.sh"

: ${CLEANUP:=true}

usage() {
	cat >&2 <<EOF
Usage: $0 COMMAND [ARG...]

Commands:
  run SHARED_DIR TOOLKIT_CONTAINER_IMAGE [-c | --no-cleanup-on-error ]
  clean SHARED_DIR
EOF
}

if [ $# -lt 2 ]; then usage; exit 1; fi

# We defined shared_dir here so that it can be used in cleanup
readonly command=${1}; shift
readonly shared_dir="${1}"; shift;

case "${command}" in
	clean) testing::cleanup; exit 0;;
	run) ;;
	*) usage; exit 0;;
esac

if [ $# -eq 0 ]; then usage; exit 1; fi

readonly toolkit_container_image="${1}"; shift

options=$(getopt -l no-cleanup-on-error -o c -- "$@")
if [[ "$?" -ne 0 ]]; then usage; exit 1; fi

# set options to positional parameters
eval set -- "${options}"
for opt in ${options}; do
	case "${opt}" in
	c | --no-cleanup-on-error) CLEANUP=false; shift;;
	--) shift; break;;
	esac
done

trap '"$CLEANUP" && testing::cleanup' ERR

readonly test_cases="${TEST_CASES:-toolkit docker crio containerd}"

testing::cleanup
for tc in ${test_cases}; do
	log INFO "=================Testing ${tc}================="
	testing::setup
	testing::${tc}::main "$@"
	testing::cleanup
done
