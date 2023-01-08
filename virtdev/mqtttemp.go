package virtdev

import (
	"github.com/mdzio/ccu-jack/mqtt"
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-mqtt/service"
)

type mqttTemperature struct {
	baseChannel
	temperatureChannel *vdevices.TemperatureChannel
	mqttServer         *mqtt.Server

	temperatureTopic           *vdevices.StringParameter
	temperaturePattern         *vdevices.StringParameter
	temperatureExtractorKind   *vdevices.IntParameter
	temperatureRegexpGroup     *vdevices.IntParameter
	temperatureSubscribedTopic string
	temperatureOnPublish       service.OnPublishFunc

	humidityTopic           *vdevices.StringParameter
	humidityPattern         *vdevices.StringParameter
	humidityExtractorKind   *vdevices.IntParameter
	humidityRegexpGroup     *vdevices.IntParameter
	humiditySubscribedTopic string
	humidityOnPublish       service.OnPublishFunc
}

func (c *mqttTemperature) start() {
	temperatureTopic := c.temperatureTopic.Value().(string)
	if temperatureTopic != "" {
		extractor, err := newExtractor(c.temperatureExtractorKind, c.temperaturePattern, c.temperatureRegexpGroup)
		if err != nil {
			log.Errorf("Creation of value extractor for temperature %s:%d failed: %v", c.Description().Parent,
				c.Description().Index, err)
			return
		}
		c.temperatureOnPublish = func(msg *message.PublishMessage) error {
			log.Debugf("Message for temperature %s:%d received: %s, %s", c.Description().Parent,
				c.Description().Index, msg.Topic(), msg.Payload())
			// lock channel while modifying parameters
			c.Lock()
			defer c.Unlock()
			value, err := extractor.Extract(msg.Payload())
			if err != nil {
				log.Warningf("Extraction of value for temperature %s:%d failed: %v", c.Description().Parent,
					c.Description().Index, err)
				// set status to unknown
				c.temperatureChannel.SetTemperatureStatus(1)
				return nil
			}
			c.temperatureChannel.SetTemperature(value)
			// set normal status
			c.temperatureChannel.SetTemperatureStatus(0)
			return nil
		}
		if err := c.mqttServer.Subscribe(temperatureTopic, message.QosExactlyOnce, &c.temperatureOnPublish); err != nil {
			log.Errorf("Subscribe failed on topic %s: %v", temperatureTopic, err)
		} else {
			c.temperatureSubscribedTopic = temperatureTopic
		}
	}

	humidityTopic := c.humidityTopic.Value().(string)
	if humidityTopic != "" {
		extractor, err := newExtractor(c.humidityExtractorKind, c.humidityPattern, c.humidityRegexpGroup)
		if err != nil {
			log.Errorf("Creation of value extractor for humidity %s:%d failed: %v", c.Description().Parent,
				c.Description().Index, err)
			return
		}
		c.humidityOnPublish = func(msg *message.PublishMessage) error {
			log.Debugf("Message for humidity %s:%d received: %s, %s", c.Description().Parent,
				c.Description().Index, msg.Topic(), msg.Payload())
			// lock channel while modifying parameters
			c.Lock()
			defer c.Unlock()
			value, err := extractor.Extract(msg.Payload())
			if err != nil {
				log.Warningf("Extraction of value for humidity %s:%d failed: %v", c.Description().Parent,
					c.Description().Index, err)
				// set status to unknown
				c.temperatureChannel.SetHumidityStatus(1)
				return nil
			}
			c.temperatureChannel.SetHumidity(int(value))
			// set normal status
			c.temperatureChannel.SetHumidityStatus(0)
			return nil
		}
		if err := c.mqttServer.Subscribe(humidityTopic, message.QosExactlyOnce, &c.humidityOnPublish); err != nil {
			log.Errorf("Subscribe failed on topic %s: %v", humidityTopic, err)
		} else {
			c.humiditySubscribedTopic = humidityTopic
		}
	}
}

func (c *mqttTemperature) stop() {
	if c.temperatureSubscribedTopic != "" {
		c.mqttServer.Unsubscribe(c.temperatureSubscribedTopic, &c.temperatureOnPublish)
		c.temperatureSubscribedTopic = ""
	}

	if c.humiditySubscribedTopic != "" {
		c.mqttServer.Unsubscribe(c.humiditySubscribedTopic, &c.humidityOnPublish)
		c.humiditySubscribedTopic = ""
	}
}

func (vd *VirtualDevices) addMQTTTemperature(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(mqttTemperature)

	// inititalize baseChannel
	ch.temperatureChannel = vdevices.NewTemperatureChannel(dev)
	ch.GenericChannel = ch.temperatureChannel
	ch.store = vd.Store
	ch.mqttServer = vd.MQTTServer

	// TEMPERATURE_TOPIC
	ch.temperatureTopic = vdevices.NewStringParameter("TEMPERATURE_TOPIC")
	ch.AddMasterParam(ch.temperatureTopic)

	// TEMPERATURE_PATTERN
	ch.temperaturePattern = vdevices.NewStringParameter("TEMPERATURE_PATTERN")
	ch.AddMasterParam(ch.temperaturePattern)

	// TEMPERATURE_EXTRACTOR
	ch.temperatureExtractorKind = newExtractorKindParameter("TEMPERATURE_EXTRACTOR")
	ch.AddMasterParam(ch.temperatureExtractorKind)

	// TEMPERATURE_REGEXP_GROUP
	ch.temperatureRegexpGroup = vdevices.NewIntParameter("TEMPERATURE_REGEXP_GROUP")
	ch.temperatureRegexpGroup.Description().Min = 0
	ch.temperatureRegexpGroup.Description().Max = 100
	ch.temperatureRegexpGroup.Description().Default = 0
	ch.AddMasterParam(ch.temperatureRegexpGroup)

	// HUMIDITY_TOPIC
	ch.humidityTopic = vdevices.NewStringParameter("HUMIDITY_TOPIC")
	ch.AddMasterParam(ch.humidityTopic)

	// HUMIDITY_PATTERN
	ch.humidityPattern = vdevices.NewStringParameter("HUMIDITY_PATTERN")
	ch.AddMasterParam(ch.humidityPattern)

	// HUMIDITY_EXTRACTOR
	ch.humidityExtractorKind = newExtractorKindParameter("HUMIDITY_EXTRACTOR")
	ch.AddMasterParam(ch.humidityExtractorKind)

	// HUMIDITY_REGEXP_GROUP
	ch.humidityRegexpGroup = vdevices.NewIntParameter("HUMIDITY_REGEXP_GROUP")
	ch.humidityRegexpGroup.Description().Min = 0
	ch.humidityRegexpGroup.Description().Max = 100
	ch.humidityRegexpGroup.Description().Default = 0
	ch.AddMasterParam(ch.humidityRegexpGroup)

	// clean up
	ch.temperatureChannel.OnDispose = ch.stop

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
