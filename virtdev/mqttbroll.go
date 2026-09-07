package virtdev

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-mqtt/service"
)

// HmIP-BROLL ACTIVITY_STATE enum values, shared by SHUTTER_TRANSMITTER (:3)
// and SHUTTER_VIRTUAL_RECEIVER (:4-:6) channels.
const (
	blindActivityUnknown = 0
	blindActivityUp      = 1
	blindActivityDown    = 2
	blindActivityStable  = 3
)

// ---- SHUTTER_TRANSMITTER (channel :3) ------------------------------------

// blindChannel bildet den SHUTTER_TRANSMITTER-Kanal (:3) ab, der Position und
// Aktivitätsstatus des Rollladens meldet.
type blindChannel struct {
	vdevices.Channel
	OnSetLevel    func(value float64) (ok bool)
	OnSetStop     func() (ok bool)
	level         *vdevices.FloatParameter
	stop          *vdevices.BoolParameter
	activityState *vdevices.IntParameter
	levelStatus   *vdevices.IntParameter
	sectionStatus *vdevices.IntParameter
	section       *vdevices.IntParameter
	process       *vdevices.IntParameter
}

// newBlindChannel erstellt den SHUTTER_TRANSMITTER-Kanal (:3), der die
// tatsächliche Rollladenposition sendet und den Statuskanal des Geräts bildet.
func newBlindChannel(device *vdevices.Device) *blindChannel {
	c := new(blindChannel)
	c.Channel.Init("SHUTTER_TRANSMITTER")
	device.AddChannel(&c.Channel)

	c.level = vdevices.NewFloatParameter("LEVEL")
	c.level.Description().Control = "SHUTTER_TRANSMITTER.LEVEL"
	c.level.Description().TabOrder = 0
	c.level.Description().Default = 0.0
	c.level.Description().Min = 0.0
	c.level.Description().Max = 1.0
	c.level.Description().Unit = "100%"
	c.level.OnSetValue = func(value float64) bool {
		if c.OnSetLevel != nil {
			return c.OnSetLevel(value)
		}
		return true
	}
	c.Channel.AddValueParam(c.level)

	c.stop = vdevices.NewBoolParameter("STOP")
	c.stop.Description().Type = itf.ParameterTypeAction
	c.stop.Description().Operations = itf.ParameterOperationWrite
	c.stop.Description().Control = "SHUTTER_TRANSMITTER.STOP"
	c.stop.Description().TabOrder = 1
	c.stop.OnSetValue = func(value bool) bool {
		if c.OnSetStop != nil {
			return c.OnSetStop()
		}
		return true
	}
	c.Channel.AddValueParam(c.stop)

	c.activityState = vdevices.NewIntParameter("ACTIVITY_STATE")
	c.activityState.Description().Type = itf.ParameterTypeEnum
	c.activityState.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	c.activityState.Description().Control = "SHUTTER_TRANSMITTER.ACTIVITY_STATE"
	c.activityState.Description().TabOrder = 2
	c.activityState.Description().Default = "UNKNOWN"
	c.activityState.Description().Min = "UNKNOWN"
	c.activityState.Description().Max = "STABLE"
	c.activityState.Description().ValueList = []string{"UNKNOWN", "UP", "DOWN", "STABLE"}
	c.Channel.AddValueParam(c.activityState)

	c.levelStatus = vdevices.NewIntParameter("LEVEL_STATUS")
	c.levelStatus.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	c.levelStatus.Description().TabOrder = 3
	c.Channel.AddValueParam(c.levelStatus)

	// SECTION_STATUS=1: combined/transmitter channel
	c.sectionStatus = vdevices.NewIntParameter("SECTION_STATUS")
	c.sectionStatus.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	c.sectionStatus.Description().Default = 1
	c.sectionStatus.InternalSetValue(1)
	c.sectionStatus.Description().TabOrder = 4
	c.Channel.AddValueParam(c.sectionStatus)

	c.section = vdevices.NewIntParameter("SECTION")
	c.section.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	c.section.Description().TabOrder = 5
	c.Channel.AddValueParam(c.section)

	c.process = vdevices.NewIntParameter("PROCESS")
	c.process.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	c.process.Description().TabOrder = 6
	c.Channel.AddValueParam(c.process)

	return c
}

// SetLevel / Level / SetActivityState / ActivityState sind interne Setzer bzw.
// Getter für Position und Aktivitätsstatus des SHUTTER_TRANSMITTER-Kanals (:3).
func (c *blindChannel) SetLevel(value float64) { c.level.InternalSetValue(value) }
func (c *blindChannel) Level() float64         { return c.level.Value().(float64) }
func (c *blindChannel) SetActivityState(v int) { c.activityState.InternalSetValue(v) }
func (c *blindChannel) ActivityState() int     { return c.activityState.Value().(int) }

// ---- SHUTTER_VIRTUAL_RECEIVER (channels :4-:6) ---------------------------

// shutterReceiverChannel bildet einen SHUTTER_VIRTUAL_RECEIVER-Kanal (:4-:6)
// ab, über den externe Systeme unabhängige Fahrbefehle absetzen können.
type shutterReceiverChannel struct {
	vdevices.Channel
	OnSetLevel    func(value float64) (ok bool)
	OnSetStop     func() (ok bool)
	level         *vdevices.FloatParameter
	stop          *vdevices.BoolParameter
	activityState *vdevices.IntParameter
	levelStatus   *vdevices.IntParameter
	sectionStatus *vdevices.IntParameter
	section       *vdevices.IntParameter
	process       *vdevices.IntParameter
}

// newShutterReceiverChannel erstellt einen SHUTTER_VIRTUAL_RECEIVER-Kanal
// (:4-:6), über den externe Systeme unabhängige Fahrbefehle empfangen können.
func newShutterReceiverChannel(device *vdevices.Device) *shutterReceiverChannel {
	c := new(shutterReceiverChannel)
	c.Channel.Init("SHUTTER_VIRTUAL_RECEIVER")
	c.Channel.Description().Direction = itf.DeviceDirectionReceiver
	c.Channel.Description().LinkTargetRoles = "LEVEL CONDITIONAL_SWITCH REMOTE_CONTROL SWITCH"
	device.AddChannel(&c.Channel)

	c.level = vdevices.NewFloatParameter("LEVEL")
	c.level.Description().Control = "SHUTTER_VIRTUAL_RECEIVER.LEVEL"
	c.level.Description().TabOrder = 0
	c.level.Description().Default = 0.0
	c.level.Description().Min = 0.0
	c.level.Description().Max = 1.0
	c.level.Description().Unit = "100%"
	c.level.OnSetValue = func(value float64) bool {
		if c.OnSetLevel != nil {
			return c.OnSetLevel(value)
		}
		return true
	}
	c.Channel.AddValueParam(c.level)

	c.stop = vdevices.NewBoolParameter("STOP")
	c.stop.Description().Type = itf.ParameterTypeAction
	c.stop.Description().Operations = itf.ParameterOperationWrite
	c.stop.Description().Control = "SHUTTER_VIRTUAL_RECEIVER.STOP"
	c.stop.Description().TabOrder = 1
	c.stop.OnSetValue = func(value bool) bool {
		if c.OnSetStop != nil {
			return c.OnSetStop()
		}
		return true
	}
	c.Channel.AddValueParam(c.stop)

	c.activityState = vdevices.NewIntParameter("ACTIVITY_STATE")
	c.activityState.Description().Type = itf.ParameterTypeEnum
	c.activityState.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	c.activityState.Description().Control = "SHUTTER_VIRTUAL_RECEIVER.ACTIVITY_STATE"
	c.activityState.Description().TabOrder = 2
	c.activityState.Description().Default = "UNKNOWN"
	c.activityState.Description().Min = "UNKNOWN"
	c.activityState.Description().Max = "STABLE"
	c.activityState.Description().ValueList = []string{"UNKNOWN", "UP", "DOWN", "STABLE"}
	c.Channel.AddValueParam(c.activityState)

	c.levelStatus = vdevices.NewIntParameter("LEVEL_STATUS")
	c.levelStatus.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	c.levelStatus.Description().Min = 0
	c.levelStatus.Description().Max = 3
	c.levelStatus.Description().Default = 0
	c.levelStatus.Description().TabOrder = 3
	c.Channel.AddValueParam(c.levelStatus)

	// SECTION_STATUS=0: virtual receiver, no section grouping
	c.sectionStatus = vdevices.NewIntParameter("SECTION_STATUS")
	c.sectionStatus.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	c.sectionStatus.Description().Min = 0
	c.sectionStatus.Description().Max = 1
	c.sectionStatus.Description().Default = 0
	c.sectionStatus.Description().TabOrder = 4
	c.Channel.AddValueParam(c.sectionStatus)

	c.section = vdevices.NewIntParameter("SECTION")
	c.section.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	c.section.Description().TabOrder = 5
	c.Channel.AddValueParam(c.section)

	c.process = vdevices.NewIntParameter("PROCESS")
	c.process.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	c.process.Description().TabOrder = 6
	c.Channel.AddValueParam(c.process)

	return c
}

// SetLevel / Level / SetActivityState sind interne Setzer bzw. Getter für
// Position und Aktivitätsstatus eines SHUTTER_VIRTUAL_RECEIVER-Kanals (:4-:6).
func (c *shutterReceiverChannel) SetLevel(v float64)     { c.level.InternalSetValue(v) }
func (c *shutterReceiverChannel) Level() float64         { return c.level.Value().(float64) }
func (c *shutterReceiverChannel) SetActivityState(v int) { c.activityState.InternalSetValue(v) }

// ---- BLIND_WEEK_PROFILE (channel :7) -------------------------------------

// blindWeekProfileChannel bildet den Wochenprofil-Kanal (:7) des HmIP-BROLL ab.
type blindWeekProfileChannel struct{ baseChannel }

// addWPSlot erzeugt alle 9 WP-Parameter eines Slots (01..75) im MASTER-Paramset
// (Bedingung, Astro-Typ/-Offset, Uhrzeit, Level, Ziel-Kanäle und Wochentag).
func addWPSlot(ch *vdevices.Channel, slot int) {
	nn := fmt.Sprintf("%02d", slot)

	cond := vdevices.NewIntParameter(nn + "_WP_CONDITION")
	cond.Description().Type = itf.ParameterTypeEnum
	cond.Description().Default = "FIXED"
	cond.Description().Min = "FIXED"
	cond.Description().Max = "LATEST_OF_FIXED_AND_ASTRO"
	cond.Description().ValueList = []string{
		"FIXED",
		"ASTRO",
		"FIXED_IF_BEFORE_ASTRO",
		"ASTRO_IF_BEFORE_FIXED",
		"FIXED_IF_AFTER_ASTRO",
		"ASTRO_IF_AFTER_FIXED",
		"EARLIEST_OF_FIXED_AND_ASTRO",
		"LATEST_OF_FIXED_AND_ASTRO",
	}
	cond.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationWrite
	ch.AddMasterParam(cond)

	astroType := vdevices.NewIntParameter(nn + "_WP_ASTRO_TYPE")
	astroType.Description().Type = itf.ParameterTypeEnum
	astroType.Description().Default = "SUNRISE"
	astroType.Description().Min = "SUNRISE"
	astroType.Description().Max = "SUNSET"
	astroType.Description().ValueList = []string{"SUNRISE", "SUNSET"}
	astroType.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationWrite
	ch.AddMasterParam(astroType)

	astroOffset := vdevices.NewIntParameter(nn + "_WP_ASTRO_OFFSET")
	astroOffset.Description().Min = -128
	astroOffset.Description().Max = 127
	astroOffset.Description().Default = 0
	astroOffset.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationWrite
	ch.AddMasterParam(astroOffset)

	fixedHour := vdevices.NewIntParameter(nn + "_WP_FIXED_HOUR")
	fixedHour.Description().Min = 0
	fixedHour.Description().Max = 23
	fixedHour.Description().Default = 0
	fixedHour.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationWrite
	ch.AddMasterParam(fixedHour)

	fixedMinute := vdevices.NewIntParameter(nn + "_WP_FIXED_MINUTE")
	fixedMinute.Description().Min = 0
	fixedMinute.Description().Max = 59
	fixedMinute.Description().Default = 0
	fixedMinute.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationWrite
	ch.AddMasterParam(fixedMinute)

	level := vdevices.NewFloatParameter(nn + "_WP_LEVEL")
	level.Description().Min = 0.0
	level.Description().Max = 1.01
	level.Description().Default = 0.0
	level.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationWrite
	ch.AddMasterParam(level)

	level2 := vdevices.NewFloatParameter(nn + "_WP_LEVEL_2")
	level2.Description().Min = 0.0
	level2.Description().Max = 1.01
	level2.Description().Default = 0.0
	level2.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationWrite
	level2.InternalSetValue(1.01) // native firmware stores 1.010000 for all slots
	ch.AddMasterParam(level2)

	targets := vdevices.NewIntParameter(nn + "_WP_TARGET_CHANNELS")
	targets.Description().Min = 0
	targets.Description().Max = 16777215
	targets.Description().Default = 0
	targets.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationWrite
	if slot == 1 {
		targets.InternalSetValue(1) // native firmware: 01_WP_TARGET_CHANNELS=1, all others=0
	}
	ch.AddMasterParam(targets)

	weekday := vdevices.NewIntParameter(nn + "_WP_WEEKDAY")
	weekday.Description().Min = 0
	weekday.Description().Max = 127
	weekday.Description().Default = 0
	weekday.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationWrite
	ch.AddMasterParam(weekday)
}

// newBlindWeekProfileChannel erstellt den BLIND_WEEK_PROFILE-Kanal (:7) mit
// 75 Wochenprofil-Slots (MASTER) sowie den VALUE-Parametern für State/Control.
func newBlindWeekProfileChannel(dev *vdevices.Device, vd *VirtualDevices) *blindWeekProfileChannel {
	c := new(blindWeekProfileChannel)
	c.virtualDevices = vd
	c.device = dev

	ch := new(vdevices.Channel)
	ch.Init("BLIND_WEEK_PROFILE")
	ch.Description().Version = 5
	dev.AddChannel(ch)
	c.GenericChannel = ch

	// 75 Slots (01..75)
	// Note: ACTIVE_PROFILE, WEEK_PROGRAM_POINTER, COMBINED_PARAMETER, WEEK_PROGRAM_CHANNEL_LOCK
	// are intentionally omitted: native getParamset(MASTER) does not return them in VALUES.
	for i := 1; i <= 75; i++ {
		addWPSlot(ch, i)
	}

	// VALUE-Parameter für State & Control / Weekly-UI
	state := vdevices.NewBoolParameter("STATE")
	state.Description().Control = "BLIND_WEEK_PROFILE.STATE"
	state.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent | itf.ParameterOperationWrite
	ch.AddValueParam(state)

	combinedValue := vdevices.NewStringParameter("COMBINED_PARAMETER")
	combinedValue.Description().Operations = itf.ParameterOperationWrite
	ch.AddValueParam(combinedValue)

	channelLocks := vdevices.NewIntParameter("WEEK_PROGRAM_CHANNEL_LOCKS")
	channelLocks.Description().Min = 0
	channelLocks.Description().Max = 16777215
	channelLocks.Description().Default = 0
	channelLocks.Description().Control = "WEEK_PROFILE.CHANNEL_LOCKS"
	channelLocks.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	ch.AddValueParam(channelLocks)

	targetLock := vdevices.NewStringParameter("WEEK_PROGRAM_TARGET_CHANNEL_LOCK")
	targetLock.Description().Type = itf.ParameterTypeEnum
	targetLock.Description().Default = "MANU_MODE"
	targetLock.Description().Min = "MANU_MODE"
	targetLock.Description().Max = "AUTO_MODE_WITHOUT_RESET"
	targetLock.Description().ValueList = []string{
		"MANU_MODE",
		"AUTO_MODE_WITH_RESET",
		"AUTO_MODE_WITHOUT_RESET",
	}
	targetLock.Description().Control = "WEEK_PROFILE.TARGET_CHANNEL_LOCK"
	targetLock.Description().Operations = itf.ParameterOperationWrite
	ch.AddValueParam(targetLock)

	targetLocks := vdevices.NewIntParameter("WEEK_PROGRAM_TARGET_CHANNEL_LOCKS")
	targetLocks.Description().Min = 0
	targetLocks.Description().Max = 16777215
	targetLocks.Description().Default = 0
	targetLocks.Description().Control = "WEEK_PROFILE.TARGET_CHANNEL_LOCKS"
	targetLocks.Description().Operations = itf.ParameterOperationWrite
	ch.AddValueParam(targetLocks)

	return c
}

// ---- MQTT_BROLL virtual device -------------------------------------------

// mqttBRoll bündelt alle Kanäle des virtuellen HmIP-BROLL-Geräts und die
// zugehörige MQTT-Bridge-Konfiguration (MASTER-Params auf Kanal :3).
type mqttBRoll struct {
	baseChannel
	shutterTx   *blindChannel
	shutterRx   *shutterReceiverChannel
	shutterRx2  *shutterReceiverChannel
	shutterRx3  *shutterReceiverChannel
	keyDown     *vdevices.KeyChannel
	keyUp       *vdevices.KeyChannel
	weekProfile *blindWeekProfileChannel

	subscribedTopic string
	onPublish       service.OnPublishFunc
	template        *template.Template

	paramRangeMin      *vdevices.FloatParameter
	paramRangeMax      *vdevices.FloatParameter
	paramCommandTopic  *vdevices.StringParameter
	paramRetain        *vdevices.BoolParameter
	paramTemplate      *vdevices.StringParameter
	paramStopTopic     *vdevices.StringParameter
	paramStopPayload   *vdevices.StringParameter
	paramFBTopic       *vdevices.StringParameter
	paramPattern       *vdevices.StringParameter
	paramExtractorKind *vdevices.IntParameter
	paramRegexpGroup   *vdevices.IntParameter
}

// activityFromLevelChange derives ACTIVITY_STATE from a position change.
func activityFromLevelChange(newLevel, prevLevel float64) int {
	const threshold = 0.005
	if newLevel > prevLevel+threshold {
		return blindActivityDown
	}
	if newLevel < prevLevel-threshold {
		return blindActivityUp
	}
	return blindActivityStable
}

// setAllActivityStates syncs ACTIVITY_STATE on all shutter channels.
func (c *mqttBRoll) setAllActivityStates(state int) {
	c.shutterTx.SetActivityState(state)
	c.shutterRx.SetActivityState(state)
	c.shutterRx2.SetActivityState(state)
	c.shutterRx3.SetActivityState(state)
}

// start parst das konfigurierte TEMPLATE und abonniert – sofern gesetzt – das
// FEEDBACK_TOPIC, um externe Positionsmeldungen auf den :3-Kanal zu spiegeln.
func (c *mqttBRoll) start() {
	tmplText := c.paramTemplate.Value().(string)
	specFuncs := createSpecificFuncs(c.virtualDevices.Devices, c.device, c)
	tmpl, err := template.New("mqttbroll").Funcs(tmplFuncs).Funcs(specFuncs).Parse(tmplText)
	if err != nil {
		log.Errorf("Invalid template '%s': %v", tmplText, err)
		tmpl = nil
	}
	c.template = tmpl

	fbTopic := c.paramFBTopic.Value().(string)
	if fbTopic == "" {
		log.Debugf("MQTT BROLL %s: no FEEDBACK_TOPIC configured, subscription skipped", c.Description().Parent)
		return
	}
	cmdTopic := c.paramCommandTopic.Value().(string)
	if matchTopic(fbTopic, cmdTopic) {
		log.Errorf("Feedback topic '%s' must not overlap with command topic '%s'", fbTopic, cmdTopic)
		return
	}
	extractor, err := newExtractor(c.paramExtractorKind, c.paramPattern, c.paramRegexpGroup)
	if err != nil {
		log.Errorf("Creation of value extractor for MQTT BROLL %s:%d failed: %v",
			c.Description().Parent, c.Description().Index, err)
		return
	}
	c.onPublish = func(msg *message.PublishMessage) error {
		log.Debugf("MQTT BROLL %s: message received on topic '%s' payload=%q",
			c.Description().Parent, msg.Topic(), msg.Payload())
		value, err := extractor.Extract(msg.Payload())
		if err != nil {
			log.Warningf("MQTT BROLL %s: value extraction from payload %q failed: %v",
				c.Description().Parent, msg.Payload(), err)
			return nil
		}
		newLevel := c.mapFromRange(value)
		actState := activityFromLevelChange(newLevel, c.shutterTx.Level())
		log.Debugf("MQTT BROLL %s: extracted value=%g -> LEVEL=%g, ACTIVITY_STATE=%d (updating :3 SHUTTER_TRANSMITTER)",
			c.Description().Parent, value, newLevel, actState)
		// Feedback: update only transmitter channel (:3).
		c.shutterTx.SetLevel(newLevel)
		c.shutterTx.SetActivityState(actState)
		return nil
	}
	if err := c.virtualDevices.MQTTServer.Subscribe(fbTopic, message.QosExactlyOnce, &c.onPublish); err != nil {
		log.Errorf("MQTT BROLL %s: subscribe failed on topic '%s': %v", c.Description().Parent, fbTopic, err)
	} else {
		c.subscribedTopic = fbTopic
		log.Debugf("MQTT BROLL %s: subscribed to FEEDBACK_TOPIC '%s' (extractor=%v, pattern=%q, group=%d)",
			c.Description().Parent, fbTopic,
			c.paramExtractorKind.Value(), c.paramPattern.Value(), c.paramRegexpGroup.Value())
	}
}

// stop hebt ein bestehendes FEEDBACK_TOPIC-Abonnement wieder auf.
func (c *mqttBRoll) stop() {
	if c.subscribedTopic != "" {
		log.Debugf("MQTT BROLL %s: unsubscribing from FEEDBACK_TOPIC '%s'", c.Description().Parent, c.subscribedTopic)
		c.virtualDevices.MQTTServer.Unsubscribe(c.subscribedTopic, &c.onPublish)
		c.subscribedTopic = ""
	}
}

// mapToRange bildet einen internen Level (0..1) auf den konfigurierten
// RANGE_MIN..RANGE_MAX-Bereich für die ausgehende MQTT-Nachricht ab.
func (c *mqttBRoll) mapToRange(value float64) float64 {
	min := c.paramRangeMin.Value().(float64)
	max := c.paramRangeMax.Value().(float64)
	return value*(max-min) + min
}

// mapFromRange bildet einen aus MQTT extrahierten Wert aus dem
// RANGE_MIN..RANGE_MAX-Bereich zurück auf einen internen Level (0..1) ab.
func (c *mqttBRoll) mapFromRange(value float64) float64 {
	min := c.paramRangeMin.Value().(float64)
	max := c.paramRangeMax.Value().(float64)
	if min == max {
		return 0.0
	}
	out := (value - min) / (max - min)
	if out < 0.0 {
		out = 0.0
	}
	if out > 1.0 {
		out = 1.0
	}
	return out
}

// publishLevelToMQTT rendert das TEMPLATE mit dem gemappten Level und
// veröffentlicht das Ergebnis auf dem COMMAND_TOPIC.
func (c *mqttBRoll) publishLevelToMQTT(value float64) {
	if c.template == nil {
		log.Warningf("Invalid template: %s", c.paramTemplate.Value().(string))
		return
	}
	mappedValue := c.mapToRange(value)
	var buf bytes.Buffer
	err := c.template.Execute(&buf, mappedValue)
	if err != nil {
		log.Errorf("Execution of template '%s' failed for value %g: %v",
			c.paramTemplate.Value().(string), mappedValue, err)
		return
	}
	cmdTopic := c.paramCommandTopic.Value().(string)
	log.Debugf("MQTT BROLL %s: publishing LEVEL=%g -> mapped=%g to COMMAND_TOPIC '%s' payload=%q retain=%v",
		c.Description().Parent, value, mappedValue, cmdTopic, buf.Bytes(), c.paramRetain.Value().(bool))
	c.virtualDevices.MQTTServer.Publish(
		cmdTopic,
		buf.Bytes(),
		message.QosExactlyOnce,
		c.paramRetain.Value().(bool),
	)
}

// publishStopToMQTT veröffentlicht das STOP_PAYLOAD auf dem STOP_TOPIC, sofern
// ein STOP_TOPIC konfiguriert ist.
func (c *mqttBRoll) publishStopToMQTT() {
	stopTopic := c.paramStopTopic.Value().(string)
	if stopTopic == "" {
		log.Debugf("MQTT BROLL %s: STOP requested but no STOP_TOPIC configured", c.Description().Parent)
		return
	}
	log.Debugf("MQTT BROLL %s: publishing STOP to STOP_TOPIC '%s' payload=%q retain=%v",
		c.Description().Parent, stopTopic, c.paramStopPayload.Value().(string), c.paramRetain.Value().(bool))
	c.virtualDevices.MQTTServer.Publish(
		stopTopic,
		[]byte(c.paramStopPayload.Value().(string)),
		message.QosExactlyOnce,
		c.paramRetain.Value().(bool),
	)
}

// addMQTTBRoll baut das komplette virtuelle HmIP-BROLL-Gerät zusammen. Die
// Kanäle werden in aufsteigender Reihenfolge instanziiert, damit
// vdevices.AddChannel die Indizes korrekt vergibt: :1 (KEY DOWN), :2 (KEY UP),
// :3 (SHUTTER_TRANSMITTER), :4-:6 (SHUTTER_VIRTUAL_RECEIVER) und
// :7 (BLIND_WEEK_PROFILE). :0 (MAINTENANCE) wird bereits in createDevice angelegt.
func (vd *VirtualDevices) addMQTTBRoll(dev *vdevices.Device) vdevices.GenericChannel {
	ch := new(mqttBRoll)
	ch.virtualDevices = vd
	ch.device = dev

	// Config has one entry (MQTT_BROLL at index 0). Override automatic
	// index calculation (channel.Index-1 = 2) to always use config index 0.
	idx := 0
	ch.configChannelIndex = &idx

	// KeyChannel helpers for :1 KEY_TRANSCEIVER (Down) and :2 KEY_TRANSCEIVER (Up)
	configureKeyChannel := func(ch *vdevices.KeyChannel) {
		ch.Description().Direction = itf.DeviceDirectionSender
		ch.Description().Version = 15
		ch.Description().LinkSourceRoles = "REMOTE_CONTROL"
		ch.Description().Flags = itf.DeviceFlagVisible

		for _, id := range []string{
			"PRESS_SHORT",
			"PRESS_LONG",
			"INSTALL_TEST",
		} {
			if p, err := ch.ValueParamset().Parameter(id); err == nil {
				p.Description().Operations = 0
				p.Description().Flags = itf.ParameterFlagInternal
			}
		}
	}

	// :1 KEY_TRANSCEIVER (Down)
	ch.keyDown = vdevices.NewKeyChannel(dev)
	configureKeyChannel(ch.keyDown)

	// :2 KEY_TRANSCEIVER (Up)
	ch.keyUp = vdevices.NewKeyChannel(dev)
	configureKeyChannel(ch.keyUp)

	// :3 SHUTTER_TRANSMITTER
	ch.shutterTx = newBlindChannel(dev)
	ch.GenericChannel = ch.shutterTx
	ch.shutterTx.Description().Version = 5

	// :4-:6 SHUTTER_VIRTUAL_RECEIVER (independent control paths)
	ch.shutterRx = newShutterReceiverChannel(dev)
	ch.shutterRx2 = newShutterReceiverChannel(dev)
	ch.shutterRx3 = newShutterReceiverChannel(dev)
	// Set identical Version and Roles for all three receivers
	receivers := []*shutterReceiverChannel{ch.shutterRx, ch.shutterRx2, ch.shutterRx3}
	for _, rx := range receivers {
		rx.Description().Version = 5
		rx.Description().LinkTargetRoles = "SWITCH CONDITIONAL_SWITCH LEVEL REMOTE_CONTROL"
	}

	// :7 BLIND_WEEK_PROFILE
	ch.weekProfile = newBlindWeekProfileChannel(dev, vd)
	// ch.weekProfile.configChannelIndex = &idx

	// --- Master parameters on channel :3 ---
	ch.paramRangeMin = vdevices.NewFloatParameter("RANGE_MIN")
	ch.paramRangeMin.Description().Default = 100.0
	ch.paramRangeMin.InternalSetValue(100.0)
	ch.AddMasterParam(ch.paramRangeMin)

	ch.paramRangeMax = vdevices.NewFloatParameter("RANGE_MAX")
	ch.paramRangeMax.Description().Default = 0.0
	ch.paramRangeMax.InternalSetValue(0.0)
	ch.AddMasterParam(ch.paramRangeMax)

	ch.paramCommandTopic = vdevices.NewStringParameter("COMMAND_TOPIC")
	ch.AddMasterParam(ch.paramCommandTopic)

	ch.paramRetain = vdevices.NewBoolParameter("RETAIN")
	ch.AddMasterParam(ch.paramRetain)

	ch.paramTemplate = vdevices.NewStringParameter("TEMPLATE")
	ch.paramTemplate.Description().Default = "{{ . }}"
	ch.paramTemplate.InternalSetValue("{{ . }}")
	ch.AddMasterParam(ch.paramTemplate)

	ch.paramStopTopic = vdevices.NewStringParameter("STOP_TOPIC")
	ch.AddMasterParam(ch.paramStopTopic)

	ch.paramStopPayload = vdevices.NewStringParameter("STOP_PAYLOAD")
	ch.paramStopPayload.Description().Default = "STOP"
	ch.paramStopPayload.InternalSetValue("STOP")
	ch.AddMasterParam(ch.paramStopPayload)

	ch.paramFBTopic = vdevices.NewStringParameter("FEEDBACK_TOPIC")
	ch.AddMasterParam(ch.paramFBTopic)

	ch.paramPattern = vdevices.NewStringParameter("PATTERN")
	ch.paramPattern.Description().Default = "[0-9]+(\\.[0-9]+)?"
	ch.paramPattern.InternalSetValue("[0-9]+(\\.[0-9]+)?")
	ch.AddMasterParam(ch.paramPattern)

	ch.paramExtractorKind = newExtractorKindParameter("EXTRACTOR")
	ch.paramExtractorKind.Description().Default = int(ExtractorRegexp)
	ch.paramExtractorKind.InternalSetValue(int(ExtractorRegexp))
	ch.AddMasterParam(ch.paramExtractorKind)

	ch.paramRegexpGroup = vdevices.NewIntParameter("REGEXP_GROUP")
	ch.paramRegexpGroup.Description().Min = 0
	ch.paramRegexpGroup.Description().Max = 100
	ch.paramRegexpGroup.Description().Default = 0
	ch.AddMasterParam(ch.paramRegexpGroup)

	onStop := func() bool {
		ch.setAllActivityStates(blindActivityStable)
		ch.publishStopToMQTT()
		return true
	}

	// Kanal :1 (DOWN) – Kurzdruck -> Level=1.0
	ch.keyDown.OnPressShort = func() bool {
		ch.shutterTx.SetLevel(1.0)
		ch.shutterTx.SetActivityState(blindActivityDown)
		ch.publishLevelToMQTT(1.0)
		return true
	}

	// Kanal :2 (UP) – Kurzdruck -> Level=0.0
	ch.keyUp.OnPressShort = func() bool {
		ch.shutterTx.SetLevel(0.0)
		ch.shutterTx.SetActivityState(blindActivityUp)
		ch.publishLevelToMQTT(0.0)
		return true
	}

	// Kanal :3 (SHUTTER_TRANSMITTER)
	ch.shutterTx.OnSetLevel = func(value float64) bool {
		actState := activityFromLevelChange(value, ch.shutterTx.Level())
		ch.shutterTx.SetLevel(value)
		ch.shutterTx.SetActivityState(actState)
		ch.publishLevelToMQTT(value)
		return true
	}
	ch.shutterTx.OnSetStop = onStop

	// Kanäle :4-:6 (SHUTTER_VIRTUAL_RECEIVER)
	bindReceiverLevel := func(rx *shutterReceiverChannel) {
		rx.OnSetLevel = func(value float64) bool {
			actState := activityFromLevelChange(value, ch.shutterTx.Level())
			ch.shutterTx.SetLevel(value)
			ch.shutterTx.SetActivityState(actState)
			rx.SetLevel(value)
			rx.SetActivityState(actState)
			ch.publishLevelToMQTT(value)
			return true
		}
	}
	bindReceiverLevel(ch.shutterRx)
	bindReceiverLevel(ch.shutterRx2)
	bindReceiverLevel(ch.shutterRx3)
	ch.shutterRx.OnSetStop = onStop
	ch.shutterRx2.OnSetStop = onStop
	ch.shutterRx3.OnSetStop = onStop

	// Start/Load-Block for Channel 3
	ch.MasterParamset().HandlePutParamset(func() {
		ch.stop()
		ch.storeMasterParamset()
		ch.start()
	})
	ch.loadMasterParamset()
	ch.start()

	// Persistence path block for Channel 7 (BLIND_WEEK_PROFILE)
	if ch.weekProfile != nil {
		ch.weekProfile.MasterParamset().HandlePutParamset(func() {
			ch.weekProfile.storeMasterParamset()
		})
		ch.weekProfile.loadMasterParamset()
	}

	return ch
}
