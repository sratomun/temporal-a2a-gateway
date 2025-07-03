package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ISO 8601 Timestamp Validation Test Suite
// Tests Agent 2's newISO8601Timestamp() implementation against containerized gateway
// Target: 100% coverage for timestamp format compliance

const (
	ISO8601Pattern = `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$`
	ContainerGatewayURL = "http://localhost:8080"
)

// Test setup for containerized gateway
func TestContainerizedGatewayTimestamps(t *testing.T) {
	// Ensure gateway is running in container
	t.Run("GatewayContainerHealth", testGatewayContainerHealth)
	
	// Core timestamp validation tests
	t.Run("TimestampFormatCompliance", testTimestampFormatCompliance)
	t.Run("TimestampPrecisionValidation", testTimestampPrecisionValidation)
	t.Run("TimestampTimezoneCompliance", testTimestampTimezoneCompliance)
	t.Run("TimestampConsistency", testTimestampConsistency)
	
	// Edge cases and regression tests
	t.Run("TimestampEdgeCases", testTimestampEdgeCases)
	t.Run("TimestampRegressionTests", testTimestampRegressionTests)
	
	// Integration tests across all A2A endpoints
	t.Run("EndToEndTimestampCompliance", testEndToEndTimestampCompliance)
}

func testGatewayContainerHealth(t *testing.T) {
	resp, err := http.Get(ContainerGatewayURL + "/health")
	require.NoError(t, err, "Failed to connect to containerized gateway")
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Gateway container not healthy")
	
	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(t, err)
	
	assert.Equal(t, "healthy", health["status"], "Gateway not healthy")
	
	// Validate health endpoint timestamp format
	timestamp := health["timestamp"].(string)
	validateISO8601Format(t, timestamp, "health endpoint timestamp")
	
	t.Log("✅ Gateway container is healthy and accessible")
}

func testTimestampFormatCompliance(t *testing.T) {
	t.Log("Testing ISO 8601 timestamp format compliance...")
	
	// Create task to get timestamps from gateway
	response, err := makeContainerA2ARequest(
		"message/send",
		map[string]interface{}{
			"agentId": "echo-agent",
			"message": map[string]interface{}{
				"text": "ISO 8601 format test",
			},
			"metadata": map[string]interface{}{
				"test": "iso8601-format",
			},
		},
		"iso8601-test-001",
	)
	require.NoError(t, err)
	require.NotNil(t, response.Result)
	
	taskResult := response.Result.(map[string]interface{})
	
	// Validate createdAt timestamp
	createdAt := taskResult["createdAt"].(string)
	validateISO8601Format(t, createdAt, "task createdAt")
	
	// Validate status timestamp
	status := taskResult["status"].(map[string]interface{})
	statusTimestamp := status["timestamp"].(string)
	validateISO8601Format(t, statusTimestamp, "task status timestamp")
	
	t.Log("✅ All timestamp fields comply with ISO 8601 format")
}

func testTimestampPrecisionValidation(t *testing.T) {
	t.Log("Testing timestamp millisecond precision...")
	
	// Create multiple tasks in quick succession
	timestamps := []string{}
	for i := 0; i < 5; i++ {
		response, err := makeContainerA2ARequest(
			"message/send",
			map[string]interface{}{
				"agentId": "echo-agent",
				"message": map[string]interface{}{
					"text": fmt.Sprintf("Precision test %d", i+1),
				},
			},
			fmt.Sprintf("precision-test-%03d", i+1),
		)
		require.NoError(t, err)
		
		taskResult := response.Result.(map[string]interface{})
		createdAt := taskResult["createdAt"].(string)
		timestamps = append(timestamps, createdAt)
		
		// Small delay to ensure different timestamps
		time.Sleep(5 * time.Millisecond)
	}
	
	// Verify all timestamps are different (millisecond precision working)
	for i := 1; i < len(timestamps); i++ {
		assert.NotEqual(t, timestamps[i-1], timestamps[i], 
			"Timestamps should be different with millisecond precision")
	}
	
	// Verify all timestamps have millisecond precision (.XXX)
	for _, timestamp := range timestamps {
		assert.Regexp(t, `\.\d{3}Z$`, timestamp, 
			"Timestamp should have 3-digit millisecond precision")
	}
	
	t.Logf("✅ Verified millisecond precision with %d unique timestamps", len(timestamps))
}

func testTimestampTimezoneCompliance(t *testing.T) {
	t.Log("Testing timestamp timezone compliance (UTC)...")
	
	response, err := makeContainerA2ARequest(
		"message/send",
		map[string]interface{}{
			"agentId": "echo-agent",
			"message": map[string]interface{}{
				"text": "Timezone test",
			},
		},
		"timezone-test-001",
	)
	require.NoError(t, err)
	
	taskResult := response.Result.(map[string]interface{})
	createdAt := taskResult["createdAt"].(string)
	
	// Verify UTC timezone (ends with Z)
	assert.True(t, len(createdAt) > 0 && createdAt[len(createdAt)-1] == 'Z',
		"Timestamp must end with 'Z' for UTC timezone")
	
	// Parse timestamp to verify it's valid UTC time
	parsedTime, err := time.Parse(time.RFC3339Nano, createdAt)
	require.NoError(t, err, "Should be valid RFC3339 timestamp")
	
	// Verify it's close to current time (within 10 seconds)
	timeDiff := time.Since(parsedTime).Abs()
	assert.True(t, timeDiff < 10*time.Second, 
		"Timestamp should be close to current time (got diff: %v)", timeDiff)
	
	// Verify it's in UTC
	assert.Equal(t, time.UTC, parsedTime.Location(), "Timestamp should be in UTC")
	
	t.Log("✅ Timestamp timezone compliance verified (UTC)")
}

func testTimestampConsistency(t *testing.T) {
	t.Log("Testing timestamp consistency across task lifecycle...")
	
	// Create task
	response, err := makeContainerA2ARequest(
		"message/send",
		map[string]interface{}{
			"agentId": "echo-agent",
			"message": map[string]interface{}{
				"text": "Consistency test",
			},
		},
		"consistency-test-001",
	)
	require.NoError(t, err)
	
	taskResult := response.Result.(map[string]interface{})
	taskID := taskResult["id"].(string)
	initialCreatedAt := taskResult["createdAt"].(string)
	
	// Wait for task completion and check timestamp consistency
	time.Sleep(3 * time.Second)
	
	getResponse, err := makeContainerA2ARequest(
		"tasks/get",
		map[string]interface{}{
			"id": taskID,
		},
		"consistency-test-002",
	)
	require.NoError(t, err)
	
	finalTask := getResponse.Result.(map[string]interface{})
	finalCreatedAt := finalTask["createdAt"].(string)
	
	// Verify createdAt timestamp doesn't change
	assert.Equal(t, initialCreatedAt, finalCreatedAt,
		"createdAt timestamp should remain consistent throughout task lifecycle")
	
	// Verify status timestamp is updated
	finalStatus := finalTask["status"].(map[string]interface{})
	finalStatusTimestamp := finalStatus["timestamp"].(string)
	
	validateISO8601Format(t, finalStatusTimestamp, "final status timestamp")
	
	t.Log("✅ Timestamp consistency verified across task lifecycle")
}

func testTimestampEdgeCases(t *testing.T) {
	t.Log("Testing timestamp edge cases...")
	
	testCases := []struct {
		name     string
		method   string
		params   map[string]interface{}
		validate func(t *testing.T, response *JSONRPCResponse)
	}{
		{
			name:   "minimal_message",
			method: "message/send",
			params: map[string]interface{}{
				"agentId": "echo-agent",
				"message": map[string]interface{}{
					"text": "",
				},
			},
			validate: func(t *testing.T, response *JSONRPCResponse) {
				taskResult := response.Result.(map[string]interface{})
				createdAt := taskResult["createdAt"].(string)
				validateISO8601Format(t, createdAt, "minimal message timestamp")
			},
		},
		{
			name:   "complex_metadata",
			method: "message/send",
			params: map[string]interface{}{
				"agentId": "echo-agent",
				"message": map[string]interface{}{
					"text": "Complex metadata test",
				},
				"metadata": map[string]interface{}{
					"nested": map[string]interface{}{
						"level": 2,
						"array": []string{"a", "b", "c"},
					},
					"timestamp": time.Now().Unix(),
				},
			},
			validate: func(t *testing.T, response *JSONRPCResponse) {
				taskResult := response.Result.(map[string]interface{})
				createdAt := taskResult["createdAt"].(string)
				validateISO8601Format(t, createdAt, "complex metadata timestamp")
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response, err := makeContainerA2ARequest(
				tc.method,
				tc.params,
				fmt.Sprintf("edge-case-%s", tc.name),
			)
			require.NoError(t, err)
			require.NotNil(t, response.Result)
			
			tc.validate(t, response)
		})
	}
	
	t.Log("✅ All timestamp edge cases passed")
}

func testTimestampRegressionTests(t *testing.T) {
	t.Log("Testing timestamp regression scenarios...")
	
	// Test concurrent task creation timestamps
	const concurrentTasks = 10
	timestamps := make(chan string, concurrentTasks)
	errors := make(chan error, concurrentTasks)
	
	for i := 0; i < concurrentTasks; i++ {
		go func(taskNum int) {
			response, err := makeContainerA2ARequest(
				"message/send",
				map[string]interface{}{
					"agentId": "echo-agent",
					"message": map[string]interface{}{
						"text": fmt.Sprintf("Concurrent test %d", taskNum),
					},
				},
				fmt.Sprintf("concurrent-%03d", taskNum),
			)
			
			if err != nil {
				errors <- err
				return
			}
			
			taskResult := response.Result.(map[string]interface{})
			createdAt := taskResult["createdAt"].(string)
			timestamps <- createdAt
		}(i)
	}
	
	// Collect results
	var collectedTimestamps []string
	for i := 0; i < concurrentTasks; i++ {
		select {
		case timestamp := <-timestamps:
			collectedTimestamps = append(collectedTimestamps, timestamp)
		case err := <-errors:
			t.Fatalf("Concurrent task failed: %v", err)
		case <-time.After(30 * time.Second):
			t.Fatalf("Timeout waiting for concurrent tasks")
		}
	}
	
	// Validate all timestamps
	assert.Equal(t, concurrentTasks, len(collectedTimestamps), 
		"Should have received all concurrent timestamps")
	
	for i, timestamp := range collectedTimestamps {
		validateISO8601Format(t, timestamp, fmt.Sprintf("concurrent task %d", i))
	}
	
	// Verify all timestamps are unique (no collision)
	timestampSet := make(map[string]bool)
	for _, timestamp := range collectedTimestamps {
		assert.False(t, timestampSet[timestamp], 
			"Duplicate timestamp found in concurrent execution: %s", timestamp)
		timestampSet[timestamp] = true
	}
	
	t.Logf("✅ Regression tests passed with %d unique concurrent timestamps", len(collectedTimestamps))
}

func testEndToEndTimestampCompliance(t *testing.T) {
	t.Log("Testing end-to-end timestamp compliance across all A2A endpoints...")
	
	// Test all standard A2A methods for timestamp compliance
	endpoints := []struct {
		name   string
		method string
		setup  func(t *testing.T) map[string]interface{}
	}{
		{
			name:   "message_send",
			method: "message/send",
			setup: func(t *testing.T) map[string]interface{} {
				return map[string]interface{}{
					"agentId": "echo-agent",
					"message": map[string]interface{}{
						"text": "End-to-end test",
					},
				}
			},
		},
		{
			name:   "tasks_get",
			method: "tasks/get",
			setup: func(t *testing.T) map[string]interface{} {
				// First create a task
				createResponse, err := makeContainerA2ARequest(
					"message/send",
					map[string]interface{}{
						"agentId": "echo-agent",
						"message": map[string]interface{}{
							"text": "Setup for tasks/get test",
						},
					},
					"setup-for-get",
				)
				require.NoError(t, err)
				
				taskResult := createResponse.Result.(map[string]interface{})
				taskID := taskResult["id"].(string)
				
				return map[string]interface{}{
					"id": taskID,
				}
			},
		},
	}
	
	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			params := endpoint.setup(t)
			
			response, err := makeContainerA2ARequest(
				endpoint.method,
				params,
				fmt.Sprintf("e2e-%s", endpoint.name),
			)
			require.NoError(t, err)
			require.NotNil(t, response.Result)
			
			// Extract and validate all timestamps in response
			validateTimestampsInResponse(t, response.Result, endpoint.method)
		})
	}
	
	t.Log("✅ End-to-end timestamp compliance verified for all A2A endpoints")
}

// Helper Functions

func makeContainerA2ARequest(method string, params interface{}, requestID string) (*JSONRPCResponse, error) {
	request := JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  method,
		Params:  params,
		ID:      requestID,
	}
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}
	
	resp, err := http.Post(
		ContainerGatewayURL+"/a2a",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}
	
	var response JSONRPCResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	
	return &response, nil
}

func validateISO8601Format(t *testing.T, timestamp, fieldName string) {
	// Validate format pattern
	iso8601Regex := regexp.MustCompile(ISO8601Pattern)
	assert.True(t, iso8601Regex.MatchString(timestamp),
		"Field '%s' timestamp '%s' does not match ISO 8601 pattern %s",
		fieldName, timestamp, ISO8601Pattern)
	
	// Validate it's parseable as RFC3339
	_, err := time.Parse(time.RFC3339Nano, timestamp)
	assert.NoError(t, err,
		"Field '%s' timestamp '%s' should be valid RFC3339 format",
		fieldName, timestamp)
	
	// Validate specific format requirements
	assert.True(t, len(timestamp) == 24,
		"Field '%s' timestamp '%s' should be exactly 24 characters",
		fieldName, timestamp)
	
	assert.True(t, timestamp[len(timestamp)-1] == 'Z',
		"Field '%s' timestamp '%s' should end with 'Z' for UTC",
		fieldName, timestamp)
}

func validateTimestampsInResponse(t *testing.T, result interface{}, context string) {
	// Recursively find and validate all timestamp fields
	switch v := result.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if key == "timestamp" || key == "createdAt" || key == "updatedAt" {
				if strValue, ok := value.(string); ok {
					validateISO8601Format(t, strValue, fmt.Sprintf("%s.%s", context, key))
				}
			} else {
				validateTimestampsInResponse(t, value, fmt.Sprintf("%s.%s", context, key))
			}
		}
	case []interface{}:
		for i, item := range v {
			validateTimestampsInResponse(t, item, fmt.Sprintf("%s[%d]", context, i))
		}
	}
}

// Benchmark tests for timestamp performance
func BenchmarkISO8601TimestampGeneration(b *testing.B) {
	// Test timestamp generation performance (should be fast)
	for i := 0; i < b.N; i++ {
		// Simulate the newISO8601Timestamp() function
		timestamp := time.Now().UTC().Format(time.RFC3339Nano)[:23] + "Z"
		_ = timestamp
	}
}

func BenchmarkISO8601TimestampValidation(b *testing.B) {
	// Test timestamp validation performance
	iso8601Regex := regexp.MustCompile(ISO8601Pattern)
	testTimestamp := "2024-07-03T14:30:00.123Z"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = iso8601Regex.MatchString(testTimestamp)
	}
}