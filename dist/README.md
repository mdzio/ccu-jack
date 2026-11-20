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
ARMv7 binary and packages it with the templates in [debmatic/](./debmatic),
based on the original Debmatic 2.7.1 package. GitHub Actions uploads
`ccu-jack-debmatic-armv7-<version>.tar.gz` as a build artifact.

Use the same archive for installation and updates. Before updating, back up
`/etc/config/addons/ccu-jack.cfg` and stop CCU-Jack with
`sudo systemctl stop ccu-jack.service`. Do not uninstall the previous version.

On the Debmatic host, replace `<version>` with the package version and run:

```sh
(
    set -e
    sudo tar --no-overwrite-dir -xzf 'ccu-jack-debmatic-armv7-<version>.tar.gz' -C /
    config=/etc/config/addons/ccu-jack.cfg
    if [ ! -e "$config" ] && [ ! -L "$config" ]; then
        sudo cp -n /usr/local/addons/ccu-jack/ccu-jack-default.cfg "$config"
    fi
    sudo systemctl daemon-reload
)
```

For a fresh installation, run `sudo systemctl enable ccu-jack.service`, then
`sudo systemctl restart debmatic.service`. After an update, run
`sudo systemctl start ccu-jack.service`. Verify with
`systemctl status ccu-jack.service`.

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
