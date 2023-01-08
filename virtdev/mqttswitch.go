package virtdev

import (
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-mqtt/message"
)

type mqttSwitch struct {
	baseChannel

	topic      *vdevices.StringParameter
	retain     *vdevices.BoolParameter
	onPayload  *vdevices.StringParameter
	offPayload *vdevices.StringParameter
}

func (vd *VirtualDevices) addMQTTSwitch(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(mqttSwitch)

	// inititalize baseChannel
	kch := vdevices.NewSwitchChannel(dev)
	ch.GenericChannel = kch
	ch.store = vd.Store

	// TOPIC
	ch.topic = vdevices.NewStringParameter("TOPIC")
	ch.AddMasterParam(ch.topic)

	// RETAIN
	ch.retain = vdevices.NewBoolParameter("RETAIN")
	ch.AddMasterParam(ch.retain)

	// ON_PAYLOAD
	ch.onPayload = vdevices.NewStringParameter("ON_PAYLOAD")
	ch.AddMasterParam(ch.onPayload)

	// OFF_PAYLOAD
	ch.offPayload = vdevices.NewStringParameter("OFF_PAYLOAD")
	ch.AddMasterParam(ch.offPayload)

	// state change
	kch.OnSetState = func(state bool) bool {
		var payload string
		if state {
			payload = ch.onPayload.Value().(string)
		} else {
			payload = ch.offPayload.Value().(string)
		}
		vd.MQTTServer.Publish(
			ch.topic.Value().(string),
			[]byte(payload),
			message.QosExactlyOnce,
			ch.retain.Value().(bool),
		)
		// update state in channel
		return true
	}

	// store master param on PutParamset
	ch.MasterParamset().HandlePutParamset(ch.storeMasterParamset)

	// load master parameters from config
	ch.loadMasterParamset()
	return ch
}
