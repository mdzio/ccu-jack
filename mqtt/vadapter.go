package mqtt

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mdzio/go-veap"
	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-mqtt/service"
)

type vadapter struct {
	// MQTT topic prefix
	mqttTopic string
	// VEAP path prefix
	veapPath string
	// read back duration (0: disabled)
	readBackDur time.Duration
	// MQTT server
	mqttBroker *Broker
	// VEAP service
	veapService veap.Service

	onSet service.OnPublishFunc
	onGet service.OnPublishFunc
	// condition variables
	cond *sync.Cond
	quit chan struct{}
	cnt  int
}

func (a *vadapter) start() {
	a.cond = sync.NewCond(new(sync.Mutex))
	a.quit = make(chan struct{})

	// handle set messages
	a.onSet = func(msg *message.PublishMessage) error {
		// count active callbacks
		if !a.enter() {
			return nil
		}
		defer a.exit()

		log.Tracef("Set message received: %s, %s", msg.Topic(), msg.Payload())

		// parse PV
		pv, err := wireToPV(msg.Payload())
		if err != nil {
			return err
		}

		// map topic to VEAP address
		setTopic := a.mqttTopic + "/set"
		topic := string(msg.Topic())
		if !strings.HasPrefix(topic, setTopic+"/") {
			return fmt.Errorf("Unexpected topic: %s", topic)
		}

		// path with leading /
		path := topic[len(setTopic):]

		// use VEAP service to write PV
		if err = a.veapService.WritePV(a.veapPath+path, pv); err != nil {
			return err
		}

		// read back current value and publish
		if a.readBackDur != 0 {
			go func() {
				// wait for timer or quit
				t := time.NewTimer(a.readBackDur)
				select {
				case <-a.quit:
					// clean up timer
					if !t.Stop() {
						<-t.C
					}
					return
				case <-t.C:
				}

				// count active callbacks
				if !a.enter() {
					return
				}
				defer a.exit()

				// read back
				pv, verr := a.veapService.ReadPV(a.veapPath + path)
				if verr != nil {
					log.Warning("Read back of %s failed: %v", a.veapPath+path, verr)
					return
				}
				// publish PV
				statusTopic := a.mqttTopic + "/status"
				err = a.mqttBroker.PublishPV(statusTopic+path, pv, message.QosAtLeastOnce, true)
				if err != nil {
					log.Warning("Publish of %s failed: %v", statusTopic+path, err)
					return
				}
			}()
		}
		return nil
	}

	// handle get messages
	a.onGet = func(msg *message.PublishMessage) error {
		// count active callbacks
		if !a.enter() {
			return nil
		}
		defer a.exit()

		log.Tracef("Get message received: %s", msg.Topic())

		// map topic to VEAP address
		getTopic := a.mqttTopic + "/get"
		topic := string(msg.Topic())
		if !strings.HasPrefix(topic, getTopic+"/") {
			return fmt.Errorf("Unexpected topic: %s", topic)
		}

		// path with leading /
		path := topic[len(getTopic):]

		// use VEAP service to read PV
		pv, err := a.veapService.ReadPV(a.veapPath + path)
		if err != nil {
			return err
		}

		// publish PV
		statusTopic := a.mqttTopic + "/status"
		return a.mqttBroker.PublishPV(statusTopic+path, pv, message.QosAtLeastOnce, true)
	}

	// subscribe topics
	a.mqttBroker.Subscribe(a.mqttTopic+"/set/+", message.QosExactlyOnce, &a.onSet)
	a.mqttBroker.Subscribe(a.mqttTopic+"/get/+", message.QosExactlyOnce, &a.onGet)
}

func (a *vadapter) stop() {
	// unsubscribe topics
	a.mqttBroker.Unsubscribe(a.mqttTopic+"/set/+", &a.onSet)
	a.mqttBroker.Unsubscribe(a.mqttTopic+"/get/+", &a.onGet)

	// disable callbacks
	a.cond.L.Lock()
	close(a.quit)

	// wait for completion of pending callbacks
	for a.cnt > 0 {
		a.cond.Wait()
	}
	a.cond.L.Unlock()
}

func (a *vadapter) enter() bool {
	// register callback
	a.cond.L.Lock()
	defer a.cond.L.Unlock()
	select {
	case <-a.quit:
		return false
	default:
	}
	a.cnt++
	return true
}

func (a *vadapter) exit() {
	// unregister callback
	a.cond.L.Lock()
	defer a.cond.L.Unlock()
	a.cnt--
	a.cond.Signal()
}
