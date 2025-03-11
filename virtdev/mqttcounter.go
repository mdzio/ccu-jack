package virtdev

import (
	"github.com/mdzio/go-hmccu/itf/vdevices"
)

type counterChannelType int

const (
	COUNTER_CHANNEL_ENERGY counterChannelType = iota
	COUNTER_CHANNEL_GAS
)

// common API between energy and gas counter
type counterChannelSetter interface {
	SetEnergyCounter(value float64)
	SetPower(value float64)
}

type mqttCounter struct {
	baseChannel
	energyCounter mqttAnalogInHandler
	power         mqttAnalogInHandler
}

func (c *mqttCounter) start() {
	c.energyCounter.start()
	c.power.start()
}

func (c *mqttCounter) stop() {
	c.energyCounter.stop()
	c.power.stop()
}

func (vd *VirtualDevices) addMQTTCounter(dev *vdevices.Device, chType counterChannelType) vdevices.GenericChannel {
	ch := new(mqttCounter)

	// inititalize baseChannel
	var setter counterChannelSetter
	switch chType {
	case COUNTER_CHANNEL_ENERGY:
		specificCh := vdevices.NewEnergyCounterChannel(dev)
		setter = specificCh
		ch.GenericChannel = specificCh
		specificCh.Channel.OnDispose = ch.stop
	case COUNTER_CHANNEL_GAS:
		specificCh := vdevices.NewGasCounterChannel(dev)
		setter = specificCh
		ch.GenericChannel = specificCh
		specificCh.Channel.OnDispose = ch.stop
	default:
		panic("invalid counterChannelType")
	}
	ch.store = vd.Store

	// setup handlers
	ch.energyCounter.channel = ch
	ch.energyCounter.targetParam = "ENERGY_COUNTER"
	ch.energyCounter.mqttServer = vd.MQTTServer
	ch.energyCounter.valueHandler = setter.SetEnergyCounter
	ch.energyCounter.statusHandler = func(_ int) {}
	ch.energyCounter.init()

	ch.power.channel = ch
	ch.power.targetParam = "POWER"
	ch.power.mqttServer = vd.MQTTServer
	ch.power.valueHandler = setter.SetPower
	ch.power.statusHandler = func(_ int) {}
	ch.power.init()

	// store master param on PutParamset, reregister topics
	ch.MasterParamset().HandlePutParamset(func() {
		ch.stop()
		ch.storeMasterParamset()
		ch.start()
	})

	// load master parameters from config
	ch.loadMasterParamset()

	// register topics
	ch.start()
	return ch
}

func (vd *VirtualDevices) addMQTTEnergyCounter(dev *vdevices.Device) vdevices.GenericChannel {
	return vd.addMQTTCounter(dev, COUNTER_CHANNEL_ENERGY)
}

func (vd *VirtualDevices) addMQTTGasCounter(dev *vdevices.Device) vdevices.GenericChannel {
	return vd.addMQTTCounter(dev, COUNTER_CHANNEL_GAS)
}
