package logger

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestLogger_InitAndOutput(t *testing.T) {
	// Test Development Init
	Init("development")
	if Logger().GetLevel() > zerolog.DebugLevel {
		t.Errorf("expected level to be at least Debug, got %v", Logger().GetLevel())
	}

	// Test Production JSON output
	var buf bytes.Buffer
	zerolog.TimeFieldFormat = time.RFC3339
	log = zerolog.New(&buf).With().Timestamp().Logger()

	Info().Str("service", "test-service").Int("count", 10).Msg("test message")

	output := buf.Bytes()
	if len(output) == 0 {
		t.Fatal("expected non-empty log output")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(output, &parsed); err != nil {
		t.Fatalf("failed to parse json log output: %v", err)
	}

	if parsed["level"] != "info" {
		t.Errorf("expected level 'info', got %v", parsed["level"])
	}
	if parsed["service"] != "test-service" {
		t.Errorf("expected service 'test-service', got %v", parsed["service"])
	}
	if parsed["message"] != "test message" {
		t.Errorf("expected message 'test message', got %v", parsed["message"])
	}

	// Test Warn and Error events
	buf.Reset()
	Warn().Msg("warning event")
	if buf.Len() == 0 {
		t.Error("expected non-empty output for Warn")
	}

	buf.Reset()
	Error().Msg("error event")
	if buf.Len() == 0 {
		t.Error("expected non-empty output for Error")
	}
}
