#!/bin/sh

docker run -i -v `pwd`:/gopath/src/github.com/fitbeard/libvirt_exporter docker.pkg.github.com/fitbeard/libvirt_exporter/libvirt_go:1.0 /bin/sh << 'EOF'
set -ex

# Build the libvirt_exporter
cd /gopath/src/github.com/fitbeard/libvirt_exporter
export GOPATH=/gopath
go get -v -t -d ./...
go build --ldflags '-extldflags "-static"'
strip libvirt_exporter
EOF
