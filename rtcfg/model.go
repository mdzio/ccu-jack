package rtcfg

import (
	"path"

	"golang.org/x/crypto/bcrypt"
)

// Root is the entry object of the runtime config.
type Root struct {
	Security Security
}

// Security holds the security configuration.
type Security struct {
	Subjects map[string]*Subject /* Identifier is key. */
}

// Authenticate authenticates a subject.
func (s *Security) Authenticate(endpoint Endpoint, identifier, password string) *Subject {
	// find user
	sub, ok := s.Subjects[identifier]
	if !ok {
		return nil
	}
	// check all permissions
	for _, per := range sub.Permissions {
		// check endpoint
		if endpoint&per.Endpoint == endpoint {
			// check password
			err := bcrypt.CompareHashAndPassword(sub.Password, []byte(password))
			if err != nil {
				return nil
			}
			return sub
		}
	}
	return nil
}

// AddSubject adds a subject to the security config.
func (s *Security) AddSubject(subject *Subject) {
	if s.Subjects == nil {
		s.Subjects = make(map[string]*Subject)
	}
	s.Subjects[subject.Identifier] = subject
}

// Subject represents a user or a device.
type Subject struct {
	Identifier  string
	Description string
	Password    []byte                 // bcrypt hash
	Permissions map[string]*Permission /* Identifier is key. */
}

// Authorized checks whether an authorization exists. The request must contain
// only a single endpoint and kind. pvPath is not yet checked.
func (s *Subject) Authorized(endpoint Endpoint, kind PermKind, pvPath string) bool {
	// check all permissions
	for _, per := range s.Permissions {
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
func (s *Subject) SetPassword(password string) error {
	// hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return err
	}
	s.Password = hash
	return nil
}

// AddPermission adds a permission to a subject.
func (s *Subject) AddPermission(per *Permission) {
	if s.Permissions == nil {
		s.Permissions = make(map[string]*Permission)
	}
	s.Permissions[per.Identifier] = per
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
	EndpointVEAP Endpoint = iota << 1
	EndpointMQTT
)

// PermKind specifies the kind if a permission.
type PermKind int

// Possible kinds of a permission.
const (
	PermConfig PermKind = iota << 1
	PermWritePV
	PermReadPV
)
