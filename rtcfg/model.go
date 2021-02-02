package rtcfg

import (
	"path"

	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-logging"
	"golang.org/x/crypto/bcrypt"
)

// Config is the entry object of the runtime config.
type Config struct {
	CCU         CCU
	Host        Host
	Logging     Logging
	HTTP        HTTP
	MQTT        MQTT
	Certificate Certificate
	Users       map[string]*User /* Identifier is key. */
}

// CCU configuration
type CCU struct {
	Address    string
	Interfaces itf.Types
	InitID     string
}

// Host configuration
type Host struct {
	Name    string
	Address string
}

// Logging configuration
type Logging struct {
	Level    logging.LogLevel
	FilePath string
}

// HTTP configuration
type HTTP struct {
	Port        int
	PortTLS     int
	CORSOrigins []string
}

// MQTT configuration
type MQTT struct {
	Port    int
	PortTLS int
}

type Certificate struct {
	CertificateFile string
	KeyFile         string
}

// Authenticate authenticates a user.
func (c *Config) Authenticate(endpoint Endpoint, identifier, password string) *User {
	// find user
	u, ok := c.Users[identifier]
	if !ok {
		return nil
	}
	// active?
	if !u.Active {
		return nil
	}
	// check all permissions
	for _, per := range u.Permissions {
		// check endpoint
		if endpoint&per.Endpoint == endpoint {
			// check password
			err := bcrypt.CompareHashAndPassword([]byte(u.EncryptedPassword), []byte(password))
			if err != nil {
				return nil
			}
			return u
		}
	}
	return nil
}

// AddUser adds a user to the security config.
func (c *Config) AddUser(u *User) {
	if c.Users == nil {
		c.Users = make(map[string]*User)
	}
	c.Users[u.Identifier] = u
}

// User represents a user or a device.
type User struct {
	Identifier        string
	Active            bool
	Description       string
	Password          string                 // unencrypted password (only temporary)
	EncryptedPassword string                 // bcrypt hash
	Permissions       map[string]*Permission /* Identifier is key. */
}

// Authorized checks whether an authorization exists. The request must contain
// only a single endpoint and kind. pvPath is not yet checked.
func (u *User) Authorized(endpoint Endpoint, kind PermKind, pvPath string) bool {
	// check all permissions
	for _, per := range u.Permissions {
		// check endpoint
		if endpoint&per.Endpoint == endpoint {
			// check kind
			if kind&per.Kind == kind {
				if per.PVFilter == "" {
					return true
				}
				// check pv path
				match, err := path.Match(per.PVFilter, pvPath)
				if err != nil {
					log.Warning("Invalid PV filter in security configuration: %s", per.PVFilter)
					return false
				}
				return match
			}
		}
	}
	// no permission matches
	return false
}

// SetPassword generates a new hash for the password.
func (u *User) SetPassword(password string) error {
	// hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return err
	}
	u.EncryptedPassword = string(hash)
	// clear unencrypted password, if any
	u.Password = ""
	return nil
}

// AddPermission adds a permission to a user.
func (u *User) AddPermission(per *Permission) {
	if u.Permissions == nil {
		u.Permissions = make(map[string]*Permission)
	}
	u.Permissions[per.Identifier] = per
}

// Permission represents a allowance to access something.
type Permission struct {
	Identifier  string
	Description string
	Endpoint    Endpoint
	Kind        PermKind

	// pattern syntax q.v. path.Match()
	PVFilter string
}

// Endpoint is a communication interface/protocol.
type Endpoint int

// Possible endpoints.
const (
	EndpointVEAP Endpoint = 1 << iota
	EndpointMQTT
)

// PermKind specifies the kind if a permission.
type PermKind int

// Possible kinds of a permission.
const (
	PermConfig PermKind = 1 << iota
	PermWritePV
	PermReadPV
)
