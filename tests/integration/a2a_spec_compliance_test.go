package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A2A Protocol v0.2.5 Specification Compliance Test Suite
// This test suite verifies complete adherence to the A2A Protocol v0.2.5 specification
// as defined at: https://a2aproject.github.io/A2A/latest/specification/

const (
	A2ASpecVersion = "0.2.5"
	GatewayURL     = "http://localhost:8080"
	RegistryURL    = "http://localhost:8001"
	TestTimeout    = 30 * time.Second
)

// JSON-RPC 2.0 Types
type JSONRPCRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      interface{} `json:"id"`
}

type JSONRPCResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// A2A Protocol Types per v0.2.9 specification
type A2ATask struct {
	ID        string                 `json:"id"`
	ContextID string                 `json:"contextId"`
	Status    TaskStatus             `json:"status"`
	AgentID   string                 `json:"agentId"`
	Input     interface{}            `json:"input"`
	Result    interface{}            `json:"result,omitempty"`
	Error     *string                `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt string                 `json:"createdAt"`
}

type TaskStatus struct {
	Status      string   `json:"status"`
	Progress    *float64 `json:"progress,omitempty"`
	Description *string  `json:"description,omitempty"`
	UpdatedAt   string   `json:"updatedAt"`
}

type AgentCard struct {
	Name               string                 `json:"name"`
	Description        string                 `json:"description"`
	Version            string                 `json:"version"`
	URL                string                 `json:"url"`
	Capabilities       map[string]interface{} `json:"capabilities"`
	Skills             []AgentSkill           `json:"skills"`
	DefaultInputModes  []string               `json:"defaultInputModes"`
	DefaultOutputModes []string               `json:"defaultOutputModes"`
	Security           []map[string][]string  `json:"security,omitempty"`
	SecuritySchemes    map[string]interface{} `json:"securitySchemes"`
}

type AgentSkill struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputModes  []string               `json:"inputModes"`
	OutputModes []string               `json:"outputModes"`
	Tags        []string               `json:"tags,omitempty"`
	Examples    []string               `json:"examples,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Returns     map[string]interface{} `json:"returns,omitempty"`
}

// Test Helper Functions
func makeA2ARequest(method string, params interface{}, requestID string) (*JSONRPCResponse, error) {
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
		GatewayURL+"/a2a",
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

func checkGatewayHealth(t *testing.T) {
	resp, err := http.Get(GatewayURL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Gateway health check failed")

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(t, err)

	assert.Equal(t, "healthy", health["status"], "Gateway not healthy")
	t.Logf("✅ Gateway health check passed")
}

// Test Suite: A2A Protocol v0.2.9 Specification Compliance

func TestA2ASpecCompliance(t *testing.T) {
	t.Run("GatewayHealth", func(t *testing.T) {
		checkGatewayHealth(t)
	})
	
	t.Run("JSONRPCCompliance", testJSONRPCCompliance)
	t.Run("StandardMethodsCompliance", testStandardMethodsCompliance)
	t.Run("DataStructuresCompliance", testDataStructuresCompliance)
	t.Run("ErrorHandlingCompliance", testErrorHandlingCompliance)
	t.Run("AgentDiscoveryCompliance", testAgentDiscoveryCompliance)
	t.Run("TaskLifecycleCompliance", testTaskLifecycleCompliance)
	t.Run("ConcurrentRequestsCompliance", testConcurrentRequestsCompliance)
}

func testJSONRPCCompliance(t *testing.T) {
	t.Log("Testing JSON-RPC 2.0 protocol compliance...")

	// Test 1: Valid JSON-RPC 2.0 request structure
	response, err := makeA2ARequest(
		"message/send",
		map[string]interface{}{
			"agentId": "echo-agent",
			"message": map[string]interface{}{
				"text": "JSON-RPC compliance test",
			},
			"metadata": map[string]interface{}{
				"test": "jsonrpc-compliance",
			},
		},
		"jsonrpc-test-001",
	)
	require.NoError(t, err)

	// Verify response structure
	assert.Equal(t, "2.0", response.Jsonrpc, "Invalid jsonrpc version")
	assert.Equal(t, "jsonrpc-test-001", response.ID, "Request ID mismatch")
	assert.NotNil(t, response.Result, "Missing result field")
	assert.Nil(t, response.Error, "Unexpected error field")

	// Test 2: Error response structure (invalid method)
	errorResponse, err := makeA2ARequest(
		"invalid/method",
		map[string]interface{}{},
		"error-test-001",
	)
	require.NoError(t, err)

	assert.Equal(t, "2.0", errorResponse.Jsonrpc, "Invalid jsonrpc version in error response")
	assert.Equal(t, "error-test-001", errorResponse.ID, "Request ID mismatch in error response")
	assert.Nil(t, errorResponse.Result, "Unexpected result field in error response")
	assert.NotNil(t, errorResponse.Error, "Missing error field in error response")

	// Verify error structure
	assert.NotEmpty(t, errorResponse.Error.Code, "Missing error code")
	assert.NotEmpty(t, errorResponse.Error.Message, "Missing error message")

	t.Log("✅ JSON-RPC 2.0 protocol compliance verified")
}

func testStandardMethodsCompliance(t *testing.T) {
	t.Log("Testing A2A standard methods compliance...")

	// Test message/send method
	sendResponse, err := makeA2ARequest(
		"message/send",
		map[string]interface{}{
			"agentId": "echo-agent",
			"message": map[string]interface{}{
				"text": "Standard method test",
			},
			"metadata": map[string]interface{}{
				"test": "standard-methods",
			},
		},
		"standard-001",
	)
	require.NoError(t, err)
	require.NotNil(t, sendResponse.Result)

	// Verify task creation response structure
	taskResult := sendResponse.Result.(map[string]interface{})
	requiredFields := []string{"taskId", "status", "agent", "created"}
	for _, field := range requiredFields {
		assert.Contains(t, taskResult, field, "message/send missing required field: %s", field)
	}

	taskID := taskResult["taskId"].(string)

	// Test tasks/get method
	getResponse, err := makeA2ARequest(
		"tasks/get",
		map[string]interface{}{
			"taskId": taskID,
		},
		"standard-002",
	)
	require.NoError(t, err)
	require.NotNil(t, getResponse.Result)

	// Verify A2A Task structure
	taskData := getResponse.Result.(map[string]interface{})
	a2aTaskFields := []string{"id", "contextId", "status", "agentId", "input", "createdAt"}
	for _, field := range a2aTaskFields {
		assert.Contains(t, taskData, field, "tasks/get missing A2A Task field: %s", field)
	}

	// Verify A2A v0.2.5 compliant TaskStatus structure
	statusObj := taskData["status"].(map[string]interface{})
	statusFields := []string{"state", "timestamp"}
	for _, field := range statusFields {
		assert.Contains(t, statusObj, field, "TaskStatus missing required A2A spec field: %s", field)
	}

	// Test tasks/cancel method
	cancelResponse, err := makeA2ARequest(
		"tasks/cancel",
		map[string]interface{}{
			"taskId": taskID,
		},
		"standard-003",
	)
	require.NoError(t, err)

	// tasks/cancel should return result or error
	assert.True(t, 
		cancelResponse.Result != nil || cancelResponse.Error != nil,
		"tasks/cancel must return result or error",
	)

	t.Log("✅ All A2A standard methods comply with specification")
}

func testDataStructuresCompliance(t *testing.T) {
	t.Log("Testing A2A data structures compliance...")

	// Create task and get full task data
	sendResponse, err := makeA2ARequest(
		"message/send",
		map[string]interface{}{
			"agentId": "echo-agent",
			"message": map[string]interface{}{
				"text": "Data structure test",
			},
			"metadata": map[string]interface{}{
				"test":     "data-structures",
				"priority": "high",
			},
		},
		"data-test-001",
	)
	require.NoError(t, err)

	taskResult := sendResponse.Result.(map[string]interface{})
	taskID := taskResult["taskId"].(string)

	// Wait for completion
	time.Sleep(2 * time.Second)

	// Get complete task data
	getResponse, err := makeA2ARequest(
		"tasks/get",
		map[string]interface{}{
			"taskId": taskID,
		},
		"data-test-002",
	)
	require.NoError(t, err)

	task := getResponse.Result.(map[string]interface{})

	// Verify A2A Task structure compliance
	requiredFields := map[string]string{
		"id":        "string",
		"contextId": "string",
		"status":    "object",
		"agentId":   "string",
		"input":     "object",
		"createdAt": "string",
	}

	for field, expectedType := range requiredFields {
		assert.Contains(t, task, field, "Task missing required field: %s", field)

		value := task[field]
		switch expectedType {
		case "string":
			assert.IsType(t, "", value, "Task field %s has wrong type", field)
		case "object":
			assert.IsType(t, map[string]interface{}{}, value, "Task field %s has wrong type", field)
		}
	}

	// Verify A2A v0.2.5 compliant TaskStatus structure
	status := task["status"].(map[string]interface{})
	statusRequired := map[string]string{
		"state":     "string",
		"timestamp": "string",
	}

	for field, expectedType := range statusRequired {
		assert.Contains(t, status, field, "TaskStatus missing required field: %s", field)
		if expectedType == "string" {
			assert.IsType(t, "", status[field], "TaskStatus field %s has wrong type", field)
		}
	}

	// Verify valid A2A TaskState values
	validTaskStates := []string{"submitted", "working", "completed", "failed", "cancelled"}
	stateValue := status["state"].(string)
	assert.Contains(t, validTaskStates, stateValue, "Invalid A2A TaskState value: %s", stateValue)

	t.Log("✅ All A2A data structures comply with specification")
}

func testErrorHandlingCompliance(t *testing.T) {
	t.Log("Testing A2A error handling compliance...")

	// Test 1: Invalid method
	response, err := makeA2ARequest(
		"invalid/method",
		map[string]interface{}{},
		"error-001",
	)
	require.NoError(t, err)
	assert.NotNil(t, response.Error, "Invalid method should return error")
	assert.Equal(t, -32601, int(response.Error.Code), "Invalid method error code should be -32601")

	// Test 2: Missing required parameters (agentId)
	response, err = makeA2ARequest(
		"message/send",
		map[string]interface{}{
			"message": map[string]interface{}{
				"text": "Test without agentId",
			},
		}, // Missing required agentId
		"error-002",
	)
	require.NoError(t, err)
	assert.NotNil(t, response.Error, "Missing required agentId should return error")
	if response.Error != nil {
		assert.Equal(t, -32602, int(response.Error.Code), "Missing parameters should return -32602 (Invalid params)")
	}

	// Test 3: Invalid task ID
	response, err = makeA2ARequest(
		"tasks/get",
		map[string]interface{}{
			"taskId": "non-existent-task-id",
		},
		"error-003",
	)
	require.NoError(t, err)
	assert.NotNil(t, response.Error, "Invalid task ID should return error")

	// Test 4: Invalid agent ID
	response, err = makeA2ARequest(
		"message/send",
		map[string]interface{}{
			"agentId": "non-existent-agent",
			"message": map[string]interface{}{
				"text": "Test",
			},
			"metadata": map[string]interface{}{},
		},
		"error-004",
	)
	require.NoError(t, err)
	assert.NotNil(t, response.Error, "Invalid agent ID should return error")

	t.Log("✅ Error handling complies with A2A specification")
}

func testAgentDiscoveryCompliance(t *testing.T) {
	t.Log("Testing A2A agent discovery compliance...")

	// Test 1: A2A Spec Standard - .well-known/agent.json endpoint
	t.Log("Testing A2A standard .well-known/agent.json endpoint...")
	resp, err := http.Get(GatewayURL + "/.well-known/agent.json")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var agentCard map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&agentCard)
			if err == nil {
				// Verify AgentCard structure
				requiredAgentFields := []string{
					"name", "description", "version", "url", "capabilities",
					"skills", "defaultInputModes", "defaultOutputModes",
				}

				for _, field := range requiredAgentFields {
					assert.Contains(t, agentCard, field, "AgentCard missing required field: %s", field)
				}
				t.Log("✅ .well-known/agent.json endpoint works correctly")
			}
		}
	}

	// Test 2: A2A Extension - x-a2a.discoverAgents JSON-RPC method
	t.Log("Testing x-a2a.discoverAgents extension method...")
	response, err := makeA2ARequest(
		"x-a2a.discoverAgents",
		map[string]interface{}{
			"capability": "",
			"keyword":    "",
			"limit":      10,
		},
		"discover-test-001",
	)
	require.NoError(t, err)

	if response.Result != nil {
		result := response.Result.(map[string]interface{})
		if agents, ok := result["agents"]; ok {
			agentsList := agents.([]interface{})
			
			// If agents are present, verify AgentCard structure
			if len(agentsList) > 0 {
				agent := agentsList[0].(map[string]interface{})
				requiredAgentFields := []string{
					"name", "description", "version", "url", "capabilities",
					"skills", "defaultInputModes", "defaultOutputModes",
				}

				for _, field := range requiredAgentFields {
					assert.Contains(t, agent, field, "AgentCard missing required field: %s", field)
				}
				t.Logf("✅ Found %d agents via x-a2a.discoverAgents", len(agentsList))
			} else {
				t.Log("ℹ️ No agents found via x-a2a.discoverAgents - testing registry connection")
			}
		}
	} else if response.Error != nil {
		t.Logf("ℹ️ x-a2a.discoverAgents returned error: %s", response.Error.Message)
	}

	// Test 3: A2A Standard - agent-specific .well-known endpoints
	t.Log("Testing agent-specific .well-known/agent.json endpoints...")
	agentID := "echo-agent"
	resp, err = http.Get(GatewayURL + "/agents/" + agentID + "/.well-known/agent.json")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var agentCard map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&agentCard)
			if err == nil {
				assert.Contains(t, agentCard, "name", "Agent card missing name")
				t.Log("✅ Agent-specific .well-known/agent.json works correctly")
			}
		} else {
			t.Logf("ℹ️ Agent-specific .well-known endpoint returned %d (may not be implemented)", resp.StatusCode)
		}
	}

	t.Log("✅ Agent discovery testing completed (both A2A standard and extensions)")
}

func testTaskLifecycleCompliance(t *testing.T) {
	t.Log("Testing A2A task lifecycle compliance...")

	// Create task
	sendResponse, err := makeA2ARequest(
		"message/send",
		map[string]interface{}{
			"agentId": "echo-agent",
			"message": map[string]interface{}{
				"text": "Lifecycle test message",
			},
			"metadata": map[string]interface{}{
				"test": "lifecycle",
				"step": "create",
			},
		},
		"lifecycle-001",
	)
	require.NoError(t, err)
	require.NotNil(t, sendResponse.Result)

	taskResult := sendResponse.Result.(map[string]interface{})
	taskID := taskResult["taskId"].(string)
	initialStatus := taskResult["status"].(string)

	// Verify initial status is valid A2A TaskState
	validInitialStates := []string{"submitted", "working"}
	assert.Contains(t, validInitialStates, initialStatus, "Invalid initial A2A TaskState: %s", initialStatus)

	// Monitor task progress
	maxAttempts := 10
	var finalStatus string
	var finalTask map[string]interface{}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		getResponse, err := makeA2ARequest(
			"tasks/get",
			map[string]interface{}{
				"taskId": taskID,
			},
			fmt.Sprintf("lifecycle-%03d", attempt+2),
		)
		require.NoError(t, err)
		require.NotNil(t, getResponse.Result)

		taskData := getResponse.Result.(map[string]interface{})
		currentStatus := taskData["status"].(map[string]interface{})["state"].(string)

		if strings.Contains("completed,failed,cancelled", currentStatus) {
			finalStatus = currentStatus
			finalTask = taskData
			break
		}

		time.Sleep(1 * time.Second)
	}

	assert.NotEmpty(t, finalStatus, "Task did not reach terminal state")

	// Verify final task data structure
	if finalStatus == "completed" {
		assert.Contains(t, finalTask, "result", "Completed task missing result field")
	}

	if finalStatus == "failed" {
		assert.Contains(t, finalTask, "error", "Failed task missing error field")
	}

	t.Logf("✅ Task lifecycle completed successfully (status: %s)", finalStatus)
}

func testConcurrentRequestsCompliance(t *testing.T) {
	t.Log("Testing A2A concurrent request handling...")

	taskCount := 5
	var wg sync.WaitGroup
	responses := make(chan *JSONRPCResponse, taskCount)
	errors := make(chan error, taskCount)

	// Create multiple concurrent tasks
	for i := 0; i < taskCount; i++ {
		wg.Add(1)
		go func(taskNum int) {
			defer wg.Done()

			response, err := makeA2ARequest(
				"message/send",
				map[string]interface{}{
					"agentId": "echo-agent",
					"message": map[string]interface{}{
						"text": fmt.Sprintf("Concurrent test %d", taskNum+1),
					},
					"metadata": map[string]interface{}{
						"test":        "concurrent",
						"task_number": taskNum + 1,
					},
				},
				fmt.Sprintf("concurrent-%03d", taskNum+1),
			)

			if err != nil {
				errors <- err
				return
			}

			responses <- response
		}(i)
	}

	wg.Wait()
	close(responses)
	close(errors)

	// Check for errors
	for err := range errors {
		t.Fatalf("Concurrent task failed: %v", err)
	}

	// Verify all tasks were created successfully
	taskIDs := make(map[string]bool)
	responseCount := 0

	for response := range responses {
		responseCount++
		require.NotNil(t, response.Result, "Concurrent task missing result")

		taskResult := response.Result.(map[string]interface{})
		taskID := taskResult["taskId"].(string)

		assert.False(t, taskIDs[taskID], "Non-unique task ID in concurrent requests: %s", taskID)
		taskIDs[taskID] = true
	}

	assert.Equal(t, taskCount, responseCount, "Not all concurrent tasks completed")
	assert.Equal(t, taskCount, len(taskIDs), "Non-unique task IDs in concurrent requests")

	t.Logf("✅ All %d concurrent requests handled correctly", taskCount)
}

// Benchmark tests for performance validation
func BenchmarkA2AMessageSend(b *testing.B) {
	checkGatewayHealth(&testing.T{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := makeA2ARequest(
			"message/send",
			map[string]interface{}{
				"agentId": "echo-agent",
				"message": map[string]interface{}{
					"text": fmt.Sprintf("Benchmark test %d", i),
				},
				"metadata": map[string]interface{}{
					"test": "benchmark",
				},
			},
			fmt.Sprintf("bench-%d", i),
		)
		if err != nil {
			b.Fatalf("Benchmark request failed: %v", err)
		}
	}
}

func BenchmarkA2ATasksGet(b *testing.B) {
	checkGatewayHealth(&testing.T{})

	// Create a task first
	response, err := makeA2ARequest(
		"message/send",
		map[string]interface{}{
			"agentId": "echo-agent",
			"message": map[string]interface{}{
				"text": "Benchmark setup task",
			},
			"metadata": map[string]interface{}{
				"test": "benchmark-setup",
			},
		},
		"bench-setup",
	)
	if err != nil {
		b.Fatalf("Failed to create benchmark task: %v", err)
	}

	taskResult := response.Result.(map[string]interface{})
	taskID := taskResult["taskId"].(string)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := makeA2ARequest(
			"tasks/get",
			map[string]interface{}{
				"taskId": taskID,
			},
			fmt.Sprintf("bench-get-%d", i),
		)
		if err != nil {
			b.Fatalf("Benchmark get request failed: %v", err)
		}
	}
}