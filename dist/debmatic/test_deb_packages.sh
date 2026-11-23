#!/bin/sh
set -eu

LC_ALL=C
export LC_ALL

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 version" >&2
  exit 1
fi

version=$1
test_dir=$(mktemp -d)
trap 'rm -rf "$test_dir"' 0

for architecture in armhf arm64 i386 amd64; do
  package="ccu-jack_${version}-1_${architecture}.deb"
  test "$(dpkg-deb -f "$package" Package)" = ccu-jack
  test "$(dpkg-deb -f "$package" Architecture)" = "$architecture"
  test "$(dpkg-deb -f "$package" Version)" = "$version-1"
  mkdir "$test_dir/$architecture"
  dpkg-deb -x "$package" "$test_dir/$architecture"
  addon="$test_dir/$architecture/usr/local/addons/ccu-jack"
  test -x "$addon/ccu-jack"
  test ! -e "$addon/bin"
  case "$architecture" in
    armhf) machine=ARM ;;
    arm64) machine=AArch64 ;;
    i386) machine='Intel 80386' ;;
    amd64) machine='Advanced Micro Devices X86-64' ;;
  esac
  readelf -h "$addon/ccu-jack" | grep -F "Machine:" | grep -F "$machine" >/dev/null
done

echo "Debian package architecture tests passed"
