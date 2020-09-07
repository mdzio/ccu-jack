package mqtt

import (
	"github.com/mdzio/ccu-jack/rtcfg"
	"github.com/mdzio/go-mqtt/auth"
)

// AuthHandler handles MQTT client authentication.
type AuthHandler struct {
	Store *rtcfg.Store
}

// Authenticate implements auth.Authenticator.
func (a *AuthHandler) Authenticate(id string, cred interface{}) error {
	passwd := cred.(string)
	return a.Store.View(func(c *rtcfg.Config) error {
		// if no user is configured, allow everything for every user
		if len(c.Users) == 0 {
			return nil
		}
		// authenticate user for MQTT
		user := c.Authenticate(rtcfg.EndpointMQTT, id, passwd)
		if user == nil {
			return auth.ErrAuthFailure
		}
		return nil
	})
}
