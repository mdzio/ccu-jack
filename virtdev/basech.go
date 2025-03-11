package virtdev

import (
	"fmt"

	"github.com/mdzio/ccu-jack/rtcfg"
	"github.com/mdzio/go-hmccu/itf/vdevices"
)

type baseChannel struct {
	vdevices.GenericChannel
	store *rtcfg.Store
}

// channelConfig returns the configuration of this channel. channelConfig must
// be called with the config store locked.
func (c *baseChannel) channelConfig() (*rtcfg.Channel, error) {
	d, ok := c.store.Config.VirtualDevices.Devices[c.Description().Parent]
	if !ok {
		return nil, fmt.Errorf("Virtual device %s not found in config", c.Description().Parent)
	}
	// maintenance channel (index 0) is skipped
	i := c.Description().Index - 1
	if i < 0 || i >= len(d.Channels) {
		return nil, fmt.Errorf("Virtual device channel %s:%d not found in config", c.Description().Parent, c.Description().Index)
	}
	return &d.Channels[i], nil
}

// loadMasterParamset sets the master parameters based on the config. The config
// store must be locked!
func (c *baseChannel) loadMasterParamset() {
	// config store is already locked
	chcfg, err := c.channelConfig()
	if err != nil {
		log.Error(err)
		return
	}
	for id, v := range chcfg.MasterParamset {
		p, err := c.MasterParamset().Parameter(id)
		if err != nil {
			log.Errorf("Master parameter %s:%d.%s in config not found in device", c.Description().Parent,
				c.Description().Index, id)
			continue
		}
		log.Debugf("Setting master parameter %s:%d.%s from config: %v", c.Description().Parent,
			c.Description().Index, id, v)
		err = p.InternalSetValue(v)
		if err != nil {
			log.Errorf("Setting master parameter %s:%d.%s to value %v failed: %v", c.Description().Parent,
				c.Description().Index, id, v, err)
			continue
		}
	}
}

// storeMasterParamset updates the value of the master parameters in the config.
// The config store will be locked!
func (c *baseChannel) storeMasterParamset() {
	// lock config store
	c.store.Lock()
	defer c.store.Unlock()
	chcfg, err := c.channelConfig()
	if err != nil {
		log.Error(err)
		return
	}
	for _, p := range c.MasterParamset().Parameters() {
		log.Debugf("Storing master parameter %s:%d.%s in config: %v", c.Description().Parent, c.Description().Index,
			p.Description().ID, p.Value())
		chcfg.MasterParamset[p.Description().ID] = p.Value()
	}
}
