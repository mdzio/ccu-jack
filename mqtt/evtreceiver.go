package mqtt

import (
	"fmt"
	"strings"
	"time"

	"github.com/mdzio/go-lib/hmccu/itf"
	"github.com/mdzio/go-lib/veap"
)

// EventReceiver accepts XMLRPC events, publishes them to the broker and then
// forwards them to the next receiver.
type EventReceiver struct {
	// Broker for publishing events.
	Broker *Broker

	// Next handler for XML-RPC events.
	Next itf.Receiver
}

// Event implements itf.Receiver.
func (r *EventReceiver) Event(interfaceID, address, valueKey string, value interface{}) error {
	// publish event
	if err := r.publishEvent(interfaceID, address, valueKey, value); err != nil {
		log.Errorf("Publish of event failed: %v", err)
	}
	// forward event
	return r.Next.Event(interfaceID, address, valueKey, value)
}

// NewDevices implements itf.Receiver.
func (r *EventReceiver) NewDevices(interfaceID string, devDescriptions []*itf.DeviceDescription) error {
	// only forward
	return r.Next.NewDevices(interfaceID, devDescriptions)
}

// DeleteDevices implements itf.Receiver.
func (r *EventReceiver) DeleteDevices(interfaceID string, addresses []string) error {
	// only forward
	return r.Next.DeleteDevices(interfaceID, addresses)
}

// UpdateDevice implements itf.Receiver.
func (r *EventReceiver) UpdateDevice(interfaceID, address string, hint int) error {
	// only forward
	return r.Next.UpdateDevice(interfaceID, address, hint)
}

// ReplaceDevice implements itf.Receiver.
func (r *EventReceiver) ReplaceDevice(interfaceID, oldDeviceAddress, newDeviceAddress string) error {
	// only forward
	return r.Next.ReplaceDevice(interfaceID, oldDeviceAddress, newDeviceAddress)
}

// ReaddedDevice implements itf.Receiver.
func (r *EventReceiver) ReaddedDevice(interfaceID string, deletedAddresses []string) error {
	// only forward
	return r.Next.ReaddedDevice(interfaceID, deletedAddresses)
}

func (r *EventReceiver) publishEvent(interfaceID, address, valueKey string, value interface{}) error {
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
	if err := r.Broker.PublishPV(topic, pv, retain); err != nil {
		return err
	}
	return nil
}
