module github.com/mdzio/ccu-jack

go 1.17

require (
	github.com/gorilla/handlers v1.5.1
	github.com/mdzio/go-hmccu v0.4.7
	github.com/mdzio/go-lib v0.1.7
	github.com/mdzio/go-logging v1.0.0
	github.com/mdzio/go-mqtt v0.1.3
	github.com/mdzio/go-veap v0.2.0
	golang.org/x/crypto v0.0.0-20220214200702-86341886e292
)

require (
	github.com/felixge/httpsnoop v1.0.2 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
	golang.org/x/text v0.3.7 // indirect
)

replace github.com/mdzio/go-mqtt => ../go-mqtt
