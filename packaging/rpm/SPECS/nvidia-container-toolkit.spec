Name: nvidia-container-toolkit
Version: %{version}
Release: %{release}
Group: Development Tools

Vendor: NVIDIA CORPORATION
Packager: NVIDIA CORPORATION <cudatools@nvidia.com>

Summary: NVIDIA Container Toolkit
URL: https://github.com/NVIDIA/nvidia-container-toolkit
License: Apache-2.0

Source0: nvidia-container-runtime-hook
Source1: nvidia-ctk
Source2: config.toml
Source3: oci-nvidia-hook
Source4: oci-nvidia-hook.json
Source5: LICENSE
Source6: nvidia-container-runtime
Source7: nvidia-container-runtime.cdi

Obsoletes: nvidia-container-runtime <= 3.5.0-1, nvidia-container-runtime-hook <= 1.4.0-2
Provides: nvidia-container-runtime
Provides: nvidia-container-runtime-hook
Requires: libnvidia-container-tools >= %{libnvidia_container_tools_version}, libnvidia-container-tools < 2.0.0
Requires: nvidia-container-toolkit-base == %{version}-%{release}

%if 0%{?suse_version}
Requires: libseccomp2
Requires: libapparmor1
%else
Requires: libseccomp
%endif

%description
Provides tools and utilities to enable GPU support in containers.

%prep
cp %{SOURCE0} %{SOURCE1} %{SOURCE2} %{SOURCE3} %{SOURCE4} %{SOURCE5} %{SOURCE6} %{SOURCE7} .

%install
mkdir -p %{buildroot}%{_bindir}
install -m 755 -t %{buildroot}%{_bindir} nvidia-container-runtime-hook
install -m 755 -t %{buildroot}%{_bindir} nvidia-container-runtime
install -m 755 -t %{buildroot}%{_bindir} nvidia-container-runtime.cdi
install -m 755 -t %{buildroot}%{_bindir} nvidia-ctk

mkdir -p %{buildroot}/etc/nvidia-container-runtime
install -m 644 -t %{buildroot}/etc/nvidia-container-runtime config.toml

mkdir -p %{buildroot}/usr/libexec/oci/hooks.d
install -m 755 -t %{buildroot}/usr/libexec/oci/hooks.d oci-nvidia-hook

mkdir -p %{buildroot}/usr/share/containers/oci/hooks.d
install -m 644 -t %{buildroot}/usr/share/containers/oci/hooks.d oci-nvidia-hook.json

%post
mkdir -p %{_localstatedir}/lib/rpm-state/nvidia-container-toolkit
cp -af %{_bindir}/nvidia-container-runtime-hook %{_localstatedir}/lib/rpm-state/nvidia-container-toolkit

%posttrans
if [ ! -e %{_bindir}/nvidia-container-runtime-hook ]; then
  # reparing lost file nvidia-container-runtime-hook
  cp -avf %{_localstatedir}/lib/rpm-state/nvidia-container-toolkit/nvidia-container-runtime-hook %{_bindir}
fi
rm -rf %{_localstatedir}/lib/rpm-state/nvidia-container-toolkit
ln -sf %{_bindir}/nvidia-container-runtime-hook %{_bindir}/nvidia-container-toolkit

%postun
if [ -L %{_bindir}/nvidia-container-toolkit ] then; rm -f %{_bindir}/nvidia-container-toolkit; fi

%files
%license LICENSE
%{_bindir}/nvidia-container-runtime-hook
/usr/libexec/oci/hooks.d/oci-nvidia-hook
/usr/share/containers/oci/hooks.d/oci-nvidia-hook.json

%changelog
# As of 1.10.0-1 we generate the release information automatically
* %{release_date} NVIDIA CORPORATION <cudatools@nvidia.com> %{version}-%{release}
- See https://gitlab.com/nvidia/container-toolkit/container-toolkit/-/blob/%{git_commit}/CHANGELOG.md
- Bump libnvidia-container dependency to libnvidia-container-tools >= %{libnvidia_container_tools_version}

# The BASE package consists of the NVIDIA Container Runtime and the NVIDIA Container Toolkit CLI.
# This allows the package to be installed on systems where no NVIDIA Container CLI is available.
%package base
Summary: NVIDIA Container Toolkit Base
Obsoletes: nvidia-container-runtime <= 3.5.0-1, nvidia-container-runtime-hook <= 1.4.0-2
Provides: nvidia-container-runtime
# Since this package allows certain components of the NVIDIA Container Toolkit to be installed separately
# it conflicts with older versions of the nvidia-container-toolkit package that also provide these files.
Conflicts: nvidia-container-toolkit <= 1.10.0-1

%description base
Provides tools such as the NVIDIA Container Runtime and NVIDIA Container Toolkit CLI to enable GPU support in containers.

%files base
%license LICENSE
%config /etc/nvidia-container-runtime/config.toml
%{_bindir}/nvidia-container-runtime
%{_bindir}/nvidia-container-runtime.cdi
%{_bindir}/nvidia-ctk
