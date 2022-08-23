ARG BASEIMAGE=quay.io/centos/centos:stream8
FROM ${BASEIMAGE}

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


RUN curl -s -L https://nvidia.github.io/libnvidia-container/centos8/libnvidia-container.repo \
    | tee /etc/yum.repos.d/nvidia-container-toolkit.repo

COPY entrypoint.sh /
COPY install_repo.sh /

ENTRYPOINT [ "/entrypoint.sh" ]