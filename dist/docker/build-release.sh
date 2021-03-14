#!/bin/sh
# used to build a stable image from the latest release from mdzio/ccu-jack
#
export BUILD_VERSION=$(curl -Ls https://api.github.com/repos/mdzio/ccu-jack/releases/latest | grep -oP '"tag_name": "v\K(.*)(?=")')
docker build --rm --no-cache \
    --build-arg BUILD_VERSION="${BUILD_VERSION}" \
    --build-arg BUILD_DATE="$(date +"%Y-%m-%dT%H:%M:%SZ")" \
    --tag ccu-jack:latest --tag ccu-jack:${BUILD_VERSION} .
