package virtdev

import (
	"github.com/mdzio/ccu-jack/mqtt"
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-mqtt/service"
)

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
	topic := c.paramTopic.Value().(string)
	if topic != "" {
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
				// set status to unknown
				c.analogChannel.SetVoltageStatus(1)
				return nil
			}
			c.analogChannel.SetVoltage(value)
			// set normal status
			c.analogChannel.SetVoltageStatus(0)
			return nil
		}
		if err := c.mqttServer.Subscribe(topic, message.QosExactlyOnce, &c.onPublish); err != nil {
			log.Errorf("Subscribe failed on topic %s: %v", topic, err)
		} else {
			c.subscribedTopic = topic
		}
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
