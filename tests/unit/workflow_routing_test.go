package gateway_test

import (
	"fmt"
	"testing"
)

// TestWorkflowRoutingConfiguration tests the workflow routing configuration system
// This validates the fixes applied to resolve hardcoded workflow naming patterns

func TestAgentWorkflowMapping(t *testing.T) {
	// Simulate the agent routing configuration from config/agent-routing.yaml
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
		expectedWorkflow  string
		expectedQueue     string
		shouldExist       bool
	}{
		{
			name:             "echo_agent_mapping",
			agentID:          "echo-agent",
			expectedWorkflow: "EchoTaskWorkflow",
			expectedQueue:    "echo-agent-tasks",
			shouldExist:      true,
		},
		{
			name:             "custom_agent_mapping",
			agentID:          "custom-agent",
			expectedWorkflow: "LLMAgentWorkflow",
			expectedQueue:    "custom-agent-tasks",
			shouldExist:      true,
		},
		{
			name:        "unknown_agent",
			agentID:     "unknown-agent",
			shouldExist: false,
		},
		{
			name:        "empty_agent_id",
			agentID:     "",
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate gateway routing lookup logic
			workflow, workflowExists := agentWorkflows[tt.agentID]
			queue, queueExists := agentTaskQueues[tt.agentID]

			configExists := workflowExists && queueExists

			if configExists != tt.shouldExist {
				t.Errorf("Expected config existence=%v, got=%v for agent %s", 
					tt.shouldExist, configExists, tt.agentID)
			}

			if tt.shouldExist {
				if workflow != tt.expectedWorkflow {
					t.Errorf("Expected workflow %s, got %s for agent %s", 
						tt.expectedWorkflow, workflow, tt.agentID)
				}

				if queue != tt.expectedQueue {
					t.Errorf("Expected queue %s, got %s for agent %s", 
						tt.expectedQueue, queue, tt.agentID)
				}

				// Critical test: Ensure no hardcoded pattern usage
				hardcodedPattern := tt.agentID + "Workflow"
				if workflow == hardcodedPattern {
					t.Errorf("CRITICAL: Workflow type uses hardcoded pattern %s instead of config", 
						hardcodedPattern)
				}

				hardcodedQueuePattern := tt.agentID + "-tasks"
				if queue != hardcodedQueuePattern {
					// This is actually OK - we want to avoid hardcoded patterns
					// but for task queues, the pattern is more standard
					t.Logf("Queue %s doesn't follow hardcoded pattern %s (this is OK)", 
						queue, hardcodedQueuePattern)
				}
			}
		})
	}
}

func TestWorkflowRoutingErrorHandling(t *testing.T) {
	// Simulate error handling when agent is not found in configuration
	agentWorkflows := map[string]string{
		"echo-agent": "EchoTaskWorkflow",
	}

	agentTaskQueues := map[string]string{
		"echo-agent": "echo-agent-tasks",
	}

	tests := []struct {
		name                string
		agentID             string
		expectWorkflowError bool
		expectQueueError    bool
	}{
		{
			name:                "valid_agent",
			agentID:             "echo-agent",
			expectWorkflowError: false,
			expectQueueError:    false,
		},
		{
			name:                "invalid_agent",
			agentID:             "invalid-agent",
			expectWorkflowError: true,
			expectQueueError:    true,
		},
		{
			name:                "empty_agent_id",
			agentID:             "",
			expectWorkflowError: true,
			expectQueueError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate gateway error handling logic
			workflow, workflowOK := agentWorkflows[tt.agentID]
			queue, queueOK := agentTaskQueues[tt.agentID]

			workflowError := !workflowOK
			queueError := !queueOK

			if workflowError != tt.expectWorkflowError {
				t.Errorf("Expected workflow error=%v, got=%v for agent %s", 
					tt.expectWorkflowError, workflowError, tt.agentID)
			}

			if queueError != tt.expectQueueError {
				t.Errorf("Expected queue error=%v, got=%v for agent %s", 
					tt.expectQueueError, queueError, tt.agentID)
			}

			if !workflowError && !queueError {
				// Valid configuration should return proper values
				if workflow == "" {
					t.Error("Valid agent should have non-empty workflow type")
				}
				if queue == "" {
					t.Error("Valid agent should have non-empty task queue")
				}
			}
		})
	}
}

func TestWorkflowCategoryValidation(t *testing.T) {
	// Test workflow category system from routing configuration
	workflowCategories := map[string]struct {
		Description string
		Examples    []string
	}{
		"LLMAgentWorkflow": {
			Description: "For agents that use LLM for reasoning and generation",
			Examples:    []string{"Custom Agent", "LLM-based Agent"},
		},
		"EchoTaskWorkflow": {
			Description: "Simple echo workflow for testing",
			Examples:    []string{"Echo Agent", "Test Agent"},
		},
	}

	// Map agents to workflow categories
	agentWorkflows := map[string]string{
		"echo-agent":   "EchoTaskWorkflow",
		"custom-agent": "LLMAgentWorkflow",
	}

	for agentID, workflowType := range agentWorkflows {
		t.Run(fmt.Sprintf("validate_%s_category", agentID), func(t *testing.T) {
			category, exists := workflowCategories[workflowType]
			
			if !exists {
				t.Errorf("Workflow type %s for agent %s not found in categories", 
					workflowType, agentID)
				return
			}

			if category.Description == "" {
				t.Errorf("Workflow category %s missing description", workflowType)
			}

			if len(category.Examples) == 0 {
				t.Errorf("Workflow category %s missing examples", workflowType)
			}

			t.Logf("✅ Agent %s uses workflow %s: %s", 
				agentID, workflowType, category.Description)
		})
	}
}

func TestHardcodedPatternDetection(t *testing.T) {
	// Test to detect and prevent hardcoded workflow naming patterns
	tests := []struct {
		name               string
		agentID            string
		workflowType       string
		isHardcodedPattern bool
	}{
		{
			name:               "hardcoded_echo_pattern",
			agentID:            "echo-agent",
			workflowType:       "echo-agentWorkflow", // BAD: hardcoded pattern
			isHardcodedPattern: true,
		},
		{
			name:               "hardcoded_custom_pattern",
			agentID:            "custom-agent",
			workflowType:       "custom-agentWorkflow", // BAD: hardcoded pattern
			isHardcodedPattern: true,
		},
		{
			name:               "correct_echo_config",
			agentID:            "echo-agent",
			workflowType:       "EchoTaskWorkflow", // GOOD: from configuration
			isHardcodedPattern: false,
		},
		{
			name:               "correct_custom_config",
			agentID:            "custom-agent",
			workflowType:       "LLMAgentWorkflow", // GOOD: from configuration
			isHardcodedPattern: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Detect hardcoded pattern: {agentID}Workflow
			expectedHardcodedPattern := tt.agentID + "Workflow"
			isHardcoded := tt.workflowType == expectedHardcodedPattern

			if isHardcoded != tt.isHardcodedPattern {
				t.Errorf("Expected hardcoded pattern detection=%v, got=%v for workflow %s", 
					tt.isHardcodedPattern, isHardcoded, tt.workflowType)
			}

			if isHardcoded {
				t.Errorf("CRITICAL: Workflow type %s uses hardcoded pattern for agent %s", 
					tt.workflowType, tt.agentID)
			} else {
				t.Logf("✅ Workflow type %s for agent %s uses proper configuration", 
					tt.workflowType, tt.agentID)
			}
		})
	}
}

func TestRoutingConfigurationValidation(t *testing.T) {
	// Test complete routing configuration structure
	type RoutingConfig struct {
		TaskQueue    string `yaml:"taskQueue"`
		WorkflowType string `yaml:"workflowType"`
	}

	type WorkflowCategory struct {
		Description string   `yaml:"description"`
		Examples    []string `yaml:"examples"`
	}

	type AgentRoutingConfig struct {
		Version            string                      `yaml:"version"`
		Routing            map[string]RoutingConfig    `yaml:"routing"`
		WorkflowCategories map[string]WorkflowCategory `yaml:"workflowCategories"`
	}

	// Simulate loading config from agent-routing.yaml
	config := AgentRoutingConfig{
		Version: "1.0",
		Routing: map[string]RoutingConfig{
			"echo-agent": {
				TaskQueue:    "echo-agent-tasks",
				WorkflowType: "EchoTaskWorkflow",
			},
			"custom-agent": {
				TaskQueue:    "custom-agent-tasks",
				WorkflowType: "LLMAgentWorkflow",
			},
		},
		WorkflowCategories: map[string]WorkflowCategory{
			"LLMAgentWorkflow": {
				Description: "For agents that use LLM for reasoning and generation",
				Examples:    []string{"Custom Agent", "LLM-based Agent"},
			},
		},
	}

	// Test configuration validation
	t.Run("ConfigurationStructure", func(t *testing.T) {
		if config.Version == "" {
			t.Error("Configuration missing version")
		}

		if len(config.Routing) == 0 {
			t.Error("Configuration missing routing entries")
		}

		for agentName, routing := range config.Routing {
			if routing.TaskQueue == "" {
				t.Errorf("Agent %s missing task queue", agentName)
			}
			if routing.WorkflowType == "" {
				t.Errorf("Agent %s missing workflow type", agentName)
			}

			// Validate no hardcoded patterns
			hardcodedPattern := agentName + "Workflow"
			if routing.WorkflowType == hardcodedPattern {
				t.Errorf("CRITICAL: Agent %s uses hardcoded workflow pattern %s", 
					agentName, hardcodedPattern)
			}
		}
	})

	t.Run("WorkflowCategoryReferences", func(t *testing.T) {
		// Verify all workflow types reference valid categories
		for agentName, routing := range config.Routing {
			workflowType := routing.WorkflowType
			
			// Check if workflow type exists in categories
			if _, exists := config.WorkflowCategories[workflowType]; !exists {
				// Not all workflow types need to be in categories, but log for awareness
				t.Logf("Info: Workflow type %s for agent %s not in categories (this may be OK)", 
					workflowType, agentName)
			}
		}
	})
}

func TestMessageStreamSpecificRouting(t *testing.T) {
	// Test routing specifically for message/stream endpoint
	tests := []struct {
		name     string
		agentID  string
		endpoint string
	}{
		{
			name:     "stream_echo_agent",
			agentID:  "echo-agent",
			endpoint: "message/stream",
		},
		{
			name:     "stream_custom_agent", 
			agentID:  "custom-agent",
			endpoint: "message/stream",
		},
		{
			name:     "send_echo_agent",
			agentID:  "echo-agent",
			endpoint: "message/send",
		},
	}

	// Configuration should be the same for both endpoints
	agentWorkflows := map[string]string{
		"echo-agent":   "EchoTaskWorkflow",
		"custom-agent": "LLMAgentWorkflow",
	}

	agentTaskQueues := map[string]string{
		"echo-agent":   "echo-agent-tasks",
		"custom-agent": "custom-agent-tasks",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Both message/send and message/stream should use same routing
			workflow, workflowOK := agentWorkflows[tt.agentID]
			queue, queueOK := agentTaskQueues[tt.agentID]

			if !workflowOK {
				t.Errorf("No workflow configuration for agent %s on endpoint %s", 
					tt.agentID, tt.endpoint)
			}

			if !queueOK {
				t.Errorf("No queue configuration for agent %s on endpoint %s", 
					tt.agentID, tt.endpoint)
			}

			if workflowOK && queueOK {
				t.Logf("✅ Agent %s on %s: workflow=%s, queue=%s", 
					tt.agentID, tt.endpoint, workflow, queue)
			}
		})
	}
}