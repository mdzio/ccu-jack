package mqtt

import (
	"fmt"
	"strings"
	"time"

	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-mqtt/service"
	"github.com/mdzio/go-veap"
)

const (
	// topic prefixes for CCU devices
	deviceStatusTopic = "device/status"
	deviceSetTopic    = "device/set"
	// path prefix for device data points in the VEAP address space
	deviceVeapPath = "/device"

	// topic prefix for system variables
	sysVarTopic = "sysvar"
	// path prefix for system variable data points in the VEAP address space
	sysVarVeapPath = "/sysvar"
	// delay time for reading back
	sysVarReadBackDur = 300 * time.Millisecond

	// topic prefix for programs
	prgTopic = "program"
	// path prefix for programs in the VEAP address space
	prgVeapPath = "/program"

	// topic prefixes for virtual devices
	virtDevStatusTopic = "virtdev/status"
	virtDevSetTopic    = "virtdev/set"
	// path prefix for virtual devices in the VEAP address space
	virtDevVeapPath = "/virtdev"
)

// Bridge connects MQTT and VEAP.
type Bridge struct {
	// MQTT server
	Server *Server

	// Service is used to write device data points and read/write system variables.
	Service veap.Service

	onSetDevice service.OnPublishFunc

	sysVarAdapter *vadapter
	prgAdapter    *vadapter
}

// Start starts the MQTT/VEAP-Bridge.
func (b *Bridge) Start() {
	// subscribe set device topics
	b.onSetDevice = func(msg *message.PublishMessage) error {
		log.Tracef("Set device message received: %s: %s", msg.Topic(), msg.Payload())

		// parse PV
		pv, err := wireToPV(msg.Payload())
		if err != nil {
			return err
		}

		// map topic to VEAP address
		var path string
		topic := string(msg.Topic())
		if strings.HasPrefix(topic, deviceSetTopic+"/") {
			path = deviceVeapPath + topic[len(deviceSetTopic):]
		} else if strings.HasPrefix(topic, virtDevSetTopic+"/") {
			path = virtDevVeapPath + topic[len(virtDevSetTopic):]
		} else {
			return fmt.Errorf("Unexpected topic: %s", topic)
		}

		// use VEAP service to write PV
		if err = b.Service.WritePV(path, pv); err != nil {
			return err
		}
		return nil
	}
	b.Server.Subscribe(deviceSetTopic+"/+/+/+", message.QosExactlyOnce, &b.onSetDevice)
	b.Server.Subscribe(virtDevSetTopic+"/+/+/+", message.QosExactlyOnce, &b.onSetDevice)

	// adapt VEAP system variables
	b.sysVarAdapter = &vadapter{
		mqttTopic:   sysVarTopic,
		veapPath:    sysVarVeapPath,
		readBackDur: sysVarReadBackDur,
		mqttServer:  b.Server,
		veapService: b.Service,
	}
	b.sysVarAdapter.start()

	// adapt VEAP programs
	b.prgAdapter = &vadapter{
		mqttTopic:   prgTopic,
		veapPath:    prgVeapPath,
		mqttServer:  b.Server,
		veapService: b.Service,
	}
	b.prgAdapter.start()
}

// Stop stops the MQTT/VEAP-Bridge.
func (b *Bridge) Stop() {
	// stop adapter
	b.prgAdapter.stop()
	b.sysVarAdapter.stop()

	b.Server.Unsubscribe(virtDevSetTopic+"/+/+/+", &b.onSetDevice)
	b.Server.Unsubscribe(deviceSetTopic+"/+/+/+", &b.onSetDevice)
}
