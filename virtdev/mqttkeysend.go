package virtdev

import (
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-mqtt/message"
)

type mqttKeySender struct {
	baseChannel

	shortTopic   *vdevices.StringParameter
	shortPayload *vdevices.StringParameter
	shortRetain  *vdevices.BoolParameter

	longTopic   *vdevices.StringParameter
	longPayload *vdevices.StringParameter
	longRetain  *vdevices.BoolParameter
}

// addMQTTKeySender must be called with config store locked.
func (vd *VirtualDevices) addMQTTKeySender(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(mqttKeySender)

	// inititalize baseChannel
	kch := vdevices.NewKeyChannel(dev)
	ch.GenericChannel = kch
	ch.store = vd.Store

	// SHORT_TOPIC
	ch.shortTopic = vdevices.NewStringParameter("SHORT_TOPIC")
	ch.AddMasterParam(ch.shortTopic)

	// SHORT_PAYLOAD
	ch.shortPayload = vdevices.NewStringParameter("SHORT_PAYLOAD")
	ch.AddMasterParam(ch.shortPayload)

	// SHORT_RETAIN
	ch.shortRetain = vdevices.NewBoolParameter("SHORT_RETAIN")
	ch.AddMasterParam(ch.shortRetain)

	// LONG_TOPIC
	ch.longTopic = vdevices.NewStringParameter("LONG_TOPIC")
	ch.AddMasterParam(ch.longTopic)

	// LONG_PAYLOAD
	ch.longPayload = vdevices.NewStringParameter("LONG_PAYLOAD")
	ch.AddMasterParam(ch.longPayload)

	// LONG_RETAIN
	ch.longRetain = vdevices.NewBoolParameter("LONG_RETAIN")
	ch.AddMasterParam(ch.longRetain)

	// PRESS_SHORT
	kch.OnPressShort = func() bool {
		vd.MQTTServer.Publish(
			ch.shortTopic.Value().(string),
			[]byte(ch.shortPayload.Value().(string)),
			message.QosExactlyOnce,
			ch.shortRetain.Value().(bool),
		)
		return true
	}

	// PRESS_LONG
	kch.OnPressLong = func() bool {
		vd.MQTTServer.Publish(
			ch.longTopic.Value().(string),
			[]byte(ch.longPayload.Value().(string)),
			message.QosExactlyOnce,
			ch.longRetain.Value().(bool),
		)
		return true
	}

	// store master param on PutParamset
	ch.MasterParamset().HandlePutParamset(ch.storeMasterParamset)

	// load master parameters from config
	ch.loadMasterParamset()
	return ch
}
