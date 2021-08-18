package vmodel

import (
	"fmt"
	"sync"
	"time"

	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-logging"
	"github.com/mdzio/go-veap"

	"github.com/mdzio/ccu-jack/rtcfg"
	"github.com/mdzio/go-lib/any"
	"github.com/mdzio/go-veap/model"
)

type Config struct {
	model.Variable
	changeListener func(*rtcfg.Config)
	mtx            sync.Mutex
}

// NewConfig creates a new configuration variable.
func NewConfig(col model.ChangeableCollection, store *rtcfg.Store) *Config {
	c := new(Config)
	c.Identifier = "config"
	c.Title = "Configuration"
	c.Description = "Configuration of the CCU-Jack"
	c.Collection = col
	c.ReadPVFunc = func() (veap.PV, veap.Error) {
		store.RLock()
		defer store.RUnlock()
		// clone current config, because config can be modified concurrently
		// after releasing lock
		var cfg rtcfg.Config
		err := store.Config.CopyTo(&cfg)
		if err != nil {
			return veap.PV{}, veap.NewError(veap.StatusInternalServerError, err)
		}
		return veap.PV{Time: time.Now(), Value: cfg, State: veap.StateGood}, nil
	}
	c.WritePVFunc = func(pv veap.PV) veap.Error {
		// update configuration
		store.Lock()
		defer store.Unlock()
		// clone current config
		var cfg rtcfg.Config
		err := store.Config.CopyTo(&cfg)
		if err != nil {
			return veap.NewError(veap.StatusInternalServerError, err)
		}
		// update clone
		err = updateConfig(&cfg, pv.Value)
		if err != nil {
			return veap.NewErrorf(veap.StatusBadRequest, "Configuration update failed: %v", err)
		}
		// update succeeded, set config active
		store.Config = cfg
		// notify listener
		c.mtx.Lock()
		defer c.mtx.Unlock()
		if c.changeListener != nil {
			c.changeListener(&store.Config)
		}
		return nil
	}
	col.PutItem(c)
	return c
}

func (c *Config) SetChangeListener(l func(*rtcfg.Config)) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.changeListener = l
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

	// VirtualDevices property present?
	if c.Has("VirtualDevices") {

		// base config
		rvd := c.Key("VirtualDevices").Map()
		cfg.VirtualDevices.Enable = rvd.Key("Enable").Bool()
		cfg.VirtualDevices.NextSerialNo = int(rvd.Key("NextSerialNo").Float64())

		// decode devices
		ds := make(map[string]*rtcfg.Device)
		for ra, rdq := range rvd.Key("Devices").Map().Wrap() {

			// decode device
			rd := rdq.Map()
			var rl rtcfg.DeviceLogic
			err := rl.UnmarshalText([]byte(rd.Key("Logic").String()))
			if err != nil {
				q.SetErr(err)
			}
			d := rtcfg.Device{
				Address:  rd.Key("Address").String(),
				HMType:   rd.Key("HMType").String(),
				Logic:    rl,
				Specific: int(rd.Key("Specific").Float64()),
				Channels: make([]rtcfg.Channel, 0, 8), // prevents JSON null for empty arrays
			}
			// add device
			ds[ra] = &d

			// decode channels
			for _, rcq := range rd.Key("Channels").Slice() {

				// decode channel
				rc := rcq.Map()
				var rk rtcfg.ChannelKind
				err = rk.UnmarshalText(([]byte)(rc.Key("Kind").String()))
				if err != nil {
					q.SetErr(err)
				}
				c := rtcfg.Channel{
					Kind:           rk,
					Specific:       int(rc.Key("Specific").Float64()),
					MasterParamset: make(map[string]interface{}),
				}
				// decode master paramset
				for n, v := range rc.Key("MasterParamset").Map().Wrap() {
					c.MasterParamset[n] = v.Unwrap()
				}
				// add channel
				d.Channels = append(d.Channels, c)
			}
		}
		// update devices in config
		cfg.VirtualDevices.Devices = ds

		// any error occured?
		if q.Err() != nil {
			return q.Err()
		}
	}
	return nil
}
