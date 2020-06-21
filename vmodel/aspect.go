package vmodel

import (
	"strings"

	"github.com/mdzio/go-hmccu/script"
	"github.com/mdzio/go-veap/model"
)

// AspectCol holds domains for the CCU rooms and functions.
type AspectCol struct {
	model.BasicObject
	model.BasicItem
	itemRole       string
	aspectProvider func() map[string]script.AspectDef
	service        *model.Service
}

// NewRoomCol creates a new AspectCol for rooms.
func NewRoomCol(col model.ChangeableCollection, reGaDom *script.ReGaDOM, service *model.Service) *AspectCol {
	d := new(AspectCol)
	d.Identifier = "room"
	d.Title = "Rooms"
	d.Description = "Rooms of the ReGaHss"
	d.Collection = col
	d.CollectionRole = "root"
	d.itemRole = "room"
	d.aspectProvider = func() map[string]script.AspectDef {
		return reGaDom.Rooms()
	}
	d.service = service
	col.PutItem(d)
	return d
}

// NewFunctionCol creates a new AspectCol for functions.
func NewFunctionCol(col model.ChangeableCollection, reGaDom *script.ReGaDOM, service *model.Service) *AspectCol {
	d := new(AspectCol)
	d.Identifier = "function"
	d.Title = "Functions"
	d.Description = "Functions of the ReGaHss"
	d.Collection = col
	d.CollectionRole = "root"
	d.itemRole = "function"
	d.aspectProvider = func() map[string]script.AspectDef {
		return reGaDom.Functions()
	}
	d.service = service
	col.PutItem(d)
	return d
}

// Items implements model.Collection.
func (ac *AspectCol) Items() []model.ItemObject {
	as := ac.aspectProvider()
	var ios []model.ItemObject
	for _, a := range as {
		// The objects exist only temporarily during the VEAP request.
		ios = append(ios, newAspectObj(ac, a, ac.service))
	}
	return ios
}

// Item implements model.Collection.
func (ac *AspectCol) Item(id string) (object model.ItemObject, ok bool) {
	as := ac.aspectProvider()
	if a, ok := as[id]; ok {
		// The object exists only temporarily during the VEAP request.
		return newAspectObj(ac, a, ac.service), true
	}
	return nil, false
}

// GetItemRole implements model.Collection.
func (ac *AspectCol) GetItemRole() string {
	return ac.itemRole
}

type aspectObj struct {
	model.BasicObject
	collection *AspectCol
	channels   []string
	service    *model.Service
}

func newAspectObj(col *AspectCol, ad script.AspectDef, service *model.Service) *aspectObj {
	a := new(aspectObj)
	a.Identifier = ad.ISEID
	a.Title = ad.DisplayName
	a.Description = ad.Comment
	a.collection = col
	a.channels = ad.Channels
	a.service = service
	return a
}

// GetCollection implements model.Item.
func (ao *aspectObj) GetCollection() model.CollectionObject {
	return ao.collection
}

// GetCollectionRole implements model.Item.
func (ao *aspectObj) GetCollectionRole() string {
	return "collection"
}

// ReadLinks implements LinkReader.
func (ao *aspectObj) ReadLinks() []model.Link {
	var links []model.Link
	for _, addr := range ao.channels {
		// split addr into device and channel
		p := strings.IndexRune(addr, ':')
		if p == -1 {
			// invalid address
			continue
		}
		devAddr := addr[0:p]
		chAddr := addr[p+1:]
		// lookup veap object
		chObj, err := ao.service.EvalPath("/device/" + devAddr + "/" + chAddr)
		if err != nil {
			// object not found
			continue
		}
		links = append(links, model.BasicLink{Target: chObj, Role: "channel"})
	}
	return links
}
