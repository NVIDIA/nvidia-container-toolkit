#!/usr/bin/env bash
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
#
# Writes THIRD_PARTY_NOTICES.md for the Go modules linked into ./cmd/... (vendored).

set -euo pipefail

# LC_ALL=C on every sort and grep below: collation and case folding must not vary
# by locale (under tr_TR glibc will not fold I to i, so LICENSE stops matching).

OUTPUT="${OUTPUT:-THIRD_PARTY_NOTICES.md}"
LICENSES_DIR="${LICENSES_DIR:-.licenses-cache}"
MULTI_ARCH_MK="${MULTI_ARCH_MK:-deployments/container/multi-arch.mk}"
MODULES_TXT="${MODULES_TXT:-vendor/modules.txt}"

# Exactly what 'make cmds' builds and ships.
PACKAGES=("./cmd/...")

# Must match the released image platforms; verify_platform_matrix fails on
# drift. go-licenses resolves one platform per run, so collection runs per
# target and merges.
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
)

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
    # -a: a license containing a NUL byte is otherwise treated as binary and
    # grep prints "Binary file ... matches" rather than the matches themselves.
    longest=$(LC_ALL=C grep -oaE '`+' "${file}" 2>/dev/null \
        | awk '{ if (length($0) > m) m = length($0) } END { print m+0 }')
    width=$(( longest + 1 ))
    (( width < 3 )) && width=3
    printf '%*s' "${width}" '' | tr ' ' '`'
}

check_prerequisites() {
    command -v go >/dev/null 2>&1 || die "go is not installed."

    # -x alone is not enough: bin/ is bind-mounted into the build image by the
    # docker-% targets, so a host-built binary is present but not executable there.
    if ./bin/go-licenses --help >/dev/null 2>&1; then
        GO_LICENSES="${PWD}/bin/go-licenses"
    elif command -v go-licenses >/dev/null 2>&1; then
        GO_LICENSES="$(command -v go-licenses)"
    else
        die "go-licenses is not installed." "Install it with 'make bin/go-licenses'."
    fi

    local f
    for f in "${MULTI_ARCH_MK}" "${MODULES_TXT}"; do
        [[ -f "${f}" ]] || die "${f} not found — run 'make third-party-notices' from the repo root."
    done

    LOCAL_MODULE=$(go list -m 2>/dev/null || true)
    [[ -n "${LOCAL_MODULE}" ]] || die "could not determine local module path via 'go list -m'."

    # CGO must stay on: with CGO_ENABLED=0 the build constraints exclude every
    # file in go-nvml/pkg/dl and internal/cuda, so go-licenses cannot load
    # ./cmd/... at all. No C compiler is needed; go-licenses never compiles.
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
        "Update the PLATFORMS array in hack/generate-third-party-notices.sh to match the released targets." \
        "  matrix (PLATFORMS): $(echo "${actual}" | paste -sd ' ' -)" \
        "  image platforms:    $(echo "${expected}" | paste -sd ' ' -)"
}

prepare_workspace() {
    # Guard the override: '', '/', '.' or '..' would make the rm -rf fatal.
    case "${LICENSES_DIR}" in
        ""|"/"|"."|"..")
            die "refusing to 'rm -rf' unsafe LICENSES_DIR='${LICENSES_DIR}'."
            ;;
    esac
    rm -rf "${LICENSES_DIR}"
    mkdir -p "${LICENSES_DIR}"

    local t="${TMPDIR:-/tmp}/nvidia-container-toolkit-notices"
    SAVE_ROOT="$(mktemp -d "${t}.XXXXXX")"
    COMBINED_CSV="$(mktemp "${t}-csv.XXXXXX")"
    INDEX_FILE="$(mktemp "${t}-idx.XXXXXX")"

    # Composed next to OUTPUT, not in TMPDIR, so the publish below is a rename.
    local out_dir
    out_dir="$(dirname "${OUTPUT}")"
    mkdir -p "${out_dir}"
    OUT_TMP="$(mktemp "${out_dir}/.$(basename "${OUTPUT}").XXXXXX")"

    trap 'rm -rf "${SAVE_ROOT}"; rm -f "${COMBINED_CSV}" "${INDEX_FILE}" "${OUT_TMP}"' EXIT
}

collect_runtime() {
    local platform goos goarch save_dir

    for platform in "${PLATFORMS[@]}"; do
        goos="${platform%/*}"
        goarch="${platform#*/}"
        log "Collecting licenses for ${goos}/${goarch}..."

        save_dir="${SAVE_ROOT}/${goos}_${goarch}"

        # Only the local module: --ignore matches raw string prefixes, not path
        # segments, so a stdlib list adds the token "go" and silently drops
        # golang.org/x/*, google.golang.org/* and gopkg.in/*.
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

# Module cache files are 0444 and cp preserves that, so the next platform's copy
# fails unless write permission is restored.
merge_licenses() {
    cp -R "$1/." "$2/"
    chmod -R u+w "$2"
}

# Licenses are joined, not picked: go-licenses emits a row per recognized license,
# so keeping one would hide filepath-securejoin's MPL-2.0 behind its BSD-3-Clause.
collapse_index() {
    LC_ALL=C sort -u "$1" | awk -F, '
        {
            pkg = $1
            if (!(pkg in url)) { url[pkg] = $2; order[++n] = pkg }
            if (!((pkg SUBSEP $3) in seen)) {
                seen[pkg SUBSEP $3] = 1
                # Count, do not test "pkg in lic": mawk instantiates the
                # assignment target before evaluating the right-hand side, so
                # that test is true on the first row and BSD awk disagrees.
                lic[pkg] = (cnt[pkg]++ ? lic[pkg] " / " : "") $3
            }
        }
        END { for (i = 1; i <= n; i++) print order[i] "," url[order[i]] "," lic[order[i]] }
    '
}

# Rows carry module@version, not a URL: in vendor mode go-licenses points into
# this repo at HEAD, which stops describing released content once main moves.
# Longest-prefix match, because a license may sit below the module root.
annotate_modules() {
    awk -v modfile="${MODULES_TXT}" '
        BEGIN {
            FS = OFS = ","
            while ((getline line < modfile) > 0) {
                if (line !~ /^# /) continue
                split(line, f, " ")
                # "# <path> <version>", optionally "=> <path> <version>". The
                # replacement is what is vendored; a filesystem replace has no
                # version, so stop rather than misstate it.
                if (f[4] == "=>" || f[3] == "=>") {
                    r = (f[4] == "=>") ? 5 : 4
                    if (f[r + 1] == "") {
                        print "ERROR: " modfile " replaces " f[2] " with a local path;" > "/dev/stderr"
                        print "teach hack/generate-third-party-notices.sh how to attribute it." > "/dev/stderr"
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
            # A read error makes getline return -1 and the loop never runs.
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

    [[ -s "${INDEX_FILE}" ]] \
        || die "go-licenses produced no entries for ${PACKAGES[*]} — refusing to write empty notices file."

    if cut -d, -f4 "${INDEX_FILE}" | LC_ALL=C grep -qx 'unknown'; then
        die "some runtime packages could not be matched to a module in ${MODULES_TXT}." \
            "Re-run 'make vendor' first; if it persists, fix annotate_modules in hack/generate-third-party-notices.sh."
    fi

    # An unclassifiable license is reported as "Unknown" with a zero exit, so
    # without this an entry that attributes nothing would ship.
    if cut -d, -f3 "${INDEX_FILE}" | LC_ALL=C grep -qE '(^| / )Unknown( / |$)'; then
        die "go-licenses could not identify a license for some dependencies." \
            "Check the entries reported as Unknown before committing the file."
    fi
}

# Filter by name: for restricted licenses 'go-licenses save' copies the whole
# module source.
license_files_for() {
    local dir="$1" f
    [[ -d "${dir}" ]] || return 0
    while IFS= read -r -d '' f; do
        if printf '%s' "$(basename "${f}")" \
            | LC_ALL=C grep -qiE '^(licen[cs]e|notice|copying|copyright|authors|patents)([-._].*)?$'; then
            printf '%s\n' "${f}"
        fi
    done < <(find "${dir}" -maxdepth 1 -type f -print0 2>/dev/null | LC_ALL=C sort -z)
}

emit_index_table() {
    local index="$1" pkg _url license module
    printf '| Package | License | Module |\n'
    printf '|---------|---------|--------|\n'

    while IFS=, read -r pkg _url license module; do
        [[ -z "${pkg}" ]] && continue
        # shellcheck disable=SC2016  # backticks are literal markdown here.
        printf '| `%s` | %s | `%s` |\n' "${pkg}" "${license:-Unknown}" "${module:-unknown}"
    done < "${index}"
}

emit_sections() {
    local index="$1" root="$2"
    local pkg _url license module files lf fence

    while IFS=, read -r pkg _url license module; do
        [[ -z "${pkg}" ]] && continue

        printf '### %s\n\n' "${pkg}"
        printf '* License: %s\n' "${license:-Unknown}"
        printf '* Module: %s\n\n' "${module:-unknown}"

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
    {
        cat <<'EOF'
# Third-Party Notices

NVIDIA Container Toolkit

This file lists every third-party dependency that NVIDIA Container Toolkit
redistributes, along with the verbatim text of each dependency's license. In
particular, this covers all **Go modules** statically linked into the commands
under `cmd/`. The `nvidia-container-runtime-hook`, `nvidia-container-runtime`,
`nvidia-container-runtime.cdi`, `nvidia-container-runtime.legacy`, `nvidia-ctk`
and `nvidia-cdi-hook` commands ship in the deb and rpm packages. The
`nvidia-ctk-installer` command ships in the `container-toolkit` image.
Go standard library packages are excluded; they are covered by the license of
the Go distribution itself.

The `container-toolkit` image uses `nvcr.io/nvidia/distroless/go` as a base image.
All of the OSS packages and source included in this image can be found at
https://developer.nvidia.com/w/distroless-oss/index.html. A statically compiled
busybox binary is added to the image, which is licensed under GPLv2.

## Go Module Index

EOF
        emit_index_table "${INDEX_FILE}"

        cat <<'EOF'

## Go Module License Texts

EOF
        emit_sections "${INDEX_FILE}" "${LICENSES_DIR}"
    } > "${OUT_TMP}"

    # mv, not cp: OUT_TMP is in OUTPUT's directory, so this is a rename(2) and
    # OUTPUT is never a partial write. mktemp creates 0600, hence the chmod.
    chmod 644 "${OUT_TMP}"
    mv "${OUT_TMP}" "${OUTPUT}"
}

main() {
    check_prerequisites
    verify_platform_matrix
    prepare_workspace

    collect_runtime
    build_indexes
    compose_document

    local runtime_count
    runtime_count=$(wc -l < "${INDEX_FILE}" | tr -d ' ')
    log "Wrote ${OUTPUT} (${runtime_count} Go packages)"
}

main "$@"
