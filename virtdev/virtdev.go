package virtdev

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/mdzio/ccu-jack/mqtt"
	"github.com/mdzio/ccu-jack/rtcfg"
	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-hmccu/itf/xmlrpc"
	"github.com/mdzio/go-logging"
)

const (
	// path to the file InterfacesList.xml on the CCU3
	itfListFile = "/etc/config/InterfacesList.xml"

	// Use /RPC3 for calls from ReGaHss. RPC2 is already used for callbacks from
	// interface processes (e.g. BidCos-RF).
	xmlrpcPath = "/RPC3"

	// Interface ID of the CCU-Jack
	InterfaceID = "CCU-Jack"
)

var log = logging.Get("virt-dev")

type VirtualDevices struct {
	// Store with the configuration must be set before calling Start.
	Store *rtcfg.Store
	// EventPublisher for receiving value change events, must be set before
	// calling Start.
	EventPublisher vdevices.EventPublisher
	// MQTT server for MQTT virtual devices, must be set before calling Start.
	MQTTServer *mqtt.Server

	// Container for virtual devices.
	Devices *vdevices.Container

	deviceHandler *vdevices.Handler
	// actually used EventPublisher by devices
	eventPublisher vdevices.EventPublisher
}

func (vd *VirtualDevices) Start() {
	log.Info("Starting virtual devices")

	// lock config
	vd.Store.RLock()
	defer vd.Store.RUnlock()
	cfg := vd.Store.Config

	// add device layer to InterfacesList.xml
	err := vdevices.AddToInterfaceList(
		itfListFile,
		itfListFile,
		InterfaceID,
		"xmlrpc://"+cfg.CCU.Address+":"+strconv.Itoa(cfg.HTTP.Port)+xmlrpcPath,
		InterfaceID,
	)
	if err != nil {
		log.Errorf("Adding CCU-Jack device layer to CCU interface list failed: %v", err)
	}

	// virtual device container
	vd.Devices = vdevices.NewContainer()

	// virtual devices handler
	vd.deviceHandler = vdevices.NewHandler(cfg.Host.Address, vd.Devices, func(address string) {
		// a device is deleted by the CCU. delete it also in the configuration.
		vd.Store.Lock()
		defer vd.Store.Unlock()
		if _, found := vd.Store.Config.VirtualDevices.Devices[address]; !found {
			log.Errorf("Unknown device deleted by CCU: %s", address)
		} else {
			log.Infof("Removing virtual device: %s", address)
			delete(vd.Store.Config.VirtualDevices.Devices, address)
		}
	})
	vd.Devices.Synchronizer = vd.deviceHandler

	// setup event publishing
	vd.eventPublisher = &vdevices.TeeEventPublisher{
		// sent to CCU
		First: vd.deviceHandler,
		// sent to MQTT
		Second: vd.EventPublisher,
	}

	// HM RPC dispatcher for device layer
	dispatcher := itf.NewDispatcher()
	dispatcher.AddDeviceLayer(vd.deviceHandler)

	// register XML-RPC handler at the HTTP server
	httpHandler := &xmlrpc.Handler{Dispatcher: dispatcher}
	http.Handle(xmlrpcPath, httpHandler)

	// add configured devices
	vd.SynchronizeDevices()
}

func (vd *VirtualDevices) Stop() {
	// only stop, if successfully started
	if vd.deviceHandler != nil {
		log.Debug("Stopping virtual device handler")
		vd.deviceHandler.Close()
		log.Debug("Shutting down virtual devices")
		vd.Devices.Dispose()
	}
}

// SynchronizeDevices updates the virtual device container based on the
// configuration. The configuration (field Store) must be locked for reading.
func (vd *VirtualDevices) SynchronizeDevices() {
	// devices in configuration
	devcfgs := vd.Store.Config.VirtualDevices.Devices

	// delete non existing devices
	for _, dev := range vd.Devices.Devices() {
		// exists device in config?
		_, exist := devcfgs[dev.Description().Address]
		if !exist {
			// if not, remove it from container
			log.Infof("Removing virtual device: %s", dev.Description().Address)
			if err := vd.Devices.RemoveDevice(dev.Description().Address); err != nil {
				log.Errorf("Remove of virtual device %s failed: %v", dev.Description().Address, err)
			}
		}
	}

	// add new devices
	for addr, devcfg := range devcfgs {
		// exists device in runtime?
		if _, err := vd.Devices.Device(addr); err != nil {
			// if not, create it
			log.Infof("Creating virtual device %s with %d channel(s)", devcfg.Address, len(devcfg.Channels))
			if err := vd.createDevice(devcfg); err != nil {
				log.Errorf("Creation of virtual device %s failed: %v", devcfg.Address, err)
			}
		}
	}
}

func (vd *VirtualDevices) createDevice(devcfg *rtcfg.Device) error {
	// create device
	dev := vdevices.NewDevice(devcfg.Address, devcfg.HMType, vd.eventPublisher)
	// add maintenance channel
	vdevices.NewMaintenanceChannel(dev)

	// create channels
	for _, chcfg := range devcfg.Channels {
		switch chcfg.Kind {

		case rtcfg.ChannelKey:
			ch := vdevices.NewKeyChannel(dev)
			log.Debugf("Created static key channel: %s", ch.Description().Address)
		case rtcfg.ChannelSwitch:
			ch := vdevices.NewSwitchChannel(dev)
			log.Debugf("Created static switch channel: %s", ch.Description().Address)
		case rtcfg.ChannelAnalog:
			ch := vdevices.NewAnalogInputChannel(dev)
			log.Debugf("Created static analog input channel: %s", ch.Description().Address)
		case rtcfg.ChannelDoorSensor:
			ch := vdevices.NewDoorSensorChannel(dev)
			log.Debugf("Created static door sensor channel: %s", ch.Description().Address)
		case rtcfg.ChannelDimmer:
			ch := vd.addStaticDimmer(dev)
			log.Debugf("Created static dimmer channel: %s", ch.Description().Address)
		case rtcfg.ChannelTemperature:
			ch := vdevices.NewTemperatureChannel(dev)
			log.Debugf("Created static temperature channel: %s", ch.Description().Address)
		case rtcfg.ChannelPowerMeter:
			ch := vdevices.NewPowerMeterChannel(dev)
			log.Debugf("Created static power meter channel: %s", ch.Description().Address)

		case rtcfg.ChannelMQTTKeySender:
			ch := vd.addMQTTKeySender(dev)
			log.Debugf("Created MQTT key sender channel: %s", ch.Description().Address)
		case rtcfg.ChannelMQTTKeyReceiver:
			ch := vd.addMQTTKeyReceiver(dev)
			log.Debugf("Created MQTT key receiver channel: %s", ch.Description().Address)
		case rtcfg.ChannelMQTTSwitch:
			ch := vd.addMQTTSwitch(dev)
			log.Debugf("Created MQTT switch channel: %s", ch.Description().Address)
		case rtcfg.ChannelMQTTSwitchFeedback:
			ch := vd.addMQTTSwitchFeedback(dev)
			log.Debugf("Created MQTT switch with feedback channel: %s", ch.Description().Address)
		case rtcfg.ChannelMQTTAnalogReceiver:
			ch := vd.addMQTTAnalogReceiver(dev)
			log.Debugf("Created MQTT analog receiver channel: %s", ch.Description().Address)
		case rtcfg.ChannelMQTTDoorSensor:
			ch := vd.addMQTTDoorSensor(dev)
			log.Debugf("Created MQTT door sensor channel: %s", ch.Description().Address)
		case rtcfg.ChannelMQTTDimmer:
			ch := vd.addMQTTDimmer(dev)
			log.Debugf("Created MQTT dimmer channel: %s", ch.Description().Address)
		case rtcfg.ChannelMQTTTemperature:
			ch := vd.addMQTTTemperature(dev)
			log.Debugf("Created MQTT temperature channel: %s", ch.Description().Address)
		case rtcfg.ChannelMQTTPowerMeter:
			ch := vd.addMQTTPowerMeter(dev)
			log.Debugf("Created MQTT power meter channel: %s", ch.Description().Address)

		default:
			return fmt.Errorf("Unsupported kind of channel in device %s: %v", devcfg.Address, chcfg.Kind)
		}
	}

	if err := vd.Devices.AddDevice(dev); err != nil {
		return fmt.Errorf("Registration failed: %v", err)
	}
	return nil
}
