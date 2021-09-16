package virtdev

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

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
	c.shortSubscribedTopic = c.paramShortTopic.Value().(string)
	if c.shortSubscribedTopic != "" {
		matcher, err := newMatcher(c.paramShortMatcherKind, c.paramShortPattern)
		if err != nil {
			log.Errorf("Creation of matcher for short keypress failed: %v", err)
		} else {
			c.shortOnPublish = func(msg *message.PublishMessage) error {
				log.Debugf("Message for %s:%d short keypress received: %s, %s", c.Description().Parent,
					c.Description().Index, msg.Topic(), msg.Payload())
				if matcher.Match(msg.Payload()) {
					log.Debugf("Triggering short keypress on %s:%d", c.Description().Parent, c.Description().Index)
					c.keyChannel.PressShort()
				}
				return nil
			}
			c.mqttServer.Subscribe(c.shortSubscribedTopic, message.QosExactlyOnce, &c.shortOnPublish)
		}
	}

	c.longSubscribedTopic = c.paramLongTopic.Value().(string)
	if c.longSubscribedTopic != "" {
		matcher, err := newMatcher(c.paramLongMatcherKind, c.paramLongPattern)
		if err != nil {
			log.Errorf("Creation of matcher for long keypress failed: %v", err)
		} else {
			c.longOnPublish = func(msg *message.PublishMessage) error {
				log.Debugf("Message for %s:%d long keypress received: %s, %s", c.Description().Parent,
					c.Description().Index, msg.Topic(), msg.Payload())
				if matcher.Match(msg.Payload()) {
					log.Debugf("Triggering long keypress on %s:%d", c.Description().Parent, c.Description().Index)
					c.keyChannel.PressLong()
				}
				return nil
			}
			c.mqttServer.Subscribe(c.longSubscribedTopic, message.QosExactlyOnce, &c.longOnPublish)
		}
	}
}

func (c *mqttKeyReceiver) stop() {
	if c.shortSubscribedTopic != "" {
		c.mqttServer.Unsubscribe(c.shortSubscribedTopic, &c.shortOnPublish)
	}
	if c.longSubscribedTopic != "" {
		c.mqttServer.Unsubscribe(c.longSubscribedTopic, &c.longOnPublish)
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
	switchChannel *vdevices.SwitchChannel
	mqttServer    *mqtt.Server

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
	c.subscribedTopic = c.paramFBTopic.Value().(string)
	if c.subscribedTopic != "" {
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
				c.switchChannel.SetState(true)
			} else if offMatcher.Match(msg.Payload()) {
				log.Debugf("Turning off switch %s:%d", c.Description().Parent, c.Description().Index)
				c.switchChannel.SetState(false)
			} else {
				log.Warningf("Invalid message for switch %s:%d received: %s", c.Description().Parent,
					c.Description().Index, msg.Payload())
			}
			return nil
		}
		c.mqttServer.Subscribe(c.subscribedTopic, message.QosExactlyOnce, &c.onPublish)
	}
}

func (c *mqttSwitchFeedback) stop() {
	if c.subscribedTopic != "" {
		c.mqttServer.Unsubscribe(c.subscribedTopic, &c.onPublish)
	}
}

func (vd *VirtualDevices) addMQTTSwitchFeedback(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(mqttSwitchFeedback)

	// inititalize baseChannel
	ch.switchChannel = vdevices.NewSwitchChannel(dev)
	ch.GenericChannel = ch.switchChannel
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
	ch.switchChannel.OnSetState = func(state bool) bool {
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
	ch.switchChannel.OnDispose = ch.stop

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
			log.Errorf("Creation of extractor for analog receiver %s:%d failed: %v", err)
			return
		}
		c.onPublish = func(msg *message.PublishMessage) error {
			log.Debugf("Message for analog receiver %s:%d received: %s, %s", c.Description().Parent,
				c.Description().Index, msg.Topic(), msg.Payload())
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

func newExtractorKindParameter(id string) *vdevices.IntParameter {
	p := vdevices.NewIntParameter(id)
	p.Description().Type = itf.ParameterTypeEnum
	// align with extractorKind constants
	p.Description().ValueList = []string{"AFTER", "BEFORE", "REGEXP"}
	p.Description().Min = 0
	p.Description().Max = 2
	p.Description().Default = 0
	return p
}

type extractorKind int

const (
	// align with function newExtractorKindParameter
	ExtractorAfter extractorKind = iota
	ExtractorBefore
	ExtractorRegexp
)

type extractor struct {
	regexp   *regexp.Regexp
	groupIdx int
}

func (e *extractor) Extract(payload []byte) (float64, error) {
	groups := e.regexp.FindStringSubmatch(string(payload))
	if groups == nil {
		return 0.0, fmt.Errorf("Regexp does not match: %s", payload)
	}
	if e.groupIdx < 0 || e.groupIdx >= len(groups) {
		return 0.0, fmt.Errorf("Invalid group index: %d", e.groupIdx)
	}
	fval, err := strconv.ParseFloat(groups[e.groupIdx], 64)
	if err != nil {
		return 0.0, fmt.Errorf("Not a valid number literal: %s", groups[e.groupIdx])
	}
	return fval, nil
}

const (
	numberPattern = `([+-]?(\d+(\.\d*)?|\.\d+))`
	skipPattern   = `[^\d.+-]*`
)

func newExtractor(kindParam *vdevices.IntParameter, patternParam *vdevices.StringParameter,
	groupParam *vdevices.IntParameter) (*extractor, error) {
	kind := extractorKind(kindParam.Value().(int))
	pattern := patternParam.Value().(string)
	groupIdx := groupParam.Value().(int)
	switch kind {
	case ExtractorAfter:
		pattern = regexp.QuoteMeta(pattern) + skipPattern + numberPattern
		groupIdx = 1
	case ExtractorBefore:
		pattern = numberPattern + skipPattern + regexp.QuoteMeta(pattern)
		groupIdx = 1
	case ExtractorRegexp:
		// nothing change
	default:
		return nil, fmt.Errorf("Invalid extractor kind: %d", kind)
	}
	log.Tracef("Creating extractor with regular expression %s and group %d", pattern, groupIdx)
	regexp, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("Invalid regular expression: %s", pattern)
	}
	return &extractor{regexp: regexp, groupIdx: groupIdx}, nil
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
