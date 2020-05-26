#!/bin/sh

set -x

REPO_URL="github.com/fitbeard/libvirt_exporter"
BINARY_NAME=libvirt-exporter

docker run --rm \
  -v "$PWD"/../go/src:/go/src -w /go/src \
  -v "$PWD":/go/src/${REPO_URL} -w /go/src/${REPO_URL} \
  -e GOOS=linux \
  -e GOARCH=amd64 \
   docker.pkg.github.com/fitbeard/libvirt_exporter/libvirt_go:3.0 go build --ldflags '-extldflags "-static"' -o ${BINARY_NAME}

strip ${BINARY_NAME}
sudo apt-get upgrade upx-ucl -y
upx ${BINARY_NAME}
