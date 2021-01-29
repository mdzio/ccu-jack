package rtcfg

import (
	"errors"
	"os"
	"testing"

	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-logging"
)

const tmpFile = "tmpFile"

func init() {
	var l logging.LogLevel
	err := l.Set(os.Getenv("LOG_LEVEL"))
	if err == nil {
		logging.SetLevel(l)
	}
}

func TestStore(t *testing.T) {
	defer func() { os.Remove(tmpFile) }()

	s := &Store{FileName: tmpFile}
	err := s.Read()
	if err == nil {
		t.Fatal("Expected error")
	}
	s.Update(func(r *Config) error {
		r.Logging.Level = logging.WarningLevel
		r.CCU.Interfaces = []itf.Type{itf.BidCosRF}
		sub := &User{Identifier: "abc"}
		sub.SetPassword("test")
		sub.AddPermission(&Permission{
			Endpoint:   EndpointMQTT | EndpointVEAP,
			Identifier: "admin",
			Kind:       PermConfig | PermReadPV | PermWritePV,
			PVFilter:   "/a/*",
		})
		r.AddUser(sub)
		return nil
	})
	err = s.Write()
	if err != nil {
		t.Fatal(err)
	}
	s.Close()

	s2 := &Store{FileName: tmpFile}
	err = s2.Read()
	if err != nil {
		t.Fatal(err)
	}
	err = s2.View(func(r *Config) error {
		if r.Logging.Level != logging.WarningLevel {
			return errors.New("Unexpected logging level")
		}
		if len(r.CCU.Interfaces) != 1 || r.CCU.Interfaces[0] != itf.BidCosRF {
			return errors.New("Unexpected interfaces")
		}
		if len(r.Users) != 1 || r.Users["abc"].Identifier != "abc" {
			return errors.New("Unexpected subjects")
		}
		sub := r.Users["abc"]
		if len(sub.Permissions) != 1 || sub.Permissions["admin"].Kind != PermConfig|PermReadPV|PermWritePV {
			return errors.New("Unexpected permissions")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPermissions(t *testing.T) {
	defer func() { os.Remove(tmpFile) }()

	s := &Store{FileName: tmpFile}
	s.Update(func(r *Config) error {
		sub := &User{
			Identifier: "sub",
			Active:     true,
		}
		sub.SetPassword("pwd")
		sub.AddPermission(&Permission{
			Identifier: "per",
			Endpoint:   EndpointVEAP,
			Kind:       PermWritePV,
			PVFilter:   "/A[01]/B",
		})
		r.AddUser(sub)
		return nil
	})

	err := s.View(func(r *Config) error {
		su := r.Authenticate(EndpointVEAP, "unkwon-sub", "pwd")
		if su != nil {
			return errors.New("Unexpected authentication (subject)")
		}
		su = r.Authenticate(EndpointMQTT, "sub", "pwd")
		if su != nil {
			return errors.New("Unexpected authentication (endpoint)")
		}
		su = r.Authenticate(EndpointVEAP, "sub", "wrong-pwd")
		if su != nil {
			return errors.New("Unexpected authentication (password)")
		}
		su = r.Authenticate(EndpointVEAP, "sub", "pwd")
		if su == nil {
			return errors.New("Authentication failed")
		}
		ok := su.Authorized(EndpointMQTT, PermWritePV, "/A0/B")
		if ok {
			return errors.New("Unexpected authorization (endpoint)")
		}
		ok = su.Authorized(EndpointVEAP, PermReadPV, "/A0/B")
		if ok {
			return errors.New("Unexpected authorization (kind)")
		}
		ok = su.Authorized(EndpointVEAP, PermWritePV, "/A2/B")
		if ok {
			return errors.New("Unexpected authorization (PV filter)")
		}
		ok = su.Authorized(EndpointVEAP, PermWritePV, "/A1/B")
		if !ok {
			return errors.New("Authorization failed")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
