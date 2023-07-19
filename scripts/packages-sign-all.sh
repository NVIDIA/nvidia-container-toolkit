#!/usr/bin/env bash

: "${ALL_DEBS:? Must set ALL_DEBS}"
: "${ALL_RPMS:? Must set ALL_RPMS}"
: "${GPG_LOCAL_USER:? Must set GPG_LOCAL_USER}"
: "${TARGETS:? Must set TARGETS}"

set -x -e

function deb-sign {
    local last_found
    for r in "$@"; do
        if [ -f "./${r}" ]; then
            last_found=${r}
        fi
    done
    if [[ -z ${last_found} ]]; then
        echo "WARNING: No expected package found in $(pwd); skipping signing of repo;"
        return
    fi
    apt-ftparchive packages . \
        | tee Packages \
        | xz > Packages.xz
    apt-ftparchive -c repo.conf release . \
        | gpg --batch --yes --expert --clearsign \
            --armor \
            --no-emit-version \
            --no-comments \
            --personal-digest-preferences sha512 \
            --local-user "${GPG_LOCAL_USER}" \
        > InRelease
}

function rpm-sign {
    for r in "$@"; do
        if [ -f "./${r}" ]; then
            rpmsign --addsign --key-id A04EA552 --digest-algo=sha512 "${r}"
        fi
    done
    createrepo -v --no-database -s sha512 --compress-type xz --revision "1.0" .
    gpg2 --batch --yes --expert --sign --detach-sign \
        --armor \
        --no-emit-version \
        --no-comments --personal-digest-preferences sha512 \
        --local-user "${GPG_LOCAL_USER}" \
    repodata/repomd.xml
}

function sign() {
    local target=$1
    local dst_root=$2
    local by_package_type=$3

    local src_dist=${target%-*}
    local dst_dist=${src_dist/amazonlinux/amzn}

    local pkg_type=unknown
    local arch=${target##*-}
    local dst_arch=${arch}

    case ${src_dist} in
    amazonlinux*) pkg_type=rpm
        ;;
    centos* | rpm) pkg_type=rpm
        ;;
    debian*) pkg_type=deb
        ;;
    fedora*) pkg_type=rpm
        ;;
    opensuse-leap*) pkg_type=rpm
        ;;
    ubuntu* | deb) pkg_type=deb
        arch=${arch//ppc64le/ppc64el}
        ;;
    *) echo "ERROR: unexpected distribution ${src_dist}"
        ;;
    esac

    if [[ x"${by_package_type}" == x"true" ]]; then
        dst_dist=${pkg_type}
    fi

    local dst=${dst_root}/${dst_dist}/${arch}

    if [[ ! -d ${dst} ]]; then
        echo "Directory ${dst} not found. Skipping"
        return
    fi

    cd "${dst}"
    if [[ -f "/etc/debian_version" ]]; then
        [[ "${pkg_type}" == "deb" ]] && deb-sign ${ALL_DEBS}
    else
        [[ "${pkg_type}" == "rpm" ]] && rpm-sign ${ALL_RPMS}
    fi
    cd -
}

for target in ${TARGETS[@]}; do
    echo "checking target=${target}"
    by_package_type=
    case ${target} in
    ubuntu18.04-* | centos7-*)
        by_package_type="true"
        ;;
    centos8-ppc64le)
        by_package_type="false"
        ;;
    *)
        echo "Skipping target ${target}"
        continue
        ;;
    esac
    sign "${target}" "$(pwd)" ${by_package_type}
done
