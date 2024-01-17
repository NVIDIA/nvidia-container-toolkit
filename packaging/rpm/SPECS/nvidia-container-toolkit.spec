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
Source2: LICENSE
Source3: nvidia-container-runtime
Source4: nvidia-container-runtime.cdi
Source5: nvidia-container-runtime.legacy

Obsoletes: nvidia-container-runtime <= 3.5.0-1, nvidia-container-runtime-hook <= 1.4.0-2
Provides: nvidia-container-runtime
Provides: nvidia-container-runtime-hook
Requires: libnvidia-container-tools >= %{libnvidia_container_tools_version}, libnvidia-container-tools < 2.0.0
Requires: nvidia-container-toolkit-base == %{version}-%{release}

%description
Provides tools and utilities to enable GPU support in containers.

%prep
cp %{SOURCE0} %{SOURCE1} %{SOURCE2} %{SOURCE3} %{SOURCE4} %{SOURCE5} .

%install
mkdir -p %{buildroot}%{_bindir}
install -m 755 -t %{buildroot}%{_bindir} nvidia-container-runtime-hook
install -m 755 -t %{buildroot}%{_bindir} nvidia-container-runtime
install -m 755 -t %{buildroot}%{_bindir} nvidia-container-runtime.cdi
install -m 755 -t %{buildroot}%{_bindir} nvidia-container-runtime.legacy
install -m 755 -t %{buildroot}%{_bindir} nvidia-ctk

%post
if [ $1 -gt 1 ]; then  # only on package upgrade
  mkdir -p %{_localstatedir}/lib/rpm-state/nvidia-container-toolkit
  cp -af %{_bindir}/nvidia-container-runtime-hook %{_localstatedir}/lib/rpm-state/nvidia-container-toolkit
fi

%posttrans
if [ ! -e %{_bindir}/nvidia-container-runtime-hook ]; then
  # repairing lost file nvidia-container-runtime-hook
  cp -avf %{_localstatedir}/lib/rpm-state/nvidia-container-toolkit/nvidia-container-runtime-hook %{_bindir}
fi
rm -rf %{_localstatedir}/lib/rpm-state/nvidia-container-toolkit
ln -sf %{_bindir}/nvidia-container-runtime-hook %{_bindir}/nvidia-container-toolkit

# Generate the default config; If this file already exists no changes are made.
%{_bindir}/nvidia-ctk --quiet config --config-file=%{_sysconfdir}/nvidia-container-runtime/config.toml --in-place

%postun
if [ "$1" = 0 ]; then  # package is uninstalled, not upgraded
  if [ -L %{_bindir}/nvidia-container-toolkit ]; then rm -f %{_bindir}/nvidia-container-toolkit; fi
fi

%files
%license LICENSE
%{_bindir}/nvidia-container-runtime-hook

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
%{_bindir}/nvidia-container-runtime
%{_bindir}/nvidia-ctk

# The OPERATOR EXTENSIONS package consists of components that are required to enable GPU support in Kubernetes.
# This package is not distributed as part of the NVIDIA Container Toolkit RPMs.
%package operator-extensions
Summary: NVIDIA Container Toolkit Operator Extensions
Requires: nvidia-container-toolkit-base == %{version}-%{release}

%description operator-extensions
Provides tools for using the NVIDIA Container Toolkit with the GPU Operator

%files operator-extensions
%license LICENSE
%{_bindir}/nvidia-container-runtime.cdi
%{_bindir}/nvidia-container-runtime.legacy
