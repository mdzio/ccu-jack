package virtdev

import (
	"github.com/mdzio/ccu-jack/mqtt"
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-mqtt/service"
)

type mqttKeyReceiver struct {
	baseChannel
	keyChannel *vdevices.KeyChannel
	mqttServer *mqtt.Server

	paramShortTopic       *vdevices.StringParameter
	paramShortPattern     *vdevices.StringParameter
	paramShortMatcherKind *vdevices.IntParameter
	shortSubscribedTopic  string
	shortOnPublish        service.OnPublishFunc

	paramLongTopic       *vdevices.StringParameter
	paramLongPattern     *vdevices.StringParameter
	paramLongMatcherKind *vdevices.IntParameter
	longSubscribedTopic  string
	longOnPublish        service.OnPublishFunc
}

func (c *mqttKeyReceiver) start() {
	shortTopic := c.paramShortTopic.Value().(string)
	if shortTopic != "" {
		matcher, err := newMatcher(c.paramShortMatcherKind, c.paramShortPattern)
		if err != nil {
			log.Errorf("Creation of matcher for short keypress failed: %v", err)
		} else {
			c.shortOnPublish = func(msg *message.PublishMessage) error {
				log.Debugf("Message for %s:%d short keypress received: %s, %s", c.Description().Parent,
					c.Description().Index, msg.Topic(), msg.Payload())
				if matcher.Match(msg.Payload()) {
					log.Debugf("Triggering short keypress on %s:%d", c.Description().Parent, c.Description().Index)
					// lock channel while modifying parameters
					c.Lock()
					c.keyChannel.PressShort()
					c.Unlock()
				}
				return nil
			}
			if err := c.mqttServer.Subscribe(shortTopic, message.QosExactlyOnce, &c.shortOnPublish); err != nil {
				log.Errorf("Subscribe failed on topic %s: %v", shortTopic, err)
			} else {
				c.shortSubscribedTopic = shortTopic
			}
		}
	}

	longTopic := c.paramLongTopic.Value().(string)
	if longTopic != "" {
		matcher, err := newMatcher(c.paramLongMatcherKind, c.paramLongPattern)
		if err != nil {
			log.Errorf("Creation of matcher for long keypress failed: %v", err)
		} else {
			c.longOnPublish = func(msg *message.PublishMessage) error {
				log.Debugf("Message for %s:%d long keypress received: %s, %s", c.Description().Parent,
					c.Description().Index, msg.Topic(), msg.Payload())
				if matcher.Match(msg.Payload()) {
					log.Debugf("Triggering long keypress on %s:%d", c.Description().Parent, c.Description().Index)
					// lock channel while modifying parameters
					c.Lock()
					c.keyChannel.PressLong()
					c.Unlock()
				}
				return nil
			}
			if err := c.mqttServer.Subscribe(longTopic, message.QosExactlyOnce, &c.longOnPublish); err != nil {
				log.Errorf("Subscribe failed on topic %s: %v", longTopic, err)
			} else {
				c.longSubscribedTopic = longTopic
			}
		}
	}
}

func (c *mqttKeyReceiver) stop() {
	if c.shortSubscribedTopic != "" {
		c.mqttServer.Unsubscribe(c.shortSubscribedTopic, &c.shortOnPublish)
		c.shortSubscribedTopic = ""
	}
	if c.longSubscribedTopic != "" {
		c.mqttServer.Unsubscribe(c.longSubscribedTopic, &c.longOnPublish)
		c.longSubscribedTopic = ""
	}
}

func (vd *VirtualDevices) addMQTTKeyReceiver(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(mqttKeyReceiver)

	// inititalize
	ch.keyChannel = vdevices.NewKeyChannel(dev)
	ch.GenericChannel = ch.keyChannel
	ch.store = vd.Store
	ch.mqttServer = vd.MQTTServer

	// SHORT_TOPIC
	ch.paramShortTopic = vdevices.NewStringParameter("SHORT_TOPIC")
	ch.AddMasterParam(ch.paramShortTopic)

	// SHORT_PATTERN
	ch.paramShortPattern = vdevices.NewStringParameter("SHORT_PATTERN")
	ch.AddMasterParam(ch.paramShortPattern)

	// SHORT_MATCHER
	ch.paramShortMatcherKind = newMatcherKindParameter("SHORT_MATCHER")
	ch.AddMasterParam(ch.paramShortMatcherKind)

	// LONG_TOPIC
	ch.paramLongTopic = vdevices.NewStringParameter("LONG_TOPIC")
	ch.AddMasterParam(ch.paramLongTopic)

	// LONG_PATTERN
	ch.paramLongPattern = vdevices.NewStringParameter("LONG_PATTERN")
	ch.AddMasterParam(ch.paramLongPattern)

	// SHORT_MATCHER
	ch.paramLongMatcherKind = newMatcherKindParameter("LONG_MATCHER")
	ch.AddMasterParam(ch.paramLongMatcherKind)

	// clean up
	ch.keyChannel.OnDispose = ch.stop

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
