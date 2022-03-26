#!/usr/bin/env bash

mkdir dist
for arch in "amd64" "arm64"
do
  GOARCH=$arch make pairmesh
  tar -czf dist/"pairmesh_$(git describe --tags --abbrev=0)_macos_$arch.tar.gz" -C bin pairmesh
done