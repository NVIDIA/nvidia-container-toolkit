#!/usr/bin/env bash
# Copyright 2026 NVIDIA CORPORATION
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
#
# Writes THIRD_PARTY_NOTICES.md: an index of every Go dependency plus the
# verbatim text of each license, for ./cmd (from vendor/) and
# deployments/devel/go.mod (from the module cache). Stdlib is excluded; the Go
# distribution covers it.
#
# go-licenses resolves only the host platform, and build-tagged sources differ
# per platform, so one run is both incomplete and host-dependent. There is no
# union mode, so run it per target and merge, sorting with LC_ALL=C.
#
# Requires go (deployments/devel/Dockerfile, via hack/golang-version.sh) and
# go-licenses (deployments/devel/go.mod; 'make notices' installs it).

set -euo pipefail

# --- configuration ---

OUTPUT="${OUTPUT:-THIRD_PARTY_NOTICES.md}"
LICENSES_DIR="${LICENSES_DIR:-.licenses-cache}"
TOOLS_DIR="${TOOLS_DIR:-deployments/devel}"
TOOLS_FILE="${TOOLS_DIR}/tools.go"
MULTI_ARCH_MK="${MULTI_ARCH_MK:-deployments/container/multi-arch.mk}"
MODULES_TXT="${MODULES_TXT:-vendor/modules.txt}"

# Exactly what 'make cmds' builds: CMDS is a wildcard over ./cmd/*, and both
# release paths run it. docker/Dockerfile.* runs 'make PREFIX=... cmds' to
# populate the deb and rpm packages, and deployments/container/Dockerfile builds
# nvidia-ctk-installer from source and then unpacks those same packages into the
# released image. Every command under cmd/ is therefore distributed.
#
# Measured against ./..., the only extra packages are four local ones
# (internal/test, internal/test/to, internal/nvsandboxutils/mock and its
# generator) plus five stdlib packages, so this scope drops no third-party
# dependency. The e2e suite under tests/ and the fixtures under testdata/ are
# separate Go modules with their own go.mod files, so their dependencies cannot
# reach this graph at all.
PACKAGES=("./cmd/...")

# Must match the released image platforms; verify_platform_matrix fails on drift
# so a new target cannot silently produce an incomplete file.
#
# The deb/rpm packages are built per architecture rather than as a multi-arch
# image, but they are released for the same two: hack/prepare-artifacts.sh
# publishes deb amd64/arm64 and rpm x86_64/aarch64 and nothing else. The
# ppc64le targets in docker/docker.mk are buildable but are not part of a
# release, so they are not in this matrix.
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
)

# Wider than the runtime matrix: host tools get built on laptops as well as CI,
# so a host-only graph would differ between macOS and Linux.
TOOLCHAIN_PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

# --- helpers ---

die() {
    printf 'ERROR: %s\n' "$1" >&2
    shift
    if (( $# > 0 )); then
        printf '%s\n' "$@" >&2
    fi
    exit 1
}

log() {
    printf '%s\n' "$*" >&2
}

# Licenses that are themselves Markdown close a fixed ``` fence early and invert
# every block after it, so open with one backtick more than the file's longest run.
fence_for() {
    local file="$1" longest width
    # -a: without it a license containing a NUL byte is treated as binary and
    # grep prints "Binary file ... matches" instead of the matches. Older greps
    # put that on stdout and newer ones on stderr, so the measured width would
    # differ by host — and would even vary with the path length. LC_ALL=C for
    # the same reason it is on every sort here.
    longest=$(LC_ALL=C grep -oaE '`+' "${file}" 2>/dev/null \
        | awk '{ if (length($0) > m) m = length($0) } END { print m+0 }')
    width=$(( longest + 1 ))
    (( width < 3 )) && width=3
    printf '%*s' "${width}" '' | tr ' ' '`'
}

# --- setup ---

check_prerequisites() {
    command -v go >/dev/null 2>&1 || die "go is not installed."

    # Prefer the pinned repo-local copy; absolute, as the toolchain pass chdirs.
    if [[ -x "./bin/go-licenses" ]]; then
        GO_LICENSES="${PWD}/bin/go-licenses"
    elif command -v go-licenses >/dev/null 2>&1; then
        GO_LICENSES="$(command -v go-licenses)"
    else
        die "go-licenses is not installed." "Install it with 'make bin/go-licenses'."
    fi

    local f
    for f in "${TOOLS_FILE}" "${MULTI_ARCH_MK}" "${MODULES_TXT}"; do
        [[ -f "${f}" ]] || die "${f} not found — run 'make notices' from the repo root."
    done

    # Ignored during collection — this is a *third-party* notices file.
    LOCAL_MODULE=$(go list -m 2>/dev/null || true)
    [[ -n "${LOCAL_MODULE}" ]] || die "could not determine local module path via 'go list -m'."

    # vendor/ keeps this offline.
    #
    # CGO stays ON because 'make cmds' does not disable it: the released
    # binaries are linked with '-extldflags=-Wl,--export-dynamic ...' and reach
    # libnvidia-container and NVML through cgo. Turning it off changes which
    # files the build constraints select and risks a silently short inventory.
    # No C compiler is needed here because go-licenses only lists and parses; it
    # never compiles, so the cross-platform passes work from any host.
    export GOFLAGS="-mod=vendor"
    export CGO_ENABLED=1
}

verify_platform_matrix() {
    local expected actual
    expected=$(sed -n 's/^DOCKER_BUILD_PLATFORM_OPTIONS[[:space:]]*?*=[[:space:]]*--platform=//p' \
        "${MULTI_ARCH_MK}" | tr ',' '\n' | sed '/^$/d' | LC_ALL=C sort -u)
    [[ -n "${expected}" ]] \
        || die "could not read DOCKER_BUILD_PLATFORM_OPTIONS from ${MULTI_ARCH_MK}."

    actual=$(printf '%s\n' "${PLATFORMS[@]}" | LC_ALL=C sort -u)
    [[ "${expected}" == "${actual}" ]] || die \
        "the PLATFORMS matrix is out of sync with ${MULTI_ARCH_MK}." \
        "Update the PLATFORMS array in hack/generate-notices.sh to match the released targets." \
        "  matrix (PLATFORMS): $(echo "${actual}" | paste -sd ' ' -)" \
        "  image platforms:    $(echo "${expected}" | paste -sd ' ' -)"
}

prepare_workspace() {
    # Rebuilt each run. Guard the override: '', '/', '.' or '..' would make this fatal.
    case "${LICENSES_DIR}" in
        ""|"/"|"."|"..")
            die "refusing to 'rm -rf' unsafe LICENSES_DIR='${LICENSES_DIR}'."
            ;;
    esac
    rm -rf "${LICENSES_DIR}"
    mkdir -p "${LICENSES_DIR}" "${LICENSES_DIR}/.tools"

    # Explicit templates: macOS mktemp ignores TMPDIR without one.
    local t="${TMPDIR:-/tmp}/nvidia-container-toolkit-notices"
    SAVE_ROOT="$(mktemp -d "${t}.XXXXXX")"
    COMBINED_CSV="$(mktemp "${t}-csv.XXXXXX")"
    INDEX_FILE="$(mktemp "${t}-idx.XXXXXX")"
    TOOLS_CSV="$(mktemp "${t}-tools-csv.XXXXXX")"
    TOOLS_INDEX="$(mktemp "${t}-tools-idx.XXXXXX")"
    OUT_TMP="$(mktemp "${t}-out.XXXXXX")"
    trap 'rm -rf "${SAVE_ROOT}"; rm -f "${COMBINED_CSV}" "${INDEX_FILE}" "${TOOLS_CSV}" "${TOOLS_INDEX}" "${OUT_TMP}"' EXIT
}

# --- collection ---

collect_runtime() {
    local platform goos goarch save_dir

    for platform in "${PLATFORMS[@]}"; do
        goos="${platform%/*}"
        goarch="${platform#*/}"
        log "Collecting licenses for ${goos}/${goarch}..."

        save_dir="${SAVE_ROOT}/${goos}_${goarch}"

        # Only the local module is ignored. go-licenses already omits the
        # standard library, and --ignore matches on plain STRING prefixes, not
        # path segments: passing stdlib top-level names silently drops every
        # dependency starting with those letters — the token "go" (from go/ast,
        # go/build) would remove golang.org/x/*, google.golang.org/*,
        # gopkg.in/* and go.opencensus.io in one go. Keep this list minimal and
        # never add a short, generic prefix.
        GOOS="${goos}" GOARCH="${goarch}" "${GO_LICENSES}" save "${PACKAGES[@]}" \
            --save_path="${save_dir}" \
            --force \
            --ignore="${LOCAL_MODULE}"

        GOOS="${goos}" GOARCH="${goarch}" "${GO_LICENSES}" csv "${PACKAGES[@]}" \
            --ignore="${LOCAL_MODULE}" \
            >> "${COMBINED_CSV}"

        merge_licenses "${save_dir}" "${LICENSES_DIR}"
    done
}

collect_toolchain() {
    local platform goos goarch save_dir

    read_tool_packages

    # Populate the module cache once; deployments/devel is not vendored
    # (mod-vendor in the top-level Makefile explicitly skips ./deployments/*).
    ( cd "${TOOLS_DIR}" && GOFLAGS="-mod=readonly" go mod download ) >&2

    for platform in "${TOOLCHAIN_PLATFORMS[@]}"; do
        goos="${platform%/*}"
        goarch="${platform#*/}"
        log "Collecting build toolchain licenses for ${goos}/${goarch}..."

        save_dir="${SAVE_ROOT}/tools/${goos}_${goarch}"
        (
            cd "${TOOLS_DIR}"
            # shellcheck disable=SC2030  # subshell-local on purpose: only the
            # devel module reads from the cache, the outer -mod=vendor stands.
            export GOFLAGS="-mod=readonly"
            GOOS="${goos}" GOARCH="${goarch}" "${GO_LICENSES}" save "${TOOL_PKGS[@]}" \
                --save_path="${save_dir}" \
                --force \
                --ignore="${LOCAL_MODULE}" >&2
            GOOS="${goos}" GOARCH="${goarch}" "${GO_LICENSES}" csv "${TOOL_PKGS[@]}" \
                --ignore="${LOCAL_MODULE}"
        ) >> "${TOOLS_CSV}"

        # Separate subtree so a module in both graphs cannot clobber the other copy.
        merge_licenses "${save_dir}" "${LICENSES_DIR}/.tools"
    done
}

# tools.go is build-tagged and imports main packages, so it cannot be listed as
# a package; read the pinned paths out of it as 'make install-tools' does.
read_tool_packages() {
    TOOL_PKGS=()
    local pkg
    while IFS= read -r pkg; do
        [[ -n "${pkg}" ]] && TOOL_PKGS+=("${pkg}")
    done < <(grep -E '^[[:space:]]*_ "' "${TOOLS_FILE}" | sed 's/.*"\(.*\)".*/\1/')

    (( ${#TOOL_PKGS[@]} > 0 )) || die "no tool imports found in ${TOOLS_FILE}."
}

# Union into the shared tree: text is identical across platforms so overwrites
# are no-ops. Module cache files are 0444 and cp preserves that, so restore
# write permission or the next platform's copy fails.
merge_licenses() {
    cp -R "$1/." "$2/"
    chmod -R u+w "$2"
}

# --- index building ---

# One row per package, joining licenses rather than picking one: go-licenses
# emits a row per recognized license, so key-only dedup would hide
# filepath-securejoin's MPL-2.0 behind its BSD-3-Clause. Key-only dedup is also
# non-deterministic — BSD sort keeps the first input row, GNU sort applies a
# whole-line tiebreak — so sort whole lines first.
collapse_index() {
    LC_ALL=C sort -u "$1" | awk -F, '
        {
            pkg = $1
            if (!(pkg in url)) { url[pkg] = $2; order[++n] = pkg }
            if (!((pkg SUBSEP $3) in seen)) {
                seen[pkg SUBSEP $3] = 1
                # Count, do not test "pkg in lic". mawk and busybox awk
                # instantiate the assignment target before evaluating the
                # right-hand side, so that test is already true on the first row
                # and every license comes out prefixed with " / ". BWK awk (the
                # macOS default) and gawk do not, so the split is by awk
                # implementation, not by OS — and mawk is /usr/bin/awk on stock
                # Debian and Ubuntu. This form is correct on all of them.
                lic[pkg] = (cnt[pkg]++ ? lic[pkg] " / " : "") $3
            }
        }
        END { for (i = 1; i <= n; i++) print order[i] "," url[order[i]] "," lic[order[i]] }
    '
}

# In vendor mode go-licenses reports a URL into this repo at HEAD, which stops
# describing released content once main moves, so append module@version from
# modules.txt. Longest-prefix match: a license may sit below the module root.
annotate_modules() {
    awk -v modfile="${MODULES_TXT}" '
        BEGIN {
            FS = OFS = ","
            while ((getline line < modfile) > 0) {
                if (line !~ /^# /) continue
                split(line, f, " ")
                # "# <path> <version>", optionally "=> <path> <version>" for a
                # replace. The replacement is what is actually vendored, so it
                # is what the notices file must name. A filesystem replace has
                # no version to report, so stop rather than misstate it.
                if (f[4] == "=>" || f[3] == "=>") {
                    r = (f[4] == "=>") ? 5 : 4
                    if (f[r + 1] == "") {
                        print "ERROR: " modfile " replaces " f[2] " with a local path;" > "/dev/stderr"
                        print "teach hack/generate-notices.sh how to attribute it." > "/dev/stderr"
                        exit 1
                    }
                    mods[++m] = f[2]
                    disp[f[2]] = f[r] "@" f[r + 1]
                } else {
                    mods[++m] = f[2]
                    disp[f[2]] = f[2] "@" f[3]
                }
            }
            close(modfile)
            # A read error makes getline return -1 and the loop never runs,
            # which would label every entry "unknown" without failing.
            if (m == 0) {
                print "ERROR: no module lines read from " modfile > "/dev/stderr"
                exit 1
            }
        }
        {
            best = ""
            for (i = 1; i <= m; i++) {
                mp = mods[i]
                if (($1 == mp || index($1, mp "/") == 1) && length(mp) > length(best)) best = mp
            }
            print $0, (best == "" ? "unknown" : disp[best])
        }
    '
}

build_indexes() {
    log "Generating dependency index..."
    collapse_index "${COMBINED_CSV}" | annotate_modules > "${INDEX_FILE}"
    collapse_index "${TOOLS_CSV}" > "${TOOLS_INDEX}"

    [[ -s "${INDEX_FILE}" ]] \
        || die "go-licenses produced no entries for ${PACKAGES[*]} — refusing to write empty notices file."
    [[ -s "${TOOLS_INDEX}" ]] \
        || die "go-licenses produced no entries for the build toolchain — refusing to write incomplete notices file."

    # A runtime row with no resolved module means modules.txt and the license
    # tree disagree; that is a bug in this script, not a dependency change.
    if cut -d, -f4 "${INDEX_FILE}" | grep -qx 'unknown'; then
        die "some runtime packages could not be matched to a module in ${MODULES_TXT}." \
            "Re-run 'make vendor' first; if it persists, fix annotate_modules in hack/generate-notices.sh."
    fi

    # Toolchain Source URLs come from go-licenses, which resolves non-github
    # hosts over the network (GET <path>?go-get=1) and falls back to "Unknown"
    # with a warning and a zero exit. Behind a proxy that blocks such a lookup
    # the document would quietly lose URLs and look like a dependency change.
    if cut -d, -f2 "${TOOLS_INDEX}" | grep -qx 'Unknown'; then
        die "go-licenses could not resolve source URLs for some toolchain modules." \
            "This usually means the network blocked a '?go-get=1' lookup. Re-run with" \
            "access to the module hosts rather than committing a degraded file."
    fi
}

# --- rendering ---

# License-bearing files, sorted. Filter by name: for restricted licenses
# 'go-licenses save' copies the whole module source, which does not belong here.
license_files_for() {
    local dir="$1" f
    [[ -d "${dir}" ]] || return 0
    while IFS= read -r -d '' f; do
        # LC_ALL=C for the same reason it is on every sort here: under a Turkish
        # locale glibc does not fold I to i, so this would stop matching LICENSE
        # and every section would silently render "License text unavailable".
        if printf '%s' "$(basename "${f}")" \
            | LC_ALL=C grep -qiE '^(licen[cs]e|notice|copying|copyright|authors|patents)([-._].*)?$'; then
            printf '%s\n' "${f}"
        fi
    done < <(find "${dir}" -maxdepth 1 -type f -print0 2>/dev/null | LC_ALL=C sort -z)
}

# ${2}: "module" for runtime entries, "source" for toolchain ones, which
# already carry a versioned upstream URL.
emit_index_table() {
    local index="$1" provenance="$2" pkg url license module
    if [[ "${provenance}" == "module" ]]; then
        printf '| Package | License | Module |\n'
    else
        printf '| Package | License | Source |\n'
    fi
    printf '|---------|---------|--------|\n'

    while IFS=, read -r pkg url license module; do
        [[ -z "${pkg}" ]] && continue
        # shellcheck disable=SC2016  # backticks are literal markdown here.
        if [[ "${provenance}" == "module" ]]; then
            printf '| `%s` | %s | `%s` |\n' "${pkg}" "${license:-Unknown}" "${module:-unknown}"
        else
            printf '| `%s` | %s | %s |\n' "${pkg}" "${license:-Unknown}" "${url:-n/a}"
        fi
    done < "${index}"
}

emit_sections() {
    local index="$1" root="$2" provenance="$3"
    local pkg url license module files lf fence

    while IFS=, read -r pkg url license module; do
        [[ -z "${pkg}" ]] && continue

        printf '### %s\n\n' "${pkg}"
        printf '* License: %s\n' "${license:-Unknown}"
        if [[ "${provenance}" == "module" ]]; then
            printf '* Module: %s\n\n' "${module:-unknown}"
        else
            printf '* Source: %s\n\n' "${url:-n/a}"
        fi

        files=()
        while IFS= read -r lf; do
            [[ -n "${lf}" ]] && files+=("${lf}")
        done < <(license_files_for "${root}/${pkg}")

        if (( ${#files[@]} == 0 )); then
            printf 'License text unavailable. See upstream source for the full license.\n'
        else
            for lf in "${files[@]}"; do
                fence="$(fence_for "${lf}")"
                printf '#### %s\n\n' "$(basename "${lf}")"
                printf '%stext\n' "${fence}"
                cat "${lf}"
                echo
                printf '%s\n' "${fence}"
                echo
            done
        fi
        echo
    done < "${index}"
}

compose_document() {
    log "Composing ${OUTPUT}..."
    # Build in a temp file and move into place, so a failure part way through
    # does not leave the committed file truncated in a developer's worktree.
    {
        cat <<'EOF'
# Third-Party Notices

NVIDIA Container Toolkit

This file covers the **Go dependencies** of this project, with the verbatim text
of each one's license. Two surfaces are covered:

* **Runtime dependencies** — Go modules statically linked into the commands
  under `cmd/`, resolved as the union across every released platform. All seven
  of those commands are distributed: `nvidia-container-runtime-hook`,
  `nvidia-container-runtime`, `nvidia-container-runtime.cdi`,
  `nvidia-container-runtime.legacy`, `nvidia-ctk` and `nvidia-cdi-hook` ship in
  the deb and rpm packages, and `nvidia-ctk-installer` ships in the
  `container-toolkit` image, which also carries the unpacked contents of those
  same packages.
* **Build toolchain** — Go modules behind the helpers pinned in
  `deployments/devel/go.mod`: `moq`, which generates the committed `*_mock.go`
  files, and `go-licenses`, which generates this document. These run at build
  time and are not shipped, but both produce artifacts that are committed to
  this repository, so the chain behind them is listed here too.

Go standard library packages are excluded; they are covered by the license of
the Go distribution itself.

## Scope

This document inventories Go dependencies only. Two things it deliberately does
not cover:

* **`third_party/libnvidia-container`.** That directory is a git submodule
  pointing at <https://github.com/NVIDIA/libnvidia-container>, a separate
  upstream C project. It is not merely vendored source: `scripts/build-packages.sh`
  initialises the submodule and `scripts/build-all-components.sh` drives its own
  `mk/docker.mk`, so `libnvidia-container1`, `libnvidia-container-tools` and
  `libnvidia-container-dev` are published in the same release tarballs and
  unpacked into the same `container-toolkit` image as the Go binaries listed
  below. Its build also fetches C dependencies of its own that are not vendored
  anywhere in this repository — elftoolchain's libelf, libtirpc and
  nvidia-modprobe-utils.

  None of that is inventoried here, because `libnvidia-container` ships its own
  compliance artifacts. `NOTICE`, `LICENSE`, `COPYING` and `COPYING.LESSER` are
  installed into each of its packages, and its `NOTICE` states the LGPL terms
  that apply when it is dynamically linked against libelf from elfutils. Those
  files, not this one, are authoritative for that component. Nothing under
  `third_party/` is built by this repository's `make cmds`, and no part of it is
  linked into the Go binaries listed below.
* **Non-Go contents of the released container image** — the packages inherited
  from the distroless base image and the statically linked BusyBox binary with
  its applet symlinks. Base image dependencies are covered by that image's own
  compliance process, and BusyBox is handled separately, including any
  source-distribution obligations it carries.

The end-to-end suite under `tests/` and the fixtures under `testdata/` are
separate Go modules that are neither built nor shipped as part of a release, so
their dependencies are not listed here either.

Go runtime entries identify their dependency as `module@version` rather than a
URL. `go-licenses` in vendor mode reports a link into this repository at `HEAD`,
which stops describing the released content once `main` advances;
`module@version` is immutable and names the upstream project directly.

This file is generated. Run `make notices` to regenerate it; `make
notices-check` verifies it is current.

## Runtime Dependency Index

EOF
        emit_index_table "${INDEX_FILE}" module

        cat <<'EOF'

## Build Toolchain Dependency Index

EOF
        emit_index_table "${TOOLS_INDEX}" source

        cat <<'EOF'

## Runtime Dependency License Texts

EOF
        emit_sections "${INDEX_FILE}" "${LICENSES_DIR}" module

        cat <<'EOF'
## Build Toolchain License Texts

EOF
        emit_sections "${TOOLS_INDEX}" "${LICENSES_DIR}/.tools" source
    } > "${OUT_TMP}"
    mkdir -p "$(dirname "${OUTPUT}")"
    cp "${OUT_TMP}" "${OUTPUT}"
    # mktemp creates 0600; the committed file should be world-readable.
    chmod 644 "${OUTPUT}"
}

# --- entry point ---

main() {
    check_prerequisites
    verify_platform_matrix
    prepare_workspace

    collect_runtime
    collect_toolchain
    build_indexes
    compose_document

    # Index rows are per package, not per module: one module can own several.
    local runtime_count toolchain_count
    runtime_count=$(wc -l < "${INDEX_FILE}" | tr -d ' ')
    toolchain_count=$(wc -l < "${TOOLS_INDEX}" | tr -d ' ')
    log "Wrote ${OUTPUT} (${runtime_count} Go runtime packages, ${toolchain_count} toolchain packages)"
}

main "$@"
