package virtdev

import (
	"github.com/mdzio/go-hmccu/itf/vdevices"
)

type mqttPowerMeter struct {
	baseChannel
	powerMeterChannel *vdevices.PowerMeterChannel

	energyCounter mqttAnalogInHandler
	power         mqttAnalogInHandler
	current       mqttAnalogInHandler
	voltage       mqttAnalogInHandler
	frequency     mqttAnalogInHandler
}

func (c *mqttPowerMeter) start() {
	c.energyCounter.start()
	c.power.start()
	c.current.start()
	c.voltage.start()
	c.frequency.start()
}

func (c *mqttPowerMeter) stop() {
	c.energyCounter.stop()
	c.power.stop()
	c.current.stop()
	c.voltage.stop()
	c.frequency.stop()
}

func (vd *VirtualDevices) addMQTTPowerMeter(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(mqttPowerMeter)

	// inititalize baseChannel
	ch.powerMeterChannel = vdevices.NewPowerMeterChannel(dev)
	ch.GenericChannel = ch.powerMeterChannel
	ch.store = vd.Store

	// setup handlers
	ch.energyCounter.channel = ch
	ch.energyCounter.targetParam = "ENERGY_COUNTER"
	ch.energyCounter.mqttServer = vd.MQTTServer
	ch.energyCounter.valueHandler = ch.powerMeterChannel.SetEnergyCounter
	ch.energyCounter.statusHandler = func(_ int) {}
	ch.energyCounter.init()

	ch.power.channel = ch
	ch.power.targetParam = "POWER"
	ch.power.mqttServer = vd.MQTTServer
	ch.power.valueHandler = ch.powerMeterChannel.SetPower
	ch.power.statusHandler = func(_ int) {}
	ch.power.init()

	ch.current.channel = ch
	ch.current.targetParam = "CURRENT"
	ch.current.mqttServer = vd.MQTTServer
	ch.current.valueHandler = ch.powerMeterChannel.SetCurrent
	ch.current.statusHandler = func(_ int) {}
	ch.current.init()

	ch.voltage.channel = ch
	ch.voltage.targetParam = "VOLTAGE"
	ch.voltage.mqttServer = vd.MQTTServer
	ch.voltage.valueHandler = ch.powerMeterChannel.SetVoltage
	ch.voltage.statusHandler = func(_ int) {}
	ch.voltage.init()

	ch.frequency.channel = ch
	ch.frequency.targetParam = "FREQUENCY"
	ch.frequency.mqttServer = vd.MQTTServer
	ch.frequency.valueHandler = ch.powerMeterChannel.SetFrequency
	ch.frequency.statusHandler = func(_ int) {}
	ch.frequency.init()

	// clean up
	ch.powerMeterChannel.OnDispose = ch.stop

	// store master param on PutParamset, reregister topics
	ch.MasterParamset().HandlePutParamset(func() {
		ch.stop()
		ch.storeMasterParamset()
		ch.start()
	})

	// load master parameters from config
	ch.loadMasterParamset()

	// register topics
	ch.Lock()
	defer ch.Unlock()
	ch.start()
	return ch
}
