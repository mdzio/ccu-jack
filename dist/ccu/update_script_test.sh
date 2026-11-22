#!/bin/sh
set -eu

script_dir=$(CDPATH= cd "$(dirname "$0")" && pwd)
test_dir=$(mktemp -d)
trap 'rm -rf "$test_dir"' 0

export TEST_ROOT="$test_dir"
mkdir -p "$test_dir/tools" "$test_dir/package/addon" "$test_dir/package/bin" \
  "$test_dir/package/rc.d" "$test_dir/package/www" "$test_dir/package/etc" \
  "$test_dir/system/addons/ccu-jack/bin" "$test_dir/system/etc/config/rc.d"

cat > "$test_dir/tools/uname" <<'EOF'
#!/bin/sh
echo "$TEST_MACHINE"
EOF
chmod +x "$test_dir/tools/uname"
PATH="$test_dir/tools:$PATH"
export PATH

sed "s|/usr/local|$test_dir/system|g" "$script_dir/update_script" > "$test_dir/package/update_script"
cat > "$test_dir/package/addon/update_hm_addons.tcl" <<'EOF'
#!/bin/sh
exit 0
EOF
chmod +x "$test_dir/package/addon/update_hm_addons.tcl"
echo default > "$test_dir/package/addon/ccu-jack-default.cfg"
cat > "$test_dir/package/rc.d/ccu-jack" <<'EOF'
#!/bin/sh
touch "$TEST_ROOT/stopped"
EOF
chmod +x "$test_dir/package/rc.d/ccu-jack"
cp "$test_dir/package/rc.d/ccu-jack" "$test_dir/system/etc/config/rc.d/ccu-jack"
touch "$test_dir/package/www/placeholder" "$test_dir/package/etc/placeholder"

addon_dir="$test_dir/system/addons/ccu-jack"
echo existing-config > "$addon_dir/ccu-jack.cfg"
echo old-launcher > "$addon_dir/ccu-jack"
echo unrelated > "$addon_dir/bin/unrelated"
for target in rm-rp0+1 ccu3-rm-rp2+3 rm-rp4 vccu-x86 vccu-x86_64; do
  printf '#!/bin/sh\n# %s\n' "$target" > "$test_dir/package/bin/ccu-jack-$target"
  chmod +x "$test_dir/package/bin/ccu-jack-$target"
  echo old > "$addon_dir/bin/ccu-jack-$target"
done

for mapping in \
  armv6l:rm-rp0+1 armv7l:ccu3-rm-rp2+3 armv8l:ccu3-rm-rp2+3 \
  aarch64:rm-rp4 arm64:rm-rp4 i386:vccu-x86 i486:vccu-x86 \
  i586:vccu-x86 i686:vccu-x86 x86:vccu-x86 \
  x86_64:vccu-x86_64 amd64:vccu-x86_64; do
  TEST_MACHINE=${mapping%%:*}
  expected=${mapping#*:}
  export TEST_MACHINE
  if ! (cd "$test_dir/package" && sh ./update_script test) > "$test_dir/output" 2>&1; then
    cat "$test_dir/output" >&2
    exit 1
  fi
  test "$(tail -n 1 "$addon_dir/ccu-jack")" = "# $expected"
  test -x "$addon_dir/ccu-jack"
done

test -f "$test_dir/stopped"
test "$(cat "$addon_dir/ccu-jack.cfg")" = existing-config
test "$(cat "$addon_dir/bin/unrelated")" = unrelated
for target in rm-rp0+1 ccu3-rm-rp2+3 rm-rp4 vccu-x86 vccu-x86_64; do
  test ! -e "$addon_dir/bin/ccu-jack-$target"
done

rm "$test_dir/stopped"
TEST_MACHINE=unsupported
export TEST_MACHINE
if (cd "$test_dir/package" && sh ./update_script test) > "$test_dir/output" 2>&1; then
  echo "Unsupported architecture was accepted" >&2
  exit 1
fi
grep -q "unsupported machine architecture: unsupported" "$test_dir/output"
test ! -e "$test_dir/stopped"
test "$(tail -n 1 "$addon_dir/ccu-jack")" = "# vccu-x86_64"

TEST_MACHINE=armv6l
export TEST_MACHINE
rm "$test_dir/package/bin/ccu-jack-rm-rp0+1"
if (cd "$test_dir/package" && sh ./update_script test) > "$test_dir/output" 2>&1; then
  echo "Missing binary was accepted" >&2
  exit 1
fi
grep -q "missing binary: bin/ccu-jack-rm-rp0+1" "$test_dir/output"
test ! -e "$test_dir/stopped"
test "$(tail -n 1 "$addon_dir/ccu-jack")" = "# vccu-x86_64"

echo "CCU add-on installation tests passed"
