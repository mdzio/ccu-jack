package vmodel

import (
	"strconv"
	"time"

	"github.com/mdzio/ccu-jack/virtdev"
	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-hmccu/script"
	"github.com/mdzio/go-veap"
	"github.com/mdzio/go-veap/model"
)

// VirtualDeviceCol provides the VEAP-API for the virtual devices.
type VirtualDeviceCol struct {
	Container    *vdevices.Container
	ModelService *model.Service
	ReGaDOM      *script.ReGaDOM

	collection model.CollectionObject
}

func NewVirtualDeviceCol(col model.ChangeableCollection) *VirtualDeviceCol {
	d := new(VirtualDeviceCol)
	d.collection = col
	col.PutItem(d)
	return d
}

// GetIdentifier implements model.Object.
func (d *VirtualDeviceCol) GetIdentifier() string {
	return "virtdev"
}

// GetTitle implements model.Object.
func (d *VirtualDeviceCol) GetTitle() string {
	return "Virtual Devices"
}

// GetDescription implements model.Object.
func (d *VirtualDeviceCol) GetDescription() string {
	return "Virtual devices of the CCU-Jack"
}

// GetCollection implements model.Item.
func (d *VirtualDeviceCol) GetCollection() model.CollectionObject {
	return d.collection
}

// GetCollectionRole implements model.Item.
func (d *VirtualDeviceCol) GetCollectionRole() string {
	return "root"
}

// Items implements model.Collection.
func (d *VirtualDeviceCol) Items() []model.ItemObject {
	cds := d.Container.Devices()
	items := make([]model.ItemObject, len(cds))
	for idx := range cds {
		items[idx] = &virtualDevice{
			collection: d,
			device:     cds[idx],
		}
	}
	return items
}

// Item implements model.Collection.
func (d *VirtualDeviceCol) Item(id string) (object model.ItemObject, ok bool) {
	cd, err := d.Container.Device(id)
	if err != nil {
		return nil, false
	}
	return &virtualDevice{
		collection: d,
		device:     cd,
	}, true
}

// GetItemRole implements model.Collection.
func (d *VirtualDeviceCol) GetItemRole() string {
	return "device"
}

type virtualDevice struct {
	collection model.CollectionObject
	device     vdevices.GenericDevice
}

// GetIdentifier implements model.Object.
func (d *virtualDevice) GetIdentifier() string {
	return d.device.Description().Address
}

// GetTitle implements model.Object.
func (d *virtualDevice) GetTitle() string {
	r := d.collection.(*VirtualDeviceCol).ReGaDOM
	rd := r.Device(d.device.Description().Address)
	if rd != nil {
		return rd.DisplayName
	}
	return d.device.Description().Address
}

// GetDescription implements model.Object.
func (d *virtualDevice) GetDescription() string {
	return ""
}

// GetCollection implements model.Item.
func (d *virtualDevice) GetCollection() model.CollectionObject {
	return d.collection
}

// GetCollectionRole implements model.Item.
func (d *virtualDevice) GetCollectionRole() string {
	return "devices"
}

// ReadAttributes implements model.AttributeReader.
func (d *virtualDevice) ReadAttributes() veap.AttrValues {
	attr := deviceDescrToAttr(d.device.Description())
	attr["interfaceType"] = virtdev.InterfaceID
	return attr
}

// Items implements model.Collection.
func (d *virtualDevice) Items() []model.ItemObject {
	cs := d.device.Channels()
	items := make([]model.ItemObject, len(cs))
	for idx := range cs {
		items[idx] = &virtualChannel{
			collection: d,
			channel:    cs[idx],
		}
	}
	return items
}

// Item implements model.Collection.
func (d *virtualDevice) Item(id string) (object model.ItemObject, ok bool) {
	c, err := d.device.Channel(id)
	if err != nil {
		return nil, false
	}
	return &virtualChannel{
		collection: d,
		channel:    c,
	}, true
}

// GetItemRole implements model.Collection.
func (d *virtualDevice) GetItemRole() string {
	return "channel"
}

type virtualChannel struct {
	collection model.CollectionObject
	channel    vdevices.GenericChannel
}

// GetIdentifier implements model.Object.
func (c *virtualChannel) GetIdentifier() string {
	// only channel index
	return strconv.Itoa(c.channel.Description().Index)
}

// GetTitle implements model.Object.
func (c *virtualChannel) GetTitle() string {
	d := c.collection.(*virtualDevice)
	r := d.collection.(*VirtualDeviceCol).ReGaDOM
	rc := r.Channel(c.channel.Description().Address)
	if rc != nil {
		return rc.DisplayName
	}
	return c.channel.Description().Address
}

// GetDescription implements model.Object.
func (c *virtualChannel) GetDescription() string {
	return ""
}

// GetCollection implements model.Item.
func (c *virtualChannel) GetCollection() model.CollectionObject {
	return c.collection
}

// GetCollectionRole implements model.Item.
func (c *virtualChannel) GetCollectionRole() string {
	return "device"
}

// ReadAttributes implements model.AttributeReader.
func (c *virtualChannel) ReadAttributes() veap.AttrValues {
	return deviceDescrToAttr(c.channel.Description())
}

// Items implements model.Collection.
func (c *virtualChannel) Items() []model.ItemObject {
	ps := c.channel.ValueParamset().Parameters()
	items := make([]model.ItemObject, len(ps))
	for idx := range ps {
		items[idx] = &virtualParameter{
			collection: c,
			parameter:  ps[idx],
		}
	}
	return items
}

// Item implements model.Collection.
func (c *virtualChannel) Item(id string) (object model.ItemObject, ok bool) {
	p, err := c.channel.ValueParamset().Parameter(id)
	if err != nil {
		return nil, false
	}
	return &virtualParameter{
		collection: c,
		parameter:  p,
	}, true
}

// GetItemRole implements model.Collection.
func (c *virtualChannel) GetItemRole() string {
	return "parameter"
}

// ReadLinks implements model.LinkReader.
func (c *virtualChannel) ReadLinks() []model.Link {
	var links []model.Link
	// add links to rooms and functions
	d := c.collection.(*virtualDevice)
	r := d.collection.(*VirtualDeviceCol)
	cdef := r.ReGaDOM.Channel(c.channel.Description().Address)
	if cdef != nil {
		for _, rID := range cdef.Rooms {
			chObj, err := r.ModelService.EvalPath("/room/" + rID)
			if err != nil {
				// object not found
				continue
			}
			links = append(links, model.BasicLink{Target: chObj, Role: "room"})
		}
		for _, fID := range cdef.Functions {
			chObj, err := r.ModelService.EvalPath("/function/" + fID)
			if err != nil {
				// object not found
				continue
			}
			links = append(links, model.BasicLink{Target: chObj, Role: "function"})
		}
	}
	return links
}

type virtualParameter struct {
	collection model.CollectionObject
	parameter  vdevices.GenericParameter
}

// GetIdentifier implements model.Object.
func (p *virtualParameter) GetIdentifier() string {
	return p.parameter.Description().ID
}

// GetTitle implements model.Object.
func (p *virtualParameter) GetTitle() string {
	chTitle := p.collection.GetTitle()
	if chTitle != "" {
		return chTitle + " - " + p.parameter.Description().ID
	}
	return p.parameter.Description().ID
}

// GetDescription implements model.Object.
func (p *virtualParameter) GetDescription() string {
	return ""
}

// GetCollection implements model.Item.
func (p *virtualParameter) GetCollection() model.CollectionObject {
	return p.collection
}

// GetCollectionRole implements model.Item.
func (p *virtualParameter) GetCollectionRole() string {
	return "channel"
}

// ReadAttributes implements model.AttributeReader.
func (p *virtualParameter) ReadAttributes() veap.AttrValues {
	ch := p.collection.(*virtualChannel)
	dev := ch.collection.(*virtualDevice)
	mqttTopic := dev.GetIdentifier() + "/" + ch.GetIdentifier() + "/" + p.GetIdentifier()
	descr := p.parameter.Description()
	attrs := veap.AttrValues{
		"type":            descr.Type,
		"operations":      descr.Operations,
		"flags":           descr.Flags,
		"default":         descr.Default,
		"maximum":         descr.Max,
		"minimum":         descr.Min,
		"unit":            descr.Unit,
		"tabOrder":        descr.TabOrder,
		"control":         descr.Control,
		"id":              descr.ID,
		"mqttStatusTopic": "virtdev/status/" + mqttTopic,
	}

	// special attributes
	switch descr.Type {
	case "FLOAT":
		fallthrough
	case "INTEGER":
		special := make([]interface{}, len(descr.Special))
		for i := range special {
			special[i] = map[string]interface{}{
				"id":    descr.Special[i].ID,
				"value": descr.Special[i].Value,
			}
		}
		attrs["special"] = special
	case "ENUM":
		valueList := make([]interface{}, len(descr.ValueList))
		for i := range valueList {
			valueList[i] = descr.ValueList[i]
		}
		attrs["valueList"] = valueList
	}

	// parameter writeable?
	if descr.Operations&itf.ParameterOperationWrite != 0 {
		attrs["mqttSetTopic"] = "virtdev/set/" + mqttTopic
	}
	return attrs
}

// ReadPV implements model.PVReader.
func (p *virtualParameter) ReadPV() (veap.PV, veap.Error) {
	return veap.PV{
		Time:  time.Now(),
		Value: p.parameter.Value(),
		State: veap.StateGood,
	}, nil
}

// WritePV implements model.PVWriter.
func (p *virtualParameter) WritePV(pv veap.PV) veap.Error {
	// get channel
	ch := p.collection.(*virtualChannel)
	// convert JSON number/float64 to int for parameters of type INTEGER/ENUM
	value := pv.Value
	f, ok := value.(float64)
	ty := p.parameter.Description().Type
	if (ty == itf.ParameterTypeEnum || ty == itf.ParameterTypeInteger) && ok {
		value = int(f)
	}
	// check data type
	err := checkType(ty, value)
	if err != nil {
		return veap.NewErrorf(veap.StatusInternalServerError, "Writing parameter %s failed: %v", ch.channel.Description().Address+"."+p.parameter.Description().ID, err)
	}
	// set value
	err = p.parameter.SetValue(value)
	if err != nil {
		return veap.NewError(veap.StatusInternalServerError, err)
	}
	return nil
}
