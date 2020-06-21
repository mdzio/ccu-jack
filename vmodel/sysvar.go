package vmodel

import (
	"time"

	"github.com/mdzio/go-hmccu/script"
	"github.com/mdzio/go-veap"
	"github.com/mdzio/go-veap/model"
	"github.com/mdzio/go-logging"
)

const (
	// exploration cycle for system variables
	sysVarExploreCycle = 30 * time.Minute
)

var sysVarLog = logging.Get("sysvar")

// SysVarCol contains domains for the CCU system variables. ScriptClient must be
// set, before start is called.
type SysVarCol struct {
	model.Domain
	ScriptClient *script.Client

	stopRequest chan struct{}
	stopped     chan struct{}
}

// NewSysVarCol creates a new SysVarCol.
func NewSysVarCol(col model.ChangeableCollection) *SysVarCol {
	d := new(SysVarCol)
	d.Identifier = "sysvar"
	d.Title = "System variables"
	d.Description = "System variables of the ReGaHss"
	d.Collection = col
	d.CollectionRole = "root"
	d.ItemRole = "sysvar"
	d.stopRequest = make(chan struct{})
	d.stopped = make(chan struct{})
	col.PutItem(d)
	return d
}

// Start starts the exploration of the CCU system variables.
func (d *SysVarCol) Start() {
	// start sysvar explorer
	go func() {
		sysVarLog.Info("Starting system variable explorer")

		// defer clean up
		defer func() {
			sysVarLog.Debug("Stopping system variable explorer")
			d.stopped <- struct{}{}
		}()

		// exploration at startup
		d.explore()

		// exploration loop
		for {
			select {
			case <-d.stopRequest:
				return
			case <-time.After(sysVarExploreCycle):
				d.explore()
			}
		}
	}()
}

// Stop stops the exploration of the CCU system variables.
func (d *SysVarCol) Stop() {
	// stop sysvar explorer
	d.stopRequest <- struct{}{}
	<-d.stopped
}

func (d *SysVarCol) explore() {
	sysVarLog.Debug("Exploring system variables")
	// retrieve system variables
	svs, err := d.ScriptClient.SystemVariables()
	if err != nil {
		sysVarLog.Error(err)
		return
	}
	// build lookup map
	svm := make(map[string]*script.SysVarDef)
	for _, sv := range svs {
		svm[sv.ISEID] = sv
	}
	// deleting missing variables
	for _, it := range d.Items() {
		_, ok := svm[it.GetIdentifier()]
		if !ok {
			// delete variable
			sysVarLog.Debugf("Deleting system variable: %s (%s)", it.GetIdentifier(), it.GetTitle())
			d.RemoveItem(it.GetIdentifier())
		}
	}
	// create new and updated variables
	for id, sv := range svm {
		it, ok := d.Item(id)
		cr := false
		if ok {
			o := it.(*sysVar)
			if !sv.Equal(o.sv) {
				sysVarLog.Debugf("Updating system variable: %s (%s)", id, sv.Name)
				cr = true
			}
		} else {
			sysVarLog.Debugf("Creating system variable: %s (%s)", id, sv.Name)
			cr = true
		}
		if cr {
			v := new(sysVar)
			v.Collection = d
			v.CollectionRole = "sysvars"
			v.sv = sv
			v.scriptClient = d.ScriptClient
			d.PutItem(v)
		}
	}
}

type sysVar struct {
	model.BasicItem
	sv           *script.SysVarDef
	scriptClient *script.Client
}

func (v *sysVar) GetIdentifier() string {
	return v.sv.ISEID
}

func (v *sysVar) GetTitle() string {
	return v.sv.Name
}

func (v *sysVar) GetDescription() string {
	return v.sv.Description
}

func (v *sysVar) ReadAttributes() veap.AttrValues {
	attr := veap.AttrValues{
		"unit":            v.sv.Unit,
		"operations":      v.sv.Operations,
		"type":            v.sv.Type,
		"mqttGetTopic":    "sysvar/get/" + v.sv.ISEID,
		"mqttStatusTopic": "sysvar/status/" + v.sv.ISEID,
		"mqttSetTopic":    "sysvar/set/" + v.sv.ISEID,
	}
	if v.sv.Minimum != nil {
		attr["minimum"] = v.sv.Minimum
	}
	if v.sv.Maximum != nil {
		attr["maximum"] = v.sv.Maximum
	}
	if v.sv.ValueName0 != nil {
		attr["valueName0"] = v.sv.ValueName0
	}
	if v.sv.ValueName1 != nil {
		attr["valueName1"] = v.sv.ValueName1
	}
	if v.sv.ValueList != nil {
		attr["valueList"] = v.sv.ValueList
	}
	return attr
}

func (v *sysVar) ReadPV() (veap.PV, veap.Error) {
	// read value
	cln := v.Collection.(*SysVarCol).ScriptClient
	value, ts, uncertain, err := cln.ReadSysVar(v.sv)
	if err != nil {
		return veap.PV{}, veap.NewError(veap.StatusInternalServerError, err)
	}
	state := veap.StateGood
	if uncertain {
		state = veap.StateUncertain
	}
	return veap.PV{Time: ts, Value: value, State: state}, nil
}

func (v *sysVar) WritePV(pv veap.PV) veap.Error {
	// convert JSON number/float64 to int for system variables of type ENUM
	value := pv.Value
	f, ok := value.(float64)
	if v.sv.Type == "ENUM" && ok {
		value = int(f)
	}

	// write value
	cln := v.Collection.(*SysVarCol).ScriptClient
	err := cln.WriteSysVar(v.sv, value)
	if err != nil {
		return veap.NewError(veap.StatusInternalServerError, err)
	}
	return nil
}
