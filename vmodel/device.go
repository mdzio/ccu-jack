package vmodel

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/script"
	"github.com/mdzio/go-logging"
	"github.com/mdzio/go-veap"
	"github.com/mdzio/go-veap/model"
)

const (
	// A buffer is needed for buffering events while exploring.
	notifBufferSize = 1000

	// delay between XMLRPC requests while exploring
	xmlRPCDelay = 50 * time.Millisecond
)

var deviceLog = logging.Get("devices")

// DeviceCol contains domains for the CCU devices. This domain implements
// itf.Receiver to receive callbacks from the CCU. Interconnector,
// ReGaDom and ModelService must be set, before start is called.
type DeviceCol struct {
	model.Domain
	Interconnector *itf.Interconnector
	ReGaDOM        *script.ReGaDOM
	ModelService   *model.Service

	notifications chan *deviceNotif
	stopRequest   chan struct{}
	stopped       chan struct{}
}

type paramEvt struct {
	address, valueKey string
	value             interface{}
}

type deviceNotif struct {
	interfaceID   string
	event         paramEvt
	newDevices    []*itf.DeviceDescription
	deleteDevices []string
}

// NewDeviceCol creates a new DeviceCol.
func NewDeviceCol(col model.ChangeableCollection) *DeviceCol {
	d := new(DeviceCol)
	d.Identifier = "device"
	d.Title = "Devices"
	d.Description = "CCU Devices"
	d.Collection = col
	d.CollectionRole = "root"
	d.ItemRole = "device"
	d.notifications = make(chan *deviceNotif, notifBufferSize)
	d.stopRequest = make(chan struct{})
	d.stopped = make(chan struct{})
	col.PutItem(d)
	return d
}

// Start starts handling notifications.
func (d *DeviceCol) Start() {
	// start handling notifications
	go func() {
		deviceLog.Info("Starting notification handler")

		// defer clean up
		defer func() {
			deviceLog.Debug("Stopping notification handler")
			d.stopped <- struct{}{}
		}()

		// handle notifications
		for {
			select {
			case <-d.stopRequest:
				return
			case n := <-d.notifications:
				// deletion of devices
				if len(n.deleteDevices) > 0 {
					d.handleDeletion(n)
				}
				// new devices
				if len(n.newDevices) > 0 {
					if !d.handleNew(n) {
						// exit while exploring devices
						return
					}
				}
				// value change event
				if len(n.event.address) > 0 {
					d.handleEvent(n)
				}
			}
		}
	}()
}

// Stop stops handling notifications.
func (d *DeviceCol) Stop() {
	// stop handling notifications
	d.stopRequest <- struct{}{}
	<-d.stopped
}

func (d *DeviceCol) handleDeletion(n *deviceNotif) {
	// separate devices and channels
	type chAddr struct{ dev, ch string }
	var channels []chAddr
	var devices []string
	for _, addr := range n.deleteDevices {
		if p := strings.IndexRune(addr, ':'); p == -1 {
			devices = append(devices, addr)
		} else {
			channels = append(channels, chAddr{
				dev: addr[0:p],
				ch:  addr[p+1:],
			})
		}
	}
	// 1. delete channels
	for _, addr := range channels {
		deviceLog.Debug("Deleting channel: ", addr.dev, ":", addr.ch)
		devItem, ok := d.Item(addr.dev)
		if !ok {
			deviceLog.Warning("Deletion of channel failed, device not found: ", addr.dev)
			continue
		}
		chItem := devItem.(model.ChangeableCollection).RemoveItem(addr.ch)
		if chItem == nil {
			deviceLog.Warning("Deletion of channel failed, channel not found: ", addr.dev, ":", addr.ch)
		}
	}
	// 2. delete devices
	for _, addr := range devices {
		deviceLog.Debug("Deleting device: ", addr)
		devItem := d.RemoveItem(addr)
		if devItem == nil {
			deviceLog.Warning("Deletion of device failed, device not found: ", addr)
		}
	}
}

func (d *DeviceCol) handleNew(n *deviceNotif) bool {
	// separate devices and channels
	type chDescr struct {
		dev, ch string
		descr   *itf.DeviceDescription
	}
	var channels []chDescr
	var devices []*itf.DeviceDescription
	for _, descr := range n.newDevices {
		if p := strings.IndexRune(descr.Address, ':'); p == -1 {
			devices = append(devices, descr)
		} else {
			channels = append(channels, chDescr{
				dev:   descr.Address[0:p],
				ch:    descr.Address[p+1:],
				descr: descr,
			})
		}
	}

	// 1. create devices
	for _, descr := range devices {
		deviceLog.Debug("Creating device: ", descr.Address)
		// get CCU interface client
		cln, err := d.Interconnector.Client(n.interfaceID)
		if err != nil {
			deviceLog.Error("Invalid interface ID in callback: ", n.interfaceID)
			return true
		}
		// create device domain
		dev := new(device)
		dev.descr = descr
		dev.itfClient = cln
		dev.Collection = d
		dev.CollectionRole = "devices"
		dev.ItemRole = "channel"
		d.PutItem(dev)
		// parameter sets of device
		for _, psID := range descr.Paramsets {
			// The parameter set MASTER can always be read and written. With the
			// others (e.g. LINK, SERVICE) this is unclear. Especially with
			// battery operated devices these cannot be read immediately.
			// Therefore currently only MASTER is supported.
			if psID == "MASTER" {
				// add parameter set as PV
				deviceLog.Debug("Creating parameter set: ", psID)
				dev.PutItem(&paramset{
					id:        psID,
					address:   descr.Address,
					itfClient: cln,
					BasicItem: model.BasicItem{
						Collection:     dev,
						CollectionRole: "device",
					},
				})
			}
		}
	}

	// 2. create channels
	for _, ch := range channels {
		deviceLog.Debug("Creating channel: ", ch.descr.Address)
		// find device
		devi, ok := d.Item(ch.dev)
		if !ok {
			deviceLog.Error("Device for channel not found: ", ch.dev)
			continue
		}
		devd := devi.(*device)
		// create channel domain
		chd := new(channel)
		chd.identifier = ch.ch
		chd.descr = ch.descr
		chd.Collection = devd
		chd.CollectionRole = "device"
		chd.ItemRole = "parameter"
		devd.PutItem(chd)
		// add parameter sets
		for _, psID := range ch.descr.Paramsets {
			if psID == "VALUES" {
				// add parameter set VALUES
				psetDescr, err := devd.itfClient.GetParamsetDescription(ch.descr.Address, psID)
				if err != nil {
					deviceLog.Error("Retrieving parameter set description failed: ", err)
					continue
				}
				// delay or stop requested?
				t := time.NewTimer(xmlRPCDelay)
				select {
				case <-d.stopRequest:
					// clean up timer
					if !t.Stop() {
						<-t.C
					}
					return false
				case <-t.C:
				}
				// create parameter domains
				for _, descr := range psetDescr {
					deviceLog.Debug("Creating parameter: ", descr.ID)
					p := new(parameter)
					p.descr = descr
					p.Collection = chd
					p.CollectionRole = "channel"
					chd.PutItem(p)
				}
			} else if psID == "MASTER" {
				// The parameter set MASTER can always be read and written. With
				// the others (e.g. LINK, SERVICE) this is unclear. Especially
				// with battery operated devices these cannot be read
				// immediately. Therefore currently only MASTER is supported.
				deviceLog.Debug("Creating parameter set: ", psID)
				chd.PutItem(&paramset{
					id:        psID,
					address:   chd.descr.Address,
					itfClient: devd.itfClient,
					BasicItem: model.BasicItem{
						Collection:     chd,
						CollectionRole: "channel",
					},
				})
			}
		}
	}
	return true
}

func (d *DeviceCol) handleEvent(n *deviceNotif) {
	// separate device and channel
	var dev, ch string
	var p int
	if p = strings.IndexRune(n.event.address, ':'); p == -1 {
		deviceLog.Warning("Device should not send event: ", n.event.address)
		return
	}
	dev = n.event.address[0:p]
	ch = n.event.address[p+1:]
	// find device
	i, ok := d.Item(dev)
	if !ok {
		deviceLog.Debug("Device for event not found: ", dev)
		return
	}
	devDom := i.(*device)
	// find channel
	i, ok = devDom.Item(ch)
	if !ok {
		deviceLog.Debug("Channel for event not found: ", n.event.address)
		return
	}
	chDom := i.(*channel)
	// find parameter
	i, ok = chDom.Item(n.event.valueKey)
	if !ok {
		deviceLog.Debug("Parameter for event not found: ", n.event.address, ".", n.event.valueKey)
		return
	}
	paramVar := i.(*parameter)
	// update parameter value
	deviceLog.Debug("Updating PV of ", n.event.address, ".", n.event.valueKey, " to ", n.event.value)
	paramVar.updatePV(n.event.value)
}

func (d *DeviceCol) sendNotification(n *deviceNotif) {
	select {
	case d.notifications <- n:
		// send ok
	default:
		// channel full
		deviceLog.Errorf("Notification lost, buffer size is too small: %d", notifBufferSize)
	}
}

// Event implements itf.Receiver.
func (d *DeviceCol) Event(interfaceID, address, valueKey string, value interface{}) error {
	// send notification
	d.sendNotification(&deviceNotif{
		interfaceID: interfaceID,
		event:       paramEvt{address, valueKey, value},
	})
	return nil
}

// NewDevices implements itf.Receiver.
func (d *DeviceCol) NewDevices(interfaceID string, devDescriptions []*itf.DeviceDescription) error {
	// send notification
	d.sendNotification(&deviceNotif{
		interfaceID: interfaceID,
		newDevices:  devDescriptions,
	})
	return nil
}

// DeleteDevices implements itf.Receiver.
func (d *DeviceCol) DeleteDevices(interfaceID string, addresses []string) error {
	// send notification
	d.sendNotification(&deviceNotif{
		interfaceID:   interfaceID,
		deleteDevices: addresses,
	})
	return nil
}

// UpdateDevice implements itf.Receiver.
func (d *DeviceCol) UpdateDevice(interfaceID, address string, hint int) error {
	// not handled at the moment
	return nil
}

// ReplaceDevice implements itf.Receiver.
func (d *DeviceCol) ReplaceDevice(interfaceID, oldDeviceAddress, newDeviceAddress string) error {
	// not handled at the moment
	return nil
}

// ReaddedDevice implements itf.Receiver.
func (d *DeviceCol) ReaddedDevice(interfaceID string, deletedAddresses []string) error {
	// not handled at the moment
	return nil
}

func deviceDescrToAttr(d *itf.DeviceDescription) veap.AttrValues {
	return veap.AttrValues{
		"type":              d.Type,
		"address":           d.Address,
		"rfAddress":         d.RFAddress,
		"children":          d.Children,
		"parent":            d.Parent,
		"parentType":        d.ParentType,
		"index":             d.Index,
		"aesActive":         d.AESActive,
		"paramsets":         d.Paramsets,
		"firmware":          d.Firmware,
		"availableFirmware": d.AvailableFirmware,
		"version":           d.Version,
		"flags":             d.Flags,
		"linkSourceRoles":   d.LinkSourceRoles,
		"linkTargetRoles":   d.LinkTargetRoles,
		"direction":         d.Direction,
		"group":             d.Group,
		"team":              d.Team,
		"teamTag":           d.TeamTag,
		"teamChannels":      d.TeamChannels,
		"interface":         d.Interface,
		"roaming":           d.Roaming,
		"rxMode":            d.RXMode,
	}
}

type device struct {
	model.BasicItem
	model.BasicCollection
	descr     *itf.DeviceDescription
	itfClient *itf.RegisteredClient
}

func (c *device) GetIdentifier() string {
	return c.descr.Address
}

func (c *device) GetTitle() string {
	devCol := c.Collection.(*DeviceCol)
	dd := devCol.ReGaDOM.Device(c.descr.Address)
	if dd != nil {
		return dd.DisplayName
	}
	return c.descr.Address
}

func (c *device) GetDescription() string {
	return ""
}

func (c *device) ReadAttributes() veap.AttrValues {
	attr := deviceDescrToAttr(c.descr)
	attr["interfaceType"] = c.itfClient.ReGaHssID
	return attr
}

type channel struct {
	model.BasicItem
	model.BasicCollection
	identifier string
	descr      *itf.DeviceDescription
}

func (c *channel) GetIdentifier() string {
	return c.identifier
}

func (c *channel) GetTitle() string {
	dev := c.Collection.(*device)
	devCol := dev.Collection.(*DeviceCol)
	cd := devCol.ReGaDOM.Channel(c.descr.Address)
	if cd != nil {
		return cd.DisplayName
	}
	return c.descr.Address
}

func (c *channel) GetDescription() string {
	return ""
}

func (c *channel) ReadAttributes() veap.AttrValues {
	return deviceDescrToAttr(c.descr)
}

func (c *channel) ReadLinks() []model.Link {
	var links []model.Link
	// add links to rooms and functions
	dev := c.Collection.(*device)
	devCol := dev.Collection.(*DeviceCol)
	chDef := devCol.ReGaDOM.Channel(c.descr.Address)
	if chDef != nil {
		for _, rID := range chDef.Rooms {
			chObj, err := devCol.ModelService.EvalPath("/room/" + rID)
			if err != nil {
				// object not found
				continue
			}
			links = append(links, model.BasicLink{Target: chObj, Role: "room"})
		}
		for _, fID := range chDef.Functions {
			chObj, err := devCol.ModelService.EvalPath("/function/" + fID)
			if err != nil {
				// object not found
				continue
			}
			links = append(links, model.BasicLink{Target: chObj, Role: "function"})
		}
	}
	return links
}

type parameter struct {
	model.BasicItem
	descr  *itf.ParameterDescription
	pv     veap.PV
	pvLock sync.RWMutex
}

func (p *parameter) GetIdentifier() string {
	return p.descr.ID
}

func (p *parameter) GetTitle() string {
	chTitle := p.Collection.GetTitle()
	if chTitle != "" {
		return chTitle + " - " + p.descr.ID
	}
	return p.descr.ID
}

func (p *parameter) GetDescription() string {
	return ""
}

func (p *parameter) ReadAttributes() veap.AttrValues {
	ch := p.Collection.(*channel)
	dev := ch.Collection.(*device)
	mqttTopic := dev.GetIdentifier() + "/" + ch.GetIdentifier() + "/" + p.GetIdentifier()
	attrs := veap.AttrValues{
		"type":            p.descr.Type,
		"operations":      p.descr.Operations,
		"flags":           p.descr.Flags,
		"default":         p.descr.Default,
		"maximum":         p.descr.Max,
		"minimum":         p.descr.Min,
		"unit":            p.descr.Unit,
		"tabOrder":        p.descr.TabOrder,
		"control":         p.descr.Control,
		"id":              p.descr.ID,
		"mqttStatusTopic": "device/status/" + mqttTopic,
	}

	// special attributes
	switch p.descr.Type {
	case "FLOAT":
		fallthrough
	case "INTEGER":
		special := make([]interface{}, len(p.descr.Special))
		for i := range special {
			special[i] = map[string]interface{}{
				"id":    p.descr.Special[i].ID,
				"value": p.descr.Special[i].Value,
			}
		}
		attrs["special"] = special
	case "ENUM":
		valueList := make([]interface{}, len(p.descr.ValueList))
		for i := range valueList {
			valueList[i] = p.descr.ValueList[i]
		}
		attrs["valueList"] = valueList
	}

	// parameter writeable?
	if p.descr.Operations&itf.ParameterOperationWrite != 0 {
		attrs["mqttSetTopic"] = "device/set/" + mqttTopic
	}
	return attrs
}

func (p *parameter) ReadPV() (veap.PV, veap.Error) {
	// if no value is present, retrieve the current value from the ReGaHss of
	// the CCU per HM script.
	if p.pv.Value == nil {
		ch := p.Collection.(*channel)
		dev := ch.Collection.(*device)
		devCol := dev.Collection.(*DeviceCol)
		client := devCol.ReGaDOM.ScriptClient
		addr := dev.itfClient.ReGaHssID + "." + ch.descr.Address + "." + p.descr.ID
		val, err := client.ReadValues([]script.ValObjDef{{ISEID: addr, Type: p.descr.Type}})
		if err != nil {
			return veap.PV{}, veap.NewError(veap.StatusInternalServerError, err)
		}
		state := veap.StateGood
		if val[0].Uncertain {
			state = veap.StateUncertain
		}
		// store and return PV
		p.pvLock.Lock()
		defer p.pvLock.Unlock()
		p.pv = veap.PV{
			Time:  val[0].Timestamp,
			Value: val[0].Value,
			State: state,
		}
		return p.pv, nil
	}
	// return PV
	p.pvLock.RLock()
	defer p.pvLock.RUnlock()
	return p.pv, nil
}

func (p *parameter) WritePV(pv veap.PV) veap.Error {
	// get channel and device
	ch := p.Collection.(*channel)
	dev := ch.Collection.(*device)
	// convert JSON number/float64 to int for parameters of type INTEGER/ENUM
	value := pv.Value
	f, ok := value.(float64)
	if (p.descr.Type == "ENUM" || p.descr.Type == "INTEGER") && ok {
		value = int(f)
	}
	// check data type
	err := checkType(p.descr.Type, value)
	if err != nil {
		return veap.NewErrorf(veap.StatusInternalServerError, "Writing parameter %s failed: %v", ch.descr.Address+"."+p.descr.ID, err)
	}
	// set value through XML-RPC
	err = dev.itfClient.SetValue(ch.descr.Address, p.descr.ID, value)
	if err != nil {
		return veap.NewError(veap.StatusInternalServerError, err)
	}
	return nil
}

// update PV with new value
func (p *parameter) updatePV(v interface{}) {
	// store PV
	p.pvLock.Lock()
	defer p.pvLock.Unlock()
	p.pv = veap.PV{
		Time:  time.Now(),
		Value: v,
		State: veap.StateGood,
	}
}

type paramset struct {
	// ID of parameter set (e.g. MASTER)
	id string
	// device or channel address
	address string
	// cached parameter set description
	cachedDescr itf.ParamsetDescription
	// XML-RPC client
	itfClient *itf.RegisteredClient

	model.BasicItem
}

func (ps *paramset) descr() itf.ParamsetDescription {
	// lazy retrieving of parameter set description
	if ps.cachedDescr == nil {
		descr, err := ps.itfClient.GetParamsetDescription(ps.address, ps.id)
		if err != nil {
			deviceLog.Error("Retrieving parameter set description failed: ", err)
			// error can't be signaled to VEAP client
			return nil
		}
		ps.cachedDescr = descr
	}
	return ps.cachedDescr
}

// GetIdentifier implements model.Object.
func (ps *paramset) GetIdentifier() string {
	// use a prefix to avoid name clashes with HomeMatic identifiers
	return "$" + ps.id
}

// GetTitle implements model.Object.
func (ps *paramset) GetTitle() string {
	return ps.Collection.GetTitle() + " - " + ps.GetIdentifier()
}

// GetDescription implements model.Object.
func (ps *paramset) GetDescription() string {
	return "Parameter set " + ps.id + " of device/channel " + ps.Collection.GetTitle()
}

// ReadAttributes implements model.Object.
func (ps *paramset) ReadAttributes() veap.AttrValues {
	// convert
	attrs := make(veap.AttrValues)
	for n, d := range ps.descr() {
		attrs[n] = map[string]interface{}{
			"type":       d.Type,
			"operations": d.Operations,
			"flags":      d.Flags,
			"default":    d.Default,
			"maximum":    d.Max,
			"minimum":    d.Min,
			"unit":       d.Unit,
			"tabOrder":   d.TabOrder,
			"control":    d.Control,
			"id":         d.ID,
		}
	}
	return attrs
}

// ReadPV implements model.PVReader.
func (ps *paramset) ReadPV() (veap.PV, veap.Error) {
	// call CCU interface API
	vs, err := ps.itfClient.GetParamset(ps.address, ps.id)
	if err != nil {
		return veap.PV{}, veap.NewError(veap.StatusInternalServerError, err)
	}
	return veap.PV{Value: vs, Time: time.Now(), State: veap.StateGood}, nil
}

// WritePV implements model.PVReader.
func (ps *paramset) WritePV(pv veap.PV) veap.Error {
	// check and convert value
	pvv, ok := pv.Value.(map[string]interface{})
	if !ok {
		return veap.NewErrorf(
			veap.StatusBadRequest,
			"Writing parameter set %s of %s failed: Invalid type for parameter set (expected JSON object)",
			ps.id, ps.address,
		)
	}
	vs := make(map[string]interface{})
	for k, v := range pvv {
		d, ok := ps.descr()[k]
		// known parameter?
		if !ok {
			return veap.NewErrorf(
				veap.StatusBadRequest,
				"Writing parameter set %s of %s failed: Unknown parameter: %s",
				ps.id, ps.address, k,
			)
		}
		// convert JSON number/float64 to int for parameters of type INTEGER/ENUM
		f, ok := v.(float64)
		if (d.Type == "ENUM" || d.Type == "INTEGER") && ok {
			v = int(f)
		}
		// check type
		err := checkType(d.Type, v)
		if err != nil {
			return veap.NewErrorf(
				veap.StatusBadRequest,
				"Writing parameter set %s of %s failed: %v",
				ps.id, ps.address, err,
			)
		}
		vs[k] = v
	}
	// call CCU interface API
	err := ps.itfClient.PutParamset(ps.address, ps.id, vs)
	if err != nil {
		return veap.NewError(veap.StatusInternalServerError, err)
	}
	return nil
}

func checkType(t string, v interface{}) error {
	switch t {
	case "BOOL":
		fallthrough
	case "ACTION":
		_, ok := v.(bool)
		if !ok {
			return fmt.Errorf("Invalid type for BOOL/ACTION: %#v", v)
		}
	case "INTEGER":
		fallthrough
	case "ENUM":
		_, ok := v.(int)
		if !ok {
			return fmt.Errorf("Invalid type for INTEGER/ENUM: %#v", v)
		}
	case "FLOAT":
		_, ok := v.(float64)
		if !ok {
			return fmt.Errorf("Invalid type for FLOAT: %#v", v)
		}
	case "STRING":
		_, ok := v.(string)
		if !ok {
			return fmt.Errorf("Invalid type for STRING: %#v", v)
		}
	default:
		return fmt.Errorf("Unsupported type: %s", t)
	}
	return nil
}
