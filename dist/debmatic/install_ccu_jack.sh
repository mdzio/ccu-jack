#!/bin/sh
set -eu

if [ "$#" -gt 1 ]; then
  echo "Usage: $0 [version]" >&2
  exit 1
fi

for command in curl dpkg sudo; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "Required command not found: $command" >&2
    exit 1
  fi
done

architecture=$(dpkg --print-architecture)
case "$architecture" in
  armhf|arm64|i386|amd64) ;;
  *) echo "Unsupported Debian architecture: $architecture" >&2; exit 1 ;;
esac

release_url=https://github.com/mdzio/ccu-jack/releases
if [ "$#" -eq 1 ]; then
  version=${1#v}
else
  if ! latest_url=$(curl --fail --silent --show-error --location \
    --output /dev/null --write-out '%{url_effective}' "$release_url/latest"); then
    echo "Could not determine the latest release" >&2
    exit 1
  fi
  case "$latest_url" in
    "$release_url"/tag/v*) version=${latest_url#"$release_url"/tag/v} ;;
    *) echo "Unexpected latest release URL: $latest_url" >&2; exit 1 ;;
  esac
fi

if ! printf '%s\n' "$version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+([~-][0-9A-Za-z.+~-]+)?$'; then
  echo "Invalid release version: $version" >&2
  exit 1
fi

package="ccu-jack_${version}-1_${architecture}.deb"
work_dir=$(mktemp -d)
trap 'rm -f "$work_dir/$package"; rmdir "$work_dir"' 0
chmod 755 "$work_dir"
echo "Downloading $package"
curl --fail --show-error --location --output "$work_dir/$package" \
  "$release_url/download/v$version/$package"
chmod 644 "$work_dir/$package"
sudo apt install --reinstall "$work_dir/$package"
