package main

import (
	"net/http"

	"github.com/mdzio/ccu-jack/rtcfg"

	"github.com/mdzio/go-logging"
)

var (
	logAuth = logging.Get("http-auth")
)

// HTTPAuthHandler wraps another http.Handler and authenticates an HTTP client.
type HTTPAuthHandler struct {
	http.Handler
	Store *rtcfg.Store

	// Realm must only contain valid characters for an HTTP header value and no
	// double quotes.
	Realm string
}

func (h *HTTPAuthHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	name, passwd, ok := req.BasicAuth()

	// read config
	var allowAll bool
	var user *rtcfg.User
	h.Store.View(func(c *rtcfg.Config) error {
		allowAll = true
		// search an active user
		for _, u := range c.Users {
			if u.Active {
				allowAll = false
				break
			}
		}
		if !allowAll {
			user = c.Authenticate(rtcfg.EndpointVEAP, name, passwd)
		}
		return nil
	})

	// if no activve user is configured, allow everything for every user
	if allowAll {
		h.Handler.ServeHTTP(rw, req)
		return
	}

	// no credentials
	if !ok {
		logAuth.Tracef("Not authenticated: %s", req.RemoteAddr)
		h.sendAuth(rw, req)
		return
	}

	// check credentials
	if user == nil {
		logAuth.Warningf("Authentication request failed: address %s, user %s", req.RemoteAddr, name)
		h.sendAuth(rw, req)
		return
	}

	// credentials ok
	h.Handler.ServeHTTP(rw, req)
}

func (h *HTTPAuthHandler) sendAuth(rw http.ResponseWriter, _ *http.Request) {
	rw.Header().Set("WWW-Authenticate", "Basic realm=\""+h.Realm+"\", charset=\"UTF-8\"")
	http.Error(rw, "Unauthorized", http.StatusUnauthorized)
}
