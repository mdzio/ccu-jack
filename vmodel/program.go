package vmodel

import (
	"sync"
	"time"

	"github.com/mdzio/go-hmccu/script"
	"github.com/mdzio/go-veap"
	"github.com/mdzio/go-veap/model"
	"github.com/mdzio/go-logging"
)

const (
	// exploration cycle for CCU programs
	prgExploreCycle = 30 * time.Minute
)

var prgLog = logging.Get("program")

// ProgramCol contains domains for the CCU programs. ScriptClient must be set,
// before start is called.
type ProgramCol struct {
	model.Domain
	ScriptClient *script.Client

	stopRequest chan struct{}
	stopped     sync.WaitGroup
}

// NewProgramCol creates a new ProgramCol.
func NewProgramCol(col model.ChangeableCollection) *ProgramCol {
	d := new(ProgramCol)
	d.Identifier = "program"
	d.Title = "Programs"
	d.Description = "Programs of the ReGaHss"
	d.Collection = col
	d.CollectionRole = "root"
	d.ItemRole = "program"
	d.stopRequest = make(chan struct{})
	col.PutItem(d)
	return d
}

// Start starts the exploration of the CCU programs.
func (pc *ProgramCol) Start() {
	// start program explorer
	pc.stopped.Add(1)
	go func() {
		prgLog.Info("Starting ReGaHss program explorer")

		// defer clean up
		defer func() {
			prgLog.Debug("Stopping ReGaHss program explorer")
			pc.stopped.Done()
		}()

		// exploration at startup
		pc.explore()

		// exploration loop
		for {
			select {
			case <-pc.stopRequest:
				return
			case <-time.After(prgExploreCycle):
				pc.explore()
			}
		}
	}()
}

// Stop stops the exploration of the CCU programs.
func (pc *ProgramCol) Stop() {
	// stop program explorer
	close(pc.stopRequest)
	pc.stopped.Wait()
}

func (pc *ProgramCol) explore() {
	prgLog.Debug("Exploring ReGaHss programs")
	// retrieve programs
	ps, err := pc.ScriptClient.Programs()
	if err != nil {
		prgLog.Error(err)
		return
	}
	// build lookup map
	psm := make(map[string]*script.ProgramDef)
	for _, p := range ps {
		psm[p.ISEID] = p
	}
	// deleting missing programs
	for _, it := range pc.Items() {
		_, ok := psm[it.GetIdentifier()]
		if !ok {
			// delete program
			prgLog.Debugf("Deleting program: %s (%s)", it.GetIdentifier(), it.GetTitle())
			pc.RemoveItem(it.GetIdentifier())
		}
	}
	// create new and updated programs
	for id, p := range psm {
		it, ok := pc.Item(id)
		cr := false
		if ok {
			o := it.(*program)
			if *p != *o.prg {
				prgLog.Debugf("Updating program: %s (%s)", id, p.DisplayName)
				cr = true
			}
		} else {
			prgLog.Debugf("Creating program: %s (%s)", id, p.DisplayName)
			cr = true
		}
		if cr {
			np := new(program)
			np.Collection = pc
			np.CollectionRole = "programs"
			np.prg = p
			np.scriptClient = pc.ScriptClient
			pc.PutItem(np)
		}
	}
}

type program struct {
	model.BasicItem
	prg          *script.ProgramDef
	scriptClient *script.Client
}

func (p *program) GetIdentifier() string {
	return p.prg.ISEID
}

func (p *program) GetTitle() string {
	return p.prg.DisplayName
}

func (p *program) GetDescription() string {
	return p.prg.Description
}

func (p *program) ReadAttributes() veap.AttrValues {
	attr := veap.AttrValues{
		"active":          p.prg.Active,
		"visible":         p.prg.Visible,
		"mqttGetTopic":    "program/get/" + p.prg.ISEID,
		"mqttStatusTopic": "program/status/" + p.prg.ISEID,
		"mqttSetTopic":    "program/set/" + p.prg.ISEID,
	}
	return attr
}

func (p *program) ReadPV() (veap.PV, veap.Error) {
	ts, err := p.scriptClient.ReadExecTime(p.prg)
	if err != nil {
		return veap.PV{}, veap.NewError(veap.StatusInternalServerError, err)
	}
	return veap.PV{Time: ts, Value: false, State: veap.StateGood}, nil
}

func (p *program) WritePV(pv veap.PV) veap.Error {
	x, ok := pv.Value.(bool)
	if !ok {
		return veap.NewErrorf(veap.StatusBadRequest, "Expected type bool: %#v", pv.Value)
	}
	// execute only if PV has value true
	if x {
		err := p.scriptClient.ExecProgram(p.prg)
		if err != nil {
			return veap.NewError(veap.StatusInternalServerError, err)
		}
	}
	return nil
}
