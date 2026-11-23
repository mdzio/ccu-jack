# Information about the target platforms

[Environment variables of the Go compiler for cross compiling to ARM.](https://github.com/golang/go/wiki/GoArm)

## Tcl

* CCU2: V8.2
* CCU3: V8.2
* OpenCCU v3.85.7.20260131: V8.6

## CCU2
```
GOOS=linux
GOARCH=arm
GOARM=5
```

## OpenCCU (Raspberry Pi A, A+, B, B+, Zero)
```
GOOS=linux
GOARCH=arm
GOARM=6
```

## CCU3 / OpenCCU (Raspberry Pi 2/3)
```
GOOS=linux
GOARCH=arm
GOARM=7
```

## CCU Add-on (all supported architectures)

The Go release build (`go run .` from `build`) creates one universal
`ccu-jack-ccu-addon-<version>.tar.gz` archive. It contains binaries for
OpenCCU Raspberry Pi 0/1 (ARMv6), Pi 2/3 (ARMv7), Pi 4/5 (ARM64), and VCCU
32-bit and 64-bit x86. The installation script selects and installs only the
binary matching `uname -m`, so users do not need to choose an
architecture-specific archive.

### Debian packages

Debmatic packages are built for `armhf`, `arm64`, `i386`, and `amd64`. Each
package contains only the executable for its architecture. CI builds the
packages and attaches them and the installer to matching tagged releases.

[create_ccu_jack_deb.sh](./debmatic/create_ccu_jack_deb.sh) builds all four
packages on one Debian/Ubuntu host. It follows
the pattern of Alexander Reinert's
[create_detect_radio_module.sh](https://github.com/alexreinert/piVCCU/blob/master/create_detect_radio_module.sh):
cross-compile per architecture, fill the [DEBIAN/](./debmatic/DEBIAN)
templates (`{PKG_VERSION}`, `{PKG_ARCH}`), then package with
`dpkg-deb --build`. No cross toolchains are needed because Go
cross-compiles natively (`GOOS`/`GOARCH`/`GOARM`, `CGO_ENABLED=0`).

The `armhf` binary uses `GOARM=7`; ARMv6 devices such as Raspberry Pi
1/Zero are not supported by Debmatic. The `arm64` binary is native, so no
32-bit multiarch support is required.

To build the packages locally from the project root:

```sh
cd dist/debmatic && ./create_ccu_jack_deb.sh   # version from build/main.go
```

For installation or updates, download `ccu-jack-install-debmatic.sh` from the
[releases](https://github.com/mdzio/ccu-jack/releases) page and run:

```sh
sh ccu-jack-install-debmatic.sh
```

The script requires `curl`, `dpkg`, and `sudo`, selects the Debian architecture
and downloads the matching package from the latest stable GitHub release. Pass
an explicit version, e.g. `sh ccu-jack-install-debmatic.sh 2.14.0`, to select
another release. The script reinstalls the package if it is already installed.
Alternatively, download the matching package and run
`sudo apt install ./ccu-jack_VERSION-1_ARCH.deb`, replacing `VERSION` and `ARCH`
with the downloaded file's version and architecture.

The package declares `/etc/config/addons/ccu-jack.cfg` as conffile, so
dpkg preserves the configuration on updates. `postinst`/`prerm`/`postrm`
handle service registration, the `hm_addons.cfg` entry, and cleanup. Use
`apt purge ccu-jack` to remove a package installation.

To migrate an existing tar.gz installation, install the matching `.deb` directly over
the existing files; do not remove the tar.gz installation first. Back up
`/etc/config/addons/ccu-jack.cfg`. If dpkg asks how to handle that file, choose
`N`/`O` or press Enter (the default) to keep the current configuration; `Y`/`I`
replaces it with the package version. If Debmatic is active, installation
restarts CCU-Jack and causes a short interruption. After migration, use only
`.deb` packages for updates.

The legacy Debmatic ARMv7 tar.gz is no longer generated. Without an APT
repository, APT will not discover new versions automatically; rerun the
installer for updates.

## Windows
```
GOOS=windows
GOARCH=amd64
```

## Linux
```
GOOS=linux
GOARCH=amd64
```

## MacOS, iOS (darwin)
```
GOOS=darwin
GOARCH=amd64
```
