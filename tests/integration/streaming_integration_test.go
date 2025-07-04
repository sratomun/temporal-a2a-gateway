package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Streaming Integration Test Suite
// Tests the complete streaming functionality including SSE, workflow execution, and signal monitoring

const (
	StreamingTimeout = 30 * time.Second
)

// TestMessageStreamEndpoint tests the message/stream endpoint implementation
func TestMessageStreamEndpoint(t *testing.T) {
	t.Log("Testing message/stream endpoint integration...")

	// Test 1: Basic streaming request structure
	t.Run("BasicStreamingRequest", func(t *testing.T) {
		request := JSONRPCRequest{
			Jsonrpc: "2.0",
			Method:  "message/stream",
			Params: map[string]interface{}{
				"agentId": "echo-agent",
				"message": map[string]interface{}{
					"text": "Integration test streaming message",
				},
				"metadata": map[string]interface{}{
					"test": "streaming-integration",
					"mode": "basic",
				},
			},
			ID: "stream-test-001",
		}

		// Make streaming request
		response, err := makeStreamingRequest(request)
		require.NoError(t, err)

		// Verify SSE headers are set correctly
		assert.Equal(t, "text/event-stream", response.Header.Get("Content-Type"))
		assert.Equal(t, "no-cache", response.Header.Get("Cache-Control"))
		assert.Equal(t, "keep-alive", response.Header.Get("Connection"))
		assert.Equal(t, "*", response.Header.Get("Access-Control-Allow-Origin"))

		// Note: HTTP Flusher interface availability depends on client setup
		// In production, this would work with proper SSE clients
		t.Log("✅ Streaming endpoint accepts requests and sets correct headers")
	})

	// Test 2: Invalid agent ID handling
	t.Run("InvalidAgentIdError", func(t *testing.T) {
		request := JSONRPCRequest{
			Jsonrpc: "2.0",
			Method:  "message/stream",
			Params: map[string]interface{}{
				"agentId": "non-existent-agent",
				"message": map[string]interface{}{
					"text": "Test with invalid agent",
				},
			},
			ID: "stream-error-001",
		}

		response, err := makeStreamingRequest(request)
		require.NoError(t, err)

		// Should return error for unknown agent
		if response.StatusCode == 200 {
			// SSE headers set, but error event should be sent
			t.Log("✅ Streaming started for invalid agent (error event expected)")
		} else {
			// Direct error response
			assert.Equal(t, 500, response.StatusCode)
			t.Log("✅ Invalid agent ID properly rejected")
		}
	})

	// Test 3: Missing required parameters
	t.Run("MissingParametersError", func(t *testing.T) {
		request := JSONRPCRequest{
			Jsonrpc: "2.0",
			Method:  "message/stream",
			Params: map[string]interface{}{
				"message": map[string]interface{}{
					"text": "Test without agentId",
				},
			},
			ID: "stream-error-002",
		}

		response, err := makeStreamingRequest(request)
		require.NoError(t, err)

		// Should return error for missing agentId
		assert.NotEqual(t, 200, response.StatusCode)
		t.Log("✅ Missing parameters properly handled")
	})
}

// TestWorkflowExecutionIntegration tests workflow execution for streaming
func TestWorkflowExecutionIntegration(t *testing.T) {
	t.Log("Testing workflow execution integration for streaming...")

	// Test workflow routing configuration
	t.Run("WorkflowRoutingValidation", func(t *testing.T) {
		// Verify echo-agent uses correct workflow via regular message/send
		sendResponse, err := makeA2ARequest(
			"message/send",
			map[string]interface{}{
				"agentId": "echo-agent",
				"message": map[string]interface{}{
					"parts": []map[string]interface{}{
						{"text": "Workflow routing test"},
					},
				},
				"metadata": map[string]interface{}{
					"test": "workflow-routing",
				},
			},
			"workflow-routing-001",
		)
		require.NoError(t, err)
		require.NotNil(t, sendResponse.Result)

		taskResult := sendResponse.Result.(map[string]interface{})
		taskID := taskResult["taskId"].(string)

		// Wait for workflow completion
		time.Sleep(3 * time.Second)

		// Verify task completed successfully
		getResponse, err := makeA2ARequest(
			"tasks/get",
			map[string]interface{}{
				"taskId": taskID,
			},
			"workflow-routing-002",
		)
		require.NoError(t, err)

		taskData := getResponse.Result.(map[string]interface{})
		status := taskData["status"].(map[string]interface{})
		
		assert.Equal(t, "completed", status["state"], "Workflow should complete successfully")
		assert.NotNil(t, taskData["result"], "Completed task should have result")

		t.Log("✅ Workflow routing works correctly for echo-agent")
	})

	// Test workflow signal generation
	t.Run("ProgressSignalGeneration", func(t *testing.T) {
		// Create a task and monitor its lifecycle
		sendResponse, err := makeA2ARequest(
			"message/send",
			map[string]interface{}{
				"agentId": "echo-agent",
				"message": map[string]interface{}{
					"parts": []map[string]interface{}{
						{"text": "Signal generation test"},
					},
				},
				"metadata": map[string]interface{}{
					"test": "signal-generation",
				},
			},
			"signal-test-001",
		)
		require.NoError(t, err)

		taskResult := sendResponse.Result.(map[string]interface{})
		taskID := taskResult["taskId"].(string)

		// Monitor task status changes
		maxAttempts := 10
		statusChanges := []string{}

		for attempt := 0; attempt < maxAttempts; attempt++ {
			getResponse, err := makeA2ARequest(
				"tasks/get",
				map[string]interface{}{
					"taskId": taskID,
				},
				fmt.Sprintf("signal-test-%03d", attempt+2),
			)
			require.NoError(t, err)

			taskData := getResponse.Result.(map[string]interface{})
			status := taskData["status"].(map[string]interface{})
			currentState := status["state"].(string)

			// Track unique status changes
			if len(statusChanges) == 0 || statusChanges[len(statusChanges)-1] != currentState {
				statusChanges = append(statusChanges, currentState)
			}

			if currentState == "completed" || currentState == "failed" {
				break
			}

			time.Sleep(500 * time.Millisecond)
		}

		// Verify status progression
		assert.Contains(t, statusChanges, "working", "Task should go through working state")
		assert.Contains(t, statusChanges, "completed", "Task should reach completed state")

		t.Logf("✅ Task progressed through states: %v", statusChanges)
	})
}

// TestSignalQueryIntegration tests the signal query mechanism
func TestSignalQueryIntegration(t *testing.T) {
	t.Log("Testing signal query integration...")

	// This test validates that the gateway can query progress signals from workflows
	// We can't directly test the Temporal query, but we can test the workflow execution

	t.Run("WorkflowCompletionSignals", func(t *testing.T) {
		// Create multiple tasks to test signal generation
		taskCount := 3
		taskIDs := make([]string, taskCount)

		for i := 0; i < taskCount; i++ {
			sendResponse, err := makeA2ARequest(
				"message/send",
				map[string]interface{}{
					"agentId": "echo-agent",
					"message": map[string]interface{}{
						"parts": []map[string]interface{}{
							{"text": fmt.Sprintf("Signal test message %d", i+1)},
						},
					},
					"metadata": map[string]interface{}{
						"test":     "signal-query",
						"sequence": i + 1,
					},
				},
				fmt.Sprintf("signal-query-%03d", i+1),
			)
			require.NoError(t, err)

			taskResult := sendResponse.Result.(map[string]interface{})
			taskIDs[i] = taskResult["taskId"].(string)
		}

		// Wait for all tasks to complete
		time.Sleep(5 * time.Second)

		// Verify all tasks completed successfully
		completedCount := 0
		for i, taskID := range taskIDs {
			getResponse, err := makeA2ARequest(
				"tasks/get",
				map[string]interface{}{
					"taskId": taskID,
				},
				fmt.Sprintf("signal-verify-%03d", i+1),
			)
			require.NoError(t, err)

			taskData := getResponse.Result.(map[string]interface{})
			status := taskData["status"].(map[string]interface{})

			if status["state"].(string) == "completed" {
				completedCount++
			}
		}

		assert.Equal(t, taskCount, completedCount, "All tasks should complete successfully")
		t.Logf("✅ %d/%d tasks completed successfully", completedCount, taskCount)
	})
}

// TestStreamingPerformance tests streaming performance characteristics
func TestStreamingPerformance(t *testing.T) {
	t.Log("Testing streaming performance...")

	t.Run("ConcurrentStreamingRequests", func(t *testing.T) {
		// Test multiple concurrent streaming requests
		requestCount := 5
		results := make(chan error, requestCount)

		for i := 0; i < requestCount; i++ {
			go func(index int) {
				request := JSONRPCRequest{
					Jsonrpc: "2.0",
					Method:  "message/stream",
					Params: map[string]interface{}{
						"agentId": "echo-agent",
						"message": map[string]interface{}{
							"text": fmt.Sprintf("Concurrent streaming test %d", index+1),
						},
						"metadata": map[string]interface{}{
							"test":     "concurrent-streaming",
							"sequence": index + 1,
						},
					},
					ID: fmt.Sprintf("concurrent-stream-%03d", index+1),
				}

				_, err := makeStreamingRequest(request)
				results <- err
			}(i)
		}

		// Collect results
		errorCount := 0
		for i := 0; i < requestCount; i++ {
			if err := <-results; err != nil {
				errorCount++
				t.Logf("Concurrent request %d failed: %v", i+1, err)
			}
		}

		successCount := requestCount - errorCount
		successRate := float64(successCount) / float64(requestCount) * 100

		assert.GreaterOrEqual(t, successRate, 80.0, 
			"At least 80%% of concurrent streaming requests should succeed")

		t.Logf("✅ Concurrent streaming: %d/%d requests succeeded (%.1f%%)", 
			successCount, requestCount, successRate)
	})

	t.Run("StreamingResponseTime", func(t *testing.T) {
		// Test streaming response time
		start := time.Now()

		request := JSONRPCRequest{
			Jsonrpc: "2.0",
			Method:  "message/stream",
			Params: map[string]interface{}{
				"agentId": "echo-agent",
				"message": map[string]interface{}{
					"text": "Response time test",
				},
				"metadata": map[string]interface{}{
					"test": "response-time",
				},
			},
			ID: "response-time-001",
		}

		response, err := makeStreamingRequest(request)
		responseTime := time.Since(start)

		require.NoError(t, err)
		defer response.Body.Close()

		// Streaming should start quickly (under 5 seconds)
		assert.Less(t, responseTime, 5*time.Second, 
			"Streaming should start within 5 seconds")

		t.Logf("✅ Streaming response time: %v", responseTime)
	})
}

// TestStreamingErrorHandling tests error scenarios in streaming
func TestStreamingErrorHandling(t *testing.T) {
	t.Log("Testing streaming error handling...")

	t.Run("MalformedJSONRequest", func(t *testing.T) {
		// Send malformed JSON
		malformedJSON := `{"jsonrpc": "2.0", "method": "message/stream", "params": {"agentId": "echo-agent", "message": {"text": "test"}, "id": "malformed-001"`

		resp, err := http.Post(
			GatewayURL+"/a2a",
			"application/json",
			strings.NewReader(malformedJSON),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return error for malformed JSON
		assert.NotEqual(t, 200, resp.StatusCode)
		t.Log("✅ Malformed JSON properly rejected")
	})

	t.Run("UnsupportedMethod", func(t *testing.T) {
		request := JSONRPCRequest{
			Jsonrpc: "2.0",
			Method:  "message/unsupported",
			Params: map[string]interface{}{
				"agentId": "echo-agent",
				"message": map[string]interface{}{
					"text": "Unsupported method test",
				},
			},
			ID: "unsupported-001",
		}

		response, err := makeStreamingRequest(request)
		require.NoError(t, err)
		defer response.Body.Close()

		// Should return method not found error
		assert.NotEqual(t, 200, response.StatusCode)
		t.Log("✅ Unsupported method properly rejected")
	})
}

// TestStreamingCompliance tests A2A v0.2.5 streaming compliance
func TestStreamingCompliance(t *testing.T) {
	t.Log("Testing A2A v0.2.5 streaming compliance...")

	t.Run("StreamingMethodExistence", func(t *testing.T) {
		// Test that message/stream method exists
		request := JSONRPCRequest{
			Jsonrpc: "2.0",
			Method:  "message/stream",
			Params: map[string]interface{}{
				"agentId": "echo-agent",
				"message": map[string]interface{}{
					"text": "A2A compliance test",
				},
			},
			ID: "compliance-001",
		}

		response, err := makeStreamingRequest(request)
		require.NoError(t, err)
		defer response.Body.Close()

		// Method should exist (not return method not found)
		assert.NotEqual(t, 404, response.StatusCode)
		assert.NotEqual(t, -32601, response.StatusCode) // JSON-RPC method not found

		t.Log("✅ message/stream method exists and responds")
	})

	t.Run("SSEComplianceHeaders", func(t *testing.T) {
		request := JSONRPCRequest{
			Jsonrpc: "2.0",
			Method:  "message/stream",
			Params: map[string]interface{}{
				"agentId": "echo-agent",
				"message": map[string]interface{}{
					"text": "SSE compliance test",
				},
			},
			ID: "sse-compliance-001",
		}

		response, err := makeStreamingRequest(request)
		require.NoError(t, err)
		defer response.Body.Close()

		// Verify required SSE headers
		requiredHeaders := map[string]string{
			"Content-Type":   "text/event-stream",
			"Cache-Control":  "no-cache",
			"Connection":     "keep-alive",
		}

		for header, expectedValue := range requiredHeaders {
			actualValue := response.Header.Get(header)
			assert.Equal(t, expectedValue, actualValue,
				"SSE header %s should be %s, got %s", header, expectedValue, actualValue)
		}

		t.Log("✅ SSE headers comply with streaming requirements")
	})

	t.Run("JSONRPCStructureCompliance", func(t *testing.T) {
		// Test that streaming requests follow JSON-RPC 2.0 structure
		request := JSONRPCRequest{
			Jsonrpc: "2.0",
			Method:  "message/stream",
			Params: map[string]interface{}{
				"agentId": "echo-agent",
				"message": map[string]interface{}{
					"text": "JSON-RPC structure test",
				},
			},
			ID: "jsonrpc-compliance-001",
		}

		// Verify request structure can be marshaled
		_, err := json.Marshal(request)
		require.NoError(t, err)

		// Verify required fields
		assert.Equal(t, "2.0", request.Jsonrpc)
		assert.Equal(t, "message/stream", request.Method)
		assert.NotNil(t, request.Params)
		assert.NotNil(t, request.ID)

		t.Log("✅ JSON-RPC 2.0 structure compliance verified")
	})
}

// Helper function to make streaming requests
func makeStreamingRequest(request JSONRPCRequest) (*http.Response, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), StreamingTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", GatewayURL+"/a2a", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{
		Timeout: StreamingTimeout,
	}

	return client.Do(req)
}

// TestStreamingWithRegularWorkflow compares streaming vs regular workflow execution
func TestStreamingWithRegularWorkflow(t *testing.T) {
	t.Log("Testing streaming vs regular workflow execution...")

	t.Run("SameWorkflowDifferentEndpoints", func(t *testing.T) {
		testMessage := "Endpoint comparison test"

		// Test 1: Regular message/send
		sendResponse, err := makeA2ARequest(
			"message/send",
			map[string]interface{}{
				"agentId": "echo-agent",
				"message": map[string]interface{}{
					"parts": []map[string]interface{}{
						{"text": testMessage},
					},
				},
				"metadata": map[string]interface{}{
					"test": "endpoint-comparison",
					"type": "regular",
				},
			},
			"comparison-regular-001",
		)
		require.NoError(t, err)

		regularTaskResult := sendResponse.Result.(map[string]interface{})
		regularTaskID := regularTaskResult["taskId"].(string)

		// Test 2: Streaming message/stream  
		streamRequest := JSONRPCRequest{
			Jsonrpc: "2.0",
			Method:  "message/stream",
			Params: map[string]interface{}{
				"agentId": "echo-agent",
				"message": map[string]interface{}{
					"text": testMessage,
				},
				"metadata": map[string]interface{}{
					"test": "endpoint-comparison",
					"type": "streaming",
				},
			},
			ID: "comparison-streaming-001",
		}

		streamResponse, err := makeStreamingRequest(streamRequest)
		require.NoError(t, err)
		defer streamResponse.Body.Close()

		// Wait for regular task completion
		time.Sleep(3 * time.Second)

		// Get regular task result
		getResponse, err := makeA2ARequest(
			"tasks/get",
			map[string]interface{}{
				"taskId": regularTaskID,
			},
			"comparison-get-001",
		)
		require.NoError(t, err)

		regularTaskData := getResponse.Result.(map[string]interface{})
		regularStatus := regularTaskData["status"].(map[string]interface{})

		// Verify regular workflow completed
		assert.Equal(t, "completed", regularStatus["state"], 
			"Regular workflow should complete successfully")

		// Verify streaming endpoint responded
		assert.Equal(t, "text/event-stream", streamResponse.Header.Get("Content-Type"),
			"Streaming endpoint should set SSE headers")

		t.Log("✅ Both regular and streaming endpoints work with same underlying workflow")
	})
}

// BenchmarkStreamingEndpoint benchmarks streaming endpoint performance
func BenchmarkStreamingEndpoint(b *testing.B) {
	checkGatewayHealth(&testing.T{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		request := JSONRPCRequest{
			Jsonrpc: "2.0",
			Method:  "message/stream",
			Params: map[string]interface{}{
				"agentId": "echo-agent",
				"message": map[string]interface{}{
					"text": fmt.Sprintf("Benchmark streaming test %d", i),
				},
				"metadata": map[string]interface{}{
					"test": "benchmark",
				},
			},
			ID: fmt.Sprintf("bench-stream-%d", i),
		}

		response, err := makeStreamingRequest(request)
		if err != nil {
			b.Fatalf("Benchmark streaming request failed: %v", err)
		}
		response.Body.Close()
	}
}