package debug

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	enabled   bool
	enabledMu sync.RWMutex
	noColor   bool
	noColorMu sync.RWMutex
)

// ANSI color codes
const (
	colorReset = "\033[0m"
	colorCyan  = "\033[36m"
	colorGray  = "\033[90m"
)

// SetDebug enables or disables debug mode
func SetDebug(enable bool) {
	enabledMu.Lock()
	defer enabledMu.Unlock()
	enabled = enable
}

// IsEnabled returns whether debug mode is enabled
func IsEnabled() bool {
	enabledMu.RLock()
	defer enabledMu.RUnlock()
	return enabled
}

// SetNoColor enables or disables colored output
func SetNoColor(disable bool) {
	noColorMu.Lock()
	defer noColorMu.Unlock()
	noColor = disable
}

// Debug prints a debug message with timestamp
func Debug(format string, args ...interface{}) {
	if !IsEnabled() {
		return
	}

	noColorMu.RLock()
	useColor := !noColor
	noColorMu.RUnlock()

	timestamp := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, args...)

	if useColor {
		fmt.Fprintf(os.Stderr, "%s[DEBUG]%s %s%s%s %s\n",
			colorCyan, colorReset, colorGray, timestamp, colorReset, msg)
	} else {
		fmt.Fprintf(os.Stderr, "[DEBUG] %s %s\n", timestamp, msg)
	}
}

// Debugf is an alias for Debug
func Debugf(format string, args ...interface{}) {
	Debug(format, args...)
}

// DebugSection prints a section header for debug output
func DebugSection(section string) {
	if !IsEnabled() {
		return
	}

	noColorMu.RLock()
	useColor := !noColor
	noColorMu.RUnlock()

	timestamp := time.Now().Format("15:04:05.000")

	if useColor {
		fmt.Fprintf(os.Stderr, "%s[DEBUG]%s %s%s%s %s=== %s ===%s\n",
			colorCyan, colorReset, colorGray, timestamp, colorReset,
			colorCyan, section, colorReset)
	} else {
		fmt.Fprintf(os.Stderr, "[DEBUG] %s === %s ===\n", timestamp, section)
	}
}

// DebugValue prints key=value style debug info
func DebugValue(key string, value interface{}) {
	if !IsEnabled() {
		return
	}

	noColorMu.RLock()
	useColor := !noColor
	noColorMu.RUnlock()

	timestamp := time.Now().Format("15:04:05.000")

	if useColor {
		fmt.Fprintf(os.Stderr, "%s[DEBUG]%s %s%s%s %s%s%s = %v\n",
			colorCyan, colorReset, colorGray, timestamp, colorReset,
			colorCyan, key, colorReset, value)
	} else {
		fmt.Fprintf(os.Stderr, "[DEBUG] %s %s = %v\n", timestamp, key, value)
	}
}

// DebugJSON prints structured data as JSON for debugging
func DebugJSON(key string, v interface{}) {
	if !IsEnabled() {
		return
	}

	noColorMu.RLock()
	useColor := !noColor
	noColorMu.RUnlock()

	timestamp := time.Now().Format("15:04:05.000")

	jsonBytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		Debug("Failed to marshal %s to JSON: %v", key, err)
		return
	}

	if useColor {
		fmt.Fprintf(os.Stderr, "%s[DEBUG]%s %s%s%s %s%s%s:\n%s\n",
			colorCyan, colorReset, colorGray, timestamp, colorReset,
			colorCyan, key, colorReset, string(jsonBytes))
	} else {
		fmt.Fprintf(os.Stderr, "[DEBUG] %s %s:\n%s\n", timestamp, key, string(jsonBytes))
	}
}
