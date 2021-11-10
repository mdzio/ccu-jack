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
	ch.oldLevel = 1.0

	// inititalize baseChannel
	dch := vdevices.NewDimmerChannel(dev)
	ch.GenericChannel = dch
	ch.store = vd.Store

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
