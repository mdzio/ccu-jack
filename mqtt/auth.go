package mqtt

import (
	"github.com/mdzio/go-mqtt/auth"
)

// SingleAuthHandler forces the specified authentication from the MQTT client.
type SingleAuthHandler struct {
	User     string
	Password string
}

// Authenticate implements auth.Authenticator.
func (a *SingleAuthHandler) Authenticate(id string, cred interface{}) error {
	passwd := cred.(string)
	if id != a.User || passwd != a.Password {
		return auth.ErrAuthFailure
	}
	return nil
}
