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

### Debian packages (all architectures)

[create_ccu_jack_deb.sh](./debmatic/create_ccu_jack_deb.sh) builds one
architecture-independent `.deb` for all architectures supported by Debmatic
(`armhf`, `arm64`, `i386`, `amd64`) on a single Debian/Ubuntu host. It follows
the pattern of Alexander Reinert's
[create_detect_radio_module.sh](https://github.com/alexreinert/piVCCU/blob/master/create_detect_radio_module.sh):
cross-compile per architecture, fill the [DEBIAN/](./debmatic/DEBIAN)
templates (`{PKG_VERSION}`, `{PKG_ARCH}`), then package with
`dpkg-deb --build`. No cross toolchains are needed because Go
cross-compiles natively (`GOOS`/`GOARCH`/`GOARM`, `CGO_ENABLED=0`).
The package contains all four binaries; its launcher selects the native one
from the installed Debian architecture. CI builds the package for its build
artifacts and attaches it to releases created from matching version tags.

The package contains native binaries for Debian `armhf`, `arm64`, `i386`, and
`amd64`. Its `armhf` binary uses `GOARM=7`; ARMv6 devices such as Raspberry Pi
1/Zero are not supported by Debmatic. The `arm64` binary is native, so no
32-bit multiarch support is required.

```sh
cd dist/debmatic && ./create_ccu_jack_deb.sh   # version from build/main.go
sudo apt install ./ccu-jack_<version>-1_all.deb
```

For an update, download the new universal package and install it with the same
command.

The package declares `/etc/config/addons/ccu-jack.cfg` as conffile, so
dpkg preserves the configuration on updates. `postinst`/`prerm`/`postrm`
handle service registration, the `hm_addons.cfg` entry, and cleanup. Use
`apt purge ccu-jack` to remove a package installation.

To migrate an existing tar.gz installation, install the `.deb` directly over
the existing files; do not remove the tar.gz installation first. Back up
`/etc/config/addons/ccu-jack.cfg`. If dpkg asks how to handle that file, choose
`N`/`O` or press Enter (the default) to keep the current configuration; `Y`/`I`
replaces it with the package version. If Debmatic is active, installation
restarts CCU-Jack and causes a short interruption. After migration, use only
`.deb` packages for updates.

The universal `.deb` is the Debmatic installation and update package for all
supported Debian architectures; the legacy ARMv7 tar.gz is no longer
generated. Without an APT repository, APT will not discover new versions
automatically.

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
