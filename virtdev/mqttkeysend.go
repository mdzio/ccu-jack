package virtdev

import (
	"bytes"
	"text/template"

	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-mqtt/message"
)

type sendButton struct {
	topic    *vdevices.StringParameter
	payload  *vdevices.StringParameter
	retain   *vdevices.BoolParameter
	template *template.Template
}

type mqttKeySender struct {
	baseChannel
	short sendButton
	long  sendButton
}

func (c *mqttKeySender) createTemplate(button *sendButton) {
	txt := button.payload.Value().(string)
	specFuncs := createSpecificFuncs(c.virtualDevices.Devices, c.device, c)
	tmpl, err := template.New("mqttkeysender").Funcs(tmplFuncs).Funcs(specFuncs).Parse(txt)
	if err != nil {
		log.Errorf("Invalid template '%s': %v", txt, err)
		return
	}
	button.template = tmpl
}

func (c *mqttKeySender) createTemplates() {
	c.createTemplate(&c.short)
	c.createTemplate(&c.long)
}

func (c *mqttKeySender) publish(button *sendButton) {
	if button.template == nil {
		log.Warningf("Invalid template: %s", button.payload.Value().(string))
		return
	}
	var buf bytes.Buffer
	err := button.template.Execute(&buf, nil)
	if err != nil {
		log.Errorf("Execution of template '%s' failed: %v", button.payload.Value().(string), err)
		return
	}
	c.virtualDevices.MQTTServer.Publish(
		button.topic.Value().(string),
		buf.Bytes(),
		message.QosExactlyOnce,
		button.retain.Value().(bool),
	)
}

// addMQTTKeySender must be called with config store locked.
func (vd *VirtualDevices) addMQTTKeySender(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(mqttKeySender)
	ch.virtualDevices = vd
	ch.device = dev

	// inititalize baseChannel
	kch := vdevices.NewKeyChannel(dev)
	ch.GenericChannel = kch

	// SHORT_TOPIC
	ch.short.topic = vdevices.NewStringParameter("SHORT_TOPIC")
	ch.AddMasterParam(ch.short.topic)

	// SHORT_PAYLOAD
	ch.short.payload = vdevices.NewStringParameter("SHORT_PAYLOAD")
	ch.AddMasterParam(ch.short.payload)

	// SHORT_RETAIN
	ch.short.retain = vdevices.NewBoolParameter("SHORT_RETAIN")
	ch.AddMasterParam(ch.short.retain)

	// LONG_TOPIC
	ch.long.topic = vdevices.NewStringParameter("LONG_TOPIC")
	ch.AddMasterParam(ch.long.topic)

	// LONG_PAYLOAD
	ch.long.payload = vdevices.NewStringParameter("LONG_PAYLOAD")
	ch.AddMasterParam(ch.long.payload)

	// LONG_RETAIN
	ch.long.retain = vdevices.NewBoolParameter("LONG_RETAIN")
	ch.AddMasterParam(ch.long.retain)

	// PRESS_SHORT
	kch.OnPressShort = func() bool { ch.publish(&ch.short); return true }

	// PRESS_LONG
	kch.OnPressLong = func() bool { ch.publish(&ch.long); return true }

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
