package virtdev

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/mdzio/ccu-jack/mqtt"
	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-lib/any"
	"github.com/mdzio/go-mqtt/message"
	"github.com/mdzio/go-mqtt/service"
)

type mqttAnalogInHandler struct {
	channel       vdevices.GenericChannel
	targetParam   string
	mqttServer    *mqtt.Server
	valueHandler  func(value float64)
	statusHandler func(status int)

	paramTopic         *vdevices.StringParameter
	paramPattern       *vdevices.StringParameter
	paramExtractorKind *vdevices.IntParameter
	paramRegexpGroup   *vdevices.IntParameter

	subscribedTopic string
	onPublish       service.OnPublishFunc
}

func (h *mqttAnalogInHandler) init() {
	// TOPIC
	h.paramTopic = vdevices.NewStringParameter(h.targetParam + "_TOPIC")
	h.channel.AddMasterParam(h.paramTopic)

	// PATTERN
	h.paramPattern = vdevices.NewStringParameter(h.targetParam + "_PATTERN")
	h.channel.AddMasterParam(h.paramPattern)

	// EXTRACTOR
	h.paramExtractorKind = newExtractorKindParameter(h.targetParam + "_EXTRACTOR")
	h.channel.AddMasterParam(h.paramExtractorKind)

	// REGEXP_GROUP
	h.paramRegexpGroup = vdevices.NewIntParameter(h.targetParam + "_REGEXP_GROUP")
	h.paramRegexpGroup.Description().Min = 0
	h.paramRegexpGroup.Description().Max = 100
	h.paramRegexpGroup.Description().Default = 0
	h.channel.AddMasterParam(h.paramRegexpGroup)
}

func (h *mqttAnalogInHandler) start() {
	topic := h.paramTopic.Value().(string)
	if topic != "" {
		extractor, err := newExtractor(h.paramExtractorKind, h.paramPattern, h.paramRegexpGroup)
		if err != nil {
			log.Errorf("Creation of value extractor for %s:%d.%s failed: %v", h.channel.Description().Parent,
				h.channel.Description().Index, h.targetParam, err)
			return
		}
		h.onPublish = func(msg *message.PublishMessage) error {
			log.Debugf("Message for %s:%d.%s received: %s, %s", h.channel.Description().Parent,
				h.channel.Description().Index, h.targetParam, msg.Topic(), msg.Payload())
			// lock channel while modifying parameters
			h.channel.Lock()
			defer h.channel.Unlock()
			value, err := extractor.Extract(msg.Payload())
			if err != nil {
				log.Warningf("Extraction of value for analog receiver %s:%d.%s failed: %v", h.channel.Description().Parent,
					h.channel.Description().Index, h.targetParam, err)
				// set status to unknown
				h.statusHandler(1)
				return nil
			}
			h.valueHandler(value)
			// set normal status
			h.statusHandler(0)
			return nil
		}
		if err := h.mqttServer.Subscribe(topic, message.QosExactlyOnce, &h.onPublish); err != nil {
			log.Errorf("Subscribe failed on topic %s: %v", topic, err)
		} else {
			h.subscribedTopic = topic
		}
	}
}

func (h *mqttAnalogInHandler) stop() {
	if h.subscribedTopic != "" {
		h.mqttServer.Unsubscribe(h.subscribedTopic, &h.onPublish)
		h.subscribedTopic = ""
	}
}

func newExtractorKindParameter(id string) *vdevices.IntParameter {
	p := vdevices.NewIntParameter(id)
	p.Description().Type = itf.ParameterTypeEnum
	// align with extractorKind constants
	p.Description().ValueList = []string{"AFTER", "BEFORE", "REGEXP", "ALL", "TEMPLATE"}
	p.Description().Min = 0
	p.Description().Max = len(p.Description().ValueList) - 1
	p.Description().Default = 0
	return p
}

type extractorKind int

const (
	// align with function newExtractorKindParameter
	ExtractorAfter extractorKind = iota
	ExtractorBefore
	ExtractorRegexp
	ExtractorAll
	ExtractorTemplate
)

type extractor interface {
	Extract(payload []byte) (float64, error)
}

type extractorRegexp struct {
	regexp   *regexp.Regexp
	groupIdx int
}

func (e *extractorRegexp) Extract(payload []byte) (float64, error) {
	groups := e.regexp.FindStringSubmatch(string(payload))
	if groups == nil {
		return 0.0, fmt.Errorf("Regexp does not match: %s", payload)
	}
	if e.groupIdx < 0 || e.groupIdx >= len(groups) {
		return 0.0, fmt.Errorf("Invalid group index: %d", e.groupIdx)
	}
	fval, err := strconv.ParseFloat(groups[e.groupIdx], 64)
	if err != nil {
		return 0.0, fmt.Errorf("Regexp returned invalid number literal: %s", groups[e.groupIdx])
	}
	return fval, nil
}

func newExtractorRegexp(pattern string, groupIdx int) (extractor, error) {
	log.Tracef("Creating extractor with regular expression %s and group %d", pattern, groupIdx)
	regexp, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("Invalid regular expression: %s", pattern)
	}
	return &extractorRegexp{regexp: regexp, groupIdx: groupIdx}, nil
}

type extractorTmpl struct {
	tmpl *template.Template
}

func parseJSON(str string) (interface{}, error) {
	var obj interface{}
	err := json.Unmarshal([]byte(str), &obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func wrapUnaryFunc(f func(a float64) (float64, error)) func(a interface{}) (float64, error) {
	return func(a interface{}) (float64, error) {
		q := any.Q(a)
		fa := q.ToFloat64()
		if q.Err() != nil {
			return 0, q.Err()
		}
		fr, err := f(fa)
		if err != nil {
			return 0, err
		}
		return fr, nil
	}
}

func wrapBinaryFunc(f func(a, b float64) (float64, error)) func(a, b interface{}) (float64, error) {
	return func(a, b interface{}) (float64, error) {
		q := any.Q(a)
		fa := q.ToFloat64()
		if q.Err() != nil {
			return 0, q.Err()
		}
		q = any.Q(b)
		fb := q.ToFloat64()
		if q.Err() != nil {
			return 0, q.Err()
		}
		fr, err := f(fa, fb)
		if err != nil {
			return 0, err
		}
		return fr, nil
	}
}

func mapRange(inMin, inMax, outMin, outMax, value interface{}) (float64, error) {
	// convert arguments
	q := any.Q(inMin)
	inMinf := q.ToFloat64()
	if q.Err() != nil {
		return 0, q.Err()
	}
	q = any.Q(inMax)
	inMaxf := q.ToFloat64()
	if q.Err() != nil {
		return 0, q.Err()
	}
	q = any.Q(outMin)
	outMinf := q.ToFloat64()
	if q.Err() != nil {
		return 0, q.Err()
	}
	q = any.Q(outMax)
	outMaxf := q.ToFloat64()
	if q.Err() != nil {
		return 0, q.Err()
	}
	q = any.Q(value)
	valuef := q.ToFloat64()
	if q.Err() != nil {
		return 0, q.Err()
	}
	// validate arguments
	if inMinf == inMaxf || outMinf == outMaxf {
		return 0, errors.New("mapRange: invalid minimum/maximum")
	}
	// validate range
	if valuef < inMinf || valuef > inMaxf {
		return 0, fmt.Errorf("mapRange: value %g is out of range", valuef)
	}
	// map range
	u := (valuef - inMinf) / (inMaxf - inMinf)
	return outMinf + u*(outMaxf-outMinf), nil
}

var tmplFuncs = template.FuncMap{
	"contains":  strings.Contains,
	"fields":    strings.Fields,
	"split":     strings.Split,
	"toLower":   strings.ToLower,
	"toUpper":   strings.ToUpper,
	"trimSpace": strings.TrimSpace,
	"parseJSON": parseJSON,
	"round":     wrapUnaryFunc(func(a float64) (float64, error) { return math.Round(a), nil }),
	"add":       wrapBinaryFunc(func(a, b float64) (float64, error) { return a + b, nil }),
	"sub":       wrapBinaryFunc(func(a, b float64) (float64, error) { return a - b, nil }),
	"mul":       wrapBinaryFunc(func(a, b float64) (float64, error) { return a * b, nil }),
	"div":       wrapBinaryFunc(func(a, b float64) (float64, error) { return a / b, nil }),
	"mapRange":  mapRange,
}

func (e *extractorTmpl) Extract(payload []byte) (float64, error) {
	var sb strings.Builder
	err := e.tmpl.Execute(&sb, string(payload))
	if err != nil {
		return 0.0, fmt.Errorf("Template execution failed for payload '%s': %v", string(payload), err)
	}
	// erase white space
	sval := strings.TrimSpace(sb.String())
	fval, err := strconv.ParseFloat(sval, 64)
	if err != nil {
		return 0.0, fmt.Errorf("Template returned invalid number literal '%s'", sb.String())
	}
	return fval, nil
}

func newExtractorTmpl(pattern string) (extractor, error) {
	tmpl, err := template.New("").Funcs(tmplFuncs).Parse(pattern)
	if err != nil {
		return nil, fmt.Errorf("Invalid template '%s': %v", pattern, err)
	}
	return &extractorTmpl{tmpl: tmpl}, nil
}

const (
	numberPattern  = `([+-]?(\d+(\.\d*)?|\.\d+))`
	skipPattern    = `[^\d.+-]*`
	startWSPattern = `^\s*`
	endWSPattern   = `\s*$`
)

func newExtractor(kindParam *vdevices.IntParameter, patternParam *vdevices.StringParameter,
	groupParam *vdevices.IntParameter) (extractor, error) {
	kind := extractorKind(kindParam.Value().(int))
	pattern := patternParam.Value().(string)
	groupIdx := groupParam.Value().(int)
	switch kind {
	case ExtractorAfter:
		return newExtractorRegexp(regexp.QuoteMeta(pattern)+skipPattern+numberPattern, 1)
	case ExtractorBefore:
		return newExtractorRegexp(numberPattern+skipPattern+regexp.QuoteMeta(pattern), 1)
	case ExtractorRegexp:
		return newExtractorRegexp(pattern, groupIdx)
	case ExtractorAll:
		return newExtractorRegexp(startWSPattern+numberPattern+endWSPattern, 1)
	case ExtractorTemplate:
		return newExtractorTmpl(pattern)
	default:
		return nil, fmt.Errorf("Invalid extractor kind: %d", kind)
	}
}

func newMatcherKindParameter(id string) *vdevices.IntParameter {
	p := vdevices.NewIntParameter(id)
	p.Description().Type = itf.ParameterTypeEnum
	// align with matcherKind constants
	p.Description().ValueList = []string{"EXACT", "CONTAINS", "REGEXP"}
	p.Description().Min = 0
	p.Description().Max = 2
	p.Description().Default = 0
	return p
}

type matcherKind int

const (
	// align with function newMatcherKindParameter
	MatcherExact matcherKind = iota
	MatcherContains
	MatcherRegexp
)

type matcher interface {
	Match(payload []byte) bool
}

func newMatcher(kindParam *vdevices.IntParameter, patternParam *vdevices.StringParameter) (matcher, error) {
	kind := matcherKind(kindParam.Value().(int))
	pattern := patternParam.Value().(string)
	switch kind {
	case MatcherExact:
		return &matcherExact{pattern: pattern}, nil
	case MatcherContains:
		return &matcherContains{pattern: pattern}, nil
	case MatcherRegexp:
		regexp, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("Invalid regular expression: %s", pattern)
		}
		return &matcherRegexp{regexp: regexp}, nil
	}
	return nil, fmt.Errorf("Invalid matcher kind: %d", kind)
}

type matcherExact struct {
	pattern string
}

func (m *matcherExact) Match(payload []byte) bool {
	return string(payload) == m.pattern
}

type matcherContains struct {
	pattern string
}

func (m *matcherContains) Match(payload []byte) bool {
	return strings.Contains(string(payload), m.pattern)
}

type matcherRegexp struct {
	regexp *regexp.Regexp
}

func (m *matcherRegexp) Match(payload []byte) bool {
	return m.regexp.Match(payload)
}

func matchLevels(pattern []string, topic []string) bool {
	if len(pattern) == 0 {
		return len(topic) == 0
	}
	if len(topic) == 0 {
		return pattern[0] == "#"
	}
	if pattern[0] == "#" {
		return true
	}
	if (pattern[0] == topic[0]) || (pattern[0] == "+") {
		return matchLevels(pattern[1:], topic[1:])
	}
	return false
}

func matchTopic(pattern, topic string) bool {
	return matchLevels(strings.Split(pattern, "/"), strings.Split(topic, "/"))
}
