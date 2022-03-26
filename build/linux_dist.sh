#!/usr/bin/env bash

mkdir dist
for arch in "amd64"
do
  GOARCH=$arch make pairmesh
  tar -czf dist/"pairmesh_$(git describe --tags --abbrev=0)_linux_$arch.tar.gz" -C bin pairmesh
done