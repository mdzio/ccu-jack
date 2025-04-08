package virtdev

import (
	"time"

	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-mqtt/service"
)

type udEvent int

const (
	udEventStop udEvent = iota
	udEventOk
	udEventError
)

type unreachDelay struct {
	okDur    time.Duration
	errorDur time.Duration
	cmd      chan udEvent
	done     chan struct{}
	onOk     func()
	onError  func()
}

func (ud *unreachDelay) start() {
	ud.cmd = make(chan udEvent)
	ud.done = make(chan struct{})
	go ud.run()
}

func (ud *unreachDelay) run() {
	var okT *time.Timer
	var errorT *time.Timer
	var okC <-chan time.Time
	var errorC <-chan time.Time
out:
	for {
		select {
		case cmd := <-ud.cmd:
			switch cmd {
			case udEventStop:
				break out
			case udEventOk:
				if errorT != nil {
					errorT.Stop()
					errorT = nil
					errorC = nil
				}
				ud.onOk()
				if ud.okDur != time.Duration(0) {
					if okT != nil {
						okT.Stop()
					}
					okT = time.NewTimer(ud.okDur)
					okC = okT.C
				}
			case udEventError:
				if errorT != nil {
					break
				}
				errorT = time.NewTimer(ud.errorDur)
				errorC = errorT.C
			}
		case <-okC:
			if errorT != nil {
				errorT.Stop()
				errorT = nil
				errorC = nil
			}
			ud.onOk()
		case <-errorC:
			errorT = nil
			errorC = nil
			ud.onError()
		}
	}
	if okT != nil {
		okT.Stop()
	}
	if errorT != nil {
		errorT.Stop()
	}
	ud.done <- struct{}{}
}

func (ud *unreachDelay) stop() {
	ud.cmd <- udEventStop
	<-ud.done
}

// Meaning of the STATE parameter: 0=contact is closed/off/OK; 1=contact is
// open/on/Error
type mqttUnreach struct {
	baseChannel
	digitalChannel *vdevices.DigitalChannel

	paramTopic        *vdevices.StringParameter
	paramErrorPattern *vdevices.StringParameter
	paramOkPattern    *vdevices.StringParameter
	paramMatcherKind  *vdevices.IntParameter
	paramOkDelay      *vdevices.FloatParameter
	paramErrorDelay   *vdevices.FloatParameter

	subscribedTopic string
	onPublish       service.OnPublishFunc
	delay           unreachDelay
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

func (c *mqttUnreach) start() {
	topic := c.paramTopic.Value().(string)
	if topic != "" {
		c.delay.okDur = time.Duration(c.paramOkDelay.Value().(float64) * float64(time.Second))
		c.delay.errorDur = time.Duration(c.paramErrorDelay.Value().(float64) * float64(time.Second))
		c.delay.start()

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
				c.delay.cmd <- udEventError
			} else if okMatcher.Match(msg.Payload()) {
				log.Debugf("Clearing connection error %s:%d", c.Description().Parent, c.Description().Index)
				c.delay.cmd <- udEventOk
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

		// stop delay
		c.delay.stop()
	}
}

func (vd *VirtualDevices) addMQTTUnreach(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(mqttUnreach)
	ch.virtualDevices = vd
	ch.device = dev
	ch.delay.onOk = func() { ch.setConnError(false) }
	ch.delay.onError = func() { ch.setConnError(true) }

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

	// OK_DELAY [s]
	ch.paramOkDelay = vdevices.NewFloatParameter("OK_DELAY")
	ch.paramOkDelay.Description().Min = 0.0
	ch.paramOkDelay.Description().Unit = "s"
	ch.AddMasterParam(ch.paramOkDelay)

	// ERROR_DELAY [s]
	ch.paramErrorDelay = vdevices.NewFloatParameter("ERROR_DELAY")
	ch.paramErrorDelay.Description().Min = 0.0
	ch.paramErrorDelay.Description().Unit = "s"
	ch.AddMasterParam(ch.paramErrorDelay)

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
