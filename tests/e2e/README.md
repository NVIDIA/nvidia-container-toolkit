<!--
SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
SPDX-License-Identifier: Apache-2.0

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->

# NVIDIA Container Toolkit – End‑to‑End (E2E) Test Suite

---

## 1  Scope & Goals
This repository contains a **Ginkgo v2 / Gomega** test harness that exercises an
NVIDIA Container Toolkit (CTK) installation on a **remote GPU‑enabled host** via
SSH.  The suite validates that:

1. CTK can be installed (or upgraded) head‑less (`INSTALL_CTK=true`).
2. The specified **container image** runs successfully under `nvidia-container-runtime`.
3. Errors and diagnostics are captured for post‑mortem analysis.

The tests are intended for continuous‑integration pipelines, nightly
compatibility runs, and pre‑release validation of new CTK builds.

---

## 2  Execution model
* The framework **does not** spin up a Kubernetes cluster; it drives a single
  host reachable over SSH.
* All commands run in a Ginkgo‑managed context (`ctx`) so they abort cleanly on
  timeout or Ctrl‑C.
* Environment discovery happens once in `TestMain` → `getTestEnv()`; parameters
  are therefore immutable for the duration of the run.

---

## 3  Prerequisites

| Item | Version / requirement |
|------|-----------------------|
| **Go toolchain** | ≥ 1.22 (for building Ginkgo helper binaries) |
| **GPU‑enabled Linux host** | Running a supported NVIDIA driver; reachable via SSH |
| **SSH connectivity** | Public‑key authentication *without* pass‑phrase for unattended CI |
| **Local OS** | Linux/macOS; POSIX shell required by the Makefile |

---

## 4  Environment variables

| Variable | Required | Example | Description |
|----------|----------|---------|-------------|
| `INSTALL_CTK` | ✖ | `true` | When `true` the test installs CTK on the remote host before running the image. When `false` it assumes CTK is already present. |
| `TOOLKIT_IMAGE` | ✔ | `nvcr.io/nvidia/cuda:12.4.0-runtime-ubi9` | Image that will be pulled & executed. |
| `SSH_KEY` | ✔ | `/home/ci/.ssh/id_rsa` | Private key used for authentication. |
| `SSH_USER` | ✔ | `ubuntu` | Username on the remote host. |
| `REMOTE_HOST` | ✔ | `gpurunner01.corp.local` | Hostname or IP address of the target node. |
| `REMOTE_PORT` | ✔ | `22` | SSH port of the target node. |

> All variables are validated at start‑up; the suite aborts early with a clear
> message if any are missing or ill‑formed.

---

## 5  Build helper binaries

Install the latest Ginkgo CLI locally so that the Makefile can invoke it:

```bash
make ginkgo  # installs ./bin/ginkgo
```

The Makefile entry mirrors the pattern used in other NVIDIA E2E suites:

```make
bin/ginkgo:
	GOBIN=$(CURDIR)/bin go install github.com/onsi/ginkgo/v2/ginkgo@latest
```

---

## 6  Running the suite

### 6.1  Basic invocation
```bash
INSTALL_CTK=true \
TOOLKIT_IMAGE=nvcr.io/nvidia/cuda:12.4.0-runtime-ubi9 \
SSH_KEY=$HOME/.ssh/id_rsa \
SSH_USER=ubuntu \
REMOTE_HOST=10.0.0.15 \
REMOTE_PORT=22 \
make test-e2e
```
This downloads the image on the remote host, installs CTK (if requested), and
executes a minimal CUDA‑based workload.

---

## 7  Internal test flow

| Phase | Key function(s) | Notes |
|-------|-----------------|-------|
| **Init** | `TestMain` → `getTestEnv` | Collects env vars, initializes `ctx`. |
| **Connection check** | `BeforeSuite` (not shown) | Verifies SSH reachability using `ssh -o BatchMode=yes`. |
| **Optional CTK install** | `installCTK == true` path | Runs the distro‑specific install script on the remote host. |
| **Runtime validation** | Leaf `It` blocks | Pulls `TOOLKIT_IMAGE`, runs `nvidia-smi` inside the container, asserts exit code `0`. |
| **Failure diagnostics** | `AfterEach` | Copies `/var/log/nvidia-container-runtime.log` & dmesg to `${LOG_ARTIFACTS_DIR}` via `scp`. |

---

## 8  Extending the suite

1. Create a new `_test.go` file under `tests/e2e`.
2. Use the Ginkgo DSL (`Describe`, `When`, `It` …). Each leaf node receives a
   `context.Context` so you can run remote commands with deadline control.
3. Helper utilities such as `runSSH`, `withSudo`, and `collectLogs` are already
   available from the shared test harness (see `ssh_helpers.go`).
4. Keep tests **idempotent** and clean any artefacts you create on the host.

---

## 9  Common issues & fixes

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `Permission denied (publickey)` | Wrong `SSH_KEY` or `SSH_USER` | Check variables; ensure key is readable by the CI user. |
| `docker: Error response from daemon: could not select device driver` | CTK not installed or wrong runtime class | Verify `INSTALL_CTK=true` or confirm CTK installation on the host. |
| Test hangs at image pull | No outbound internet on remote host | Pre‑load the image or use a local registry mirror. |

## 10  License
Distributed under the terms of the **Apache License 2.0** (see header).

