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
The templates come from the original Debmatic 2.7.1 package and are
independent of the local `build\deploy` staging directory.

The `-install.tar.gz` archive is for fresh installations only. It includes a
configuration template at the live configuration path. The `-update.tar.gz`
archive omits the configuration, systemd unit, and Debmatic menu integration.
GitHub Actions uploads both archives as build artifacts.

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
