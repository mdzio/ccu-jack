package virtdev

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/mdzio/ccu-jack/mqtt"
	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-mqtt/service"
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

type mqttDigitalInput struct {
	baseChannel
	digitalChannel *vdevices.DigitalChannel
	mqttServer     *mqtt.Server

	paramTopic       *vdevices.StringParameter
	paramOnPattern   *vdevices.StringParameter
	paramOffPattern  *vdevices.StringParameter
	paramMatcherKind *vdevices.IntParameter

	subscribedTopic string
	onPublish       service.OnPublishFunc
}

func (c *mqttDigitalInput) start() {
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
				// lock channel while modifying parameters
				c.Lock()
				c.digitalChannel.SetState(true)
				c.Unlock()
			} else if offMatcher.Match(msg.Payload()) {
				log.Debugf("Turning off digital input %s:%d", c.Description().Parent, c.Description().Index)
				// lock channel while modifying parameters
				c.Lock()
				c.digitalChannel.SetState(false)
				c.Unlock()
			} else {
				log.Warningf("Invalid message for digital input %s:%d received: %s", c.Description().Parent,
					c.Description().Index, msg.Payload())
			}
			return nil
		}
		if err := c.mqttServer.Subscribe(topic, message.QosExactlyOnce, &c.onPublish); err != nil {
			log.Errorf("Subscribe failed on topic %s: %v", topic, err)
			return
		}
		c.subscribedTopic = topic
	}
}

func (c *mqttDigitalInput) stop() {
	if c.subscribedTopic != "" {
		c.mqttServer.Unsubscribe(c.subscribedTopic, &c.onPublish)
		c.subscribedTopic = ""
	}
}

func (vd *VirtualDevices) addMQTTDoorSensor(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(mqttDigitalInput)

	// inititalize baseChannel
	ch.digitalChannel = vdevices.NewDoorSensorChannel(dev)
	ch.GenericChannel = ch.digitalChannel
	ch.store = vd.Store
	ch.mqttServer = vd.MQTTServer

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
	ch.Lock()
	defer ch.Unlock()
	ch.start()
	return ch
}

type mqttAnalogReceiver struct {
	baseChannel
	analogChannel *vdevices.AnalogInputChannel
	mqttServer    *mqtt.Server

	paramTopic         *vdevices.StringParameter
	paramPattern       *vdevices.StringParameter
	paramExtractorKind *vdevices.IntParameter
	paramRegexpGroup   *vdevices.IntParameter

	subscribedTopic string
	onPublish       service.OnPublishFunc
}

func (c *mqttAnalogReceiver) start() {
	c.subscribedTopic = c.paramTopic.Value().(string)
	if c.subscribedTopic != "" {
		extractor, err := newExtractor(c.paramExtractorKind, c.paramPattern, c.paramRegexpGroup)
		if err != nil {
			log.Errorf("Creation of value extractor for analog receiver %s:%d failed: %v", c.Description().Parent,
				c.Description().Index, err)
			return
		}
		c.onPublish = func(msg *message.PublishMessage) error {
			log.Debugf("Message for analog receiver %s:%d received: %s, %s", c.Description().Parent,
				c.Description().Index, msg.Topic(), msg.Payload())
			// lock channel while modifying parameters
			c.Lock()
			defer c.Unlock()
			value, err := extractor.Extract(msg.Payload())
			if err != nil {
				log.Warningf("Extraction of value for analog receiver %s:%d failed: %v", c.Description().Parent,
					c.Description().Index, err)
				// set overflow status
				c.analogChannel.SetVoltageStatus(2)
				return nil
			}
			c.analogChannel.SetVoltage(value)
			// set normal status
			c.analogChannel.SetVoltageStatus(0)
			return nil
		}
		c.mqttServer.Subscribe(c.subscribedTopic, message.QosExactlyOnce, &c.onPublish)
	}
}

func (c *mqttAnalogReceiver) stop() {
	if c.subscribedTopic != "" {
		c.mqttServer.Unsubscribe(c.subscribedTopic, &c.onPublish)
		c.subscribedTopic = ""
	}
}

func (vd *VirtualDevices) addMQTTAnalogReceiver(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(mqttAnalogReceiver)

	// inititalize baseChannel
	ch.analogChannel = vdevices.NewAnalogInputChannel(dev)
	ch.GenericChannel = ch.analogChannel
	ch.store = vd.Store
	ch.mqttServer = vd.MQTTServer

	// TOPIC
	ch.paramTopic = vdevices.NewStringParameter("TOPIC")
	ch.AddMasterParam(ch.paramTopic)

	// PATTERN
	ch.paramPattern = vdevices.NewStringParameter("PATTERN")
	ch.AddMasterParam(ch.paramPattern)

	// EXTRACTOR
	ch.paramExtractorKind = newExtractorKindParameter("EXTRACTOR")
	ch.AddMasterParam(ch.paramExtractorKind)

	// REGEXP_GROUP
	ch.paramRegexpGroup = vdevices.NewIntParameter("REGEXP_GROUP")
	ch.paramRegexpGroup.Description().Min = 0
	ch.paramRegexpGroup.Description().Max = 100
	ch.paramRegexpGroup.Description().Default = 0
	ch.AddMasterParam(ch.paramRegexpGroup)

	// clean up
	ch.analogChannel.OnDispose = ch.stop

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

type mqttDimmer struct {
	baseChannel
	dimmerChannel   *vdevices.DimmerChannel
	mqttServer      *mqtt.Server
	subscribedTopic string
	onPublish       service.OnPublishFunc
	oldLevel        float64
	template        *template.Template

	// range parameters
	paramRangeMin *vdevices.FloatParameter
	paramRangeMax *vdevices.FloatParameter

	// command parameters
	paramCommandTopic *vdevices.StringParameter
	paramRetain       *vdevices.BoolParameter
	paramTemplate     *vdevices.StringParameter

	// feedback parameters
	paramFBTopic       *vdevices.StringParameter
	paramPattern       *vdevices.StringParameter
	paramExtractorKind *vdevices.IntParameter
	paramRegexpGroup   *vdevices.IntParameter
}

func (c *mqttDimmer) start() {
	tmplText := c.paramTemplate.Value().(string)
	tmpl, err := template.New("mqttdimmer").Funcs(tmplFuncs).Parse(tmplText)
	if err != nil {
		log.Errorf("Invalid template '%s': %v", tmplText, err)
		tmpl = nil
	}
	c.template = tmpl

	fbTopic := c.paramFBTopic.Value().(string)
	if fbTopic != "" {
		cmdTopic := c.paramCommandTopic.Value().(string)
		if matchTopic(fbTopic, cmdTopic) {
			log.Errorf("Feedback topic '%s' must not overlap with command topic '%s'", fbTopic, cmdTopic)
			return
		}
		extractor, err := newExtractor(c.paramExtractorKind, c.paramPattern, c.paramRegexpGroup)
		if err != nil {
			log.Errorf("Creation of value extractor for MQTT dimmer %s:%d failed: %v", c.Description().Parent,
				c.Description().Index, err)
			return
		}
		c.onPublish = func(msg *message.PublishMessage) error {
			log.Debugf("Message for MQTT dimmer %s:%d received: %s, %s", c.Description().Parent,
				c.Description().Index, msg.Topic(), msg.Payload())
			// lock channel while modifying parameters
			c.Lock()
			defer c.Unlock()
			value, err := extractor.Extract(msg.Payload())
			if err != nil {
				log.Warningf("Extraction of value for MQTT dimmer %s:%d failed: %v", c.Description().Parent,
					c.Description().Index, err)
				// nothing can be done
				return nil
			}
			mappedValue := c.mapFromRange(value)
			c.dimmerChannel.SetLevel(mappedValue)
			return nil
		}
		if err := c.mqttServer.Subscribe(fbTopic, message.QosExactlyOnce, &c.onPublish); err != nil {
			log.Errorf("Subscribe failed on topic %s: %v", fbTopic, err)
		} else {
			c.subscribedTopic = fbTopic
		}

	}
}

func (c *mqttDimmer) stop() {
	if c.subscribedTopic != "" {
		c.mqttServer.Unsubscribe(c.subscribedTopic, &c.onPublish)
		c.subscribedTopic = ""
	}
}

func (c *mqttDimmer) mapToRange(value float64) float64 {
	min := c.paramRangeMin.Value().(float64)
	max := c.paramRangeMax.Value().(float64)
	return value*(max-min) + min
}

func (c *mqttDimmer) mapFromRange(value float64) float64 {
	min := c.paramRangeMin.Value().(float64)
	max := c.paramRangeMax.Value().(float64)
	if min == max {
		return 0.0
	}
	out := (value - min) / (max - min)
	if out < 0.0 {
		out = 0.0
	}
	if out > 1.0 {
		out = 1.0
	}
	return out
}

func (c *mqttDimmer) publishToMQTT(value float64) {
	if c.template == nil {
		log.Warningf("Invalid template: %s", c.paramTemplate.Value().(string))
		return
	}
	mappedValue := c.mapToRange(value)
	var buf bytes.Buffer
	err := c.template.Execute(&buf, mappedValue)
	if err != nil {
		log.Errorf("Execution of template '%s' failed for value %g: %v", c.paramTemplate.Value().(string), mappedValue, err)
		return
	}
	c.mqttServer.Publish(
		c.paramCommandTopic.Value().(string),
		buf.Bytes(),
		message.QosExactlyOnce,
		c.paramRetain.Value().(bool),
	)
}

func (vd *VirtualDevices) addMQTTDimmer(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(mqttDimmer)

	// inititalize baseChannel
	ch.dimmerChannel = vdevices.NewDimmerChannel(dev)
	ch.GenericChannel = ch.dimmerChannel
	ch.store = vd.Store
	ch.mqttServer = vd.MQTTServer

	// RANGE_MIN
	ch.paramRangeMin = vdevices.NewFloatParameter("RANGE_MIN")
	ch.paramRangeMin.Description().Default = 0.0
	ch.paramRangeMin.InternalSetValue(0.0)
	ch.AddMasterParam(ch.paramRangeMin)

	// RANGE_MAX
	ch.paramRangeMax = vdevices.NewFloatParameter("RANGE_MAX")
	ch.paramRangeMax.Description().Default = 1.0
	ch.paramRangeMax.InternalSetValue(1.0)
	ch.AddMasterParam(ch.paramRangeMax)

	// COMMAND_TOPIC
	ch.paramCommandTopic = vdevices.NewStringParameter("COMMAND_TOPIC")
	ch.AddMasterParam(ch.paramCommandTopic)

	// RETAIN
	ch.paramRetain = vdevices.NewBoolParameter("RETAIN")
	ch.AddMasterParam(ch.paramRetain)

	// TEMPLATE
	ch.paramTemplate = vdevices.NewStringParameter("TEMPLATE")
	ch.paramTemplate.Description().Default = "{{ . }}"
	ch.paramTemplate.InternalSetValue("{{ . }}")
	ch.AddMasterParam(ch.paramTemplate)

	// FEEDBACK_TOPIC
	ch.paramFBTopic = vdevices.NewStringParameter("FEEDBACK_TOPIC")
	ch.AddMasterParam(ch.paramFBTopic)

	// PATTERN
	ch.paramPattern = vdevices.NewStringParameter("PATTERN")
	ch.AddMasterParam(ch.paramPattern)

	// EXTRACTOR
	ch.paramExtractorKind = newExtractorKindParameter("EXTRACTOR")
	ch.AddMasterParam(ch.paramExtractorKind)

	// REGEXP_GROUP
	ch.paramRegexpGroup = vdevices.NewIntParameter("REGEXP_GROUP")
	ch.paramRegexpGroup.Description().Min = 0
	ch.paramRegexpGroup.Description().Max = 100
	ch.paramRegexpGroup.Description().Default = 0
	ch.AddMasterParam(ch.paramRegexpGroup)

	// level change
	ch.dimmerChannel.OnSetLevel = func(value float64) bool {
		if value != 0.0 {
			// remember previous dimmer level
			ch.oldLevel = value
		}
		ch.publishToMQTT(value)
		return true
	}

	// set old level
	ch.dimmerChannel.OnSetOldLevel = func() bool {
		// restore previous dimmer level
		ch.dimmerChannel.SetLevel(ch.oldLevel)
		ch.publishToMQTT(ch.oldLevel)
		return true
	}

	// clean up
	ch.dimmerChannel.OnDispose = ch.stop

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

func newExtractorKindParameter(id string) *vdevices.IntParameter {
	p := vdevices.NewIntParameter(id)
	p.Description().Type = itf.ParameterTypeEnum
	// align with extractorKind constants
	p.Description().ValueList = []string{"AFTER", "BEFORE", "REGEXP", "ALL", "TEMPLATE"}
	p.Description().Min = 0
	p.Description().Max = len(p.Description().ValueList) - 1
	p.Description().Default = 0
	return p
}

type extractorKind int

const (
	// align with function newExtractorKindParameter
	ExtractorAfter extractorKind = iota
	ExtractorBefore
	ExtractorRegexp
	ExtractorAll
	ExtractorTemplate
)

type extractor interface {
	Extract(payload []byte) (float64, error)
}

type extractorRegexp struct {
	regexp   *regexp.Regexp
	groupIdx int
}

func (e *extractorRegexp) Extract(payload []byte) (float64, error) {
	groups := e.regexp.FindStringSubmatch(string(payload))
	if groups == nil {
		return 0.0, fmt.Errorf("Regexp does not match: %s", payload)
	}
	if e.groupIdx < 0 || e.groupIdx >= len(groups) {
		return 0.0, fmt.Errorf("Invalid group index: %d", e.groupIdx)
	}
	fval, err := strconv.ParseFloat(groups[e.groupIdx], 64)
	if err != nil {
		return 0.0, fmt.Errorf("Regexp returned invalid number literal: %s", groups[e.groupIdx])
	}
	return fval, nil
}

func newExtractorRegexp(pattern string, groupIdx int) (extractor, error) {
	log.Tracef("Creating extractor with regular expression %s and group %d", pattern, groupIdx)
	regexp, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("Invalid regular expression: %s", pattern)
	}
	return &extractorRegexp{regexp: regexp, groupIdx: groupIdx}, nil
}

type extractorTmpl struct {
	tmpl *template.Template
}

var tmplFuncs = template.FuncMap{
	"round": math.Round,
	"parseJSON": func(str string) (interface{}, error) {
		var obj interface{}
		err := json.Unmarshal([]byte(str), &obj)
		if err != nil {
			return nil, err
		}
		return obj, nil
	},
	"contains":  strings.Contains,
	"fields":    strings.Fields,
	"split":     strings.Split,
	"toLower":   strings.ToLower,
	"toUpper":   strings.ToUpper,
	"trimSpace": strings.TrimSpace,
}

func (e *extractorTmpl) Extract(payload []byte) (float64, error) {
	var sb strings.Builder
	err := e.tmpl.Execute(&sb, string(payload))
	if err != nil {
		return 0.0, fmt.Errorf("Template execution failed for payload '%s': %v", string(payload), err)
	}
	fval, err := strconv.ParseFloat(sb.String(), 64)
	if err != nil {
		return 0.0, fmt.Errorf("Template returned invalid number literal: %s", sb.String())
	}
	return fval, nil
}

func newExtractorTmpl(pattern string) (extractor, error) {
	tmpl, err := template.New("").Funcs(tmplFuncs).Parse(pattern)
	if err != nil {
		return nil, fmt.Errorf("Invalid template '%s': %v", pattern, err)
	}
	return &extractorTmpl{tmpl: tmpl}, nil
}

const (
	numberPattern  = `([+-]?(\d+(\.\d*)?|\.\d+))`
	skipPattern    = `[^\d.+-]*`
	startWSPattern = `^\s*`
	endWSPattern   = `\s*$`
)

func newExtractor(kindParam *vdevices.IntParameter, patternParam *vdevices.StringParameter,
	groupParam *vdevices.IntParameter) (extractor, error) {
	kind := extractorKind(kindParam.Value().(int))
	pattern := patternParam.Value().(string)
	groupIdx := groupParam.Value().(int)
	switch kind {
	case ExtractorAfter:
		return newExtractorRegexp(regexp.QuoteMeta(pattern)+skipPattern+numberPattern, 1)
	case ExtractorBefore:
		return newExtractorRegexp(numberPattern+skipPattern+regexp.QuoteMeta(pattern), 1)
	case ExtractorRegexp:
		return newExtractorRegexp(pattern, groupIdx)
	case ExtractorAll:
		return newExtractorRegexp(startWSPattern+numberPattern+endWSPattern, 1)
	case ExtractorTemplate:
		return newExtractorTmpl(pattern)
	default:
		return nil, fmt.Errorf("Invalid extractor kind: %d", kind)
	}
}

func newMatcherKindParameter(id string) *vdevices.IntParameter {
	p := vdevices.NewIntParameter(id)
	p.Description().Type = itf.ParameterTypeEnum
	// align with matcherKind constants
	p.Description().ValueList = []string{"EXACT", "CONTAINS", "REGEXP"}
	p.Description().Min = 0
	p.Description().Max = 2
	p.Description().Default = 0
	return p
}

type matcherKind int

const (
	// align with function newMatcherKindParameter
	MatcherExact matcherKind = iota
	MatcherContains
	MatcherRegexp
)

type matcher interface {
	Match(payload []byte) bool
}

func newMatcher(kindParam *vdevices.IntParameter, patternParam *vdevices.StringParameter) (matcher, error) {
	kind := matcherKind(kindParam.Value().(int))
	pattern := patternParam.Value().(string)
	switch kind {
	case MatcherExact:
		return &matcherExact{pattern: pattern}, nil
	case MatcherContains:
		return &matcherContains{pattern: pattern}, nil
	case MatcherRegexp:
		regexp, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("Invalid regular expression: %s", pattern)
		}
		return &matcherRegexp{regexp: regexp}, nil
	}
	return nil, fmt.Errorf("Invalid matcher kind: %d", kind)
}

type matcherExact struct {
	pattern string
}

func (m *matcherExact) Match(payload []byte) bool {
	return string(payload) == m.pattern
}

type matcherContains struct {
	pattern string
}

func (m *matcherContains) Match(payload []byte) bool {
	return strings.Contains(string(payload), m.pattern)
}

type matcherRegexp struct {
	regexp *regexp.Regexp
}

func (m *matcherRegexp) Match(payload []byte) bool {
	return m.regexp.Match(payload)
}

func matchLevels(pattern []string, topic []string) bool {
	if len(pattern) == 0 {
		return len(topic) == 0
	}
	if len(topic) == 0 {
		return pattern[0] == "#"
	}
	if pattern[0] == "#" {
		return true
	}
	if (pattern[0] == topic[0]) || (pattern[0] == "+") {
		return matchLevels(pattern[1:], topic[1:])
	}
	return false
}

func matchTopic(pattern, topic string) bool {
	return matchLevels(strings.Split(pattern, "/"), strings.Split(topic, "/"))
}
