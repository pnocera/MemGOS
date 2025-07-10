package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/memtensor/memgos/pkg/interfaces"
)

func TestNoOpMetrics(t *testing.T) {
	t.Run("NoOpMetrics Creation", func(t *testing.T) {
		metrics := &NoOpMetrics{}
		assert.NotNil(t, metrics)
	})

	t.Run("NoOpMetrics Interface Implementation", func(t *testing.T) {
		var _ interfaces.Metrics = &NoOpMetrics{}
		// This test passes if the code compiles
	})

	t.Run("Counter", func(t *testing.T) {
		metrics := &NoOpMetrics{}
		labels := map[string]string{"service": "test"}
		
		// Should not panic and should do nothing
		assert.NotPanics(t, func() {
			metrics.Counter("test_counter", 1.0, labels)
			metrics.Counter("test_counter", 5.0, nil)
		})
	})

	t.Run("Gauge", func(t *testing.T) {
		metrics := &NoOpMetrics{}
		labels := map[string]string{"instance": "test"}
		
		assert.NotPanics(t, func() {
			metrics.Gauge("test_gauge", 42.5, labels)
			metrics.Gauge("test_gauge", 0.0, nil)
		})
	})

	t.Run("Histogram", func(t *testing.T) {
		metrics := &NoOpMetrics{}
		labels := map[string]string{"method": "GET"}
		
		assert.NotPanics(t, func() {
			metrics.Histogram("test_histogram", 0.123, labels)
			metrics.Histogram("test_histogram", 1.456, nil)
		})
	})

	t.Run("Timer", func(t *testing.T) {
		metrics := &NoOpMetrics{}
		labels := map[string]string{"operation": "database_query"}
		
		assert.NotPanics(t, func() {
			metrics.Timer("test_timer", 100.5, labels)
			metrics.Timer("test_timer", 250.0, nil)
		})
	})
}

func TestPlaceholderMetrics(t *testing.T) {
	t.Run("PlaceholderMetrics Creation", func(t *testing.T) {
		metrics := &PlaceholderMetrics{Port: 8080}
		assert.NotNil(t, metrics)
		assert.Equal(t, 8080, metrics.Port)
	})

	t.Run("PlaceholderMetrics Interface Implementation", func(t *testing.T) {
		var _ interfaces.Metrics = &PlaceholderMetrics{}
		// This test passes if the code compiles
	})

	t.Run("Counter", func(t *testing.T) {
		metrics := &PlaceholderMetrics{Port: 9090}
		labels := map[string]string{"service": "memgos"}
		
		assert.NotPanics(t, func() {
			metrics.Counter("requests_total", 1.0, labels)
			metrics.Counter("requests_total", 1.0, nil)
		})
	})

	t.Run("Gauge", func(t *testing.T) {
		metrics := &PlaceholderMetrics{Port: 9090}
		labels := map[string]string{"type": "memory"}
		
		assert.NotPanics(t, func() {
			metrics.Gauge("memory_usage_bytes", 1024.0, labels)
			metrics.Gauge("cpu_usage_percent", 75.5, nil)
		})
	})

	t.Run("Histogram", func(t *testing.T) {
		metrics := &PlaceholderMetrics{Port: 9090}
		labels := map[string]string{"endpoint": "/api/search"}
		
		assert.NotPanics(t, func() {
			metrics.Histogram("request_duration_seconds", 0.025, labels)
			metrics.Histogram("response_size_bytes", 2048.0, nil)
		})
	})

	t.Run("Timer", func(t *testing.T) {
		metrics := &PlaceholderMetrics{Port: 9090}
		labels := map[string]string{"operation": "search"}
		
		assert.NotPanics(t, func() {
			metrics.Timer("operation_duration_ms", 150.0, labels)
			metrics.Timer("operation_duration_ms", 300.0, nil)
		})
	})
}

func TestFactoryFunctions(t *testing.T) {
	t.Run("NewNoOpMetrics", func(t *testing.T) {
		metrics := NewNoOpMetrics()
		assert.NotNil(t, metrics)
		
		// Verify it returns a NoOpMetrics instance
		noOpMetrics, ok := metrics.(*NoOpMetrics)
		assert.True(t, ok)
		assert.NotNil(t, noOpMetrics)
		
		// Verify it implements the interface
		var _ interfaces.Metrics = metrics
	})

	t.Run("NewPrometheusMetrics", func(t *testing.T) {
		metrics := NewPrometheusMetrics()
		assert.NotNil(t, metrics)
		
		// Currently returns PlaceholderMetrics
		placeholderMetrics, ok := metrics.(*PlaceholderMetrics)
		assert.True(t, ok)
		assert.NotNil(t, placeholderMetrics)
		assert.Equal(t, 9090, placeholderMetrics.Port)
		
		// Verify it implements the interface
		var _ interfaces.Metrics = metrics
	})

	t.Run("NewTestMetrics", func(t *testing.T) {
		metrics := NewTestMetrics()
		assert.NotNil(t, metrics)
		
		// Should return NoOpMetrics for testing
		noOpMetrics, ok := metrics.(*NoOpMetrics)
		assert.True(t, ok)
		assert.NotNil(t, noOpMetrics)
		
		// Verify it implements the interface
		var _ interfaces.Metrics = metrics
	})
}

func TestMetricsUsagePatterns(t *testing.T) {
	testCases := []struct {
		name    string
		metrics interfaces.Metrics
	}{
		{
			name:    "NoOpMetrics",
			metrics: NewNoOpMetrics(),
		},
		{
			name:    "PlaceholderMetrics",
			metrics: &PlaceholderMetrics{Port: 8080},
		},
		{
			name:    "TestMetrics",
			metrics: NewTestMetrics(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("BasicUsage", func(t *testing.T) {
				assert.NotPanics(t, func() {
					// Simulate typical metrics usage
					tc.metrics.Counter("api_requests_total", 1, map[string]string{
						"method": "POST",
						"path":   "/api/search",
					})
					
					tc.metrics.Gauge("active_connections", 25, map[string]string{
						"service": "memgos",
					})
					
					tc.metrics.Histogram("request_duration_seconds", 0.123, map[string]string{
						"method": "GET",
						"status": "200",
					})
					
					tc.metrics.Timer("operation_duration_ms", 456.78, map[string]string{
						"operation": "memory_search",
					})
				})
			})

			t.Run("WithNilLabels", func(t *testing.T) {
				assert.NotPanics(t, func() {
					tc.metrics.Counter("simple_counter", 1, nil)
					tc.metrics.Gauge("simple_gauge", 42, nil)
					tc.metrics.Histogram("simple_histogram", 0.5, nil)
					tc.metrics.Timer("simple_timer", 100, nil)
				})
			})

			t.Run("WithEmptyLabels", func(t *testing.T) {
				emptyLabels := map[string]string{}
				assert.NotPanics(t, func() {
					tc.metrics.Counter("empty_labels_counter", 1, emptyLabels)
					tc.metrics.Gauge("empty_labels_gauge", 42, emptyLabels)
					tc.metrics.Histogram("empty_labels_histogram", 0.5, emptyLabels)
					tc.metrics.Timer("empty_labels_timer", 100, emptyLabels)
				})
			})

			t.Run("WithZeroValues", func(t *testing.T) {
				labels := map[string]string{"test": "zero"}
				assert.NotPanics(t, func() {
					tc.metrics.Counter("zero_counter", 0, labels)
					tc.metrics.Gauge("zero_gauge", 0, labels)
					tc.metrics.Histogram("zero_histogram", 0, labels)
					tc.metrics.Timer("zero_timer", 0, labels)
				})
			})

			t.Run("WithNegativeValues", func(t *testing.T) {
				labels := map[string]string{"test": "negative"}
				assert.NotPanics(t, func() {
					tc.metrics.Counter("negative_counter", -1, labels)
					tc.metrics.Gauge("negative_gauge", -42.5, labels)
					tc.metrics.Histogram("negative_histogram", -0.123, labels)
					tc.metrics.Timer("negative_timer", -100, labels)
				})
			})

			t.Run("WithLargeValues", func(t *testing.T) {
				labels := map[string]string{"test": "large"}
				assert.NotPanics(t, func() {
					tc.metrics.Counter("large_counter", 1e9, labels)
					tc.metrics.Gauge("large_gauge", 1e12, labels)
					tc.metrics.Histogram("large_histogram", 1e6, labels)
					tc.metrics.Timer("large_timer", 1e8, labels)
				})
			})
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	metrics := NewNoOpMetrics()
	
	t.Run("ConcurrentCounter", func(t *testing.T) {
		done := make(chan bool, 10)
		
		// Start multiple goroutines calling Counter concurrently
		for i := 0; i < 10; i++ {
			go func(id int) {
				defer func() { done <- true }()
				for j := 0; j < 100; j++ {
					metrics.Counter("concurrent_counter", 1, map[string]string{
						"goroutine": string(rune(id)),
					})
				}
			}(i)
		}
		
		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
		
		// Should not panic or race
	})

	t.Run("ConcurrentMixedOperations", func(t *testing.T) {
		done := make(chan bool, 4)
		
		// Counter goroutine
		go func() {
			defer func() { done <- true }()
			for i := 0; i < 50; i++ {
				metrics.Counter("mixed_counter", float64(i), nil)
			}
		}()
		
		// Gauge goroutine
		go func() {
			defer func() { done <- true }()
			for i := 0; i < 50; i++ {
				metrics.Gauge("mixed_gauge", float64(i*2), nil)
			}
		}()
		
		// Histogram goroutine
		go func() {
			defer func() { done <- true }()
			for i := 0; i < 50; i++ {
				metrics.Histogram("mixed_histogram", float64(i)/10.0, nil)
			}
		}()
		
		// Timer goroutine
		go func() {
			defer func() { done <- true }()
			for i := 0; i < 50; i++ {
				metrics.Timer("mixed_timer", float64(i*5), nil)
			}
		}()
		
		// Wait for all goroutines to complete
		for i := 0; i < 4; i++ {
			<-done
		}
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("EmptyMetricName", func(t *testing.T) {
		metrics := NewNoOpMetrics()
		
		assert.NotPanics(t, func() {
			metrics.Counter("", 1, nil)
			metrics.Gauge("", 42, nil)
			metrics.Histogram("", 0.5, nil)
			metrics.Timer("", 100, nil)
		})
	})

	t.Run("SpecialCharactersInMetricName", func(t *testing.T) {
		metrics := NewTestMetrics()
		
		assert.NotPanics(t, func() {
			metrics.Counter("metric-with-dashes", 1, nil)
			metrics.Gauge("metric_with_underscores", 42, nil)
			metrics.Histogram("metric.with.dots", 0.5, nil)
			metrics.Timer("metric:with:colons", 100, nil)
		})
	})

	t.Run("SpecialCharactersInLabels", func(t *testing.T) {
		metrics := NewPrometheusMetrics()
		
		labels := map[string]string{
			"label-with-dashes":      "value-1",
			"label_with_underscores": "value_2",
			"label.with.dots":        "value.3",
			"label:with:colons":      "value:4",
			"label with spaces":      "value with spaces",
			"unicode_label_测试":      "unicode_value_测试",
		}
		
		assert.NotPanics(t, func() {
			metrics.Counter("special_chars_counter", 1, labels)
			metrics.Gauge("special_chars_gauge", 42, labels)
			metrics.Histogram("special_chars_histogram", 0.5, labels)
			metrics.Timer("special_chars_timer", 100, labels)
		})
	})

	t.Run("LongMetricNames", func(t *testing.T) {
		metrics := NewNoOpMetrics()
		longName := "very_long_metric_name_that_might_exceed_normal_limits_but_should_still_be_handled_gracefully_without_causing_any_issues"
		
		assert.NotPanics(t, func() {
			metrics.Counter(longName, 1, nil)
			metrics.Gauge(longName, 42, nil)
			metrics.Histogram(longName, 0.5, nil)
			metrics.Timer(longName, 100, nil)
		})
	})
}

// Benchmark tests
func BenchmarkNoOpMetricsCounter(b *testing.B) {
	metrics := NewNoOpMetrics()
	labels := map[string]string{"service": "test"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.Counter("benchmark_counter", 1, labels)
	}
}

func BenchmarkNoOpMetricsGauge(b *testing.B) {
	metrics := NewNoOpMetrics()
	labels := map[string]string{"instance": "test"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.Gauge("benchmark_gauge", float64(i), labels)
	}
}

func BenchmarkNoOpMetricsHistogram(b *testing.B) {
	metrics := NewNoOpMetrics()
	labels := map[string]string{"method": "GET"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.Histogram("benchmark_histogram", float64(i)/1000.0, labels)
	}
}

func BenchmarkNoOpMetricsTimer(b *testing.B) {
	metrics := NewNoOpMetrics()
	labels := map[string]string{"operation": "test"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.Timer("benchmark_timer", float64(i), labels)
	}
}

func BenchmarkPlaceholderMetricsCounter(b *testing.B) {
	metrics := &PlaceholderMetrics{Port: 9090}
	labels := map[string]string{"service": "test"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.Counter("benchmark_counter", 1, labels)
	}
}

func BenchmarkMetricsCreation(b *testing.B) {
	b.Run("NewNoOpMetrics", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewNoOpMetrics()
		}
	})
	
	b.Run("NewTestMetrics", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewTestMetrics()
		}
	})
	
	b.Run("NewPrometheusMetrics", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewPrometheusMetrics()
		}
	})
}

// Integration tests
func TestMetricsIntegration(t *testing.T) {
	t.Run("TypicalApplicationUsage", func(t *testing.T) {
		metrics := NewPrometheusMetrics()
		
		// Simulate typical application metrics
		assert.NotPanics(t, func() {
			// Application startup
			metrics.Counter("app_starts_total", 1, map[string]string{
				"version": "1.0.0",
				"env":     "test",
			})
			
			// Request processing
			for i := 0; i < 10; i++ {
				metrics.Counter("http_requests_total", 1, map[string]string{
					"method": "POST",
					"path":   "/api/search",
					"status": "200",
				})
				
				metrics.Histogram("http_request_duration_seconds", float64(i)*0.01, map[string]string{
					"method": "POST",
					"path":   "/api/search",
				})
			}
			
			// System metrics
			metrics.Gauge("memory_usage_bytes", 1024*1024*100, map[string]string{
				"type": "heap",
			})
			
			metrics.Gauge("cpu_usage_percent", 45.5, nil)
			
			// Operation timing
			metrics.Timer("memory_search_duration_ms", 123.45, map[string]string{
				"operation": "textual_search",
				"cube_id":   "cube123",
			})
			
			// Error tracking
			metrics.Counter("errors_total", 1, map[string]string{
				"type":    "validation",
				"service": "memgos",
			})
		})
	})

	t.Run("MetricsChaining", func(t *testing.T) {
		// Test that different metrics implementations can be used interchangeably
		implementations := []interfaces.Metrics{
			NewNoOpMetrics(),
			NewTestMetrics(),
			NewPrometheusMetrics(),
		}
		
		for i, metrics := range implementations {
			assert.NotPanics(t, func() {
				metrics.Counter("chaining_test", float64(i), map[string]string{
					"implementation": string(rune(i)),
				})
			})
		}
	})
}

// Example tests for documentation
func ExampleNewNoOpMetrics() {
	metrics := NewNoOpMetrics()
	
	// All operations are no-ops
	metrics.Counter("requests_total", 1, map[string]string{"method": "GET"})
	metrics.Gauge("memory_usage", 1024, nil)
	metrics.Histogram("request_duration", 0.123, nil)
	metrics.Timer("operation_time", 456.78, nil)
	
	// These calls don't actually record any metrics
}

func ExamplePlaceholderMetrics() {
	metrics := &PlaceholderMetrics{Port: 9090}
	
	// Record various types of metrics
	metrics.Counter("api_requests_total", 1, map[string]string{
		"method": "POST",
		"path":   "/api/chat",
	})
	
	metrics.Gauge("active_connections", 25, map[string]string{
		"service": "memgos",
	})
	
	metrics.Histogram("response_time_seconds", 0.234, map[string]string{
		"endpoint": "/search",
	})
	
	metrics.Timer("database_query_ms", 150.5, map[string]string{
		"query_type": "select",
	})
}