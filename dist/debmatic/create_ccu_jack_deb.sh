#!/bin/bash

# Baut Debian-Pakete von CCU-Jack für Debmatic auf einem einzigen
# Debian/Ubuntu-Host für alle von Debmatic unterstützten Architekturen
# (armhf, arm64, i386, amd64). Statt Cross-Toolchains wird der native
# Crosscompile-Support von Go genutzt (GOOS/GOARCH/GOARM, CGO_ENABLED=0).
#
# Aufruf:   ./create_ccu_jack_deb.sh [Version]
#   Ohne Argument wird die Version aus build/main.go (appVersion) gelesen.
# Benötigt: go, dpkg-deb (dpkg >= 1.19.0 wegen --root-owner-group)

PKG_BUILD=1
PKG_ARCH=all

# Debian-Architektur => GOARCH. Debmatic armhf setzt ARMv7 voraus.
declare -A architectures=(["armhf"]="arm" ["arm64"]="arm64" ["i386"]="386" ["amd64"]="amd64")

function throw {
  echo "$1" >&2
  exit 1
}

function run {
  local description=$1
  local output
  local status
  shift
  echo -n "$description ... "
  if output=$("$@" 2>&1); then
    echo -e "\033[0;32mDone\033[0;0m"
  else
    status=$?
    echo -e "\033[1;91mFAILED\033[0;0m"
    echo "$output" >&2
    return "$status"
  fi
}

set -euo pipefail

command -v go >/dev/null || throw "go not found in PATH"
command -v dpkg-deb >/dev/null || throw "dpkg-deb not found in PATH"

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
PROJECT_DIR=$(cd "$SCRIPT_DIR/../.." && pwd)
CURRENT_DIR=$(pwd)
WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

APP_VERSION=${1:-$(sed -n 's/^[[:space:]]*appVersion = "\([^"]*\)".*/\1/p' "$PROJECT_DIR/build/main.go")}
[ -n "$APP_VERSION" ] || throw "appVersion not found in build/main.go, pass version as argument"
PKG_VERSION=$APP_VERSION-$PKG_BUILD

STAGE_DIR=$WORK_DIR/stage
ADDON_DIR=$STAGE_DIR/usr/local/addons/ccu-jack

# Architekturunabhängigen Paketinhalt einmalig zusammenstellen.
function build_stage {
  mkdir -p "$ADDON_DIR"
  mkdir -p "$ADDON_DIR/bin"
  mkdir -p "$STAGE_DIR/etc/config/addons/www/ccu-jack"
  mkdir -p "$STAGE_DIR/usr/lib/systemd/system"

  run "Copy ccu-jack.cfg" cp "$SCRIPT_DIR/ccu-jack.cfg" "$STAGE_DIR/etc/config/addons/"
  run "Copy config.cgi" cp "$SCRIPT_DIR/config.cgi" "$STAGE_DIR/etc/config/addons/www/ccu-jack/"
  run "Copy ccu-jack.service" cp "$SCRIPT_DIR/ccu-jack.service" "$STAGE_DIR/usr/lib/systemd/system/"
  run "Copy update-hm_addons" cp "$SCRIPT_DIR/update-hm_addons" "$ADDON_DIR/"
  run "Copy launcher" cp "$SCRIPT_DIR/ccu-jack-launcher" "$ADDON_DIR/ccu-jack"
  run "Copy update_hm_addons.tcl" cp "$PROJECT_DIR/dist/update_hm_addons.tcl" "$ADDON_DIR/"
  run "Copy webui" cp -R "$PROJECT_DIR/wd/webui" "$ADDON_DIR/"
  run "Copy third-party-licenses" cp -R "$PROJECT_DIR/third-party-licenses" "$ADDON_DIR/"
  run "Copy README.md" cp "$PROJECT_DIR/README.md" "$ADDON_DIR/"
  run "Copy LICENSE.txt" cp "$PROJECT_DIR/LICENSE.txt" "$ADDON_DIR/"

  echo -n "$APP_VERSION" > "$ADDON_DIR/VERSION"

  # Dateirechte unabhängig vom Checkout setzen.
  find "$STAGE_DIR" -type d -exec chmod 755 {} +
  find "$STAGE_DIR" -type f -exec chmod 644 {} +
  chmod 755 "$ADDON_DIR/ccu-jack" "$ADDON_DIR/update-hm_addons" "$ADDON_DIR/update_hm_addons.tcl"
  chmod 755 "$STAGE_DIR/etc/config/addons/www/ccu-jack/config.cgi"
}
echo -n "Build package stage ... "
build_stage
echo -e "\033[0;32mDone\033[0;0m"

function build_binary {
  cd "$PROJECT_DIR"
  GOOS=linux GOARCH=$GOARCH GOARM=$GOARM CGO_ENABLED=0 \
    go build -ldflags "-s -w -X main.appVersion=$APP_VERSION" \
    -o "$BINARY" .
}

function build_binaries {
  for ARCH in armhf arm64 i386 amd64
  do
    GOARCH=${architectures[$ARCH]}
    GOARM=
    [ "$GOARCH" == "arm" ] && GOARM=7
    BINARY="$ADDON_DIR/bin/ccu-jack-$ARCH"

    run "Compile binary ($ARCH)" build_binary
    chmod 755 "$BINARY"
  done
}
echo -n "Build architecture binaries ... "
build_binaries
echo -e "\033[0;32mDone\033[0;0m"

function build_package {
  TARGET_DIR="$WORK_DIR/ccu-jack_${PKG_VERSION}_$PKG_ARCH"
  run "Copy stage" cp -pR "$STAGE_DIR" "$TARGET_DIR"

  mkdir -p "$TARGET_DIR/DEBIAN"
  run "Copy DEBIAN templates" cp "$SCRIPT_DIR"/DEBIAN/* "$TARGET_DIR/DEBIAN/"
  for file in "$TARGET_DIR"/DEBIAN/*; do
    sed -i "s/{PKG_VERSION}/$PKG_VERSION/g" "$file"
    sed -i "s/{PKG_ARCH}/$PKG_ARCH/g" "$file"
  done
  chmod 644 "$TARGET_DIR/DEBIAN/control" "$TARGET_DIR/DEBIAN/conffiles"
  chmod 755 "$TARGET_DIR/DEBIAN/postinst" "$TARGET_DIR/DEBIAN/prerm" "$TARGET_DIR/DEBIAN/postrm"

  cd "$WORK_DIR"

  dpkg-deb --build --root-owner-group -Zxz "ccu-jack_${PKG_VERSION}_$PKG_ARCH" || throw "Error on dpkg-deb"
}

echo -n "Build universal package ... "
build_package
echo -e "\033[0;32mDone\033[0;0m"

cd "$WORK_DIR"

run "Copy packages to local directory" cp "$WORK_DIR"/ccu-jack_*.deb "$CURRENT_DIR"
