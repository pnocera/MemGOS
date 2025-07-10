package logger

import (
	"bytes"
	"errors"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/memtensor/memgos/pkg/interfaces"
)

func TestPlaceholderLogger(t *testing.T) {
	t.Run("NewConsoleLogger", func(t *testing.T) {
		logger := NewConsoleLogger("info")
		assert.NotNil(t, logger)
		
		placeholderLogger, ok := logger.(*PlaceholderLogger)
		require.True(t, ok)
		assert.Equal(t, "info", placeholderLogger.Level)
		assert.Empty(t, placeholderLogger.File)
	})

	t.Run("NewTestLogger", func(t *testing.T) {
		logger := NewTestLogger()
		assert.NotNil(t, logger)
		
		placeholderLogger, ok := logger.(*PlaceholderLogger)
		require.True(t, ok)
		assert.Equal(t, "debug", placeholderLogger.Level)
	})
}

func TestLoggingLevels(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	t.Run("Debug", func(t *testing.T) {
		buf.Reset()
		
		// Debug level logger should log debug messages
		logger := &PlaceholderLogger{Level: "debug"}
		logger.Debug("debug message")
		
		output := buf.String()
		assert.Contains(t, output, "[DEBUG]")
		assert.Contains(t, output, "debug message")
		
		buf.Reset()
		
		// Info level logger should not log debug messages
		logger = &PlaceholderLogger{Level: "info"}
		logger.Debug("debug message")
		
		output = buf.String()
		assert.Empty(t, output)
	})

	t.Run("Info", func(t *testing.T) {
		buf.Reset()
		
		logger := &PlaceholderLogger{Level: "info"}
		logger.Info("info message")
		
		output := buf.String()
		assert.Contains(t, output, "[INFO]")
		assert.Contains(t, output, "info message")
	})

	t.Run("Warn", func(t *testing.T) {
		buf.Reset()
		
		logger := &PlaceholderLogger{Level: "info"}
		logger.Warn("warning message")
		
		output := buf.String()
		assert.Contains(t, output, "[WARN]")
		assert.Contains(t, output, "warning message")
	})

	t.Run("Error", func(t *testing.T) {
		buf.Reset()
		
		logger := &PlaceholderLogger{Level: "info"}
		testErr := errors.New("test error")
		logger.Error("error occurred", testErr)
		
		output := buf.String()
		assert.Contains(t, output, "[ERROR]")
		assert.Contains(t, output, "error occurred")
		assert.Contains(t, output, "error=test error")
	})

	t.Run("ErrorWithoutError", func(t *testing.T) {
		buf.Reset()
		
		logger := &PlaceholderLogger{Level: "info"}
		logger.Error("error occurred", nil)
		
		output := buf.String()
		assert.Contains(t, output, "[ERROR]")
		assert.Contains(t, output, "error occurred")
		assert.NotContains(t, output, "error=")
	})
}

func TestLoggingWithFields(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	t.Run("InfoWithFields", func(t *testing.T) {
		buf.Reset()
		
		logger := &PlaceholderLogger{Level: "info"}
		fields := map[string]interface{}{
			"user_id":   "123",
			"operation": "login",
			"count":     42,
		}
		
		logger.Info("user action", fields)
		
		output := buf.String()
		assert.Contains(t, output, "[INFO]")
		assert.Contains(t, output, "user action")
		assert.Contains(t, output, "user_id=123")
		assert.Contains(t, output, "operation=login")
		assert.Contains(t, output, "count=42")
	})

	t.Run("ErrorWithFieldsAndError", func(t *testing.T) {
		buf.Reset()
		
		logger := &PlaceholderLogger{Level: "info"}
		testErr := errors.New("connection failed")
		fields := map[string]interface{}{
			"host": "localhost",
			"port": 5432,
		}
		
		logger.Error("database connection failed", testErr, fields)
		
		output := buf.String()
		assert.Contains(t, output, "[ERROR]")
		assert.Contains(t, output, "database connection failed")
		assert.Contains(t, output, "error=connection failed")
		assert.Contains(t, output, "host=localhost")
		assert.Contains(t, output, "port=5432")
	})

	t.Run("MultipleFieldMaps", func(t *testing.T) {
		buf.Reset()
		
		logger := &PlaceholderLogger{Level: "info"}
		fields1 := map[string]interface{}{"key1": "value1"}
		fields2 := map[string]interface{}{"key2": "value2"}
		
		logger.Info("test message", fields1, fields2)
		
		output := buf.String()
		assert.Contains(t, output, "key1=value1")
		assert.Contains(t, output, "key2=value2")
	})

	t.Run("EmptyFields", func(t *testing.T) {
		buf.Reset()
		
		logger := &PlaceholderLogger{Level: "info"}
		logger.Info("test message")
		
		output := buf.String()
		assert.Contains(t, output, "[INFO]")
		assert.Contains(t, output, "test message")
		// Should not contain any field markers
		lines := strings.Split(output, " ")
		for _, line := range lines {
			assert.False(t, strings.Contains(line, "="))
		}
	})
}

func TestWithFields(t *testing.T) {
	t.Run("WithFields", func(t *testing.T) {
		logger := &PlaceholderLogger{Level: "info"}
		fields := map[string]interface{}{
			"user_id": "123",
			"session": "abc",
		}
		
		// WithFields should return a logger (in this simple implementation, the same logger)
		loggerWithFields := logger.WithFields(fields)
		assert.NotNil(t, loggerWithFields)
		
		// Verify it implements the Logger interface
		var _ interfaces.Logger = loggerWithFields
	})
}

func TestFatal(t *testing.T) {
	// Note: We can't easily test Fatal as it calls os.Exit(1)
	// In a real implementation, we would use dependency injection
	// or test doubles to avoid actually exiting
	
	t.Run("FatalInterface", func(t *testing.T) {
		logger := &PlaceholderLogger{Level: "info"}
		
		// Just verify the method exists and can be called
		// We can't actually call it in tests as it would exit
		assert.NotNil(t, logger.Fatal)
	})
}

func TestLogWithFields(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	t.Run("logWithFields", func(t *testing.T) {
		buf.Reset()
		
		logger := &PlaceholderLogger{Level: "debug"}
		fields := map[string]interface{}{
			"string_val":  "test",
			"int_val":     42,
			"bool_val":    true,
			"float_val":   3.14,
			"nil_val":     nil,
		}
		
		logger.logWithFields("TEST", "test message", fields)
		
		output := buf.String()
		assert.Contains(t, output, "[TEST]")
		assert.Contains(t, output, "test message")
		assert.Contains(t, output, "string_val=test")
		assert.Contains(t, output, "int_val=42")
		assert.Contains(t, output, "bool_val=true")
		assert.Contains(t, output, "float_val=3.14")
		assert.Contains(t, output, "nil_val=<nil>")
	})

	t.Run("logWithMultipleFieldMaps", func(t *testing.T) {
		buf.Reset()
		
		logger := &PlaceholderLogger{Level: "debug"}
		fields1 := map[string]interface{}{"field1": "value1"}
		fields2 := map[string]interface{}{"field2": "value2"}
		fields3 := map[string]interface{}{"field3": "value3"}
		
		logger.logWithFields("MULTI", "multi test", fields1, fields2, fields3)
		
		output := buf.String()
		assert.Contains(t, output, "[MULTI]")
		assert.Contains(t, output, "multi test")
		assert.Contains(t, output, "field1=value1")
		assert.Contains(t, output, "field2=value2")
		assert.Contains(t, output, "field3=value3")
	})

	t.Run("logWithFileLogging", func(t *testing.T) {
		buf.Reset()
		
		// Test file logging path (though it still goes to log output in this implementation)
		logger := &PlaceholderLogger{
			Level: "debug",
			File:  "/tmp/test.log",
		}
		
		logger.logWithFields("FILE", "file test", map[string]interface{}{"key": "value"})
		
		output := buf.String()
		assert.Contains(t, output, "[FILE]")
		assert.Contains(t, output, "file test")
		assert.Contains(t, output, "key=value")
	})
}

func TestLoggerLevels(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	testCases := []struct {
		name         string
		loggerLevel  string
		method       func(interfaces.Logger)
		shouldLog    bool
		expectedText string
	}{
		{
			name:        "debug logger logs debug",
			loggerLevel: "debug",
			method: func(l interfaces.Logger) {
				l.Debug("debug test")
			},
			shouldLog:    true,
			expectedText: "[DEBUG]",
		},
		{
			name:        "info logger skips debug",
			loggerLevel: "info",
			method: func(l interfaces.Logger) {
				l.Debug("debug test")
			},
			shouldLog:    false,
			expectedText: "[DEBUG]",
		},
		{
			name:        "info logger logs info",
			loggerLevel: "info",
			method: func(l interfaces.Logger) {
				l.Info("info test")
			},
			shouldLog:    true,
			expectedText: "[INFO]",
		},
		{
			name:        "info logger logs warn",
			loggerLevel: "info",
			method: func(l interfaces.Logger) {
				l.Warn("warn test")
			},
			shouldLog:    true,
			expectedText: "[WARN]",
		},
		{
			name:        "info logger logs error",
			loggerLevel: "info",
			method: func(l interfaces.Logger) {
				l.Error("error test", nil)
			},
			shouldLog:    true,
			expectedText: "[ERROR]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()
			
			logger := &PlaceholderLogger{Level: tc.loggerLevel}
			tc.method(logger)
			
			output := buf.String()
			if tc.shouldLog {
				assert.Contains(t, output, tc.expectedText)
			} else {
				assert.NotContains(t, output, tc.expectedText)
			}
		})
	}
}

func TestComplexLogging(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	t.Run("ComplexFieldTypes", func(t *testing.T) {
		buf.Reset()
		
		logger := &PlaceholderLogger{Level: "debug"}
		
		// Test complex field types
		complexFields := map[string]interface{}{
			"slice":   []string{"a", "b", "c"},
			"map":     map[string]string{"nested": "value"},
			"struct":  struct{ Name string }{Name: "test"},
			"pointer": &struct{ ID int }{ID: 123},
		}
		
		logger.Info("complex fields test", complexFields)
		
		output := buf.String()
		assert.Contains(t, output, "[INFO]")
		assert.Contains(t, output, "complex fields test")
		// These should be formatted by Go's fmt package
		assert.Contains(t, output, "slice=")
		assert.Contains(t, output, "map=")
		assert.Contains(t, output, "struct=")
		assert.Contains(t, output, "pointer=")
	})

	t.Run("LongMessage", func(t *testing.T) {
		buf.Reset()
		
		logger := &PlaceholderLogger{Level: "info"}
		longMessage := strings.Repeat("This is a very long message. ", 50)
		
		logger.Info(longMessage)
		
		output := buf.String()
		assert.Contains(t, output, "[INFO]")
		assert.Contains(t, output, longMessage)
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		buf.Reset()
		
		logger := &PlaceholderLogger{Level: "info"}
		specialMessage := "Message with special chars: \n\t\r\\\"'"
		
		logger.Info(specialMessage, map[string]interface{}{
			"special_field": "value with spaces and \n newlines",
		})
		
		output := buf.String()
		assert.Contains(t, output, "[INFO]")
		// The exact formatting depends on Go's fmt package
		assert.Contains(t, output, "special_field=")
	})
}

// Benchmark tests
func BenchmarkLoggerInfo(b *testing.B) {
	// Discard log output to avoid I/O overhead in benchmarks
	log.SetOutput(bytes.NewBuffer(nil))
	defer log.SetOutput(os.Stderr)
	
	logger := &PlaceholderLogger{Level: "info"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark test message")
	}
}

func BenchmarkLoggerInfoWithFields(b *testing.B) {
	log.SetOutput(bytes.NewBuffer(nil))
	defer log.SetOutput(os.Stderr)
	
	logger := &PlaceholderLogger{Level: "info"}
	fields := map[string]interface{}{
		"user_id":   "12345",
		"operation": "benchmark",
		"count":     42,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark test message", fields)
	}
}

func BenchmarkLoggerDebugSkipped(b *testing.B) {
	log.SetOutput(bytes.NewBuffer(nil))
	defer log.SetOutput(os.Stderr)
	
	logger := &PlaceholderLogger{Level: "info"} // Debug messages will be skipped
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Debug("debug message that should be skipped")
	}
}

func BenchmarkLoggerError(b *testing.B) {
	log.SetOutput(bytes.NewBuffer(nil))
	defer log.SetOutput(os.Stderr)
	
	logger := &PlaceholderLogger{Level: "info"}
	testErr := errors.New("benchmark error")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Error("error message", testErr)
	}
}

// Integration tests
func TestLoggerIntegration(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	t.Run("TypicalUsagePattern", func(t *testing.T) {
		buf.Reset()
		
		logger := NewConsoleLogger("debug")
		
		// Simulate typical application logging
		logger.Info("Application starting", map[string]interface{}{
			"version": "1.0.0",
			"env":     "test",
		})
		
		logger.Debug("Processing request", map[string]interface{}{
			"request_id": "req-123",
			"user_id":    "user-456",
		})
		
		logger.Warn("High memory usage detected", map[string]interface{}{
			"usage_percent": 85,
			"threshold":     80,
		})
		
		err := errors.New("database connection lost")
		logger.Error("Database error occurred", err, map[string]interface{}{
			"host":        "db.example.com",
			"retry_count": 3,
		})
		
		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		assert.Len(t, lines, 4)
		
		assert.Contains(t, lines[0], "[INFO]")
		assert.Contains(t, lines[0], "Application starting")
		assert.Contains(t, lines[0], "version=1.0.0")
		
		assert.Contains(t, lines[1], "[DEBUG]")
		assert.Contains(t, lines[1], "Processing request")
		assert.Contains(t, lines[1], "request_id=req-123")
		
		assert.Contains(t, lines[2], "[WARN]")
		assert.Contains(t, lines[2], "High memory usage")
		assert.Contains(t, lines[2], "usage_percent=85")
		
		assert.Contains(t, lines[3], "[ERROR]")
		assert.Contains(t, lines[3], "Database error")
		assert.Contains(t, lines[3], "error=database connection lost")
		assert.Contains(t, lines[3], "host=db.example.com")
	})

	t.Run("LoggerChaining", func(t *testing.T) {
		buf.Reset()
		
		logger := NewTestLogger()
		
		// Test that WithFields returns a functional logger
		childLogger := logger.WithFields(map[string]interface{}{
			"component": "test",
			"session":   "sess-789",
		})
		
		childLogger.Info("Child logger test")
		
		output := buf.String()
		assert.Contains(t, output, "[INFO]")
		assert.Contains(t, output, "Child logger test")
	})
}

// Example tests for documentation
func ExampleNewConsoleLogger() {
	logger := NewConsoleLogger("info")
	logger.Info("Application started successfully")
	// Output will go to standard log output
}

func ExamplePlaceholderLogger_Info() {
	logger := &PlaceholderLogger{Level: "info"}
	logger.Info("User logged in", map[string]interface{}{
		"user_id":    "12345",
		"ip_address": "192.168.1.1",
	})
	// Output will include: [INFO] User logged in user_id=12345 ip_address=192.168.1.1
}

func ExamplePlaceholderLogger_Error() {
	logger := &PlaceholderLogger{Level: "info"}
	err := errors.New("connection timeout")
	logger.Error("Failed to connect to database", err, map[string]interface{}{
		"host":    "localhost",
		"timeout": "30s",
	})
	// Output will include: [ERROR] Failed to connect to database error=connection timeout host=localhost timeout=30s
}