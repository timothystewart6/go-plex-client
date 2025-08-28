package plex

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

// Logger is a minimal structured logger used by the package.
type Logger interface {
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
	Debug(msg string, fields map[string]interface{})
}

type jsonLogger struct {
	out io.Writer
	mu  sync.Mutex
}

func NewJSONLogger(out io.Writer) Logger {
	return &jsonLogger{out: out}
}

func (l *jsonLogger) log(level, msg string, fields map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	m := map[string]interface{}{
		"time":  time.Now().UTC().Format(time.RFC3339),
		"level": level,
		"msg":   msg,
	}

	if fields != nil {
		for k, v := range fields {
			m[k] = v
		}
	}

	b, _ := json.Marshal(m)
	l.out.Write(append(b, '\n'))
}

func (l *jsonLogger) Info(msg string, fields map[string]interface{})  { l.log("info", msg, fields) }
func (l *jsonLogger) Warn(msg string, fields map[string]interface{})  { l.log("warn", msg, fields) }
func (l *jsonLogger) Error(msg string, fields map[string]interface{}) { l.log("error", msg, fields) }
func (l *jsonLogger) Debug(msg string, fields map[string]interface{}) { l.log("debug", msg, fields) }

var logger Logger = NewJSONLogger(os.Stderr)

// SetLogger replaces the package-level logger. Passing nil resets to the default JSON logger.
func SetLogger(l Logger) {
	if l == nil {
		logger = NewJSONLogger(os.Stderr)
		return
	}
	logger = l
}
