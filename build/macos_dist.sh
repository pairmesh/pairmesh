#!/usr/bin/env bash

# Prepare packager
cd tools/appify && go build -o ../../bin/ . && cd ../..

# Prepare distribution directory
mkdir -p dist

# Remove previous building
rm -rf PairMesh.app

# Build and package application
for arch in "amd64" "arm64"
do
  # Build PairMesh entry process binary
  GO111MODULE=on CGO_ENABLED=1 GOARCH=$arch go build -ldflags "-s -w" -o bin/macos ./tools/macos
  bin/appify -name PairMesh -author "PairMesh.com" -version "$(git describe --always --tags --abbrev=0)" -id com.PairMesh.app -icon ./node/resources/icon_darwin.png bin/macos

  # Build PairMesh daemon process binary
  GOARCH=$arch make pairmesh
  mkdir PairMesh.app/Contents/Daemon
  cp bin/pairmesh PairMesh.app/Contents/Daemon/PairMesh
  zip dist/"pairmesh-$(git describe --tags --abbrev=0)-macos-$arch.zip" -r PairMesh.app
  rm -rf PairMesh.app
done
