package main

import (
	"container/ring"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/mdzio/go-veap"
	"github.com/mdzio/go-veap/model"
)

const (
	// number of log messages to buffer for diagnostics
	logBufferSize = 500
	// limit the size of buffered log messages
	logBufferMaxMsgSize = 250
)

type Diagnostics struct {
	// table with log messages (timestamp, severity, module, message)
	Log [][]string
}

// NewDiagnostics creates a new diagnostics variable.
func NewDiagnostics(col model.ChangeableCollection) *model.ROVariable {
	return model.NewROVariable(&model.ROVariableCfg{
		Identifier:  "diagnostics",
		Title:       "Diagnostics",
		Description: "Diagnostic information about CCU-Jack components and connections",
		Collection:  col,
		ReadPVFunc: func() (veap.PV, veap.Error) {
			v := Diagnostics{
				Log: logBuffer.Messages(),
			}
			return veap.PV{Time: time.Now(), Value: v, State: veap.StateGood}, nil
		},
	})
}

// LogBuffer is a circular buffer for log messages. LogBuffer implements
// io.Writer. Log messages are also forwarded to Next, if not nil.
type LogBuffer struct {
	Next io.Writer

	sync.RWMutex
	// ring points to the newest log message, Next() points to the previous
	ring *ring.Ring
}

// NewLogBuffer creates a LogBuffer of the specified size.
func NewLogBuffer() *LogBuffer {
	return &LogBuffer{ring: ring.New(logBufferSize)}
}

// Write implements interface io.Writer.
func (b *LogBuffer) Write(p []byte) (n int, err error) {
	if b.Next != nil {
		b.Next.Write(p)
	}
	b.Lock()
	defer b.Unlock()
	b.ring = b.ring.Prev()
	if len(p) > logBufferMaxMsgSize {
		var sb strings.Builder
		// ignore possibly truncating a multi byte UTF8 character
		sb.Write(p[:logBufferMaxMsgSize])
		sb.WriteString("…")
		b.ring.Value = sb.String()
	} else {
		b.ring.Value = string(p)
	}
	return len(p), nil
}

// Messages returns all buffered messages from newest to oldest.
func (b *LogBuffer) Messages() [][]string {
	ms := make([][]string, 0, logBufferSize)
	b.RLock()
	defer b.RUnlock()
	b.ring.Do(func(m interface{}) {
		if m != nil {
			fs := strings.SplitN(m.(string), "|", 4)
			for idx := 1; idx < 4; idx++ {
				fs[idx] = strings.TrimSpace(fs[idx])
			}
			ms = append(ms, fs)
		}
	})
	return ms
}
