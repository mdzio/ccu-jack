module github.com/mdzio/ccu-jack

go 1.23.1

require (
	github.com/gorilla/handlers v1.5.2
	github.com/mdzio/go-hmccu v1.5.3
	github.com/mdzio/go-lib v0.2.2
	github.com/mdzio/go-logging v1.0.0
	github.com/mdzio/go-mqtt v1.0.4
	github.com/mdzio/go-veap v0.5.1
	golang.org/x/crypto v0.37.0
)

require (
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/text v0.24.0 // indirect
)

replace github.com/mdzio/go-mqtt => ../go-mqtt
