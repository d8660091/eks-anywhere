# should have the certs mount, and etcd endpoint passed arguments
# docker run -it -v .:/opt 7f9f4513 bash ./opt/hello.sh
echo running etcdctl
etcdctl version
