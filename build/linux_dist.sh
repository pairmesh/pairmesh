#!/usr/bin/env bash

mkdir dist
for arch in "amd64" "arm64"
do
  unset -v CC
  if [ $arch == "arm64" ]
  then
    export CC=aarch64-linux-gnu-gcc
  fi
  GOARCH=$arch make pairmesh
  tar -czf dist/"pairmesh-$(git describe --tags --abbrev=0)-linux-$arch.tar.gz" -C bin pairmesh
done
