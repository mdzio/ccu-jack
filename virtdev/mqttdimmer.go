package virtdev

import (
	"bytes"
	"text/template"

	"github.com/mdzio/ccu-jack/mqtt"
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-mqtt/service"
)

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
