package vmodel

import (
	"github.com/mdzio/go-veap"
	"github.com/mdzio/go-veap/model"
)

// NewRefreshVar adds a VEAP boolean variable to the specified collection. If
// the refresh variable is set to true, the supplied function is executed.
func NewRefreshVar(col model.ChangeableCollection, refresh func()) {
	model.NewVariable(&model.VariableCfg{
		Identifier:  "refresh",
		Title:       "Refresh",
		Description: "Refreshes the meta information from the CCU.",
		Collection:  col,
		ReadPVFunc: func() (veap.PV, veap.Error) {
			return veap.PV{Value: false}, nil
		},
		WritePVFunc: func(pv veap.PV) veap.Error {
			if pv.State.Bad() {
				return nil
			}
			b, ok := pv.Value.(bool)
			if !ok {
				return veap.NewErrorf(veap.StatusBadRequest, "Process value is not of type boolean")
			}
			if b {
				refresh()
			}
			return nil
		},
	})
}
