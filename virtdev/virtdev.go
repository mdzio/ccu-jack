package virtdev

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/mdzio/ccu-jack/rtcfg"
	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-hmccu/itf/xmlrpc"
	"github.com/mdzio/go-logging"
)

const (
	// path to the file InterfacesList.xml on the CCU3
	itfListFile = "/etc/config/InterfacesList.xml"

	// template for a new interface entry
	itfTmpl = "\t<ipc>\n\t \t<name>%s</name>\n\t \t<url>%s</url>\n\t \t<info>%s</info>\n\t</ipc>\n"

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
	err := addToInterfaceList(
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
	vd.SynchronizeDevices(cfg.VirtualDevices.Devices)
}

func (vd *VirtualDevices) Stop() {
	// only stop, if successfully started
	if vd.deviceHandler != nil {
		log.Debug("Stopping virtual devices")
		vd.deviceHandler.Close()
	}
}

// SynchronizeDevices updates the virtual device container based on the
// configuration. The configuration must be locked for reading.
func (vd *VirtualDevices) SynchronizeDevices(devices map[string]*rtcfg.Device) {
	// delete non existing devices
	for _, dev := range vd.Devices.Devices() {
		// exists device in config?
		_, exist := devices[dev.Description().Address]
		if !exist {
			// if not, remove it from container
			log.Infof("Removing virtual device: %s", dev.Description().Address)
			vd.Devices.RemoveDevice(dev.Description().Address)
		}
	}

	// add new devices
	for addr, dev := range devices {
		// exists device in runtime?
		if _, err := vd.Devices.Device(addr); err != nil {
			// if not, create it
			vd.createDevice(dev)
		}
	}
}

func (vd *VirtualDevices) createDevice(device *rtcfg.Device) {
	log.Infof("Creating virtual device %s with logic %s and %d channel(s)", device.Address, device.Logic, len(device.Channels))
	var dev vdevices.GenericDevice
	var err error
	switch device.Logic {
	case rtcfg.LogicStatic:
		dev, err = createStaticDevice(device, vd.eventPublisher)
	default:
		err = fmt.Errorf("Virtual device logic not implemented: %v", device.Logic)
	}
	if err == nil {
		err = vd.Devices.AddDevice(dev)
	}
	if err != nil {
		log.Errorf("Creation of virtual device failed: %v", err)
		return
	}
}

func addToInterfaceList(inFilePath, outFilePath, name, url, info string) error {
	// read file
	bs, err := os.ReadFile(inFilePath)
	if err != nil {
		return err
	}
	in := string(bs)

	// generate entry
	e := fmt.Sprintf(itfTmpl, name, url, info)
	log.Tracef("Inserting into %s: %s", inFilePath, e)

	// insert entry
	p := strings.Index(in, "</interfaces>")
	if p == -1 {
		return fmt.Errorf("Invalid file format: %s", inFilePath)
	}
	out := in[:p] + e + in[p:]

	// write file
	err = os.WriteFile(outFilePath, []byte(out), 0644)
	if err != nil {
		return err
	}
	return nil
}
