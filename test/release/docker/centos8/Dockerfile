ARG BASEIMAGE=centos:8
FROM ${BASEIMAGE}

ARG BASEIMAGE
# See https://www.centos.org/centos-linux-eol/
# and https://stackoverflow.com/a/70930049 for move to vault.centos.org
# and https://serverfault.com/questions/1093922/failing-to-run-yum-update-in-centos-8 for move to vault.epel.cloud
RUN [[ "${BASEIMAGE}" != "centos:8" ]] || \
    ( \
      sed -i 's/mirrorlist/#mirrorlist/g' /etc/yum.repos.d/CentOS-Linux-* && \
      sed -i 's|#baseurl=http://mirror.centos.org|baseurl=http://vault.epel.cloud|g' /etc/yum.repos.d/CentOS-Linux-* \
    )

RUN yum install -y \
    yum-utils \
    ruby-devel \
    gcc \
    make \
    rpm-build \
    rubygems \
    createrepo

RUN gem install --no-document fpm

# We create and install a dummy docker package since these dependencies are out of
# scope for the tests performed here.
RUN fpm -s empty \
    -t rpm \
    --description "A dummy package for docker-ce_18.06.3.ce-3.el7" \
    -n docker-ce --version 18.06.3.ce-3.el7 \
    -p /tmp/docker.rpm \
    && \
    yum localinstall -y /tmp/docker.rpm \
    && \
    rm -f /tmp/docker.rpm


ARG WORKFLOW=nvidia-docker
RUN curl -s -L https://nvidia.github.io/${WORKFLOW}/centos8/nvidia-docker.repo \
    | tee /etc/yum.repos.d/nvidia-docker.repo

COPY entrypoint.sh /
COPY install_repo.sh /

ENTRYPOINT [ "/entrypoint.sh" ]