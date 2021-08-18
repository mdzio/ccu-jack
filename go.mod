module github.com/mdzio/ccu-jack

go 1.15

require (
	github.com/gorilla/handlers v1.5.1
	github.com/mdzio/go-hmccu v0.3.2
	github.com/mdzio/go-lib v0.1.6
	github.com/mdzio/go-logging v1.0.0
	github.com/mdzio/go-mqtt v0.1.2
	github.com/mdzio/go-veap v0.1.2
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97
)

replace github.com/mdzio/go-hmccu => ../go-hmccu
