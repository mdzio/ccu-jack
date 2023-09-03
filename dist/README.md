# Information about the target platforms

[Environment variables of the Go compiler for cross compiling to ARM.](https://github.com/golang/go/wiki/GoArm)

## CCU2

* Tcl version 8.2

```
GOOS=linux
GOARCH=arm
GOARM=5
```

## RaspberryMatic (Raspberry Pi A, A+, B, B+, Zero)
```
GOOS=linux
GOARCH=arm
GOARM=6
```

## CCU3 / RaspberryMatic (Raspberry Pi 2/3)
```
GOOS=linux
GOARCH=arm
GOARM=7
```

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
