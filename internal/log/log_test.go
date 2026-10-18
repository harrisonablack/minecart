package log

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestNewDefaultsToTint(t *testing.T) {
	t.Setenv("MINECART_LOG_FORMAT", "")

	var buf bytes.Buffer
	logger := New(&buf)
	logger.Info("hello world")

	if out := buf.String(); !strings.Contains(out, "hello world") {
		t.Errorf("output missing message: %q", out)
	}
}

func TestNewJSON(t *testing.T) {
	t.Setenv("MINECART_LOG_FORMAT", "json")

	var buf bytes.Buffer
	logger := New(&buf)
	logger.Info("hello", "key", "value")

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if m["msg"] != "hello" {
		t.Errorf("msg = %v, want %q", m["msg"], "hello")
	}
	if m["key"] != "value" {
		t.Errorf("key = %v, want %q", m["key"], "value")
	}
}

func TestLevelFiltering(t *testing.T) {
	t.Setenv("MINECART_LOG_LEVEL", "error")

	var buf bytes.Buffer
	logger := New(&buf)
	logger.Info("filtered out")
	logger.Error("kept")

	out := buf.String()
	if strings.Contains(out, "filtered out") {
		t.Errorf("info message should be filtered at error level: %q", out)
	}
	if !strings.Contains(out, "kept") {
		t.Errorf("error message should be present: %q", out)
	}
}

func TestLevel(t *testing.T) {
	tests := []struct {
		env  string
		want slog.Level
	}{
		{"", slog.LevelDebug},
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"DEBUG", slog.LevelDebug}, // case-insensitive
		{"bogus", slog.LevelDebug}, // unknown falls back to debug
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv("MINECART_LOG_LEVEL", tt.env)
			if got := level(); got != tt.want {
				t.Errorf("level() = %v, want %v", got, tt.want)
			}
		})
	}
}
