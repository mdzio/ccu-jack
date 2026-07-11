package virtdev

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

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
	// Use internal ports
	UseInternalPorts bool
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

	// add device layer to InterfacesList.xml only if not already present
	// (prevents duplicate entries on every restart)
	itfContent, readErr := os.ReadFile(itfListFile)
	if readErr != nil || !strings.Contains(string(itfContent), "<name>"+InterfaceID+"</name>") {
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
	} else {
		log.Debug("CCU-Jack already present in InterfacesList.xml, skipping insert")
	}

	// virtual device container
	vd.Devices = vdevices.NewContainer()

	// virtual devices handler
	vd.deviceHandler = vdevices.NewHandler(cfg.CCU.Address, vd.UseInternalPorts, vd.Devices, func(address string) {
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

	// Override getParamsetId: go-hmccu's dispatcher always returns an empty
	// string. A native HmIP device instead returns an identifier of the form
	// "<hmtype>_<channel-index>_<paramset>" (e.g. "hmip-broll_7_master"), which
	// the CCU WebUI uses to locate a matching config easymode. Provide the same
	// identifier so the virtual device answers faithfully like the real one.
	dispatcher.HandleFunc("getParamsetId", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		address := q.Idx(0).String()
		paramsetKey := q.Idx(1).String()
		if q.Err() != nil {
			return &xmlrpc.Value{}, nil
		}
		deviceAddr, channelAddr := itf.SplitAddress(address)
		dev, err := vd.Devices.Device(deviceAddr)
		if err != nil {
			return &xmlrpc.Value{}, nil
		}
		hmType := strings.ToLower(dev.Description().Type)
		key := strings.ToLower(paramsetKey)
		// Only channel paramsets follow the verified native format.
		if channelAddr == "" {
			return &xmlrpc.Value{}, nil
		}
		return xmlrpc.NewString(fmt.Sprintf("%s_%s_%s", hmType, channelAddr, key)), nil
	})

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
	if devcfg.HMType == string([]byte{72, 109, 73, 80, 45, 66, 82, 79, 76, 76}) {
		dev.Description().Version = 5
		dev.Description().Firmware = string([]byte{49, 46, 49, 48, 46, 49, 54})
		dev.Description().Interface = string([]byte{72, 109, 73, 80, 45, 82, 70})
	}
	// add maintenance channel
	maintenance := vdevices.NewMaintenanceChannel(dev)
	if devcfg.HMType == string([]byte{72, 109, 73, 80, 45, 66, 82, 79, 76, 76}) {
		maintenance.Description().Version = 5
		maintenance.Description().Flags = itf.DeviceFlagInternal
	}

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
		case rtcfg.ChannelEnergyCounter:
			ch := vdevices.NewEnergyCounterChannel(dev)
			log.Debugf("Created static energy counter channel: %s", ch.Description().Address)
		case rtcfg.ChannelGasCounter:
			ch := vdevices.NewGasCounterChannel(dev)
			log.Debugf("Created static gas counter channel: %s", ch.Description().Address)
		case rtcfg.ChannelUnreach:
			ch := vd.addStaticUnreach(dev)
			log.Debugf("Created static unreach channel: %s", ch.Description().Address)

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
		case rtcfg.ChannelMQTTEnergyCounter:
			ch := vd.addMQTTEnergyCounter(dev)
			log.Debugf("Created MQTT energy counter channel: %s", ch.Description().Address)
		case rtcfg.ChannelMQTTGasCounter:
			ch := vd.addMQTTGasCounter(dev)
			log.Debugf("Created MQTT gas counter channel: %s", ch.Description().Address)
		case rtcfg.ChannelMQTTUnreach:
			ch := vd.addMQTTUnreach(dev)
			log.Debugf("Created MQTT connection error channel: %s", ch.Description().Address)
		case rtcfg.ChannelMQTTBRoll:
			ch := vd.addMQTTBRoll(dev)
			log.Debugf("Created MQTT BROLL device starting at channel: %s", ch.Description().Address)
		default:
			return fmt.Errorf("Unsupported kind of channel in device %s: %v", devcfg.Address, chcfg.Kind)
		}
	}

	if err := vd.Devices.AddDevice(dev); err != nil {
		return fmt.Errorf("Registration failed: %v", err)
	}
	return nil
}
