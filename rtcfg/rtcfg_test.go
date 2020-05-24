package rtcfg

import (
	"errors"
	"os"
	"testing"

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
	s.Update(func(r *Root) error {
		r.Security.AddSubject(&Subject{Identifier: "abc"})
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
	err = s2.View(func(r *Root) error {
		if len(r.Security.Subjects) != 1 || r.Security.Subjects["abc"].Identifier != "abc" {
			return errors.New("Unexpected content")
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
	s.Update(func(r *Root) error {
		sub := &Subject{Identifier: "sub"}
		sub.SetPassword("pwd")
		sub.AddPermission(&Permission{
			Identifier: "per",
			Endpoint:   EndpointVEAP,
			Kind:       PermWritePV,
			PVFilter:   "/A[01]/B",
		})
		r.Security.AddSubject(sub)
		return nil
	})

	err := s.View(func(r *Root) error {
		su := r.Security.Authenticate(EndpointVEAP, "unkwon-sub", "pwd")
		if su != nil {
			return errors.New("Unexpected authentication (subject)")
		}
		su = r.Security.Authenticate(EndpointMQTT, "sub", "pwd")
		if su != nil {
			return errors.New("Unexpected authentication (endpoint)")
		}
		su = r.Security.Authenticate(EndpointVEAP, "sub", "wrong-pwd")
		if su != nil {
			return errors.New("Unexpected authentication (password)")
		}
		su = r.Security.Authenticate(EndpointVEAP, "sub", "pwd")
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
