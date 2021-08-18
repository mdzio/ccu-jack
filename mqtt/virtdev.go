package mqtt

import (
	"fmt"
	"strings"
	"time"

	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-veap"
)

// VirtDevEventReceiver publishes value change events from virtual devices to
// the MQTT server.
type VirtDevEventReceiver struct {
	// Server for publishing events.
	Server *Server
}

// PublishEvent implements vdevices.EventPublisher.
func (t *VirtDevEventReceiver) PublishEvent(address, valueKey string, value interface{}) {
	// separate device and channel
	var dev, ch string
	var p int
	if p = strings.IndexRune(address, ':'); p == -1 {
		log.Errorf("Unexpected event from a virtual device: %s", address)
		return
	}
	dev = address[0:p]
	ch = address[p+1:]

	// build topic
	topic := fmt.Sprintf("%s/%s/%s/%s", virtDevStatusTopic, dev, ch, valueKey)

	// build PV
	pv := veap.PV{
		Time:  time.Now(),
		Value: value,
		State: veap.StateGood,
	}

	// select qos and retain
	var qos byte
	var retain bool
	if valueKey != "INSTALL_TEST" && !strings.HasPrefix(valueKey, "PRESS_") {
		retain = true
		qos = message.QosAtLeastOnce
	} else {
		retain = false
		qos = message.QosExactlyOnce
	}

	// publish
	if err := t.Server.PublishPV(topic, pv, qos, retain); err != nil {
		log.Error(err)
		return
	}
}
