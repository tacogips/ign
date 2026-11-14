package debug

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestSetDebug(t *testing.T) {
	// Initially disabled
	SetDebug(false)
	if IsEnabled() {
		t.Error("Debug should be disabled initially")
	}

	// Enable
	SetDebug(true)
	if !IsEnabled() {
		t.Error("Debug should be enabled")
	}

	// Disable again
	SetDebug(false)
	if IsEnabled() {
		t.Error("Debug should be disabled again")
	}
}

func TestDebugOutput(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	SetDebug(true)
	SetNoColor(true)

	Debug("test message %s", "arg")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "[DEBUG]") {
		t.Errorf("Output should contain [DEBUG] prefix, got: %s", output)
	}

	if !strings.Contains(output, "test message arg") {
		t.Errorf("Output should contain message, got: %s", output)
	}

	// Should contain timestamp
	if !strings.Contains(output, ":") {
		t.Errorf("Output should contain timestamp, got: %s", output)
	}
}

func TestDebugDisabled(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	SetDebug(false)
	Debug("this should not appear")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output != "" {
		t.Errorf("Debug output should be empty when disabled, got: %s", output)
	}
}

func TestDebugSection(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	SetDebug(true)
	SetNoColor(true)

	DebugSection("Test Section")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "[DEBUG]") {
		t.Errorf("Output should contain [DEBUG] prefix, got: %s", output)
	}

	if !strings.Contains(output, "=== Test Section ===") {
		t.Errorf("Output should contain section header, got: %s", output)
	}
}

func TestDebugValue(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	SetDebug(true)
	SetNoColor(true)

	DebugValue("key", "value")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "[DEBUG]") {
		t.Errorf("Output should contain [DEBUG] prefix, got: %s", output)
	}

	if !strings.Contains(output, "key = value") {
		t.Errorf("Output should contain key=value, got: %s", output)
	}
}

func TestDebugJSON(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	SetDebug(true)
	SetNoColor(true)

	testData := map[string]interface{}{
		"foo": "bar",
		"num": 42,
	}
	DebugJSON("testData", testData)

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "[DEBUG]") {
		t.Errorf("Output should contain [DEBUG] prefix, got: %s", output)
	}

	if !strings.Contains(output, "testData:") {
		t.Errorf("Output should contain key, got: %s", output)
	}

	if !strings.Contains(output, "\"foo\"") {
		t.Errorf("Output should contain JSON data, got: %s", output)
	}
}
