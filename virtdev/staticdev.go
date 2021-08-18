package virtdev

import (
	"fmt"

	"github.com/mdzio/ccu-jack/rtcfg"
	"github.com/mdzio/go-hmccu/itf/vdevices"
)

func createStaticDevice(devcfg *rtcfg.Device, publisher vdevices.EventPublisher) (vdevices.GenericDevice, error) {
	// create device
	dev := vdevices.NewDevice(devcfg.Address, devcfg.HMType, publisher)
	// add maintenance channel
	vdevices.NewMaintenanceChannel(dev)

	// add configured channels
	for _, chcfg := range devcfg.Channels {
		switch chcfg.Kind {
		case rtcfg.ChannelKey:
			ch := vdevices.NewKeyChannel(dev)
			log.Debugf("Created key channel: %s", ch.Description().Address)
		case rtcfg.ChannelSwitch:
			ch := vdevices.NewSwitchChannel(dev)
			log.Debugf("Created switch channel: %s", ch.Description().Address)
		case rtcfg.ChannelAnalogInput:
			ch := vdevices.NewAnalogInputChannel(dev)
			log.Debugf("Created analog input channel: %s", ch.Description().Address)
		default:
			return nil, fmt.Errorf("Unsupported kind of channel for a %v device:  %v", devcfg.Logic, chcfg.Kind)
		}
	}
	return dev, nil
}
