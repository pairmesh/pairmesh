#!/usr/bin/env bash

mkdir dist
for arch in "amd64" "arm64"
do
  GOARCH=$arch make pairmesh
  tar -czf dist/"pairmesh-$(git describe --tags --abbrev=0)-macos-$arch.tar.gz" -C bin pairmesh
done