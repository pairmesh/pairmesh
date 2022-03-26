#!/usr/bin/env bash

# Prepare packager
cd tools/appify && go build .
cd ../..

# Prepare distribution directory
mkdir dist

# Build and package application
for arch in "amd64" "arm64"
do
  make clean
  GOARCH=$arch make pairmesh
  tools/appify/appify  -name PairMesh -author "PairMesh.com" -version "$(git describe --always --tags --abbrev=0)" -id com.PairMesh.app -icon ./node/resources/icon_darwin.png bin/pairmesh
  zip dist/"pairmesh-$(git describe --tags --abbrev=0)-macos-$arch.zip" -r PairMesh.app
  rm -rf PairMesh.app
done
