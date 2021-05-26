#!/bin/sh
# used to build a "nightly" image from the latest source code at mdzio/ccu-jack which might not be a release yet
#
# TODO: multi-arch  
# export BUILD_VERSION=$(curl -Ls https://api.github.com/repos/mdzio/ccu-jack/releases/latest | grep -oP '"tag_name": "v\K(.*)(?=")')
export BUILD_VERSION=$(curl -Ls https://raw.githubusercontent.com/mdzio/ccu-jack/master/build/main.go | grep -oP 'appVersion = "\K(.*)(?=")')
docker build --rm --no-cache \
    --build-arg BUILD_VERSION="${BUILD_VERSION}" \
    --build-arg BUILD_DATE="$(date +"%Y-%m-%dT%H:%M:%SZ")" \
    --build-arg BUILD_VERSION_NIGHTLY="${BUILD_VERSION}-$(date +"%Y-%m-%d")" \
    --tag ccu-jack:${BUILD_VERSION}_nightly \
    -f Dockerfile.nightly .
