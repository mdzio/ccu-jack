package mqtt

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/mdzio/go-lib/any"

	"github.com/mdzio/go-veap"
	"github.com/mdzio/go-veap/model"
	"github.com/mdzio/go-logging"
	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-mqtt/service"
)

const (
	deviceStatusTopic = "device/status"
	deviceSetTopic    = "device/set"
	// path prefix for device data points in the VEAP address space
	deviceVeapPath = "/device"

	// topic prefix for system variables
	sysVarTopic = "sysvar"
	// path prefix for system variable data points in the VEAP address space
	sysVarVeapPath = "/sysvar"
	// delay time for reading back
	sysVarReadBackDur = 300 * time.Millisecond
	// cycle time for reading system variables
	sysVarReadCycle = 1000 * time.Millisecond

	// topic prefix for programs
	prgTopic = "program"
	// path prefix for programs in the VEAP address space
	prgVeapPath = "/program"
)

var log = logging.Get("mqtt-broker")

// Broker for MQTT. Broker implements itf.Receiver to receive XML-RPC events.
type Broker struct {
	// Binding address for serving MQTT.
	Addr string
	// Binding address for serving Secure MQTT.
	AddrTLS string
	// Certificate file for Secure MQTT.
	CertFile string
	// Private key file for Secure MQTT.
	KeyFile string
	// Authenticator specifies the authenticator. Default is "mockSuccess".
	Authenticator string
	// When an error happens while serving (e.g. binding of port fails), this
	// error is sent to the channel ServeErr.
	ServeErr chan<- error
	// Service is used to set device data points and system variables.
	Service veap.Service

	server     *service.Server
	doneServer sync.WaitGroup

	onSetDevice service.OnPublishFunc

	stopSVReader chan struct{}
	doneSVReader chan struct{}

	sysVarAdapter *vadapter
	prgAdapter    *vadapter
}

// Start starts the MQTT broker.
func (b *Broker) Start() {
	b.server = &service.Server{
		Authenticator: b.Authenticator,
	}

	// start MQTT listener
	if b.Addr != "" {
		b.doneServer.Add(1)
		go func() {
			log.Infof("Starting MQTT listener on address %s", b.Addr)
			err := b.server.ListenAndServe(b.Addr)
			// signal server is down
			b.doneServer.Done()
			// check for error
			if err != nil {
				// signal error while serving
				if b.ServeErr != nil {
					b.ServeErr <- fmt.Errorf("Running MQTT server failed: %v", err)
				}
			}
		}()
	}

	// start Secure MQTT listener
	if b.AddrTLS != "" {
		b.doneServer.Add(1)
		go func() {
			log.Infof("Starting Secure MQTT listener on address %s", b.AddrTLS)
			// TLS configuration
			cer, err := tls.LoadX509KeyPair(b.CertFile, b.KeyFile)
			if err != nil {
				// signal error while serving
				b.doneServer.Done()
				if b.ServeErr != nil {
					b.ServeErr <- fmt.Errorf("Running Secure MQTT server failed: %v", err)
				}
				return
			}
			config := &tls.Config{Certificates: []tls.Certificate{cer}}
			// start server
			err = b.server.ListenAndServeTLS(b.AddrTLS, config)
			// signal server is down
			b.doneServer.Done()
			// check for error
			if err != nil {
				// signal error while serving
				if b.ServeErr != nil {
					b.ServeErr <- fmt.Errorf("Running Secure MQTT server failed: %v", err)
				}
			}
		}()
	}

	// start system variable reader
	b.startSysVarReader()

	// subscribe set device topics
	b.onSetDevice = func(msg *message.PublishMessage) error {
		log.Tracef("Set device message received: %s: %s", msg.Topic(), msg.Payload())

		// parse PV
		pv, err := wireToPV(msg.Payload())
		if err != nil {
			return err
		}

		// map topic to VEAP address
		topic := string(msg.Topic())
		if !strings.HasPrefix(topic, deviceSetTopic+"/") {
			return fmt.Errorf("Unexpected topic: %s", topic)
		}

		// path with leading /
		path := topic[len(deviceSetTopic):]

		// use VEAP service to write PV
		if err = b.Service.WritePV(deviceVeapPath+path, pv); err != nil {
			return err
		}
		return nil
	}
	b.server.Subscribe(deviceSetTopic+"/+/+/+", message.QosExactlyOnce, &b.onSetDevice)

	// adapt VEAP system variables
	b.sysVarAdapter = &vadapter{
		mqttTopic:   sysVarTopic,
		veapPath:    sysVarVeapPath,
		readBackDur: sysVarReadBackDur,
		mqttBroker:  b,
		veapService: b.Service,
	}
	b.sysVarAdapter.start()

	// adapt VEAP programs
	b.prgAdapter = &vadapter{
		mqttTopic:   prgTopic,
		veapPath:    prgVeapPath,
		mqttBroker:  b,
		veapService: b.Service,
	}
	b.prgAdapter.start()
}

// Stop stops the MQTT broker.
func (b *Broker) Stop() {
	// stop adapter
	b.prgAdapter.stop()
	b.sysVarAdapter.stop()

	// stop system variable reader
	close(b.stopSVReader)
	<-b.doneSVReader

	// stop broker
	log.Debugf("Stopping MQTT broker")
	b.server.Unsubscribe(deviceSetTopic+"/+/+/+", &b.onSetDevice)
	_ = b.server.Close()

	// wait for stop
	b.doneServer.Wait()
}

// PublishPV publishes a PV.
func (b *Broker) PublishPV(topic string, pv veap.PV, qos byte, retain bool) error {
	pl, err := pvToWire(pv)
	if err != nil {
		return err
	}
	if err := b.Publish(topic, pl, qos, retain); err != nil {
		return err
	}
	return nil
}

// Publish publishes a generic payload.
func (b *Broker) Publish(topic string, payload []byte, qos byte, retain bool) error {
	log.Tracef("Publishing %s: %s", topic, string(payload))
	pm := message.NewPublishMessage()
	if err := pm.SetTopic([]byte(topic)); err != nil {
		return fmt.Errorf("Invalid topic: %v", err)
	}
	if err := pm.SetQoS(qos); err != nil {
		return fmt.Errorf("Invalid QoS: %v", err)
	}
	pm.SetRetain(retain)
	pm.SetPayload(payload)
	if err := b.server.Publish(pm); err != nil {
		return fmt.Errorf("Publish failed: %v", err)
	}
	return nil
}

// Subscribe subscribes a topic.
func (b *Broker) Subscribe(topic string, qos byte, onPublish *service.OnPublishFunc) error {
	return b.server.Subscribe(topic, qos, onPublish)
}

// Unsubscribe unsubscribes a topic.
func (b *Broker) Unsubscribe(topic string, onPublish *service.OnPublishFunc) error {
	return b.server.Unsubscribe(topic, onPublish)
}

func (b *Broker) startSysVarReader() {
	log.Debug("Starting system variable reader")
	b.stopSVReader = make(chan struct{})
	b.doneSVReader = make(chan struct{})
	go func() {
		// defer clean up
		defer func() {
			log.Debug("Stopping system variable reader")
			b.doneSVReader <- struct{}{}
		}()

		// PV cache
		pvCache := make(map[string]veap.PV)

		for {
			// get list of system variables
			_, links, err := b.Service.ReadProperties(sysVarVeapPath)
			if err != nil {
				log.Errorf("System variable reader: %v", err)
				return
			}

			// get attributes of each system variable
			sleepDone := false
			for _, l := range links {
				if l.Role == "sysvar" {
					p := path.Join(sysVarVeapPath, l.Target)
					attrs, _, err := b.Service.ReadProperties(p)
					if err != nil {
						log.Errorf("System variable reader: %v", err)
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

						// read PV
						pv, err := b.Service.ReadPV(p)
						if err != nil {
							log.Errorf("System variable reader: %v", err)
						} else {

							// PV changed?
							prevPV, ok := pvCache[l.Target]
							if !ok || !pv.Equal(prevPV) {

								// publish PV
								topic := sysVarTopic + "/status" + p[len(sysVarVeapPath):]
								if err := b.PublishPV(topic, pv, message.QosAtLeastOnce, true); err != nil {
									log.Errorf("System variable reader: %v", err)
								} else {
									pvCache[l.Target] = pv
								}
							}
						}
						if sleep(b.stopSVReader, sysVarReadCycle) == errStop {
							return
						}
						sleepDone = true
					}
				}
			}

			// sleep if no system variables found
			if !sleepDone {
				if sleep(b.stopSVReader, sysVarReadCycle) == errStop {
					return
				}
			}
		}
	}()
}

var errStop = errors.New("Stop request")

func sleep(stop <-chan struct{}, duration time.Duration) error {
	select {
	case <-stop:
		return errStop
	case <-time.After(duration):
		return nil
	}
}

type wirePV struct {
	Time  int64       `json:"ts"`
	Value interface{} `json:"v"`
	State veap.State  `json:"s"`
}

var errUnexpectetContent = errors.New("Unexpectet content")

func wireToPV(payload []byte) (veap.PV, error) {
	// try to convert JSON to wirePV
	var w wirePV
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	err := dec.Decode(&w)
	if err == nil {
		// check for unexpected content
		c, err2 := ioutil.ReadAll(dec.Buffered())
		if err2 != nil {
			return veap.PV{}, fmt.Errorf("ReadAll failed: %v", err)
		}
		// allow only white space
		cs := strings.TrimSpace(string(c))
		if len(cs) != 0 {
			err = errUnexpectetContent
		}
	}

	// if parsing failed, take whole payload as JSON value
	if err != nil {
		var v interface{}
		err = json.Unmarshal(payload, &v)
		if err == nil {
			w = wirePV{Value: v}
		} else {
			// if no valid JSON content is found, use the whole payload as string
			w = wirePV{Value: string(payload)}
		}
	}

	// if no timestamp is provided, use current time
	var ts time.Time
	if w.Time == 0 {
		ts = time.Now()
	} else {
		ts = time.Unix(0, w.Time*1000000)
	}

	// if no state is provided, state is implicit GOOD
	return veap.PV{
		Time:  ts,
		Value: w.Value,
		State: w.State,
	}, nil
}

func pvToWire(pv veap.PV) ([]byte, error) {
	var w wirePV
	w.Time = pv.Time.UnixNano() / 1000000
	w.Value = pv.Value
	w.State = pv.State
	pl, err := json.Marshal(w)
	if err != nil {
		return nil, fmt.Errorf("Conversion of PV to JSON failed: %v", err)
	}
	return pl, nil
}
