#!/usr/bin/env bash

: "${ALL_DEBS:? Must set ALL_DEBS}"
: "${ALL_RPMS:? Must set ALL_RPMS}"
: "${GPG_LOCAL_USER:? Must set GPG_LOCAL_USER}"
: "${TARGETS:? Must set TARGETS}"

set -x -e

function deb-sign {
	local last_found
	for r in ${*}; do
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
			--local-user ${GPG_LOCAL_USER} \
		> InRelease
}

function rpm-sign {
	for r in ${*}; do
		if [ -f "./${r}" ]; then
			rpmsign --addsign --key-id A04EA552 --digest-algo=sha512 "${r}"
		fi
	done
	createrepo -v --no-database -s sha512 --compress-type xz --revision "1.0" .
	gpg2 --batch --yes --expert --sign --detach-sign \
		--armor \
		--no-emit-version \
		--no-comments --personal-digest-preferences sha512 \
		--local-user ${GPG_LOCAL_USER} \
	repodata/repomd.xml
}

function sign() {
	local target=$1
    local dst_root=$2

	local src_dist=${target%-*}
    local dist=${src_dist/amazonlinux/amzn}

	local pkg_type
    case ${src_dist} in
    amazonlinux*) pkg_type=rpm
        ;;
    centos*) pkg_type=rpm
        ;;
    debian*) pkg_type=deb
        ;;
    opensuse-leap*) pkg_type=rpm
        ;;
    ubuntu*) pkg_type=deb
        ;;
    *) echo "ERROR: unexpected distribution ${src_dist}"
        ;;
    esac

    local arch=${target##*-}
    case ${src_dist} in
    ubuntu*) arch=${arch//ppc64le/ppc64el}
    esac

    local dst=${dst_root}/${dist}/${arch}

	if [[ ! -d ${dst} ]]; then
		echo "Directory ${dst} not found. Skipping"
		return
	fi

	cd ${dst}
	if [[ -f "/etc/debian_version" ]]; then
		[[ ${pkg_type} == "deb" ]] && deb-sign ${ALL_DEBS}
	else
		[[ ${pkg_type} == "rpm" ]] && rpm-sign ${ALL_RPMS}
	fi
	cd -
}

for target in ${TARGETS[@]}; do
    sign ${target} $(pwd)
done
