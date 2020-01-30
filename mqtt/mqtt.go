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

var log = logging.Get("mqtt-broker")

// Broker for MQTT. Broker implements itf.Receiver.
type Broker struct {
	// Binding address for serving MQTT.
	Addr string

	// When an error happens while serving (e.g. binding of port fails), this
	// error is sent to the channel ServeErr.
	ServeErr chan<- error

	// Next handler for XMLRPC events.
	Next itf.Receiver

	server *service.Server
	done   chan struct{}
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
}

// Stop stops the MQTT broker.
func (b *Broker) Stop() {
	log.Debugf("Stopping MQTT broker")
	_ = b.server.Close()
	// wait for shutdown (must match number of listeners/servers)
	<-b.done
}

// PublishPV publishes a PV.
func (b *Broker) PublishPV(topic string, pv veap.PV, retain bool) error {
	// build payload
	t := pv.Time
	if t.IsZero() {
		t = time.Now()
	}
	wpv := wirePV{
		Time:  t.UnixNano() / 1000000,
		Value: pv.Value,
		State: pv.State,
	}
	pl, err := json.Marshal(wpv)
	if err != nil {
		return fmt.Errorf("Conversion of PV to JSON failed: %v", err)
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
	topic := fmt.Sprintf("device/%s/%s/%s", dev, ch, valueKey)

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
