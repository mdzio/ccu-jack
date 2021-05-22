package vmodel

import (
	"fmt"
	"time"

	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-logging"

	"github.com/mdzio/ccu-jack/rtcfg"
	"github.com/mdzio/go-lib/any"
	"github.com/mdzio/go-veap"
	"github.com/mdzio/go-veap/model"
)

// NewConfig creates a new configuration variable.
func NewConfig(col model.ChangeableCollection, store *rtcfg.Store) *model.Variable {
	v := model.NewVariable(&model.VariableCfg{
		Identifier:  "config",
		Title:       "Configuration",
		Description: "Configuration of the CCU-Jack",
		Collection:  col,
		ReadPVFunc: func() (veap.PV, veap.Error) {
			store.RLock()
			defer store.RUnlock()
			// export configuration
			v := exportConfig(&store.Config)
			return veap.PV{Time: time.Now(), Value: v, State: veap.StateGood}, nil
		},
		WritePVFunc: func(pv veap.PV) veap.Error {
			// update configuration
			store.Lock()
			defer store.Unlock()
			err := updateConfig(&store.Config, pv.Value)
			if err != nil {
				return veap.NewErrorf(veap.StatusBadRequest, "Configuration update failed: %v", err)
			}
			return nil
		},
	})
	return v
}

func exportConfig(s *rtcfg.Config) *rtcfg.Config {
	d := *s
	// clone users into new map
	d.Users = make(map[string]*rtcfg.User)
	for id, u := range s.Users {
		d.Users[id] = exportUser(u)
	}
	return &d
}

func exportUser(s *rtcfg.User) *rtcfg.User {
	d := *s
	// clone permissions into new map
	d.Permissions = make(map[string]*rtcfg.Permission)
	for id, sp := range s.Permissions {
		dp := *sp
		d.Permissions[id] = &dp
	}
	return &d
}

func updateConfig(cfg *rtcfg.Config, v interface{}) error {
	// v must be an JSON object
	q := any.Q(v)
	c := q.Map()
	if q.Err() != nil {
		return q.Err()
	}

	// CCU property present?
	if c.Has("CCU") {
		// CCU interface list
		ts := make(itf.Types, 0) // no nil slice
		is := c.Key("CCU").Map().Key("Interfaces").Slice()
		for _, i := range is {
			var t itf.Type
			in := i.String()
			// not a string?
			if q.Err() != nil {
				return q.Err()
			}
			err := t.Set(in)
			// invalid interface name
			if err != nil {
				return err
			}
			ts = append(ts, t)
		}
		cfg.CCU.Interfaces = ts
	}

	// Users property present?
	if c.Has("Users") {
		users := make(map[string]*rtcfg.User)
		for id, u := range c.Key("Users").Map().Wrap() {
			// fill user data
			uo := u.Map()
			user := &rtcfg.User{
				Identifier:  uo.Key("Identifier").String(),
				Active:      uo.Key("Active").Bool(),
				Description: uo.TryKey("Description").String(),
			}
			pwd := uo.TryKey("Password").String()
			epwd := uo.TryKey("EncryptedPassword").String()
			// valid user data?
			if q.Err() != nil {
				return q.Err()
			}
			if id != user.Identifier {
				return fmt.Errorf("User identifier mismatches: %s", user.Identifier)
			}
			if pwd == "" && epwd == "" {
				return fmt.Errorf("No password provided for user: %s", user.Identifier)
			}
			// handle unencrypted password
			if pwd != "" {
				user.SetPassword(pwd)
			} else {
				user.EncryptedPassword = epwd
			}
			// set for now all permissions
			user.AddPermission(&rtcfg.Permission{
				Identifier:  "all",
				Description: "All permissions",
				Endpoint:    rtcfg.EndpointVEAP | rtcfg.EndpointMQTT,
				Kind:        rtcfg.PermConfig | rtcfg.PermReadPV | rtcfg.PermWritePV,
			})
			users[id] = user
		}
		if q.Err() != nil {
			return q.Err()
		}
		cfg.Users = users
	}

	// Logging property present?
	if c.Has("Logging") {
		lvltxt := c.Key("Logging").Map().TryKey("Level").String()
		if q.Err() != nil {
			return q.Err()
		}
		// logging level present?
		if lvltxt != "" {
			var lvl logging.LogLevel
			err := lvl.Set(lvltxt)
			if err != nil {
				return fmt.Errorf("Invalid logging level: %s", lvltxt)
			}
			cfg.Logging.Level = lvl
			// activate log level
			logging.SetLevel(lvl)
		}
	}
	return nil
}
