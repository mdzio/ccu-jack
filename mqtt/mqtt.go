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

// Event implements itf.Receiver.
func (b *Broker) Event(interfaceID, address, valueKey string, value interface{}) error {
	// publish event
	b.publishEvent(interfaceID, address, valueKey, value)
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

type payloadPV struct {
	Time  int64       `json:"ts"`
	Value interface{} `json:"v"`
	State veap.State  `json:"s"`
}

func (b *Broker) publishEvent(interfaceID, address, valueKey string, value interface{}) {
	// build payload
	pv := payloadPV{
		Time:  time.Now().UnixNano() / 1000000,
		Value: value,
		State: veap.StateGood,
	}
	pl, err := json.Marshal(pv)
	if err != nil {
		log.Error("Conversion of PV to JSON failed: %v", err)
		return
	}

	// separate device and channel
	var dev, ch string
	var p int
	if p = strings.IndexRune(address, ':'); p == -1 {
		log.Warning("Device should not send event: ", address)
		return
	}
	dev = address[0:p]
	ch = address[p+1:]

	// build topic
	topic := fmt.Sprintf("device/%s/%s/%s", dev, ch, valueKey)

	// publish message
	log.Tracef("Publishing %s: %s", topic, string(pl))
	pm := message.NewPublishMessage()
	if err := pm.SetTopic([]byte(topic)); err != nil {
		log.Error("Invalid topic: %v", err)
		return
	}
	if err := pm.SetQoS(0); err != nil {
		log.Error("Invalid QoS: %v", err)
		return
	}
	if valueKey == "INSTALL_TEST" || strings.HasPrefix(valueKey, "PRESS_") {
		// do not retain actions
		pm.SetRetain(false)
	} else {
		pm.SetRetain(true)
	}
	pm.SetPayload(pl)
	if err := b.server.Publish(pm, nil); err != nil {
		log.Error("Publish failed: %v", err)
		return
	}
}
