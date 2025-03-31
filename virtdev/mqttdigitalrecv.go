package virtdev

import (
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-mqtt/service"
)

type mqttDigitalReceiver struct {
	baseChannel
	digitalChannel *vdevices.DigitalChannel

	paramTopic       *vdevices.StringParameter
	paramOnPattern   *vdevices.StringParameter
	paramOffPattern  *vdevices.StringParameter
	paramMatcherKind *vdevices.IntParameter

	subscribedTopic string
	onPublish       service.OnPublishFunc
}

func (c *mqttDigitalReceiver) start() {
	topic := c.paramTopic.Value().(string)
	if topic != "" {
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
			log.Debugf("Message for digital input %s:%d received: %s, %s", c.Description().Parent,
				c.Description().Index, msg.Topic(), msg.Payload())
			if onMatcher.Match(msg.Payload()) {
				log.Debugf("Turning on digital input %s:%d", c.Description().Parent, c.Description().Index)
				c.digitalChannel.SetState(true)
			} else if offMatcher.Match(msg.Payload()) {
				log.Debugf("Turning off digital input %s:%d", c.Description().Parent, c.Description().Index)
				c.digitalChannel.SetState(false)
			} else {
				log.Warningf("Invalid message for digital input %s:%d received: %s", c.Description().Parent,
					c.Description().Index, msg.Payload())
			}
			return nil
		}
		if err := c.virtualDevices.MQTTServer.Subscribe(topic, message.QosExactlyOnce, &c.onPublish); err != nil {
			log.Errorf("Subscribe failed on topic %s: %v", topic, err)
			return
		}
		c.subscribedTopic = topic
	}
}

func (c *mqttDigitalReceiver) stop() {
	if c.subscribedTopic != "" {
		c.virtualDevices.MQTTServer.Unsubscribe(c.subscribedTopic, &c.onPublish)
		c.subscribedTopic = ""
	}
}

func (vd *VirtualDevices) addMQTTDoorSensor(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(mqttDigitalReceiver)
	ch.virtualDevices = vd
	ch.device = dev

	// inititalize baseChannel
	ch.digitalChannel = vdevices.NewDoorSensorChannel(dev)
	ch.GenericChannel = ch.digitalChannel

	// TOPIC
	ch.paramTopic = vdevices.NewStringParameter("TOPIC")
	ch.AddMasterParam(ch.paramTopic)

	// OPEN_PATTERN
	ch.paramOnPattern = vdevices.NewStringParameter("OPEN_PATTERN")
	ch.AddMasterParam(ch.paramOnPattern)

	// CLOSED_PATTERN
	ch.paramOffPattern = vdevices.NewStringParameter("CLOSED_PATTERN")
	ch.AddMasterParam(ch.paramOffPattern)

	// MATCHER
	ch.paramMatcherKind = newMatcherKindParameter("MATCHER")
	ch.AddMasterParam(ch.paramMatcherKind)

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
