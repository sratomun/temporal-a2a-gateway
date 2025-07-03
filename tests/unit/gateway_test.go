package gateway_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// Import the gateway package types
// In a real test, this would import from the gateway module
type AgentCapabilities struct {
	Streaming              *bool `json:"streaming,omitempty"`
	PushNotifications      *bool `json:"pushNotifications,omitempty"`
	StateTransitionHistory *bool `json:"stateTransitionHistory,omitempty"`
}

type AgentCard struct {
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Version      string             `json:"version"`
	URL          string             `json:"url,omitempty"`
	Capabilities *AgentCapabilities `json:"capabilities,omitempty"`
}

type JSONRPCRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      interface{} `json:"id"`
}

// Helper function to create bool pointers
func boolPtr(b bool) *bool {
	return &b
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// Test AgentCard based on A2A protocol specification
func createMockAgentCard() AgentCard {
	return AgentCard{
		Name:        "Test Agent",
		Description: "A test agent for JSON-RPC parsing",
		Version:     "1.0.0",
		URL:         "http://test-agent:8080",
		Capabilities: &AgentCapabilities{
			Streaming:              boolPtr(false),
			PushNotifications:      boolPtr(true),
			StateTransitionHistory: boolPtr(true),
		},
	}
}

// Test basic agent card parsing
func TestBasicAgentCardParsing(t *testing.T) {
	agentCard := AgentCard{
		Name:        "Basic Test Agent",
		Description: "Basic test",
		Version:     "1.0.0",
	}

	// Test JSON marshaling/unmarshaling
	data, err := json.Marshal(agentCard)
	if err != nil {
		t.Fatalf("Failed to marshal basic agent card: %v", err)
	}

	var parsed AgentCard
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal basic agent card: %v", err)
	}

	if parsed.Name != agentCard.Name {
		t.Errorf("Expected name %s, got %s", agentCard.Name, parsed.Name)
	}
}

// Test agent card with capabilities parsing
func TestCapabilitiesAgentCardParsing(t *testing.T) {
	agentCard := createMockAgentCard()

	// Test JSON marshaling/unmarshaling
	data, err := json.Marshal(agentCard)
	if err != nil {
		t.Fatalf("Failed to marshal capabilities agent card: %v", err)
	}

	var parsed AgentCard
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal capabilities agent card: %v", err)
	}

	if parsed.Name != agentCard.Name {
		t.Errorf("Expected name %s, got %s", agentCard.Name, parsed.Name)
	}

	if parsed.Capabilities == nil {
		t.Error("Expected capabilities to be parsed, got nil")
	} else {
		if parsed.Capabilities.Streaming == nil || *parsed.Capabilities.Streaming != false {
			t.Error("Expected streaming to be false")
		}
		if parsed.Capabilities.PushNotifications == nil || *parsed.Capabilities.PushNotifications != true {
			t.Error("Expected pushNotifications to be true")
		}
	}
}

// Test JSON-RPC parameter parsing according to A2A specification
func TestJSONRPCParameterParsing(t *testing.T) {
	agentCard := createMockAgentCard()

	// Create JSON-RPC request payload
	rpcRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "x-a2a.registerAgent",
		"params": map[string]interface{}{
			"agentCard": agentCard,
		},
		"id": "test-001",
	}

	// Marshal to JSON
	requestBytes, err := json.Marshal(rpcRequest)
	if err != nil {
		t.Fatalf("Failed to marshal RPC request: %v", err)
	}

	// Parse as JSONRPCRequest
	var req JSONRPCRequest
	err = json.Unmarshal(requestBytes, &req)
	if err != nil {
		t.Fatalf("Failed to unmarshal RPC request: %v", err)
	}

	// Test parameter parsing approach used by gateway
	paramBytes, err := json.Marshal(req.Params)
	if err != nil {
		t.Fatalf("Failed to marshal params: %v", err)
	}

	var params struct {
		AgentCard AgentCard `json:"agentCard"`
	}

	err = json.Unmarshal(paramBytes, &params)
	if err != nil {
		t.Fatalf("Failed to unmarshal params to struct: %v", err)
	}

	// Validate parsed agent card
	if params.AgentCard.Name != agentCard.Name {
		t.Errorf("Expected name %s, got %s", agentCard.Name, params.AgentCard.Name)
	}

	if params.AgentCard.Capabilities == nil {
		t.Error("Expected capabilities to be parsed")
	}
}

// Test HTTP request simulation
func TestHTTPJSONRPCRequest(t *testing.T) {
	agentCard := createMockAgentCard()

	// Create JSON-RPC request
	rpcRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "x-a2a.registerAgent",
		"params": map[string]interface{}{
			"agentCard": agentCard,
		},
		"id": "http-test-001",
	}

	requestBytes, err := json.Marshal(rpcRequest)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Create HTTP request
	req := httptest.NewRequest("POST", "/a2a", bytes.NewReader(requestBytes))
	req.Header.Set("Content-Type", "application/json")

	// Parse request body
	var jsonReq JSONRPCRequest
	err = json.NewDecoder(req.Body).Decode(&jsonReq)
	if err != nil {
		t.Fatalf("Failed to decode HTTP request: %v", err)
	}

	// Test parameter parsing
	paramBytes, err := json.Marshal(jsonReq.Params)
	if err != nil {
		t.Fatalf("Failed to marshal params: %v", err)
	}

	var params struct {
		AgentCard AgentCard `json:"agentCard"`
	}

	err = json.Unmarshal(paramBytes, &params)
	if err != nil {
		t.Fatalf("Failed to unmarshal params: %v", err)
	}

	// Validate
	if params.AgentCard.Name != agentCard.Name {
		t.Errorf("Expected name %s, got %s", agentCard.Name, params.AgentCard.Name)
	}
}

// Test different capability combinations
func TestVariousCapabilityConfigurations(t *testing.T) {
	testCases := []struct {
		name         string
		capabilities *AgentCapabilities
	}{
		{
			name: "all_true",
			capabilities: &AgentCapabilities{
				Streaming:              boolPtr(true),
				PushNotifications:      boolPtr(true),
				StateTransitionHistory: boolPtr(true),
			},
		},
		{
			name: "all_false",
			capabilities: &AgentCapabilities{
				Streaming:              boolPtr(false),
				PushNotifications:      boolPtr(false),
				StateTransitionHistory: boolPtr(false),
			},
		},
		{
			name: "mixed",
			capabilities: &AgentCapabilities{
				Streaming:              boolPtr(true),
				PushNotifications:      boolPtr(false),
				StateTransitionHistory: boolPtr(true),
			},
		},
		{
			name:         "nil_capabilities",
			capabilities: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			agentCard := AgentCard{
				Name:         "Test Agent",
				Description:  "Test",
				Version:      "1.0.0",
				Capabilities: tc.capabilities,
			}

			// Test marshaling
			data, err := json.Marshal(agentCard)
			if err != nil {
				t.Fatalf("Failed to marshal agent card: %v", err)
			}

			// Test unmarshaling
			var parsed AgentCard
			err = json.Unmarshal(data, &parsed)
			if err != nil {
				t.Fatalf("Failed to unmarshal agent card: %v", err)
			}

			// Validate
			if parsed.Name != agentCard.Name {
				t.Errorf("Expected name %s, got %s", agentCard.Name, parsed.Name)
			}

			// Check capabilities
			if tc.capabilities == nil && parsed.Capabilities != nil {
				t.Error("Expected nil capabilities, got non-nil")
			}
			if tc.capabilities != nil && parsed.Capabilities == nil {
				t.Error("Expected non-nil capabilities, got nil")
			}
		})
	}
}

// Test the exact JSON-RPC flow the gateway uses
func TestGatewayJSONRPCFlow(t *testing.T) {
	// Create test agent cards of increasing complexity
	testCases := []struct {
		name      string
		agentCard AgentCard
	}{
		{
			name: "basic",
			agentCard: AgentCard{
				Name:        "Basic Agent",
				Description: "Basic test agent",
				Version:     "1.0.0",
			},
		},
		{
			name: "with_url",
			agentCard: AgentCard{
				Name:        "URL Agent",
				Description: "Agent with URL",
				Version:     "1.0.0",
				URL:         "http://test:8080",
			},
		},
		{
			name: "with_capabilities",
			agentCard: AgentCard{
				Name:        "Capabilities Agent",
				Description: "Agent with capabilities",
				Version:     "1.0.0",
				Capabilities: &AgentCapabilities{
					Streaming:              boolPtr(false),
					PushNotifications:      boolPtr(true),
					StateTransitionHistory: boolPtr(true),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate exact gateway flow
			rpcRequest := JSONRPCRequest{
				Jsonrpc: "2.0",
				Method:  "x-a2a.registerAgent",
				Params: map[string]interface{}{
					"agentCard": tc.agentCard,
				},
				ID: "gateway-test",
			}

			// Step 1: Marshal req.Params (what gateway does)
			paramBytes, err := json.Marshal(rpcRequest.Params)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}

			// Step 2: Unmarshal to struct (what gateway does)
			var params struct {
				AgentCard AgentCard `json:"agentCard"`
			}

			err = json.Unmarshal(paramBytes, &params)
			if err != nil {
				t.Fatalf("Failed to unmarshal params to struct: %v", err)
			}

			// Step 3: Validate
			if params.AgentCard.Name != tc.agentCard.Name {
				t.Errorf("Expected name %s, got %s", tc.agentCard.Name, params.AgentCard.Name)
			}

			t.Logf("Successfully parsed %s agent", tc.name)
		})
	}
}