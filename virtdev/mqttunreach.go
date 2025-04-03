package virtdev

import (
	"sync"
	"time"

	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-mqtt/service"
)

// Meaning of the STATE parameter: 0=contact is closed/off/OK; 1=contact is
// open/on/Error
type mqttUnreach struct {
	baseChannel
	digitalChannel *vdevices.DigitalChannel

	paramTopic        *vdevices.StringParameter
	paramErrorPattern *vdevices.StringParameter
	paramOkPattern    *vdevices.StringParameter
	paramMatcherKind  *vdevices.IntParameter
	paramDelay        *vdevices.FloatParameter

	subscribedTopic string
	onPublish       service.OnPublishFunc
	delayTimer      *time.Timer
	delayMutex      sync.Mutex
}

func (c *mqttUnreach) setConnError(connError bool) {
	// update this channel
	c.digitalChannel.SetState(connError)

	// set parameter UNREACH of channel 0 (maintenance channel)
	gch, err := c.device.Channel("0")
	if err != nil {
		log.Errorf("Maintenance channel (0) not found: %v", err)
		return
	}
	mch, ok := gch.(*vdevices.MaintenanceChannel)
	if !ok {
		log.Errorf("Channel (0) is not a maintenance channel")
		return
	}
	mch.SetUnreach(connError)
}

func (c *mqttUnreach) delayConnError(connError bool) {
	// no delay
	delayTime := c.paramDelay.Value().(float64)
	if delayTime <= 0.0 {
		c.setConnError(connError)
		return
	}

	// use timer
	c.delayMutex.Lock()
	defer c.delayMutex.Unlock()
	if connError {
		// start timer, if not already running
		if c.delayTimer == nil {
			c.delayTimer = time.AfterFunc(
				time.Duration(delayTime*float64(time.Second)),
				func() {
					c.delayMutex.Lock()
					defer c.delayMutex.Unlock()
					// check, if not stopped in the meantime
					if c.delayTimer != nil {
						c.setConnError(true)
					}
				},
			)
		}
	} else {
		// stop timer
		if c.delayTimer != nil {
			c.delayTimer.Stop()
			c.delayTimer = nil
		}
		// reset conn. error
		c.setConnError(false)
	}
}

func (c *mqttUnreach) start() {
	topic := c.paramTopic.Value().(string)
	if topic != "" {
		errorMatcher, err := newMatcher(c.paramMatcherKind, c.paramErrorPattern)
		if err != nil {
			log.Errorf("Creation of matcher for 'error' failed: %v", err)
			return
		}
		okMatcher, err := newMatcher(c.paramMatcherKind, c.paramOkPattern)
		if err != nil {
			log.Errorf("Creation of matcher for 'ok' failed: %v", err)
			return
		}
		c.onPublish = func(msg *message.PublishMessage) error {
			log.Debugf("Message for connection state %s:%d received: %s, %s", c.Description().Parent,
				c.Description().Index, msg.Topic(), msg.Payload())
			if errorMatcher.Match(msg.Payload()) {
				log.Debugf("Setting connection error %s:%d", c.Description().Parent, c.Description().Index)
				c.delayConnError(true)
			} else if okMatcher.Match(msg.Payload()) {
				log.Debugf("Clearing connection error %s:%d", c.Description().Parent, c.Description().Index)
				c.delayConnError(false)
			} else {
				log.Warningf("Invalid message for connection state %s:%d received: %s", c.Description().Parent,
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

func (c *mqttUnreach) stop() {
	if c.subscribedTopic != "" {
		// unsubscribe
		c.virtualDevices.MQTTServer.Unsubscribe(c.subscribedTopic, &c.onPublish)
		c.subscribedTopic = ""

		// stop timer
		c.delayMutex.Lock()
		defer c.delayMutex.Unlock()
		if c.delayTimer != nil {
			c.delayTimer.Stop()
			c.delayTimer = nil
		}
	}
}

func (vd *VirtualDevices) addMQTTUnreach(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(mqttUnreach)
	ch.virtualDevices = vd
	ch.device = dev

	// inititalize baseChannel
	ch.digitalChannel = vdevices.NewDoorSensorChannel(dev)
	ch.GenericChannel = ch.digitalChannel

	// TOPIC
	ch.paramTopic = vdevices.NewStringParameter("TOPIC")
	ch.AddMasterParam(ch.paramTopic)

	// ERROR_PATTERN (contact is open/on/Error)
	ch.paramErrorPattern = vdevices.NewStringParameter("ERROR_PATTERN")
	ch.AddMasterParam(ch.paramErrorPattern)

	// OK_PATTERN (contact is closed/off/OK)
	ch.paramOkPattern = vdevices.NewStringParameter("OK_PATTERN")
	ch.AddMasterParam(ch.paramOkPattern)

	// MATCHER
	ch.paramMatcherKind = newMatcherKindParameter("MATCHER")
	ch.AddMasterParam(ch.paramMatcherKind)

	// DELAY [s]
	ch.paramDelay = vdevices.NewFloatParameter("DELAY_TIME")
	ch.paramDelay.Description().Min = 0.0
	ch.paramDelay.Description().Unit = "s"
	ch.AddMasterParam(ch.paramDelay)

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
