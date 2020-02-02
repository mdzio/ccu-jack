package mqtt

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mdzio/go-lib/hmccu/itf"
	"github.com/mdzio/go-lib/veap"
	"github.com/mdzio/go-logging"
	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-mqtt/service"
)

const (
	deviceStatusTopic = "device/status"
	deviceSetTopic    = "device/set"
	// path prefix for device data points in the VEAP address space
	deviceServicePath = "/device"

	sysVarStatusTopic = "sysvar/status"
	sysVarSetTopic    = "sysvar/set"
	sysVarGetTopic    = "sysvar/get"
	// path prefix for system variable data points in the VEAP address space
	sysVarServicePath = "/sysvar"
)

var log = logging.Get("mqtt-broker")

// Broker for MQTT. Broker implements itf.Receiver to receive XML-RPC events.
type Broker struct {
	// Binding address for serving MQTT.
	Addr string

	// When an error happens while serving (e.g. binding of port fails), this
	// error is sent to the channel ServeErr.
	ServeErr chan<- error

	// Next handler for XML-RPC events.
	Next itf.Receiver

	// Service is used to set device data points and system variables.
	Service veap.Service

	server *service.Server
	done   chan struct{}

	onSetDevice service.OnPublishFunc
	onSetSysVar service.OnPublishFunc
	onGetSysVar service.OnPublishFunc
}

// Start starts the MQTT broker.
func (b *Broker) Start() {
	b.server = &service.Server{}

	// capacity must match the number of listeners/servers
	b.done = make(chan struct{}, 1)

	// start listeners
	go func() {
		log.Infof("Starting MQTT listener on address %s", b.Addr)
		err := b.server.ListenAndServe(b.Addr)
		// signal server is down (must not block)
		b.done <- struct{}{}
		// check for error
		if err != nil {
			// signal error while serving (block does not harm)
			if b.ServeErr != nil {
				b.ServeErr <- fmt.Errorf("Running MQTT server failed: %v", err)
			}
		}
	}()

	// subscribe set device topics
	b.onSetDevice = func(msg *message.PublishMessage) error {
		log.Tracef("Set device message received: %s: %s", msg.Topic(), msg.Payload())

		// parse PV
		pv, err := wireToPV(msg.Payload())
		if err != nil {
			return err
		}

		// map topic to VEAP address
		topic := string(msg.Topic())
		if !strings.HasPrefix(topic, deviceSetTopic+"/") {
			return fmt.Errorf("Unexpected topic: %s", topic)
		}

		// path with leading /
		path := topic[len(deviceSetTopic):]

		// use VEAP service to write PV
		if err = b.Service.WritePV(deviceServicePath+path, pv); err != nil {
			return err
		}
		return nil
	}
	b.server.Subscribe(deviceSetTopic+"/+/+/+", message.QosExactlyOnce, &b.onSetDevice)

	// subscribe set sysvar topics
	b.onSetSysVar = func(msg *message.PublishMessage) error {
		log.Tracef("Set sysvar message received: %s: %s", msg.Topic(), msg.Payload())

		// parse PV
		pv, err := wireToPV(msg.Payload())
		if err != nil {
			return err
		}

		// map topic to VEAP address
		topic := string(msg.Topic())
		if !strings.HasPrefix(topic, sysVarSetTopic+"/") {
			return fmt.Errorf("Unexpected topic: %s", topic)
		}

		// path with leading /
		path := topic[len(sysVarSetTopic):]

		// use VEAP service to write PV
		if err = b.Service.WritePV(sysVarServicePath+path, pv); err != nil {
			return err
		}
		return nil
	}
	b.server.Subscribe(sysVarSetTopic+"/+", message.QosExactlyOnce, &b.onSetSysVar)

	// subscribe get sysvar topics
	b.onGetSysVar = func(msg *message.PublishMessage) error {
		log.Tracef("Get sysvar message received: %s: %s", msg.Topic(), msg.Payload())

		// map topic to VEAP address
		topic := string(msg.Topic())
		if !strings.HasPrefix(topic, sysVarGetTopic+"/") {
			return fmt.Errorf("Unexpected topic: %s", topic)
		}

		// path with leading /
		path := topic[len(sysVarGetTopic):]

		// use VEAP service to read PV
		pv, err := b.Service.ReadPV(sysVarServicePath + path)
		if err != nil {
			return err
		}

		// publish PV
		return b.PublishPV(sysVarStatusTopic+path, pv, true)
	}
	b.server.Subscribe(sysVarGetTopic+"/+", message.QosExactlyOnce, &b.onGetSysVar)
}

// Stop stops the MQTT broker.
func (b *Broker) Stop() {
	log.Debugf("Stopping MQTT broker")
	b.server.Unsubscribe(deviceSetTopic+"/+/+/+", &b.onSetDevice)
	b.server.Unsubscribe(sysVarSetTopic+"/+", &b.onSetSysVar)
	b.server.Unsubscribe(sysVarGetTopic+"/+", &b.onGetSysVar)
	_ = b.server.Close()

	// wait for shutdown (must match number of listeners/servers)
	<-b.done
}

// PublishPV publishes a PV.
func (b *Broker) PublishPV(topic string, pv veap.PV, retain bool) error {
	pl, err := pvToWire(pv)
	if err != nil {
		return err
	}
	if err := b.Publish(topic, pl, retain); err != nil {
		return err
	}
	return nil
}

// Publish publishes a generic payload.
func (b *Broker) Publish(topic string, payload []byte, retain bool) error {
	log.Tracef("Publishing %s: %s", topic, string(payload))
	pm := message.NewPublishMessage()
	if err := pm.SetTopic([]byte(topic)); err != nil {
		return fmt.Errorf("Invalid topic: %v", err)
	}
	if err := pm.SetQoS(0); err != nil {
		return fmt.Errorf("Invalid QoS: %v", err)
	}
	pm.SetRetain(retain)
	pm.SetPayload(payload)
	if err := b.server.Publish(pm, nil); err != nil {
		return fmt.Errorf("Publish failed: %v", err)
	}
	return nil
}

// Event implements itf.Receiver.
func (b *Broker) Event(interfaceID, address, valueKey string, value interface{}) error {
	// publish event
	if err := b.publishEvent(interfaceID, address, valueKey, value); err != nil {
		log.Errorf("Publish of event failed: %v", err)
	}
	// forward event
	return b.Next.Event(interfaceID, address, valueKey, value)
}

// NewDevices implements itf.Receiver.
func (b *Broker) NewDevices(interfaceID string, devDescriptions []*itf.DeviceDescription) error {
	// only forward
	return b.Next.NewDevices(interfaceID, devDescriptions)
}

// DeleteDevices implements itf.Receiver.
func (b *Broker) DeleteDevices(interfaceID string, addresses []string) error {
	// only forward
	return b.Next.DeleteDevices(interfaceID, addresses)
}

// UpdateDevice implements itf.Receiver.
func (b *Broker) UpdateDevice(interfaceID, address string, hint int) error {
	// only forward
	return b.Next.UpdateDevice(interfaceID, address, hint)
}

// ReplaceDevice implements itf.Receiver.
func (b *Broker) ReplaceDevice(interfaceID, oldDeviceAddress, newDeviceAddress string) error {
	// only forward
	return b.Next.ReplaceDevice(interfaceID, oldDeviceAddress, newDeviceAddress)
}

// ReaddedDevice implements itf.Receiver.
func (b *Broker) ReaddedDevice(interfaceID string, deletedAddresses []string) error {
	// only forward
	return b.Next.ReaddedDevice(interfaceID, deletedAddresses)
}

type wirePV struct {
	Time  int64       `json:"ts"`
	Value interface{} `json:"v"`
	State veap.State  `json:"s"`
}

func (b *Broker) publishEvent(interfaceID, address, valueKey string, value interface{}) error {
	// separate device and channel
	var dev, ch string
	var p int
	if p = strings.IndexRune(address, ':'); p == -1 {
		return fmt.Errorf("Unexpected event from a device: %s", address)
	}
	dev = address[0:p]
	ch = address[p+1:]

	// build topic
	topic := fmt.Sprintf("%s/%s/%s/%s", deviceStatusTopic, dev, ch, valueKey)

	// build PV
	pv := veap.PV{
		Time:  time.Now(),
		Value: value,
		State: veap.StateGood,
	}

	// retain all except actions
	retain := false
	if valueKey != "INSTALL_TEST" && !strings.HasPrefix(valueKey, "PRESS_") {
		retain = true
	}

	// publish
	if err := b.PublishPV(topic, pv, retain); err != nil {
		return err
	}
	return nil
}

func wireToPV(payload []byte) (veap.PV, error) {
	// convert JSON to PV
	var w wirePV
	err := json.Unmarshal(payload, &w)
	if err != nil {
		return veap.PV{}, fmt.Errorf("Conversion of JSON to PV failed: %v", err)
	}
	if w.Value == nil {
		return veap.PV{}, fmt.Errorf("Conversion of JSON to PV failed: No value property (v) found")
	}

	// if no timestamp is provided, use current time
	var ts time.Time
	if w.Time == 0 {
		ts = time.Now()
	} else {
		ts = time.Unix(0, w.Time*1000000)
	}

	// if no state is provided, state is implicit GOOD
	return veap.PV{
		Time:  ts,
		Value: w.Value,
		State: w.State,
	}, nil
}

func pvToWire(pv veap.PV) ([]byte, error) {
	var w wirePV
	w.Time = pv.Time.UnixNano() / 1000000
	w.Value = pv.Value
	w.State = pv.State
	pl, err := json.Marshal(w)
	if err != nil {
		return nil, fmt.Errorf("Conversion of PV to JSON failed: %v", err)
	}
	return pl, nil
}
