package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mdzio/ccu-jack/rtcfg"
	"github.com/mdzio/go-lib/conc"
	"github.com/mdzio/go-logging"
	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-mqtt/service"
)

var logBridge = logging.Get("mqtt-bridge")

const (
	bridgeKeepAlive       = 60 * time.Second
	bridgeRecoverDuration = 60 * time.Second
)

// Bridge connects the embedded MQTT server with a remote one. Messages on
// configurable topics are exchanged between the servers.
type Bridge struct {
	EmbeddedServer *Server

	address    string
	port       int
	useTLS     bool
	caCertFile string
	insecure   bool
	bufferSize int64

	connMsg *message.ConnectMessage

	cancel func()
	in     []rtcfg.MQTTSharedTopic
	out    []rtcfg.MQTTSharedTopic
}

// Start starts the bridge with the specified configuration. The configuration
// must be locked for reading.
func (b *Bridge) Start(cfg *rtcfg.MQTTBridge) {
	// bridge enabled?
	if !cfg.Enable {
		return
	}

	// setup connection parameters
	b.address = cfg.Address
	b.port = cfg.Port
	b.useTLS = cfg.UseTLS
	b.caCertFile = cfg.CACertFile
	b.insecure = cfg.Insecure
	b.bufferSize = cfg.BufferSize

	// setup connection message
	b.connMsg = message.NewConnectMessage()
	b.connMsg.SetCleanSession(cfg.CleanSession)
	b.connMsg.SetClientID([]byte(cfg.ClientID))
	b.connMsg.SetPassword([]byte(cfg.Password))
	b.connMsg.SetUsername([]byte(cfg.Username))
	b.connMsg.SetVersion(0x4) // MQTT V3.1.1
	b.connMsg.SetKeepAlive(uint16(bridgeKeepAlive / time.Second))

	// clone shared topics
	b.in = cloneSharedTopics(cfg.Incoming)
	b.out = cloneSharedTopics(cfg.Outgoing)

	// run daemon
	b.cancel = conc.DaemonFunc(b.run)
}

func (b *Bridge) Stop() {
	// only stop, if enabled
	if b.cancel != nil {
		b.cancel()
		b.cancel = nil
	}
}

func (b *Bridge) run(ctx conc.Context) {
	logBridge.Info("Starting MQTT bridge")
	defer logBridge.Debug("Stopping MQTT bridge")
	// rerun client on errors
	for {
		if err := b.runClient(ctx); err != nil {
			logBridge.Error(err)
			if err := ctx.Sleep(bridgeRecoverDuration); err != nil {
				// bridge should stop
				return
			}
		} else {
			// bridge should stop
			return
		}
	}
}

func (b *Bridge) runClient(ctx conc.Context) error {
	// create client and connect
	client := &service.Client{
		BufferSize: b.bufferSize,
	}
	addr := "tcp://" + b.address + ":" + strconv.Itoa(b.port)
	if b.useTLS {
		logBridge.Debugf("Connecting to secure MQTT server on %s with client ID %s", addr, string(b.connMsg.ClientID()))
		tls := &tls.Config{
			ServerName: b.address,
		}
		// allow insecure connections?
		if b.insecure {
			tls.InsecureSkipVerify = true
		}
		// CA certificates provided?
		if b.caCertFile != "" {
			caCerts := x509.NewCertPool()
			data, err := os.ReadFile(b.caCertFile)
			if err != nil {
				return fmt.Errorf("Loading of CA certificates from file %s failed: %w", b.caCertFile, err)
			}
			ok := caCerts.AppendCertsFromPEM(data)
			if !ok {
				return fmt.Errorf("Loading of CA certificates from file %s failed: Invalid file format", b.caCertFile)
			}
			tls.RootCAs = caCerts
		}
		if err := client.ConnectTLS(addr, b.connMsg, tls); err != nil {
			return fmt.Errorf("Connecting to secure MQTT server on address %s failed: %w", addr, err)
		}
	} else {
		logBridge.Debugf("Connecting to MQTT server on %s with client ID %s", addr, string(b.connMsg.ClientID()))
		if err := client.Connect(addr, b.connMsg); err != nil {
			return fmt.Errorf("Connecting to MQTT server on address %s failed: %w", addr, err)
		}
	}
	defer client.Disconnect()

	// subscribe remote topics and publish local
	for _, tt := range b.in {
		t := tt // clone for callbacks
		submsg := message.NewSubscribeMessage()
		if err := submsg.AddTopic([]byte(t.RemotePrefix+t.Pattern), t.QoS); err != nil {
			return fmt.Errorf("Adding remote topic %s failed: %w", t.RemotePrefix+t.Pattern, err)
		}
		var onComplete service.OnCompleteFunc = func(msg, ack message.Message, err error) error {
			if err != nil {
				logBridge.Errorf("Subscribing remote topic %s failed: %v", t.RemotePrefix+t.Pattern, err)
			}
			return nil
		}
		var onPublish service.OnPublishFunc = func(pubmsg *message.PublishMessage) error {
			rt := string(pubmsg.Topic())
			logBridge.Tracef("Incoming remote message on topic %s with retain %t, QoS %d and payload %s", rt, pubmsg.Retain(), pubmsg.QoS(), string(pubmsg.Payload()))
			// replace topic prefix
			lt := t.LocalPrefix + strings.TrimPrefix(rt, t.RemotePrefix)
			// publish on local server
			if err := b.EmbeddedServer.Publish(lt, pubmsg.Payload(), pubmsg.QoS(), pubmsg.Retain()); err != nil {
				logBridge.Errorf("Publishing message on local topic %s failed: %v", lt, err)
			}
			return nil
		}
		if err := client.Subscribe(submsg, onComplete, onPublish); err != nil {
			return fmt.Errorf("Subscribing remote topic %s failed: %w", t.RemotePrefix+t.Pattern, err)
		}
	}

	// subscribe local topics and publish remote
	for _, tt := range b.out {
		t := tt // clone for callbacks
		var onComplete service.OnCompleteFunc = func(msg, ack message.Message, err error) error {
			if err != nil {
				logBridge.Errorf("Publishing on remote topic failed: %v", err)
			}
			return nil
		}
		var onPublish service.OnPublishFunc = func(msg *message.PublishMessage) error {
			lt := string(msg.Topic())
			logBridge.Tracef("Outgoing local message on topic %s with retain %t, QoS %d and payload %s", lt, msg.Retain(), msg.QoS(), string(msg.Payload()))
			// replace topic prefix
			rt := t.RemotePrefix + strings.TrimPrefix(lt, t.LocalPrefix)
			// publish on remote server
			pubmsg := message.NewPublishMessage()
			if err := pubmsg.SetTopic([]byte(rt)); err != nil {
				logBridge.Errorf("Invalid remote topic %s: %v", rt, err)
				return nil
			}
			pubmsg.SetPayload(msg.Payload())
			pubmsg.SetQoS(msg.QoS())
			pubmsg.SetRetain(msg.Retain())
			logBridge.Tracef("Publishing on remote topic %s: %s", rt, string(msg.Payload()))
			if err := client.Publish(pubmsg, onComplete); err != nil {
				logBridge.Errorf("Publishing message on remote topic %s failed: %v", rt, err)
			}
			return nil
		}
		if err := b.EmbeddedServer.Subscribe(t.LocalPrefix+t.Pattern, t.QoS, &onPublish); err != nil {
			return fmt.Errorf("Subscribing outgoing local topic %s failed: %w", t.LocalPrefix+t.Pattern, err)
		}
		// remove subscriptions on stop
		defer b.EmbeddedServer.Unsubscribe(t.LocalPrefix+t.Pattern, &onPublish)
	}

	// send keep alive pings
	for {
		logBridge.Trace("Sending ping")
		if err := client.Ping(nil); err != nil {
			return fmt.Errorf("Ping failed: %w", err)
		}
		if err := ctx.Sleep(bridgeKeepAlive); err != nil {
			// bridge should stop
			return nil
		}
	}
}

func cloneSharedTopics(ts []rtcfg.MQTTSharedTopic) []rtcfg.MQTTSharedTopic {
	var cts []rtcfg.MQTTSharedTopic
	for _, t := range ts {
		cts = append(cts, rtcfg.MQTTSharedTopic{
			Pattern:      t.Pattern,
			LocalPrefix:  t.LocalPrefix,
			RemotePrefix: t.RemotePrefix,
			QoS:          t.QoS,
		})
	}
	return cts
}
