package virtdev

import (
	"github.com/mdzio/ccu-jack/mqtt"
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-mqtt/service"
)

type mqttSwitchFeedback struct {
	baseChannel
	digitalChannel *vdevices.DigitalChannel
	mqttServer     *mqtt.Server

	// sending parameters
	paramCommandTopic *vdevices.StringParameter
	paramRetain       *vdevices.BoolParameter
	paramOnPayload    *vdevices.StringParameter
	paramOffPayload   *vdevices.StringParameter

	// feedback parameters
	paramFBTopic     *vdevices.StringParameter
	paramOnPattern   *vdevices.StringParameter
	paramOffPattern  *vdevices.StringParameter
	paramMatcherKind *vdevices.IntParameter

	subscribedTopic string
	onPublish       service.OnPublishFunc
}

func (c *mqttSwitchFeedback) start() {
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
				// lock channel while modifying parameters
				c.Lock()
				c.digitalChannel.SetState(true)
				c.Unlock()
			} else if offMatcher.Match(msg.Payload()) {
				log.Debugf("Turning off switch %s:%d", c.Description().Parent, c.Description().Index)
				// lock channel while modifying parameters
				c.Lock()
				c.digitalChannel.SetState(false)
				c.Unlock()
			} else {
				log.Warningf("Invalid message for switch %s:%d received: %s", c.Description().Parent,
					c.Description().Index, msg.Payload())
			}
			return nil
		}
		if err := c.mqttServer.Subscribe(fbTopic, message.QosExactlyOnce, &c.onPublish); err != nil {
			log.Errorf("Subscribe failed on topic %s: %v", fbTopic, err)
		} else {
			c.subscribedTopic = fbTopic
		}
	}
}

func (c *mqttSwitchFeedback) stop() {
	if c.subscribedTopic != "" {
		c.mqttServer.Unsubscribe(c.subscribedTopic, &c.onPublish)
		c.subscribedTopic = ""
	}
}

func (vd *VirtualDevices) addMQTTSwitchFeedback(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(mqttSwitchFeedback)

	// inititalize baseChannel
	ch.digitalChannel = vdevices.NewSwitchChannel(dev)
	ch.GenericChannel = ch.digitalChannel
	ch.store = vd.Store
	ch.mqttServer = vd.MQTTServer

	// COMMAND_TOPIC
	ch.paramCommandTopic = vdevices.NewStringParameter("COMMAND_TOPIC")
	ch.AddMasterParam(ch.paramCommandTopic)

	// RETAIN
	ch.paramRetain = vdevices.NewBoolParameter("RETAIN")
	ch.AddMasterParam(ch.paramRetain)

	// ON_PAYLOAD
	ch.paramOnPayload = vdevices.NewStringParameter("ON_PAYLOAD")
	ch.AddMasterParam(ch.paramOnPayload)

	// OFF_PAYLOAD
	ch.paramOffPayload = vdevices.NewStringParameter("OFF_PAYLOAD")
	ch.AddMasterParam(ch.paramOffPayload)

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
		var payload string
		if state {
			payload = ch.paramOnPayload.Value().(string)
		} else {
			payload = ch.paramOffPayload.Value().(string)
		}
		vd.MQTTServer.Publish(
			ch.paramCommandTopic.Value().(string),
			[]byte(payload),
			message.QosExactlyOnce,
			ch.paramRetain.Value().(bool),
		)
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
	ch.Lock()
	defer ch.Unlock()
	ch.start()
	return ch
}
