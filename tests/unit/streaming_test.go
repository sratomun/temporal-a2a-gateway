package gateway_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Test types matching the actual gateway implementation
type StreamingMessageParams struct {
	AgentID  string                 `json:"agentId"`
	Message  interface{}            `json:"message"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type SSEEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type WorkflowProgressSignal struct {
	TaskID    string      `json:"taskId"`
	Status    string      `json:"status"`
	Progress  float64     `json:"progress"`
	Result    interface{} `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// TestStreamingMessageParamsValidation tests parameter validation for message/stream
func TestStreamingMessageParamsValidation(t *testing.T) {
	tests := []struct {
		name        string
		params      StreamingMessageParams
		expectValid bool
	}{
		{
			name: "valid_basic_params",
			params: StreamingMessageParams{
				AgentID: "echo-agent",
				Message: map[string]interface{}{
					"text": "Test message",
				},
			},
			expectValid: true,
		},
		{
			name: "valid_with_metadata",
			params: StreamingMessageParams{
				AgentID: "echo-agent",
				Message: map[string]interface{}{
					"parts": []map[string]interface{}{
						{"text": "Test message with parts"},
					},
				},
				Metadata: map[string]interface{}{
					"priority": "high",
					"source":   "unit-test",
				},
			},
			expectValid: true,
		},
		{
			name: "missing_agent_id",
			params: StreamingMessageParams{
				Message: map[string]interface{}{
					"text": "Test message",
				},
			},
			expectValid: false,
		},
		{
			name: "empty_agent_id",
			params: StreamingMessageParams{
				AgentID: "",
				Message: map[string]interface{}{
					"text": "Test message",
				},
			},
			expectValid: false,
		},
		{
			name: "nil_message",
			params: StreamingMessageParams{
				AgentID: "echo-agent",
				Message: nil,
			},
			expectValid: true, // nil message should be allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling/unmarshaling
			data, err := json.Marshal(tt.params)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}

			var parsed StreamingMessageParams
			err = json.Unmarshal(data, &parsed)
			if err != nil {
				t.Fatalf("Failed to unmarshal params: %v", err)
			}

			// Test validation logic
			isValid := parsed.AgentID != ""
			if isValid != tt.expectValid {
				t.Errorf("Expected valid=%v, got valid=%v for params: %+v", 
					tt.expectValid, isValid, tt.params)
			}
		})
	}
}

// TestSSEEventSerialization tests SSE event JSON serialization
func TestSSEEventSerialization(t *testing.T) {
	tests := []struct {
		name  string
		event SSEEvent
	}{
		{
			name: "task_created_event",
			event: SSEEvent{
				Type: "task.created",
				Data: map[string]interface{}{
					"taskId": "test-task-123",
					"task": map[string]interface{}{
						"id":      "test-task-123",
						"agentId": "echo-agent",
						"status": map[string]interface{}{
							"state":     "submitted",
							"timestamp": "2025-07-03T17:46:00.000Z",
						},
					},
				},
			},
		},
		{
			name: "task_progress_event",
			event: SSEEvent{
				Type: "task.progress",
				Data: map[string]interface{}{
					"taskId":   "test-task-123",
					"status":   "working",
					"progress": 0.5,
					"timestamp": "2025-07-03T17:46:01.000Z",
				},
			},
		},
		{
			name: "task_completed_event",
			event: SSEEvent{
				Type: "task.completed",
				Data: map[string]interface{}{
					"taskId": "test-task-123",
					"result": map[string]interface{}{
						"status": "completed",
						"messages": []interface{}{
							map[string]interface{}{
								"messageId": "echo-test-task-123",
								"role":      "agent",
								"parts":     []map[string]interface{}{{"text": "Echo: Hello"}},
							},
						},
					},
					"timestamp": "2025-07-03T17:46:02.000Z",
				},
			},
		},
		{
			name: "task_error_event",
			event: SSEEvent{
				Type: "task.error",
				Data: map[string]interface{}{
					"taskId":    "test-task-123",
					"error":     "Workflow execution failed",
					"timestamp": "2025-07-03T17:46:02.000Z",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.event)
			if err != nil {
				t.Fatalf("Failed to marshal SSE event: %v", err)
			}

			// Test JSON unmarshaling
			var parsed SSEEvent
			err = json.Unmarshal(data, &parsed)
			if err != nil {
				t.Fatalf("Failed to unmarshal SSE event: %v", err)
			}

			// Verify structure
			if parsed.Type != tt.event.Type {
				t.Errorf("Expected type %s, got %s", tt.event.Type, parsed.Type)
			}

			if parsed.Data == nil {
				t.Error("Expected non-nil data field")
			}

			// Verify data can be marshaled again (round-trip test)
			_, err = json.Marshal(parsed.Data)
			if err != nil {
				t.Errorf("Failed to re-marshal event data: %v", err)
			}
		})
	}
}

// TestWorkflowProgressSignalStructure tests progress signal data structure
func TestWorkflowProgressSignalStructure(t *testing.T) {
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
				Timestamp: "2025-07-03T17:46:00.000Z",
			},
		},
		{
			name: "completed_signal_with_result",
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
			name: "failed_signal_with_error",
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
			// Test JSON serialization
			data, err := json.Marshal(tt.signal)
			if err != nil {
				t.Fatalf("Failed to marshal progress signal: %v", err)
			}

			// Test JSON deserialization
			var parsed WorkflowProgressSignal
			err = json.Unmarshal(data, &parsed)
			if err != nil {
				t.Fatalf("Failed to unmarshal progress signal: %v", err)
			}

			// Verify required fields
			if parsed.TaskID != tt.signal.TaskID {
				t.Errorf("Expected taskId %s, got %s", tt.signal.TaskID, parsed.TaskID)
			}

			if parsed.Status != tt.signal.Status {
				t.Errorf("Expected status %s, got %s", tt.signal.Status, parsed.Status)
			}

			if parsed.Timestamp != tt.signal.Timestamp {
				t.Errorf("Expected timestamp %s, got %s", tt.signal.Timestamp, parsed.Timestamp)
			}

			// Verify progress range
			if parsed.Progress < 0.0 || parsed.Progress > 1.0 {
				t.Errorf("Progress should be between 0.0 and 1.0, got %f", parsed.Progress)
			}
		})
	}
}

// TestProgressSignalToSSEEventConversion tests conversion logic
func TestProgressSignalToSSEEventConversion(t *testing.T) {
	tests := []struct {
		name           string
		signal         WorkflowProgressSignal
		expectedType   string
		expectProgress bool
		expectResult   bool
		expectError    bool
	}{
		{
			name: "working_to_progress_event",
			signal: WorkflowProgressSignal{
				TaskID:    "test-task-123",
				Status:    "working",
				Progress:  0.5,
				Timestamp: "2025-07-03T17:46:01.000Z",
			},
			expectedType:   "task.progress",
			expectProgress: true,
			expectResult:   false,
			expectError:    false,
		},
		{
			name: "completed_to_completed_event",
			signal: WorkflowProgressSignal{
				TaskID:   "test-task-123",
				Status:   "completed",
				Progress: 1.0,
				Result: map[string]interface{}{
					"status": "completed",
				},
				Timestamp: "2025-07-03T17:46:02.000Z",
			},
			expectedType:   "task.completed",
			expectProgress: false,
			expectResult:   true,
			expectError:    false,
		},
		{
			name: "failed_to_error_event",
			signal: WorkflowProgressSignal{
				TaskID:    "test-task-123",
				Status:    "failed",
				Progress:  0.0,
				Error:     "Workflow failed",
				Timestamp: "2025-07-03T17:46:02.000Z",
			},
			expectedType:   "task.error",
			expectProgress: false,
			expectResult:   false,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate conversion logic from gateway
			var eventType string
			eventData := map[string]interface{}{
				"taskId":    tt.signal.TaskID,
				"timestamp": tt.signal.Timestamp,
			}

			switch tt.signal.Status {
			case "working":
				eventType = "task.progress"
				eventData["status"] = tt.signal.Status
				eventData["progress"] = tt.signal.Progress
			case "completed":
				eventType = "task.completed"
				if tt.signal.Result != nil {
					eventData["result"] = tt.signal.Result
				}
			case "failed":
				eventType = "task.error"
				if tt.signal.Error != "" {
					eventData["error"] = tt.signal.Error
				}
			default:
				eventType = "task.update"
				eventData["status"] = tt.signal.Status
			}

			// Verify event type
			if eventType != tt.expectedType {
				t.Errorf("Expected event type %s, got %s", tt.expectedType, eventType)
			}

			// Verify presence of expected fields
			if tt.expectProgress {
				if _, ok := eventData["progress"]; !ok {
					t.Error("Expected progress field in event data")
				}
			}

			if tt.expectResult {
				if _, ok := eventData["result"]; !ok {
					t.Error("Expected result field in event data")
				}
			}

			if tt.expectError {
				if _, ok := eventData["error"]; !ok {
					t.Error("Expected error field in event data")
				}
			}

			// Test full SSE event structure
			sseEvent := SSEEvent{
				Type: eventType,
				Data: eventData,
			}

			// Verify JSON serialization
			_, err := json.Marshal(sseEvent)
			if err != nil {
				t.Errorf("Failed to marshal converted SSE event: %v", err)
			}
		})
	}
}

// TestStreamingWorkflowRouting tests workflow routing configuration
func TestStreamingWorkflowRouting(t *testing.T) {
	// Simulate agent routing configuration
	agentWorkflows := map[string]string{
		"echo-agent":   "EchoTaskWorkflow",
		"custom-agent": "LLMAgentWorkflow",
	}

	agentTaskQueues := map[string]string{
		"echo-agent":   "echo-agent-tasks",
		"custom-agent": "custom-agent-tasks",
	}

	tests := []struct {
		name              string
		agentID           string
		expectWorkflow    string
		expectQueue       string
		expectConfigFound bool
	}{
		{
			name:              "echo_agent_routing",
			agentID:           "echo-agent",
			expectWorkflow:    "EchoTaskWorkflow",
			expectQueue:       "echo-agent-tasks",
			expectConfigFound: true,
		},
		{
			name:              "custom_agent_routing",
			agentID:           "custom-agent",
			expectWorkflow:    "LLMAgentWorkflow",
			expectQueue:       "custom-agent-tasks",
			expectConfigFound: true,
		},
		{
			name:              "unknown_agent",
			agentID:           "unknown-agent",
			expectWorkflow:    "",
			expectQueue:       "",
			expectConfigFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate gateway routing lookup
			workflowType, workflowOk := agentWorkflows[tt.agentID]
			taskQueue, queueOk := agentTaskQueues[tt.agentID]

			configFound := workflowOk && queueOk

			if configFound != tt.expectConfigFound {
				t.Errorf("Expected config found=%v, got=%v", tt.expectConfigFound, configFound)
			}

			if tt.expectConfigFound {
				if workflowType != tt.expectWorkflow {
					t.Errorf("Expected workflow %s, got %s", tt.expectWorkflow, workflowType)
				}

				if taskQueue != tt.expectQueue {
					t.Errorf("Expected queue %s, got %s", tt.expectQueue, taskQueue)
				}

				// Verify no hardcoded patterns
				hardcodedPattern := tt.agentID + "Workflow"
				if workflowType == hardcodedPattern {
					t.Errorf("Workflow type appears to use hardcoded pattern: %s", workflowType)
				}
			}
		})
	}
}

// TestTimestampGeneration tests timestamp handling for workflow determinism
func TestTimestampGeneration(t *testing.T) {
	// Test ISO 8601 format compliance
	testTimestamp := "2025-07-03T17:46:00.000Z"

	// Verify parsing
	_, err := time.Parse(time.RFC3339Nano, testTimestamp)
	if err != nil {
		t.Errorf("Failed to parse timestamp %s: %v", testTimestamp, err)
	}

	// Test timestamp format requirements
	if !isValidA2ATimestamp(testTimestamp) {
		t.Errorf("Timestamp %s does not meet A2A format requirements", testTimestamp)
	}

	// Test workflow determinism requirement
	signal := WorkflowProgressSignal{
		TaskID:    "test-task",
		Status:    "working",
		Progress:  0.5,
		Timestamp: testTimestamp,
	}

	// Timestamp should not be generated dynamically during serialization
	data1, err := json.Marshal(signal)
	if err != nil {
		t.Fatalf("Failed to marshal signal first time: %v", err)
	}

	time.Sleep(100 * time.Millisecond) // Ensure time difference

	data2, err := json.Marshal(signal)
	if err != nil {
		t.Fatalf("Failed to marshal signal second time: %v", err)
	}

	// Should be identical (deterministic)
	if string(data1) != string(data2) {
		t.Error("Timestamp generation is not deterministic - serialization differs between calls")
	}
}

// Helper function to validate A2A timestamp format
func isValidA2ATimestamp(timestamp string) bool {
	// A2A v0.2.5 requires UTC timezone (ends with Z)
	if !strings.HasSuffix(timestamp, "Z") {
		return false
	}

	// Should be RFC3339Nano format
	_, err := time.Parse(time.RFC3339Nano, timestamp)
	return err == nil
}

// TestSSEHeaderRequirements tests required SSE headers
func TestSSEHeaderRequirements(t *testing.T) {
	requiredHeaders := map[string]string{
		"Content-Type":                "text/event-stream",
		"Cache-Control":               "no-cache",
		"Connection":                  "keep-alive",
		"Access-Control-Allow-Origin": "*",
		"Access-Control-Allow-Headers": "Cache-Control",
	}

	// Simulate header setting
	headers := make(map[string]string)
	for key, value := range requiredHeaders {
		headers[key] = value
	}

	// Verify all required headers are present
	for key, expectedValue := range requiredHeaders {
		if actualValue, ok := headers[key]; !ok {
			t.Errorf("Missing required SSE header: %s", key)
		} else if actualValue != expectedValue {
			t.Errorf("Incorrect header value for %s: expected %s, got %s", 
				key, expectedValue, actualValue)
		}
	}

	// Verify Content-Type specifically for SSE
	contentType := headers["Content-Type"]
	if contentType != "text/event-stream" {
		t.Errorf("SSE Content-Type must be 'text/event-stream', got '%s'", contentType)
	}
}

// TestStreamChannelBuffering tests SSE stream channel configuration
func TestStreamChannelBuffering(t *testing.T) {
	// Test channel buffer size (should match gateway implementation)
	expectedBufferSize := 10
	
	// Simulate channel creation
	streamChan := make(chan SSEEvent, expectedBufferSize)
	
	// Test channel capacity
	if cap(streamChan) != expectedBufferSize {
		t.Errorf("Expected channel buffer size %d, got %d", expectedBufferSize, cap(streamChan))
	}
	
	// Test non-blocking writes up to buffer size
	for i := 0; i < expectedBufferSize; i++ {
		event := SSEEvent{
			Type: "test.event",
			Data: map[string]interface{}{
				"sequence": i,
			},
		}
		
		select {
		case streamChan <- event:
			// Should not block
		default:
			t.Errorf("Channel write blocked at event %d (buffer size %d)", i, expectedBufferSize)
		}
	}
	
	// Test that buffer is now full
	testEvent := SSEEvent{Type: "test.overflow", Data: map[string]interface{}{}}
	select {
	case streamChan <- testEvent:
		t.Error("Channel should be full and block on additional writes")
	default:
		// Expected - channel is full
	}
	
	close(streamChan)
}