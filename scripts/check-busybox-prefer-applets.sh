#!/bin/sh

# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

# Verifies that the busybox binary was NOT built with
# CONFIG_FEATURE_PREFER_APPLETS=y.
#
# With PREFER_APPLETS=y, 'chroot <host> modprobe' runs the busybox modprobe
# applet instead of the host's modprobe. The applet ignores softdeps, which
# breaks loading of kernel modules that rely on them.
#
# busybox does not embed its build config, so this checks behavior: a fake
# 'modprobe' placed first on PATH must be executed by 'busybox chroot' rather
# than the applet.
#
# This script is meant to run inside the image being checked (see the 'shell'
# stage of deployments/container/Dockerfile), so that a failure fails the
# image build.
#
# Background: https://github.com/NVIDIA/k8s-driver-manager/issues/299
#
# Usage: check-busybox-prefer-applets.sh [busybox-path]

set -eu

BUSYBOX=${1:-/busybox/busybox}
FAKEBIN=$(mktemp -d)
trap 'rm -rf "$FAKEBIN"' EXIT

printf '#!%s sh\necho FAKE-MODPROBE\n' "$BUSYBOX" > "$FAKEBIN/modprobe"
chmod +x "$FAKEBIN/modprobe"

output=$(PATH="$FAKEBIN:$PATH" "$BUSYBOX" chroot / modprobe x 2>&1 | head -n1 || true)

if [ "$output" != "FAKE-MODPROBE" ]; then
	echo "FAIL: 'chroot' ran the busybox modprobe applet instead of the fake modprobe in PATH" >&2
	echo "      expected output:    FAKE-MODPROBE" >&2
	echo "      actual output:      $output" >&2
	echo "$BUSYBOX is built with CONFIG_FEATURE_PREFER_APPLETS=y" >&2
	exit 1
fi
echo "OK: $BUSYBOX is built with CONFIG_FEATURE_PREFER_APPLETS=n"
