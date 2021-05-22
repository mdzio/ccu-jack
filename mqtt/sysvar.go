package mqtt

import (
	"path"
	"strings"
	"time"

	"github.com/mdzio/go-hmccu/script"
	"github.com/mdzio/go-lib/any"
	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-veap"
	"github.com/mdzio/go-veap/model"
)

const (
	// cycle time for reading system variables
	sysVarReadCycle = 3000 * time.Millisecond
)

// SysVarReader reads cyclic system variables.
type SysVarReader struct {
	// Service is used to explore the system variables.
	Service veap.Service
	// ScriptClient is used to bulk read system variables.
	ScriptClient *script.Client
	// Server is used for publishing value changes.
	Server *Server

	stop chan struct{}
	done chan struct{}
}

// Start starts the system variable reader.
func (r *SysVarReader) Start() {
	log.Debug("Starting system variable reader")
	r.stop = make(chan struct{})
	r.done = make(chan struct{})
	go func() {
		// defer clean up
		defer func() {
			log.Debug("Stopping system variable reader")
			r.done <- struct{}{}
		}()

		// PV cache
		pvCache := make(map[string]veap.PV)

		for {
			// sleep before next read
			select {
			case <-r.stop:
				return
			case <-time.After(sysVarReadCycle):
			}

			// get list of system variables
			_, links, verr := r.Service.ReadProperties(sysVarVeapPath)
			if verr != nil {
				log.Errorf("System variable reader: %v", verr)
				return
			}

			// find system variables with "mqtt" in description?
			var sysVars []script.ValObjDef
			for _, l := range links {
				if l.Role == "sysvar" {
					p := path.Join(sysVarVeapPath, l.Target)
					attrs, _, verr := r.Service.ReadProperties(p)
					if verr != nil {
						log.Errorf("System variable reader: %v", verr)
						return
					}
					q := any.Q(map[string]interface{}(attrs))
					descr := q.Map().TryKey(model.DescriptionProperty).String()
					if q.Err() != nil {
						log.Errorf("System variable reader: %v", q.Err())
						return
					}

					// "mqtt" in description?
					if strings.Contains(strings.ToLower(descr), "mqtt") {
						iseID := q.Map().Key(model.IdentifierProperty).String()
						dataType := q.Map().Key("type").String()
						if q.Err() != nil {
							log.Errorf("System variable reader: %v", q.Err())
							return
						}
						sysVars = append(sysVars, script.ValObjDef{
							ISEID: iseID,
							Type:  dataType,
						})
					}
				}
			}

			// nothing to do?
			if len(sysVars) == 0 {
				continue
			}

			// bulk read system variables
			results, err := r.ScriptClient.ReadValues(sysVars)
			if err != nil {
				log.Errorf("System variable reader: %v", err)
				continue
			}
			for idx := range sysVars {
				err = results[idx].Err
				if err != nil {
					log.Errorf("System variable reader: %v", err)
					continue
				}
				iseID := sysVars[idx].ISEID

				// create PV
				state := veap.StateGood
				if results[idx].Uncertain {
					state = veap.StateUncertain
				}
				pv := veap.PV{Time: results[idx].Timestamp, Value: results[idx].Value, State: state}

				// PV changed?
				prevPV, ok := pvCache[iseID]
				if !ok || !pv.Equal(prevPV) {

					// publish PV
					topic := sysVarTopic + "/status/" + iseID
					if err := r.Server.PublishPV(topic, pv, message.QosExactlyOnce, true); err != nil {
						log.Errorf("System variable reader: %v", err)
					} else {
						pvCache[iseID] = pv
					}
				}
			}
		}
	}()
}

// Stop stops the system variable reader.
func (r *SysVarReader) Stop() {
	// stop system variable reader
	close(r.stop)
	<-r.done
}
