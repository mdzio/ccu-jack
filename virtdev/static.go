package virtdev

import (
	"github.com/mdzio/go-hmccu/itf/vdevices"
)

type staticDimmer struct {
	baseChannel
	oldLevel float64
}

func (vd *VirtualDevices) addStaticDimmer(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(staticDimmer)
	ch.virtualDevices = vd
	ch.device = dev
	ch.oldLevel = 1.0

	// inititalize baseChannel
	dch := vdevices.NewDimmerChannel(dev)
	ch.GenericChannel = dch

	// LEVEL
	dch.OnSetLevel = func(value float64) bool {
		if value != 0.0 {
			// remember previous dimmer level
			ch.oldLevel = value
		}
		return true
	}

	// OLD_LEVEL
	dch.OnSetOldLevel = func() bool {
		// restore previous dimmer level
		dch.SetLevel(ch.oldLevel)
		return true
	}

	return ch
}

func (vd *VirtualDevices) addStaticUnreach(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(baseChannel)
	ch.virtualDevices = vd
	ch.device = dev
	swch := vdevices.NewSwitchChannel(dev)
	ch.GenericChannel = swch

	swch.OnSetState = func(value bool) (ok bool) {
		// set parameter UNREACH of channel 0 (maintenance channel)
		gch, err := ch.device.Channel("0")
		if err != nil {
			log.Errorf("Maintenance channel (0) not found: %v", err)
			return false
		}
		mch, ok := gch.(*vdevices.MaintenanceChannel)
		if !ok {
			log.Errorf("Channel (0) is not a maintenance channel")
			return false
		}
		mch.SetUnreach(value)
		return true
	}

	return ch
}
