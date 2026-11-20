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

## Debmatic (ARMv7)

The Go release build (`go run .` from `build`) reuses the CCU3/OpenCCU
ARMv7 binary and packages it with the templates in [debmatic/](./debmatic).
The layout follows the [original Debmatic package](https://homematic-forum.de/forum/viewtopic.php?p=777847#p777847).
GitHub Actions uploads `ccu-jack-debmatic-rp2+3-<version>.tar.gz` as a build
artifact. The binary targets ARMv7 (Raspberry Pi 2/3). On ARM64, the host
must support running 32-bit ARM executables.

### Fresh installation

Replace `<version>` with the package version. Use this command only on a
fresh installation: the archive includes `/etc/config/addons/ccu-jack.cfg`
and extracting it without an exclusion overwrites any existing configuration.

```sh
sudo tar -C / -xzf 'ccu-jack-debmatic-rp2+3-<version>.tar.gz' &&
    sudo systemctl daemon-reload &&
    sudo systemctl enable ccu-jack.service &&
    sudo systemctl restart debmatic.service
```

Open CCU-Jack from the Debmatic control panel or at `http://<CCU-IP>:2121`.

### Update

Back up `/etc/config/addons/ccu-jack.cfg` before updating. Do not uninstall the
previous version. Use the same archive, but exclude the configuration entry
to preserve settings and virtual devices:

```sh
sudo systemctl stop ccu-jack.service &&
    sudo tar --no-overwrite-dir --exclude='etc/config/addons/ccu-jack.cfg' \
        -C / -xzf 'ccu-jack-debmatic-rp2+3-<version>.tar.gz' &&
    sudo systemctl daemon-reload &&
    sudo systemctl start ccu-jack.service
```

### Uninstall

Stop and disable `ccu-jack.service`, then remove the CCU-Jack files listed
below and run `sudo systemctl daemon-reload`. Back up the configuration
before deleting it. Remove only the `ccu-jack` entry from the shared add-on
list, not the entries of other add-ons.

| Path | Purpose |
| --- | --- |
| `/usr/local/addons/ccu-jack/` | Application files and Web UI |
| `/etc/config/addons/ccu-jack.cfg` | Active configuration |
| `/etc/config/addons/www/ccu-jack/` | Debmatic control panel integration |
| `/usr/lib/systemd/system/ccu-jack.service` | systemd unit |
| `/etc/config/hm_addons.cfg` | Shared add-on list; edit only the CCU-Jack entry |
| `/var/log/ccu-jack.log` | Log file; keep or remove as needed |

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
