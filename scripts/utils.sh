
# package_type returns the packaging type (deb or rpm) for the specfied distribution.
# An error is returned if the ditribution is unsupported.
function package_type() {
    local pkg_type
    case ${1} in
    amazonlinux*) pkg_type=rpm
        ;;
    centos*) pkg_type=rpm
        ;;
    debian*) pkg_type=deb
        ;;
    fedora*) pkg_type=rpm
        ;;
    opensuse-leap*) pkg_type=rpm
        ;;
    ubuntu*) pkg_type=deb
        ;;
    deb) pkg_type=deb
        ;;
    rpm) pkg_type=rpm
        ;;
    *) exit 1
        ;;
    esac
    echo "${pkg_type}"
}

function get_artifactory_repository() {
    local pkg_type=$(package_type $1)

    case ${pkg_type} in
    deb) echo "sw-gpu-cloudnative-debian-local"
        ;;
    rpm) echo "sw-gpu-cloudnative-rpm-local"
        ;;
    *) echo "sw-gpu-cloudnative-generic-local"
        ;;
    esac
}

function get_package_target() {
    local target=$1
    local dist=${target%-*}
    local arch=${target##*-}

    case ${target} in
    deb) echo ""
        ;;
    rpm) echo ""
        ;;
    *) echo "${dist}/${arch}"
        ;;
    esac
}
