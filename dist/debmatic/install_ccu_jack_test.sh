#!/bin/sh
set -eu

script_dir=$(CDPATH= cd "$(dirname "$0")" && pwd)
test_dir=$(mktemp -d)
trap 'rm -rf "$test_dir"' 0
export TEST_ROOT="$test_dir"
mkdir -p "$test_dir/tools"

cat > "$test_dir/tools/dpkg" <<'EOF'
#!/bin/sh
test "$1" = --print-architecture
echo "$TEST_ARCH"
EOF
cat > "$test_dir/tools/curl" <<'EOF'
#!/bin/sh
case "$*" in
  *releases/latest)
    test "${TEST_FAIL:-}" != latest || exit 22
    printf '%s' "${TEST_LATEST_URL:-https://github.com/mdzio/ccu-jack/releases/tag/v2.14.0}"
    ;;
  *)
    test "${TEST_FAIL:-}" != download || exit 22
    while [ "$#" -gt 0 ]; do
      case "$1" in
        --output) output=$2; shift 2 ;;
        https://*) url=$1; shift ;;
        *) shift ;;
      esac
    done
    printf '%s\n' "$url" > "$TEST_ROOT/download-url"
    printf '%s\n' package > "$output"
    ;;
esac
EOF
cat > "$test_dir/tools/sudo" <<'EOF'
#!/bin/sh
test "$1" = apt && test "$2" = install && test "$3" = --reinstall
test "$(cat "$4")" = package
printf '%s\n' "${4##*/}" > "$TEST_ROOT/installed"
EOF
chmod +x "$test_dir/tools/dpkg" "$test_dir/tools/curl" "$test_dir/tools/sudo"
PATH="$test_dir/tools:$PATH"
export PATH

for TEST_ARCH in armhf arm64 i386 amd64; do
  export TEST_ARCH
  sh "$script_dir/install_ccu_jack.sh" > "$test_dir/output" 2>&1
  package="ccu-jack_2.14.0-1_${TEST_ARCH}.deb"
  test "$(cat "$test_dir/installed")" = "$package"
  test "$(cat "$test_dir/download-url")" = \
    "https://github.com/mdzio/ccu-jack/releases/download/v2.14.0/$package"
done

sh "$script_dir/install_ccu_jack.sh" v2.15.0-rc.1 > "$test_dir/output" 2>&1
test "$(cat "$test_dir/installed")" = ccu-jack_2.15.0-rc.1-1_amd64.deb
rm "$test_dir/installed"

for failure in unsupported invalid latest download; do
  TEST_ARCH=arm64
  TEST_FAIL=
  export TEST_ARCH TEST_FAIL
  case "$failure" in
    unsupported) TEST_ARCH=armel ;;
    invalid) TEST_LATEST_URL=https://example.com/tag/v2.14.0; export TEST_LATEST_URL ;;
    latest|download) TEST_FAIL=$failure ;;
  esac
  if sh "$script_dir/install_ccu_jack.sh" > "$test_dir/output" 2>&1; then
    echo "Installer accepted $failure" >&2
    exit 1
  fi
  test ! -e "$test_dir/installed"
  unset TEST_LATEST_URL
done

echo "Debmatic installer tests passed"
