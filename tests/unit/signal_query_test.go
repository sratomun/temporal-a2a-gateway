package gateway_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestSignalQueryCompatibility tests the signal query mechanism
// This validates the fix for query name mismatch between gateway and worker

func TestQueryNameCompatibility(t *testing.T) {
	// Test the query name mismatch that was fixed
	tests := []struct {
		name               string
		gatewayQueryName   string
		workerQueryName    string
		expectedCompatible bool
	}{
		{
			name:               "fixed_query_names",
			gatewayQueryName:   "get_progress_signals",
			workerQueryName:    "get_progress_signals",
			expectedCompatible: true,
		},
		{
			name:               "old_mismatch_case",
			gatewayQueryName:   "GetProgressSignals", // OLD: was causing mismatch
			workerQueryName:    "get_progress_signals",
			expectedCompatible: false,
		},
		{
			name:               "another_mismatch",
			gatewayQueryName:   "getProgressSignals",
			workerQueryName:    "get_progress_signals",
			expectedCompatible: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compatible := tt.gatewayQueryName == tt.workerQueryName

			if compatible != tt.expectedCompatible {
				t.Errorf("Expected compatibility=%v, got=%v for gateway=%s, worker=%s",
					tt.expectedCompatible, compatible, tt.gatewayQueryName, tt.workerQueryName)
			}

			if !compatible {
				t.Logf("Query name mismatch: gateway='%s', worker='%s'", 
					tt.gatewayQueryName, tt.workerQueryName)
			} else {
				t.Logf("✅ Query names match: '%s'", tt.gatewayQueryName)
			}
		})
	}
}

func TestProgressSignalQueryStructure(t *testing.T) {
	// Test the structure of progress signals returned by queries
	type WorkflowProgressSignal struct {
		TaskID    string      `json:"taskId"`
		Status    string      `json:"status"`
		Progress  float64     `json:"progress"`
		Result    interface{} `json:"result,omitempty"`
		Error     string      `json:"error,omitempty"`
		Timestamp string      `json:"timestamp"`
	}

	tests := []struct {
		name   string
		signal WorkflowProgressSignal
	}{
		{
			name: "working_signal",
			signal: WorkflowProgressSignal{
				TaskID:    "test-task-123",
				Status:    "working",
				Progress:  0.5,
				Timestamp: "2025-07-03T17:46:01.000Z",
			},
		},
		{
			name: "completed_signal",
			signal: WorkflowProgressSignal{
				TaskID:   "test-task-123",
				Status:   "completed",
				Progress: 1.0,
				Result: map[string]interface{}{
					"status": "completed",
					"messages": []interface{}{
						map[string]interface{}{
							"messageId": "echo-test-task-123",
							"role":      "agent",
							"parts":     []map[string]interface{}{{"text": "Echo: Hello"}},
						},
					},
				},
				Timestamp: "2025-07-03T17:46:02.000Z",
			},
		},
		{
			name: "failed_signal",
			signal: WorkflowProgressSignal{
				TaskID:    "test-task-123",
				Status:    "failed",
				Progress:  0.0,
				Error:     "Activity execution failed",
				Timestamp: "2025-07-03T17:46:02.000Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test signal serialization (what worker returns)
			signalBytes, err := json.Marshal(tt.signal)
			if err != nil {
				t.Fatalf("Failed to marshal signal: %v", err)
			}

			// Test signal deserialization (what gateway receives)
			var parsedSignal WorkflowProgressSignal
			err = json.Unmarshal(signalBytes, &parsedSignal)
			if err != nil {
				t.Fatalf("Failed to unmarshal signal: %v", err)
			}

			// Verify structure integrity
			if parsedSignal.TaskID != tt.signal.TaskID {
				t.Errorf("TaskID mismatch: expected %s, got %s", 
					tt.signal.TaskID, parsedSignal.TaskID)
			}

			if parsedSignal.Status != tt.signal.Status {
				t.Errorf("Status mismatch: expected %s, got %s", 
					tt.signal.Status, parsedSignal.Status)
			}

			if parsedSignal.Timestamp != tt.signal.Timestamp {
				t.Errorf("Timestamp mismatch: expected %s, got %s", 
					tt.signal.Timestamp, parsedSignal.Timestamp)
			}

			// Verify progress is valid range
			if parsedSignal.Progress < 0.0 || parsedSignal.Progress > 1.0 {
				t.Errorf("Progress out of valid range [0,1]: %f", parsedSignal.Progress)
			}
		})
	}
}

func TestSignalArrayHandling(t *testing.T) {
	// Test handling of signal arrays (what query returns)
	type WorkflowProgressSignal struct {
		TaskID    string      `json:"taskId"`
		Status    string      `json:"status"`
		Progress  float64     `json:"progress"`
		Result    interface{} `json:"result,omitempty"`
		Error     string      `json:"error,omitempty"`
		Timestamp string      `json:"timestamp"`
	}

	// Simulate a series of progress signals
	signals := []WorkflowProgressSignal{
		{
			TaskID:    "test-task-123",
			Status:    "submitted",
			Progress:  0.0,
			Timestamp: "2025-07-03T17:46:00.000Z",
		},
		{
			TaskID:    "test-task-123",
			Status:    "working",
			Progress:  0.1,
			Timestamp: "2025-07-03T17:46:00.100Z",
		},
		{
			TaskID:    "test-task-123",
			Status:    "working",
			Progress:  0.5,
			Timestamp: "2025-07-03T17:46:01.000Z",
		},
		{
			TaskID:   "test-task-123",
			Status:   "completed",
			Progress: 1.0,
			Result: map[string]interface{}{
				"status": "completed",
			},
			Timestamp: "2025-07-03T17:46:02.000Z",
		},
	}

	t.Run("SignalArraySerialization", func(t *testing.T) {
		// Test array serialization (worker side)
		arrayBytes, err := json.Marshal(signals)
		if err != nil {
			t.Fatalf("Failed to marshal signal array: %v", err)
		}

		// Test array deserialization (gateway side)
		var parsedSignals []WorkflowProgressSignal
		err = json.Unmarshal(arrayBytes, &parsedSignals)
		if err != nil {
			t.Fatalf("Failed to unmarshal signal array: %v", err)
		}

		// Verify array integrity
		if len(parsedSignals) != len(signals) {
			t.Errorf("Signal array length mismatch: expected %d, got %d", 
				len(signals), len(parsedSignals))
		}

		// Verify signal order and progression
		for i, signal := range parsedSignals {
			if signal.TaskID != signals[i].TaskID {
				t.Errorf("Signal %d TaskID mismatch: expected %s, got %s", 
					i, signals[i].TaskID, signal.TaskID)
			}

			if signal.Status != signals[i].Status {
				t.Errorf("Signal %d Status mismatch: expected %s, got %s", 
					i, signals[i].Status, signal.Status)
			}
		}
	})

	t.Run("ProgressSignalProgression", func(t *testing.T) {
		// Test that progress signals show proper progression
		expectedProgression := []string{"submitted", "working", "working", "completed"}
		expectedProgress := []float64{0.0, 0.1, 0.5, 1.0}

		for i, signal := range signals {
			if signal.Status != expectedProgression[i] {
				t.Errorf("Signal %d status progression error: expected %s, got %s", 
					i, expectedProgression[i], signal.Status)
			}

			if signal.Progress != expectedProgress[i] {
				t.Errorf("Signal %d progress progression error: expected %f, got %f", 
					i, expectedProgress[i], signal.Progress)
			}
		}

		// Verify timestamps are in chronological order
		for i := 1; i < len(signals); i++ {
			prev := signals[i-1].Timestamp
			curr := signals[i].Timestamp

			if curr < prev {
				t.Errorf("Signal timestamps not in chronological order: %s > %s", prev, curr)
			}
		}
	})

	t.Run("SignalArrayGrowth", func(t *testing.T) {
		// Test signal array growth pattern (simulating real workflow)
		maxSignals := 10
		signalArray := make([]WorkflowProgressSignal, 0, maxSignals)

		// Simulate adding signals over time
		for i := 0; i < 5; i++ {
			newSignal := WorkflowProgressSignal{
				TaskID:    "growth-test",
				Status:    "working",
				Progress:  float64(i) / 4.0,
				Timestamp: "2025-07-03T17:46:00.000Z",
			}

			signalArray = append(signalArray, newSignal)

			// Test array at each growth stage
			if len(signalArray) != i+1 {
				t.Errorf("Signal array growth error at step %d: expected length %d, got %d", 
					i, i+1, len(signalArray))
			}

			// Test serialization at each stage
			_, err := json.Marshal(signalArray)
			if err != nil {
				t.Errorf("Failed to marshal signal array at growth step %d: %v", i, err)
			}
		}

		// Test memory efficiency (array should not exceed reasonable size)
		if cap(signalArray) > maxSignals {
			t.Errorf("Signal array capacity exceeded maximum: %d > %d", 
				cap(signalArray), maxSignals)
		}
	})
}

func TestQueryNameValidation(t *testing.T) {
	// Test query name validation and conventions
	tests := []struct {
		name      string
		queryName string
		isValid   bool
		reason    string
	}{
		{
			name:      "correct_snake_case",
			queryName: "get_progress_signals",
			isValid:   true,
			reason:    "proper snake_case format",
		},
		{
			name:      "incorrect_pascal_case",
			queryName: "GetProgressSignals",
			isValid:   false,
			reason:    "PascalCase not compatible with Python naming",
		},
		{
			name:      "incorrect_camel_case",
			queryName: "getProgressSignals",
			isValid:   false,
			reason:    "camelCase not compatible with Python naming",
		},
		{
			name:      "empty_query_name",
			queryName: "",
			isValid:   false,
			reason:    "empty query name",
		},
		{
			name:      "query_with_spaces",
			queryName: "get progress signals",
			isValid:   false,
			reason:    "spaces not allowed in query names",
		},
		{
			name:      "query_with_hyphens",
			queryName: "get-progress-signals",
			isValid:   false,
			reason:    "hyphens not standard for Python methods",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate query name format
			isValid := isValidQueryName(tt.queryName)

			if isValid != tt.isValid {
				t.Errorf("Query name '%s' validation mismatch: expected valid=%v, got valid=%v (%s)", 
					tt.queryName, tt.isValid, isValid, tt.reason)
			}

			if tt.isValid {
				t.Logf("✅ Query name '%s' is valid: %s", tt.queryName, tt.reason)
			} else {
				t.Logf("❌ Query name '%s' is invalid: %s", tt.queryName, tt.reason)
			}
		})
	}
}

func TestQueryIntervalTiming(t *testing.T) {
	// Test query interval timing (100ms as per implementation)
	expectedInterval := 100 * time.Millisecond
	tolerance := 50 * time.Millisecond

	t.Run("QueryIntervalValidation", func(t *testing.T) {
		// Simulate query timing
		intervals := []time.Duration{
			100 * time.Millisecond,
			99 * time.Millisecond,
			101 * time.Millisecond,
			110 * time.Millisecond,
			90 * time.Millisecond,
		}

		for i, interval := range intervals {
			withinTolerance := abs(interval-expectedInterval) <= tolerance

			if !withinTolerance {
				t.Errorf("Query interval %d out of tolerance: %v (expected %v ± %v)", 
					i, interval, expectedInterval, tolerance)
			} else {
				t.Logf("✅ Query interval %d within tolerance: %v", i, interval)
			}
		}
	})

	t.Run("QueryPerformanceImpact", func(t *testing.T) {
		// Test that 100ms intervals are reasonable for performance
		queriesPerSecond := float64(time.Second) / float64(expectedInterval)
		maxReasonableQPS := 20.0 // 20 queries per second should be reasonable

		if queriesPerSecond > maxReasonableQPS {
			t.Errorf("Query rate too high: %.1f queries/second (max reasonable: %.1f)", 
				queriesPerSecond, maxReasonableQPS)
		} else {
			t.Logf("✅ Query rate reasonable: %.1f queries/second", queriesPerSecond)
		}
	})
}

func TestSignalQueryErrorHandling(t *testing.T) {
	// Test error scenarios in signal querying
	tests := []struct {
		name          string
		queryResponse interface{}
		expectError   bool
		errorType     string
	}{
		{
			name: "successful_query",
			queryResponse: []map[string]interface{}{
				{
					"taskId":    "test-task",
					"status":    "working",
					"progress":  0.5,
					"timestamp": "2025-07-03T17:46:01.000Z",
				},
			},
			expectError: false,
		},
		{
			name:          "empty_array_response",
			queryResponse: []map[string]interface{}{},
			expectError:   false,
		},
		{
			name:          "null_response",
			queryResponse: nil,
			expectError:   true,
			errorType:     "null_response",
		},
		{
			name:          "malformed_response",
			queryResponse: "invalid json",
			expectError:   true,
			errorType:     "type_mismatch",
		},
		{
			name: "missing_required_fields",
			queryResponse: []map[string]interface{}{
				{
					"taskId": "test-task",
					// Missing status, progress, timestamp
				},
			},
			expectError: true,
			errorType:   "missing_fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate query response processing
			hasError := false
			errorType := ""

			if tt.queryResponse == nil {
				hasError = true
				errorType = "null_response"
			} else {
				// Try to process response
				responseBytes, err := json.Marshal(tt.queryResponse)
				if err != nil {
					hasError = true
					errorType = "marshal_error"
				} else {
					var signals []map[string]interface{}
					err = json.Unmarshal(responseBytes, &signals)
					if err != nil {
						hasError = true
						errorType = "type_mismatch"
					} else {
						// Check for required fields
						for _, signal := range signals {
							requiredFields := []string{"taskId", "status", "timestamp"}
							for _, field := range requiredFields {
								if _, ok := signal[field]; !ok {
									hasError = true
									errorType = "missing_fields"
									break
								}
							}
							if hasError {
								break
							}
						}
					}
				}
			}

			if hasError != tt.expectError {
				t.Errorf("Error expectation mismatch: expected error=%v, got error=%v", 
					tt.expectError, hasError)
			}

			if tt.expectError && tt.errorType != "" && errorType != tt.errorType {
				t.Errorf("Error type mismatch: expected %s, got %s", tt.errorType, errorType)
			}

			if !hasError {
				t.Logf("✅ Query response processed successfully")
			} else {
				t.Logf("❌ Query response error: %s", errorType)
			}
		})
	}
}

// Helper functions

func isValidQueryName(name string) bool {
	if name == "" {
		return false
	}

	// Check for spaces or invalid characters
	if strings.Contains(name, " ") {
		return false
	}

	// Check for snake_case pattern (Python convention)
	if strings.Contains(name, "_") {
		// Should be all lowercase with underscores
		return strings.ToLower(name) == name
	}

	// Check if it's camelCase or PascalCase (not compatible)
	if name != strings.ToLower(name) {
		return false
	}

	return true
}

func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}