package virtdev

import (
	"bytes"
	"text/template"

	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-mqtt/message"
)

type switchPayload struct {
	payload  *vdevices.StringParameter
	template *template.Template
}

type mqttSwitch struct {
	baseChannel

	topic  *vdevices.StringParameter
	retain *vdevices.BoolParameter
	on     switchPayload
	off    switchPayload
}

func (c *mqttSwitch) createTemplate(sw *switchPayload) {
	txt := sw.payload.Value().(string)
	specFuncs := createSpecificFuncs(c.virtualDevices.Devices, c.device, c)
	tmpl, err := template.New("mqttswitch").Funcs(tmplFuncs).Funcs(specFuncs).Parse(txt)
	if err != nil {
		log.Errorf("Invalid template '%s': %v", txt, err)
		return
	}
	sw.template = tmpl
}

func (c *mqttSwitch) createTemplates() {
	c.createTemplate(&c.on)
	c.createTemplate(&c.off)
}

func (c *mqttSwitch) publish(sw *switchPayload, value interface{}) {
	if sw.template == nil {
		log.Warningf("Invalid template: %s", sw.payload.Value().(string))
		return
	}
	var buf bytes.Buffer
	err := sw.template.Execute(&buf, value)
	if err != nil {
		log.Errorf("Execution of template '%s' failed: %v", sw.payload.Value().(string), err)
		return
	}
	c.virtualDevices.MQTTServer.Publish(
		c.topic.Value().(string),
		buf.Bytes(),
		message.QosExactlyOnce,
		c.retain.Value().(bool),
	)
}

func (vd *VirtualDevices) addMQTTSwitch(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(mqttSwitch)
	ch.virtualDevices = vd
	ch.device = dev

	// inititalize baseChannel
	kch := vdevices.NewSwitchChannel(dev)
	ch.GenericChannel = kch

	// TOPIC
	ch.topic = vdevices.NewStringParameter("TOPIC")
	ch.AddMasterParam(ch.topic)

	// RETAIN
	ch.retain = vdevices.NewBoolParameter("RETAIN")
	ch.AddMasterParam(ch.retain)

	// ON_PAYLOAD
	ch.on.payload = vdevices.NewStringParameter("ON_PAYLOAD")
	ch.AddMasterParam(ch.on.payload)

	// OFF_PAYLOAD
	ch.off.payload = vdevices.NewStringParameter("OFF_PAYLOAD")
	ch.AddMasterParam(ch.off.payload)

	// state change
	kch.OnSetState = func(state bool) bool {
		// update state for usage in template
		kch.SetState(state)
		if state {
			ch.publish(&ch.on, true)
		} else {
			ch.publish(&ch.off, false)
		}
		// update state in channel and publish event to CCU
		return true
	}

	// store master param on PutParamset
	ch.MasterParamset().HandlePutParamset(func() {
		ch.storeMasterParamset()
		ch.createTemplates()
	})

	// load master parameters from config
	ch.loadMasterParamset()

	// create templates
	ch.createTemplates()
	return ch
}
