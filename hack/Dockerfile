FROM alpine:3.14

RUN apk add --no-cache bash

# Set the version of etcdctl
ENV ETCDCTL_VERSION=3.5.0

# Download etcdctl
RUN wget https://github.com/etcd-io/etcd/releases/download/v${ETCDCTL_VERSION}/etcd-v${ETCDCTL_VERSION}-linux-amd64.tar.gz

# Extract etcdctl
RUN tar xzvf etcd-v${ETCDCTL_VERSION}-linux-amd64.tar.gz && \
    mv etcd-v${ETCDCTL_VERSION}-linux-amd64/etcdctl /usr/local/bin/etcdctl && \
    rm -rf etcd-v${ETCDCTL_VERSION}-linux-amd64.tar.gz etcd-v${ETCDCTL_VERSION}-linux-amd64

