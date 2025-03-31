package virtdev

import (
	"bytes"
	"text/template"

	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-mqtt/service"
)

type mqttSwitchFeedback struct {
	baseChannel
	digitalChannel *vdevices.DigitalChannel

	// sending parameters
	paramCommandTopic *vdevices.StringParameter
	paramRetain       *vdevices.BoolParameter
	on                switchPayload
	off               switchPayload

	// feedback parameters
	paramFBTopic     *vdevices.StringParameter
	paramOnPattern   *vdevices.StringParameter
	paramOffPattern  *vdevices.StringParameter
	paramMatcherKind *vdevices.IntParameter

	subscribedTopic string
	onPublish       service.OnPublishFunc
}

func (c *mqttSwitchFeedback) createTemplate(sw *switchPayload) {
	txt := sw.payload.Value().(string)
	specFuncs := createSpecificFuncs(c.virtualDevices.Devices, c.device, c)
	tmpl, err := template.New("mqttswitch").Funcs(tmplFuncs).Funcs(specFuncs).Parse(txt)
	if err != nil {
		log.Errorf("Invalid template '%s': %v", txt, err)
		return
	}
	sw.template = tmpl
}

func (c *mqttSwitchFeedback) start() {
	c.createTemplate(&c.on)
	c.createTemplate(&c.off)

	fbTopic := c.paramFBTopic.Value().(string)
	if fbTopic != "" {
		cmdTopic := c.paramCommandTopic.Value().(string)
		if matchTopic(fbTopic, cmdTopic) {
			log.Errorf("Feedback topic '%s' must not overlap with command topic '%s'", fbTopic, cmdTopic)
			return
		}
		onMatcher, err := newMatcher(c.paramMatcherKind, c.paramOnPattern)
		if err != nil {
			log.Errorf("Creation of matcher for 'on' failed: %v", err)
			return
		}
		offMatcher, err := newMatcher(c.paramMatcherKind, c.paramOffPattern)
		if err != nil {
			log.Errorf("Creation of matcher for 'off' failed: %v", err)
			return
		}
		c.onPublish = func(msg *message.PublishMessage) error {
			log.Debugf("Message for switch %s:%d received: %s, %s", c.Description().Parent,
				c.Description().Index, msg.Topic(), msg.Payload())
			if onMatcher.Match(msg.Payload()) {
				log.Debugf("Turning on switch %s:%d", c.Description().Parent, c.Description().Index)
				c.digitalChannel.SetState(true)
			} else if offMatcher.Match(msg.Payload()) {
				log.Debugf("Turning off switch %s:%d", c.Description().Parent, c.Description().Index)
				c.digitalChannel.SetState(false)
			} else {
				log.Warningf("Invalid message for switch %s:%d received: %s", c.Description().Parent,
					c.Description().Index, msg.Payload())
			}
			return nil
		}
		if err := c.virtualDevices.MQTTServer.Subscribe(fbTopic, message.QosExactlyOnce, &c.onPublish); err != nil {
			log.Errorf("Subscribe failed on topic %s: %v", fbTopic, err)
		} else {
			c.subscribedTopic = fbTopic
		}
	}
}

func (c *mqttSwitchFeedback) publish(sw *switchPayload, value interface{}) {
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
		c.paramCommandTopic.Value().(string),
		buf.Bytes(),
		message.QosExactlyOnce,
		c.paramRetain.Value().(bool),
	)
}

func (c *mqttSwitchFeedback) stop() {
	if c.subscribedTopic != "" {
		c.virtualDevices.MQTTServer.Unsubscribe(c.subscribedTopic, &c.onPublish)
		c.subscribedTopic = ""
	}
}

func (vd *VirtualDevices) addMQTTSwitchFeedback(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(mqttSwitchFeedback)
	ch.virtualDevices = vd
	ch.device = dev

	// inititalize baseChannel
	ch.digitalChannel = vdevices.NewSwitchChannel(dev)
	ch.GenericChannel = ch.digitalChannel

	// COMMAND_TOPIC
	ch.paramCommandTopic = vdevices.NewStringParameter("COMMAND_TOPIC")
	ch.AddMasterParam(ch.paramCommandTopic)

	// RETAIN
	ch.paramRetain = vdevices.NewBoolParameter("RETAIN")
	ch.AddMasterParam(ch.paramRetain)

	// ON_PAYLOAD
	ch.on.payload = vdevices.NewStringParameter("ON_PAYLOAD")
	ch.AddMasterParam(ch.on.payload)

	// OFF_PAYLOAD
	ch.off.payload = vdevices.NewStringParameter("OFF_PAYLOAD")
	ch.AddMasterParam(ch.off.payload)

	// FEEDBACK_TOPIC
	ch.paramFBTopic = vdevices.NewStringParameter("FEEDBACK_TOPIC")
	ch.AddMasterParam(ch.paramFBTopic)

	// ON_PATTERN
	ch.paramOnPattern = vdevices.NewStringParameter("ON_PATTERN")
	ch.AddMasterParam(ch.paramOnPattern)

	// OFF_PATTERN
	ch.paramOffPattern = vdevices.NewStringParameter("OFF_PATTERN")
	ch.AddMasterParam(ch.paramOffPattern)

	// MATCHER
	ch.paramMatcherKind = newMatcherKindParameter("MATCHER")
	ch.AddMasterParam(ch.paramMatcherKind)

	// state change
	ch.digitalChannel.OnSetState = func(state bool) bool {
		if state {
			ch.publish(&ch.on, true)
		} else {
			ch.publish(&ch.off, false)
		}
		// do not update state in channel
		return false
	}

	// clean up
	ch.digitalChannel.OnDispose = ch.stop

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
