package plex

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewLogger_FormatsJSONAndFields(t *testing.T) {
	var buf bytes.Buffer

	// Create a logger that writes to our buffer
	l := NewLogger(&buf)

	// Emit a log with fields
	l.Info("test message", zap.String("foo", "bar"), zap.Int("num", 42))

	out := strings.TrimSpace(buf.String())
	if out == "" {
		t.Fatalf("expected log output, got empty string")
	}

	// The logger writes JSON per-line; decode to map
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("expected valid JSON log line; unmarshal error: %v; output: %s", err, out)
	}

	// Check message
	if msg, ok := m["msg"].(string); !ok || msg != "test message" {
		t.Fatalf("expected msg to be %q, got %#v", "test message", m["msg"])
	}

	// Check fields
	if foo, ok := m["foo"].(string); !ok || foo != "bar" {
		t.Fatalf("expected foo field to be %q, got %#v", "bar", m["foo"])
	}

	// JSON numbers decode to float64
	if num, ok := m["num"].(float64); !ok || num != 42 {
		t.Fatalf("expected num field to be 42, got %#v", m["num"])
	}
}

func TestNewLogger_DebugSuppressedAtInfoLevel(t *testing.T) {
	var buf bytes.Buffer

	// Create a logger at Info level
	l := NewLoggerWithLevel(&buf, zapcore.InfoLevel)

	// Emit a Debug log and an Info log
	l.Debug("debug message", zap.String("k", "v"))
	l.Info("info message")

	out := strings.TrimSpace(buf.String())
	if out == "" {
		t.Fatalf("expected some log output, got empty string")
	}

	// The output should only contain the info message, not the debug message
	if strings.Contains(out, "debug message") {
		t.Fatalf("debug message should be suppressed at Info level; output: %s", out)
	}

	if !strings.Contains(out, "info message") {
		t.Fatalf("info message expected but not found; output: %s", out)
	}
}
