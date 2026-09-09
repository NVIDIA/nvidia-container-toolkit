# AGENTS.md

Guidance for AI coding agents working in this repository. Human contributors
should also read [CONTRIBUTING.md](CONTRIBUTING.md) and the
[documentation site](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/index.html)
which are the source of truth.

## Project Summary

The NVIDIA Container Toolkit is a collection of utilities that allow you to
run GPU-accelerated containers. It is designed to work with any container
runtime, e.g. docker, containerd, cri-o, podman. It is written primarily in Go.

Historically, NVIDIA GPUs were injected into containers via an OCI
[preStart hook](https://github.com/opencontainers/runtime-spec/blob/6999a89a76a0329f440d5740497bedb9dd431297/config.md#prestart).
The preStart hook would invoke the `nvidia-container-cli` which wraps the
NVIDIA Container Runtime library, `libnvidia-container`, written in C.
The preStart hook would discover GPUs on the host (during every container
startup) and perform a number of operations to make the requested GPU(s)
available in the container. Most of these operations, e.g. setting up cgroups
for devices, bind mounting files, are what low-level runtimes, like runc,
specialize in. This is the primary motivation for the development of the
[Container Device Interface (CDI)](https://github.com/cncf-tags/container-device-interface),
which provides a standard specification for defining "what" access to a device
means. Device vendors no longer need to write and maintain their own hooks to
enable device access. Instead, they define a CDI spec for their devices, and
runtime like runc take care of "injecting" devices based on the spec.

The NVIDIA Container Toolkit defaults to using CDI for enabling GPU
support in containers. Current and future development is focused on the 
CDI-based implementation. The preStart hook-based implementation is not
leveraged by default, but users can opt-in to using it by configuring
NVIDIA Container Toolkit to run in `legacy` mode.

## Repository layout

- `cmd/` — binary entrypoints.
- `internal/` — private modules shared across commands.
- `pkg/` — public modules intended for broader reuse.
- `api/` — versioned config schema types.
- `tests/` — integration and e2e tests; this is a **separate Go module**
  (`tests/go.mod`) from the root module.
- `packaging/` — `.deb`/`.rpm` packaging metadata.
- `deployments/` — container/systemd/udev artifacts and the `devel` build
  image used by `make docker-*` targets.
- `hack/`, `scripts/` — release and CI helper scripts.
- `third_party/` — vendored submodules; treat as upstream, not project code.
- `testdata/` — data used for unit tests.

## Build, test, and lint

All standard tasks go through the Makefile. Prefer make targets over invoking
tools directly so CI and local runs stay consistent.

```sh
make build          # go build ./...
make cmds           # build all cmd/ binaries
make test           # run unit tests
make lint           # golangci-lint run ./... (config is in .golangci.yml)
make fmt            # gofmt -s -l -w
make goimports      # goimports -local github.com/NVIDIA/nvidia-container-toolkit
```

Always run `make build`, `make fmt` and `make test` before considering Go 
changes complete.

Any `make <target>` can be run inside the project's build container instead
of on the host with `make docker-<target>` (e.g. `make docker-test`,
`make docker-lint`) — this matches the environment CI uses, so prefer it if
local Go/golangci-lint versions are in doubt.

## Coding conventions

- Comments explain **why**, not **what**. Identifier names should carry the
  "what."
- Keep changes scoped to the task. No drive-by refactors, speculative
  abstractions, or unrelated formatting churn — one concern per PR.
- When adding new files, add the Apache 2.0 license header to the top of the
  file. Match the header used in other files exactly rather than inventing a 
  variant.
- Follow existing patterns in `cmds/`, `pkg`, and `internal/` for logging and 
  error wrapping rather than introducing new libraries.
- Vendor directory (`vendor/`) is checked in; run `go mod tidy`/`go mod vendor`
  after dependency changes and do not hand-edit vendored code.
- Mocks are generated (files end in `_mock.go`); don't hand-edit generated mocks
  — regenerate via `make generate` (`go generate ./...`) instead.
- Standard Go formatting (`gofmt -s -l -w`) and import grouping via `goimports
  -local github.com/NVIDIA/nvidia-container-toolkit`.

## Testing conventions

- Unit tests are co-located `*_test.go` files using `testify` 
  (`require`/`assert`), typically table-driven  with a `map[string]struct{...}`
  of cases and `t.Run(name, ...)`. Run via `make test`. New behavior needs 
  appropriate test coverage.
- When fixing a bug, add a regression test that fails without the fix.

## Contribution process (see `CONTRIBUTING.md`)

- For any significant change (architectural change, new feature, breaking
  change, non-trivial bug fix), an issue should exist describing the 
  problem/proposal before implementation begins — check for or ask about a
  linked issue rather than assuming a PR alone is sufficient.
- All commits must be signed off (DCO): `git commit -s`, producing a trailing
  `Signed-off-by: Name <email>` line. Do not fabricate a sign-off identity — use
  the configured git user's identity.
- Do not open, push to, or comment on GitHub issues/PRs without explicit user
  confirmation.
- Keep PR titles short and imperative; the body should explain motivation
  ("why"), not just restate the diff.

## Things to avoid

- Never commit credentials, API keys, tokens, passwords, kubeconfigs, or private
  keys.
- Do not hand-edit generated files. Run `make generate` to regenerate Go mocks. 
  Run `go mod tidy`/`go mod vendor` after dependency changes and do not
  hand-edit vendored code.
- Do not modify anything in `third_party`. These are vendored Git submodules
  that track upstream repos, namely `libnvidia-container`. If a change is
  required in a Git submodule, a separate PR needs to be raised against
  that repository.
- Do not commit built binaries, `coverage.out`, or anything the
  [.gitignore](.gitignore) already excludes.
- Do not modify [CODEOWNERS](CODEOWNERS) or [GOVERNANCE.md](GOVERNANCE.md) unless the task
  is explicitly about that.
