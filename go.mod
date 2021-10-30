module github.com/mdzio/ccu-jack

go 1.15

require (
	github.com/gorilla/handlers v1.5.1
	github.com/mdzio/go-hmccu v0.4.3
	github.com/mdzio/go-lib v0.1.7
	github.com/mdzio/go-logging v1.0.0
	github.com/mdzio/go-mqtt v0.1.2
	github.com/mdzio/go-veap v0.1.2
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
)

replace github.com/mdzio/go-veap => ../go-veap
