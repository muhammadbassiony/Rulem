package logging

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
)

func TestDebug_DisabledInProduction(t *testing.T) {
	var buf bytes.Buffer

	logger := log.NewWithOptions(&buf, log.Options{
		ReportTimestamp: false,
		ReportCaller:    false,
	})
	logger.SetLevel(log.DebugLevel)

	appLogger := &AppLogger{
		logger: logger,
		debug:  false, // Production mode
	}

	appLogger.Debug("debug message that should not appear")

	output := buf.String()
	if strings.Contains(output, "debug message that should not appear") {
		t.Errorf("Expected debug message to be suppressed in production mode, got: %s", output)
	}
}

func TestLogMessage(t *testing.T) {
	logger, buf := NewTestLogger()

	// Test with a KeyMsg
	keyMsg := tea.KeyMsg{
		Type:  tea.KeySpace,
		Runes: []rune{' '},
	}

	logger.LogMessage(keyMsg)

	output := buf.String()
	if !strings.Contains(output, "Message received") {
		t.Errorf("Expected log output to contain 'Message received', got: %s", output)
	}
	if !strings.Contains(output, "tea.KeyMsg") {
		t.Errorf("Expected log output to contain message type 'tea.KeyMsg', got: %s", output)
	}
}

func TestLogMessage_DisabledInProduction(t *testing.T) {
	var buf bytes.Buffer

	logger := log.NewWithOptions(&buf, log.Options{
		ReportTimestamp: false,
		ReportCaller:    false,
	})
	logger.SetLevel(log.DebugLevel)

	appLogger := &AppLogger{
		logger: logger,
		debug:  false, // Production mode
	}

	keyMsg := tea.KeyMsg{Type: tea.KeySpace}
	appLogger.LogMessage(keyMsg)

	output := buf.String()
	if strings.Contains(output, "Message received") {
		t.Errorf("Expected message logging to be suppressed in production mode, got: %s", output)
	}
}

func TestDebugObject(t *testing.T) {
	logger, buf := NewTestLogger()

	testObj := struct {
		Name  string
		Value int
	}{
		Name:  "test",
		Value: 42,
	}

	logger.DebugObject("test_object", testObj)

	output := buf.String()
	if !strings.Contains(output, "Object dump") {
		t.Errorf("Expected log output to contain 'Object dump', got: %s", output)
	}
	if !strings.Contains(output, "test_object") {
		t.Errorf("Expected log output to contain object name, got: %s", output)
	}
	if !strings.Contains(output, "test") {
		t.Errorf("Expected log output to contain object data, got: %s", output)
	}
}

func TestLogPerformance(t *testing.T) {
	logger, buf := NewTestLogger()

	start := time.Now()
	time.Sleep(1 * time.Millisecond) // Small delay for measurable duration
	logger.LogPerformance("test_operation", start)

	output := buf.String()
	if !strings.Contains(output, "Performance") {
		t.Errorf("Expected log output to contain 'Performance', got: %s", output)
	}
	if !strings.Contains(output, "test_operation") {
		t.Errorf("Expected log output to contain operation name, got: %s", output)
	}
	if !strings.Contains(output, "duration") {
		t.Errorf("Expected log output to contain duration, got: %s", output)
	}
}

func TestLogStateTransition(t *testing.T) {
	logger, buf := NewTestLogger()

	logger.LogStateTransition("MainModel", "StateMenu", "StateSettings")

	output := buf.String()
	if !strings.Contains(output, "State transition") {
		t.Errorf("Expected log output to contain 'State transition', got: %s", output)
	}
	if !strings.Contains(output, "MainModel") {
		t.Errorf("Expected log output to contain component name, got: %s", output)
	}
	if !strings.Contains(output, "StateMenu") {
		t.Errorf("Expected log output to contain 'from' state, got: %s", output)
	}
	if !strings.Contains(output, "StateSettings") {
		t.Errorf("Expected log output to contain 'to' state, got: %s", output)
	}
}

func TestLogUserAction(t *testing.T) {
	logger, buf := NewTestLogger()

	logger.LogUserAction("menu_selection", "import_rules")

	output := buf.String()
	if !strings.Contains(output, "User action") {
		t.Errorf("Expected log output to contain 'User action', got: %s", output)
	}
	if !strings.Contains(output, "menu_selection") {
		t.Errorf("Expected log output to contain action, got: %s", output)
	}
	if !strings.Contains(output, "import_rules") {
		t.Errorf("Expected log output to contain context, got: %s", output)
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	// Reset the singleton for testing
	defaultLogger = nil
	once = sync.Once{}

	// Set debug mode for testing
	os.Setenv("DEBUG", "1")
	defer os.Unsetenv("DEBUG")

	// Test that package-level functions work
	Info("package level info")
	Warn("package level warn")
	Error("package level error")
	Debug("package level debug")

	// Test LogMessage at package level
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	LogMessage(keyMsg)

	// Test LogPerformance at package level
	start := time.Now()
	LogPerformance("package_operation", start)

	// If we get here without panics, the package-level functions work
}

func TestGetDefault_Singleton(t *testing.T) {
	// Reset the singleton for testing
	defaultLogger = nil
	once = sync.Once{}

	logger1 := GetDefault()
	logger2 := GetDefault()

	if logger1 != logger2 {
		t.Error("Expected GetDefault() to return the same instance (singleton)")
	}
}

// Benchmark tests
func BenchmarkInfo(b *testing.B) {
	logger, _ := NewTestLogger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "iteration", i)
	}
}

func BenchmarkDebug(b *testing.B) {
	logger, _ := NewTestLogger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Debug("benchmark debug message", "iteration", i)
	}
}

func BenchmarkLogMessage(b *testing.B) {
	logger, _ := NewTestLogger()
	keyMsg := tea.KeyMsg{Type: tea.KeySpace}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogMessage(keyMsg)
	}
}
