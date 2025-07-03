package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.temporal.io/sdk/client"
	"gopkg.in/yaml.v2"
)

// A2A Gateway Service in Go
// Implements JSON-RPC 2.0 A2A Protocol with Temporal workflow orchestration

// A2A Protocol v0.2.5 compliant ISO 8601 timestamp generator
// Returns current time in UTC with millisecond precision: 2024-07-03T14:30:00.000Z
func newISO8601Timestamp() string {
	return time.Now().UTC().Format(time.RFC3339Nano)[:23] + "Z"
}

// A2A Protocol deprecation middleware
// Adds deprecation warnings for legacy methods with 3-month transition period
func addDeprecationWarnings(w http.ResponseWriter, method string) {
	// Set deprecation HTTP headers
	w.Header().Set("Deprecation", "true")
	w.Header().Set("Sunset", "2024-10-03T00:00:00Z") // 3 months from July 2024
	w.Header().Set("Link", `</docs/api.md>; rel="sunset"`)
	
	// Log deprecation usage for monitoring/analytics
	log.Printf("⚠️  DEPRECATED METHOD USED: %s - Will be removed 2024-10-03. Use A2A v0.2.5 methods instead.", method)
}

// A2A Protocol v0.2.5 unified timestamp validation
// Validates ISO 8601 format: 2024-07-03T14:30:00.000Z (UTC + milliseconds)
func validateISO8601Timestamp(timestamp string) error {
	if timestamp == "" {
		return fmt.Errorf("timestamp cannot be empty")
	}
	
	// Parse using RFC3339Nano format (supports milliseconds)
	_, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return fmt.Errorf("invalid ISO 8601 timestamp format: %s (expected: 2024-07-03T14:30:00.000Z)", timestamp)
	}
	
	// Additional validation: ensure UTC timezone (ends with Z)
	if !strings.HasSuffix(timestamp, "Z") {
		return fmt.Errorf("timestamp must be in UTC timezone (end with Z): %s", timestamp)
	}
	
	return nil
}

type Gateway struct {
	temporalClient client.Client
	redisClient    *redis.Client
	agentRegistryURL string
	port           string
}

// Agent Registry response types
type RegisterAgentResponse struct {
	AgentID string `json:"agentId"`
}

// Note: Removed DiscoverAgentsResponse and AgentDiscoveryInfo types
// The gateway now passes through the registry's Google SDK-compatible AgentCard format directly

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

type CreateTaskParams struct {
	AgentID  string                 `json:"agentId"`
	Input    interface{}            `json:"input"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// A2A Protocol v0.2.5 compliant parameter types
type GetTaskParams struct {
	ID string `json:"id"`
}

type SendMessageParams struct {
	ID      string      `json:"id"`
	Message interface{} `json:"message"`
}

type CancelTaskParams struct {
	ID string `json:"id"`
}

type RegisterAgentParams struct {
	AgentCard AgentCard `json:"agentCard"`
}

type DiscoverAgentsParams struct {
	Capability string `json:"capability,omitempty"`
	Keyword    string `json:"keyword,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type GetTasksByMetadataParams struct {
	MetadataKey   string      `json:"metadataKey"`
	MetadataValue interface{} `json:"metadataValue"`
	Limit         int         `json:"limit,omitempty"`
}

// Standard A2A Protocol Parameter Types
type MessageSendParams struct {
	AgentID  string                 `json:"agentId"`
	Message  interface{}            `json:"message"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// A2A Protocol v0.2.5 compliant parameter types
type TasksGetParams struct {
	ID string `json:"id"`
}

type TasksCancelParams struct {
	ID string `json:"id"`
}

// Official A2A AgentCapabilities from a2a-samples
type AgentCapabilities struct {
	Streaming              *bool `json:"streaming,omitempty"`
	PushNotifications      *bool `json:"pushNotifications,omitempty"`
	StateTransitionHistory *bool `json:"stateTransitionHistory,omitempty"`
}

// Test AgentCard with just capabilities field
type AgentCard struct {
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Version      string             `json:"version"`
	URL          string             `json:"url,omitempty"`
	Capabilities *AgentCapabilities `json:"capabilities,omitempty"`
}

type AgentCapability struct {
	Name               string      `json:"name"`
	Description        string      `json:"description"`
	InputSchema        interface{} `json:"inputSchema"`
	OutputSchema       interface{} `json:"outputSchema"`
	StreamingSupported bool        `json:"streamingSupported"`
	AsyncSupported     bool        `json:"asyncSupported"`
}

// NEW: Google SDK Compatible Types
type AgentSkill struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters,omitempty"`
	Returns     interface{} `json:"returns,omitempty"`
}

type SecurityScheme struct {
	OAuth2  *OAuth2SecurityScheme  `json:"oauth2,omitempty"`
	APIKey  *APIKeySecurityScheme  `json:"apiKey,omitempty"`
	HTTP    *HTTPSecurityScheme    `json:"http,omitempty"`
	OpenID  *OpenIDSecurityScheme  `json:"openIdConnect,omitempty"`
}

type OAuth2SecurityScheme struct {
	Type             string            `json:"type"`
	Flows            OAuth2Flows       `json:"flows"`
	Scopes           map[string]string `json:"scopes,omitempty"`
}

type OAuth2Flows struct {
	AuthorizationCode *OAuth2Flow `json:"authorizationCode,omitempty"`
	ClientCredentials *OAuth2Flow `json:"clientCredentials,omitempty"`
}

type OAuth2Flow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes,omitempty"`
}

type APIKeySecurityScheme struct {
	Type string `json:"type"`
	Name string `json:"name"`
	In   string `json:"in"` // header, query, cookie
}

type HTTPSecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme"`       // bearer, basic
	BearerFormat string `json:"bearerFormat,omitempty"`
}

type OpenIDSecurityScheme struct {
	Type             string `json:"type"`
	OpenIDConnectURL string `json:"openIdConnectUrl"`
}

type AuthenticationMethod struct {
	Type     string      `json:"type"`
	Required bool        `json:"required"`
	Schema   interface{} `json:"schema"`
}

type AgentEndpoint struct {
	Type           string `json:"type"`
	URL            string `json:"url"`
	HealthCheckURL string `json:"healthCheckUrl"`
	Timeout        int    `json:"timeout"`
}

type TaskResult struct {
	TaskID  string      `json:"taskId"`
	Status  string      `json:"status"`
	Agent   AgentInfo   `json:"agent"`
	Created string      `json:"created"`
	Output  string      `json:"output,omitempty"`
	Messages []interface{} `json:"messages,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type AgentInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// A2A Protocol v0.2.5 Compliant Data Structures
type TaskStatus struct {
	State     string  `json:"state"`               // A2A spec: required TaskState
	Message   *string `json:"message,omitempty"`   // A2A spec: optional Message
	Timestamp string  `json:"timestamp,omitempty"` // A2A spec: optional ISO 8601 datetime
}

type A2ATask struct {
	ID        string                 `json:"id"`
	ContextID string                 `json:"contextId"`
	Status    TaskStatus             `json:"status"`
	Kind      string                 `json:"kind"`                // A2A v0.2.5 required field
	AgentID   string                 `json:"agentId"`
	Input     interface{}            `json:"input"`
	Result    interface{}            `json:"result,omitempty"`
	Error     *string                `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt string                 `json:"createdAt"`
}

// StoredTask embeds A2ATask and adds internal storage fields
type StoredTask struct {
	A2ATask                           // Embedded A2A compliant task
	WorkflowID string `json:"workflowId"` // Internal Temporal workflow ID
}

// Agent routing configuration structures
type RoutingConfig struct {
	TaskQueue    string `yaml:"taskQueue"`
	WorkflowType string `yaml:"workflowType"`
}

type WorkflowCategory struct {
	Description string   `yaml:"description"`
	Examples    []string `yaml:"examples"`
}

type AgentRoutingYAML struct {
	Version            string                      `yaml:"version"`
	Routing            map[string]RoutingConfig    `yaml:"routing"`
	WorkflowCategories map[string]WorkflowCategory `yaml:"workflowCategories,omitempty"`
}

// Global routing configuration
var agentTaskQueues = make(map[string]string)
var agentWorkflows = make(map[string]string)

func loadAgentRouting() error {
	log.Printf("📋 Loading agent routing configuration...")
	
	configPath := "config/agent-routing.yaml"
	if envPath := os.Getenv("AGENT_ROUTING_CONFIG"); envPath != "" {
		configPath = envPath
	}
	
	// Read YAML file (required)
	yamlFile, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read routing config %s: %v", configPath, err)
	}
	
	// Parse YAML
	var config AgentRoutingYAML
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return fmt.Errorf("failed to parse routing config: %v", err)
	}
	
	// Load routing into global maps
	agentCount := 0
	for agentName, routing := range config.Routing {
		agentTaskQueues[agentName] = routing.TaskQueue
		agentWorkflows[agentName] = routing.WorkflowType
		agentCount++
	}
	
	log.Printf("✅ Loaded routing for %d agents from %s (version %s)", 
		agentCount, configPath, config.Version)
	
	// Log workflow categories if present
	if len(config.WorkflowCategories) > 0 {
		log.Printf("📊 Available workflow categories:")
		for category, info := range config.WorkflowCategories {
			log.Printf("   - %s: %s", category, info.Description)
		}
	}
	
	return nil
}

func validateEnvironment() error {
	log.Printf("🔍 Validating environment configuration...")
	
	// Required environment variables
	required := map[string]string{
		"TEMPORAL_HOST":       "Temporal server hostname",
		"TEMPORAL_PORT":       "Temporal server port", 
		"TEMPORAL_NAMESPACE":  "Temporal namespace",
		"A2A_PORT":           "A2A Gateway port",
		"REDIS_URL":          "Redis connection URL",
		"AGENT_REGISTRY_URL": "Agent Registry service URL",
	}
	
	var missingVars []string
	var invalidVars []string
	
	for envVar, description := range required {
		value := os.Getenv(envVar)
		if value == "" {
			missingVars = append(missingVars, fmt.Sprintf("%s (%s)", envVar, description))
		} else {
			// Validate specific formats
			switch envVar {
			case "TEMPORAL_PORT", "A2A_PORT":
				if len(value) == 0 || value[0] < '0' || value[0] > '9' {
					invalidVars = append(invalidVars, fmt.Sprintf("%s: must be numeric (got: %s)", envVar, value))
				}
			case "REDIS_URL":
				if !strings.HasPrefix(value, "redis://") && !strings.HasPrefix(value, "rediss://") {
					invalidVars = append(invalidVars, fmt.Sprintf("%s: must start with redis:// or rediss:// (got: %s)", envVar, value))
				}
			case "AGENT_REGISTRY_URL":
				if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
					invalidVars = append(invalidVars, fmt.Sprintf("%s: must be valid HTTP URL (got: %s)", envVar, value))
				}
			}
		}
	}
	
	// Optional but recommended environment variables
	optional := map[string]string{
		"JWT_SECRET":     "JWT signing secret (defaults to insecure value)",
		"LOG_LEVEL":      "Logging level (debug, info, warn, error)",
		"DATABASE_URL":   "PostgreSQL connection URL",
	}
	
	var warnings []string
	for envVar, description := range optional {
		value := os.Getenv(envVar)
		if value == "" {
			warnings = append(warnings, fmt.Sprintf("%s (%s)", envVar, description))
		} else {
			// Validate JWT secret strength
			if envVar == "JWT_SECRET" && (len(value) < 32 || strings.Contains(value, "default") || strings.Contains(value, "secret")) {
				warnings = append(warnings, fmt.Sprintf("%s: should be a strong, unique secret (current value appears weak)", envVar))
			}
		}
	}
	
	// Report findings
	if len(missingVars) > 0 {
		return fmt.Errorf("❌ Missing required environment variables:\n  - %s", strings.Join(missingVars, "\n  - "))
	}
	
	if len(invalidVars) > 0 {
		return fmt.Errorf("❌ Invalid environment variable values:\n  - %s", strings.Join(invalidVars, "\n  - "))
	}
	
	if len(warnings) > 0 {
		log.Printf("⚠️ Environment warnings (recommended to set):\n  - %s", strings.Join(warnings, "\n  - "))
	}
	
	log.Printf("✅ Environment validation passed")
	return nil
}

func NewGateway() (*Gateway, error) {
	// Validate environment first - fail fast
	if err := validateEnvironment(); err != nil {
		return nil, err
	}
	
	// Load agent routing configuration
	if err := loadAgentRouting(); err != nil {
		return nil, fmt.Errorf("failed to load agent routing: %w", err)
	}
	
	temporalHost := getEnv("TEMPORAL_HOST", "localhost")
	temporalPort := getEnv("TEMPORAL_PORT", "7233")
	namespace := getEnv("TEMPORAL_NAMESPACE", "default")
	port := getEnv("A2A_PORT", "3000")

	// Connect to Temporal
	temporalClient, err := client.Dial(client.Options{
		HostPort:  fmt.Sprintf("%s:%s", temporalHost, temporalPort),
		Namespace: namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Temporal: %w", err)
	}

	log.Printf("✅ Connected to Temporal at %s:%s, namespace: %s", temporalHost, temporalPort, namespace)

	// Initialize Redis client
	redisURL := getEnv("REDIS_URL", "redis://redis:6379")
	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}
	
	redisClient := redis.NewClient(redisOpts)
	
	// Test Redis connection
	ctx := context.Background()
	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		log.Printf("⚠️ Failed to connect to Redis: %v", err)
		// Continue without Redis for now (fallback to in-memory)
	} else {
		log.Printf("✅ Connected to Redis at %s", redisURL)
	}

	// Get Agent Registry URL
	agentRegistryURL := getEnv("AGENT_REGISTRY_URL", "http://agent-registry:8001")
	
	// Test Agent Registry connection
	resp, err := http.Get(agentRegistryURL + "/health")
	if err != nil {
		log.Printf("⚠️ Failed to connect to Agent Registry: %v", err)
		// Continue without Agent Registry for now
	} else {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			log.Printf("✅ Connected to Agent Registry at %s", agentRegistryURL)
		} else {
			log.Printf("⚠️ Agent Registry health check failed with status: %d", resp.StatusCode)
		}
	}

	return &Gateway{
		temporalClient: temporalClient,
		redisClient:    redisClient,
		agentRegistryURL: agentRegistryURL,
		port:           port,
	}, nil
}

// Agent Registry Helper Methods
func (g *Gateway) callAgentRegistry(method string, path string, body interface{}) ([]byte, error) {
	var req *http.Request
	var err error
	
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		req, err = http.NewRequest(method, g.agentRegistryURL+path, strings.NewReader(string(data)))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, g.agentRegistryURL+path, nil)
		if err != nil {
			return nil, err
		}
	}
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	if resp.StatusCode >= 400 {
		log.Printf("🔍 Registry returned HTTP %d: %s", resp.StatusCode, string(respBody))
		return nil, fmt.Errorf("agent registry error: %s", string(respBody))
	}
	
	return respBody, nil
}


// Redis Task Storage Methods
func (g *Gateway) storeTaskInRedis(task *StoredTask) error {
	if g.redisClient == nil {
		return fmt.Errorf("Redis client not available")
	}

	ctx := context.Background()
	taskKey := fmt.Sprintf("task:%s", task.A2ATask.ID)
	
	// Use pipeline for atomic operations
	pipe := g.redisClient.Pipeline()
	
	// Validate input timestamps
	if err := validateISO8601Timestamp(task.A2ATask.CreatedAt); err != nil {
		return fmt.Errorf("invalid CreatedAt timestamp: %w", err)
	}
	if task.A2ATask.Status.Timestamp != "" {
		if err := validateISO8601Timestamp(task.A2ATask.Status.Timestamp); err != nil {
			return fmt.Errorf("invalid Status.Timestamp: %w", err)
		}
	}
	
	// Generate and validate updated timestamp
	updatedTimestamp := newISO8601Timestamp()
	if err := validateISO8601Timestamp(updatedTimestamp); err != nil {
		return fmt.Errorf("generated timestamp validation failed: %w", err)
	}
	
	// Store main task data
	errorStr := ""
	if task.A2ATask.Error != nil {
		errorStr = *task.A2ATask.Error
	}
	
	taskData := map[string]interface{}{
		"agent_id":    task.A2ATask.AgentID,
		"workflow_id": task.WorkflowID,
		"status":      task.A2ATask.Status.State,
		"created":     task.A2ATask.CreatedAt,
		"updated":     updatedTimestamp,
		"error":       errorStr,
	}
	
	if task.A2ATask.Input != nil {
		inputJSON, err := json.Marshal(task.A2ATask.Input)
		if err != nil {
			return fmt.Errorf("failed to marshal task input: %w", err)
		}
		taskData["input"] = string(inputJSON)
	}
	
	if task.A2ATask.Result != nil {
		resultJSON, err := json.Marshal(task.A2ATask.Result)
		if err != nil {
			return fmt.Errorf("failed to marshal task result: %w", err)
		}
		taskData["result"] = string(resultJSON)
	}
	
	if task.A2ATask.Metadata != nil {
		metadataJSON, err := json.Marshal(task.A2ATask.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal task metadata: %w", err)
		}
		taskData["metadata"] = string(metadataJSON)
	}
	
	pipe.HSet(ctx, taskKey, taskData)
	
	// Add to indexes for querying
	createdTime, err := time.Parse(time.RFC3339, task.A2ATask.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to parse task creation time: %w", err)
	}
	pipe.ZAdd(ctx, "tasks:by_created", &redis.Z{
		Score:  float64(createdTime.Unix()),
		Member: task.A2ATask.ID,
	})
	
	pipe.SAdd(ctx, fmt.Sprintf("tasks:by_status:%s", task.A2ATask.Status.State), task.A2ATask.ID)
	pipe.SAdd(ctx, fmt.Sprintf("tasks:by_agent:%s", task.A2ATask.AgentID), task.A2ATask.ID)
	
	// Index by metadata key-value pairs (generic metadata indexing)
	if task.A2ATask.Metadata != nil {
		for key, value := range task.A2ATask.Metadata {
			// Convert value to string for indexing
			valueStr := fmt.Sprintf("%v", value)
			pipe.SAdd(ctx, fmt.Sprintf("tasks:by_metadata:%s:%s", key, valueStr), task.A2ATask.ID)
		}
	}
	
	// Set TTL based on status
	if task.A2ATask.Status.State == "completed" || task.A2ATask.Status.State == "failed" || task.A2ATask.Status.State == "canceled" {
		pipe.Expire(ctx, taskKey, 24*time.Hour) // 1 day for completed
	} else {
		pipe.Expire(ctx, taskKey, 7*24*time.Hour) // 7 days for active
	}
	
	_, err = pipe.Exec(ctx)
	return err
}

func (g *Gateway) getTaskFromRedis(taskID string) (*StoredTask, error) {
	if g.redisClient == nil {
		return nil, fmt.Errorf("Redis client not available")
	}

	ctx := context.Background()
	taskKey := fmt.Sprintf("task:%s", taskID)
	
	result := g.redisClient.HGetAll(ctx, taskKey)
	if result.Err() != nil {
		return nil, result.Err()
	}
	
	data := result.Val()
	if len(data) == 0 {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	
	// Create A2A spec compliant TaskStatus
	status := data["status"]
	
	// Use current time for timestamp in ISO 8601 format (A2A spec compliant)
	timestamp := newISO8601Timestamp()
	if data["createdAt"] != "" {
		// Try to parse created time, fallback to current time
		if createdTime, err := time.Parse(time.RFC3339, data["createdAt"]); err == nil {
			timestamp = createdTime.Format(time.RFC3339)
		}
	}

	taskStatus := TaskStatus{
		State:     status,        // A2A spec compliant: use 'state'
		Timestamp: timestamp,     // A2A spec compliant: use 'timestamp'
	}

	// Use contextId from storage
	contextId := data["contextId"]
	if contextId == "" {
		contextId = fmt.Sprintf("ctx-%s", taskID[:8])
	}

	var errorPtr *string
	if data["error"] != "" {
		errorStr := data["error"]
		errorPtr = &errorStr
	}

	task := &StoredTask{
		A2ATask: A2ATask{
			ID:        taskID,
			ContextID: contextId,
			Status:    taskStatus,
			Kind:      "task",           // A2A v0.2.5 required field
			AgentID:   data["agentId"],
			Error:     errorPtr,
			CreatedAt: data["createdAt"],
		},
		WorkflowID: data["workflowId"],
	}
	
	if data["input"] != "" {
		json.Unmarshal([]byte(data["input"]), &task.A2ATask.Input)
	}
	
	if data["result"] != "" {
		json.Unmarshal([]byte(data["result"]), &task.A2ATask.Result)
	}
	
	if data["metadata"] != "" {
		json.Unmarshal([]byte(data["metadata"]), &task.A2ATask.Metadata)
		
		// Check if contextId is in metadata
		if task.A2ATask.Metadata != nil {
			if ctxId, exists := task.A2ATask.Metadata["contextId"]; exists {
				if ctxIdStr, ok := ctxId.(string); ok {
					task.A2ATask.ContextID = ctxIdStr
				}
			}
		}
	}
	
	return task, nil
}

func (g *Gateway) updateTaskStatusInRedis(taskID, status string, result interface{}, errorMsg string) error {
	if g.redisClient == nil {
		return fmt.Errorf("Redis client not available")
	}

	ctx := context.Background()
	taskKey := fmt.Sprintf("task:%s", taskID)
	
	pipe := g.redisClient.Pipeline()
	
	// Generate and validate timestamp
	timestamp := newISO8601Timestamp()
	if err := validateISO8601Timestamp(timestamp); err != nil {
		return fmt.Errorf("generated timestamp validation failed: %w", err)
	}
	
	// Update task fields
	updates := map[string]interface{}{
		"status":  status,
		"updated": timestamp,
		"error":   errorMsg,
	}
	
	var err error
	if result != nil {
		var resultJSON []byte
		resultJSON, err = json.Marshal(result)
		if err != nil {
			log.Printf("❌ Failed to marshal task result: %v", err)
			return fmt.Errorf("failed to marshal task result: %w", err)
		}
		updates["result"] = string(resultJSON)
	}
	
	pipe.HSet(ctx, taskKey, updates)
	
	// Update status index (remove from old, add to new)
	oldStatus := g.redisClient.HGet(ctx, taskKey, "status").Val()
	if oldStatus != "" && oldStatus != status {
		pipe.SRem(ctx, fmt.Sprintf("tasks:by_status:%s", oldStatus), taskID)
	}
	pipe.SAdd(ctx, fmt.Sprintf("tasks:by_status:%s", status), taskID)
	
	_, err = pipe.Exec(ctx)
	return err
}

// Redis Cleanup Methods
func (g *Gateway) cleanupExpiredTasks() error {
	if g.redisClient == nil {
		return fmt.Errorf("Redis client not available")
	}

	ctx := context.Background()
	log.Printf("🧹 Starting Redis task cleanup...")

	// Get all task keys
	taskKeys, err := g.redisClient.Keys(ctx, "task:*").Result()
	if err != nil {
		return fmt.Errorf("failed to get task keys: %w", err)
	}

	cleanedCount := 0
	errorCount := 0

	for _, taskKey := range taskKeys {
		// Check if task exists (TTL expired tasks are automatically removed)
		exists, err := g.redisClient.Exists(ctx, taskKey).Result()
		if err != nil {
			log.Printf("⚠️ Failed to check existence of %s: %v", taskKey, err)
			errorCount++
			continue
		}

		if exists == 0 {
			// Task was TTL expired, clean up indexes
			taskID := strings.TrimPrefix(taskKey, "task:")
			if err := g.cleanupTaskIndexes(taskID); err != nil {
				log.Printf("⚠️ Failed to clean indexes for %s: %v", taskID, err)
				errorCount++
			} else {
				cleanedCount++
			}
		}
	}

	log.Printf("✅ Redis cleanup completed: %d tasks cleaned, %d errors", cleanedCount, errorCount)
	return nil
}

func (g *Gateway) cleanupTaskIndexes(taskID string) error {
	if g.redisClient == nil {
		return fmt.Errorf("Redis client not available")
	}

	ctx := context.Background()
	pipe := g.redisClient.Pipeline()

	// Remove from all possible indexes
	pipe.ZRem(ctx, "tasks:by_created", taskID)

	// Remove from status indexes
	statusIndexes := []string{"submitted", "running", "in_progress", "completed", "failed", "canceled"}
	for _, status := range statusIndexes {
		pipe.SRem(ctx, fmt.Sprintf("tasks:by_status:%s", status), taskID)
	}

	// Remove from agent indexes (we need to check which agents exist)
	agentKeys, err := g.redisClient.Keys(ctx, "tasks:by_agent:*").Result()
	if err == nil {
		for _, agentKey := range agentKeys {
			pipe.SRem(ctx, agentKey, taskID)
		}
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (g *Gateway) cleanupOldCompletedTasks(olderThanHours int) error {
	if g.redisClient == nil {
		return fmt.Errorf("Redis client not available")
	}

	ctx := context.Background()
	cutoffTime := time.Now().Add(-time.Duration(olderThanHours) * time.Hour)
	cutoffUnix := float64(cutoffTime.Unix())

	log.Printf("🧹 Cleaning completed tasks older than %d hours (%s)", olderThanHours, cutoffTime.Format(time.RFC3339))

	// Get old tasks from the created time index
	oldTaskIDs, err := g.redisClient.ZRangeByScore(ctx, "tasks:by_created", &redis.ZRangeBy{
		Min: "0",
		Max: fmt.Sprintf("%.0f", cutoffUnix),
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to get old tasks: %w", err)
	}

	cleanedCount := 0
	for _, taskID := range oldTaskIDs {
		// Check if task is completed/failed/cancelled
		taskKey := fmt.Sprintf("task:%s", taskID)
		status, err := g.redisClient.HGet(ctx, taskKey, "status").Result()
		if err != nil {
			continue // Skip if we can't get status
		}

		if status == "completed" || status == "failed" || status == "canceled" {
			// Force remove the task and its indexes
			pipe := g.redisClient.Pipeline()
			pipe.Del(ctx, taskKey)
			pipe.ZRem(ctx, "tasks:by_created", taskID)
			pipe.SRem(ctx, fmt.Sprintf("tasks:by_status:%s", status), taskID)

			// Remove from agent indexes
			agentID, err := g.redisClient.HGet(ctx, taskKey, "agent_id").Result()
			if err == nil {
				pipe.SRem(ctx, fmt.Sprintf("tasks:by_agent:%s", agentID), taskID)
			}

			_, err = pipe.Exec(ctx)
			if err == nil {
				cleanedCount++
			}
		}
	}

	log.Printf("✅ Cleaned %d old completed tasks", cleanedCount)
	return nil
}

func (g *Gateway) startRedisCleanupScheduler() {
	if g.redisClient == nil {
		log.Printf("⚠️ Redis cleanup scheduler not started - Redis client not available")
		return
	}

	log.Printf("🕐 Starting Redis cleanup scheduler...")

	go func() {
		// Run cleanup every hour
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Cleanup expired task indexes
				if err := g.cleanupExpiredTasks(); err != nil {
					log.Printf("❌ Redis cleanup failed: %v", err)
				}

				// Cleanup old completed tasks (older than 7 days)
				if err := g.cleanupOldCompletedTasks(7 * 24); err != nil {
					log.Printf("❌ Old task cleanup failed: %v", err)
				}
			}
		}
	}()

	log.Printf("✅ Redis cleanup scheduler started")
}

func (g *Gateway) handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.sendError(w, &req, ErrorParseError, "Parse error")
		return
	}

	// JSON-RPC 2.0 validation
	if req.Jsonrpc != "2.0" {
		g.sendError(w, &req, ErrorInvalidRequest, "Invalid Request - missing or invalid jsonrpc field")
		return
	}

	if req.Method == "" {
		g.sendError(w, &req, ErrorInvalidRequest, "Invalid Request - missing method field")
		return
	}

	log.Printf("📨 Received A2A request: %s", req.Method)

	switch req.Method {
	// Standard A2A Protocol Methods (Required)
	case "message/send":
		g.handleA2AMessageSend(w, &req)
	case "tasks/get":
		g.handleA2ATasksGet(w, &req)
	case "tasks/cancel":
		g.handleA2ATasksCancel(w, &req)
	
	// Backward compatibility for existing tests
	case "a2a.createTask":
		addDeprecationWarnings(w, "a2a.createTask")
		g.handleCreateTask(w, &req)
	case "a2a.getTask":
		addDeprecationWarnings(w, "a2a.getTask")
		g.handleGetTask(w, &req)
	case "a2a.cancelTask":
		addDeprecationWarnings(w, "a2a.cancelTask")
		g.handleCancelTask(w, &req)
		
	default:
		g.sendError(w, &req, ErrorMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (g *Gateway) handleSendMessage(w http.ResponseWriter, req *JSONRPCRequest) {
	var params SendMessageParams
	paramBytes, err := json.Marshal(req.Params)
	if err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Failed to marshal request params")
		return
	}
	if err := json.Unmarshal(paramBytes, &params); err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Invalid params")
		return
	}

	if params.ID == "" {
		g.sendError(w, req, ErrorInvalidParams, "Missing id parameter")
		return
	}

	log.Printf("💬 Sending message to task %s", params.ID)

	// Get task from Redis
	storedTask, err := g.getTaskFromRedis(params.ID)
	if err != nil {
		g.sendError(w, req, ErrorTaskNotFound, fmt.Sprintf("Task not found: %s", params.ID))
		return
	}

	// Check if task is in a state that can receive messages
	if storedTask.A2ATask.Status.State != "working" && storedTask.A2ATask.Status.State != "running" {
		g.sendError(w, req, ErrorTaskStateInvalid, fmt.Sprintf("Invalid task state: %s", storedTask.A2ATask.Status.State))
		return
	}

	// For now, add the message to the task's input (simplified implementation)
	// In a full implementation, this would send the message to the running workflow
	messageID := uuid.New().String()
	
	g.sendResult(w, req, map[string]interface{}{
		"messageId": messageID,
		"status":    storedTask.A2ATask.Status.State,
		"message":   "Message queued for delivery",
	})
}

func (g *Gateway) handleCancelTask(w http.ResponseWriter, req *JSONRPCRequest) {
	var params CancelTaskParams
	paramBytes, err := json.Marshal(req.Params)
	if err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Failed to marshal request params")
		return
	}
	if err = json.Unmarshal(paramBytes, &params); err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Invalid params")
		return
	}

	if params.ID == "" {
		g.sendError(w, req, -32602, "Missing id parameter")
		return
	}

	log.Printf("🚫 Cancelling task %s", params.ID)

	// Get task from Redis
	storedTask, err := g.getTaskFromRedis(params.ID)
	if err != nil {
		g.sendError(w, req, ErrorTaskNotFound, fmt.Sprintf("Task not found: %s", params.ID))
		return
	}

	// Check if task can be cancelled
	if storedTask.A2ATask.Status.State == "completed" || storedTask.A2ATask.Status.State == "failed" || storedTask.A2ATask.Status.State == "canceled" {
		g.sendError(w, req, ErrorTaskStateInvalid, fmt.Sprintf("Cannot cancel task in state: %s", storedTask.A2ATask.Status.State))
		return
	}

	// Cancel the Temporal workflow
	ctx := context.Background()
	err = g.temporalClient.CancelWorkflow(ctx, params.ID, "")
	if err != nil {
		log.Printf("❌ Failed to cancel workflow %s: %v", params.ID, err)
		g.sendError(w, req, ErrorTaskCancelFailed, fmt.Sprintf("Failed to cancel task: %v", err))
		return
	}

	// Update task status in Redis
	err = g.updateTaskStatusInRedis(params.ID, "canceled", nil, "")
	if err != nil {
		log.Printf("⚠️ Failed to update task status in Redis: %v", err)
	}

	g.sendDeprecatedResult(w, req, map[string]interface{}{
		"status": "canceled",
	})
}

func (g *Gateway) handleRegisterAgent(w http.ResponseWriter, req *JSONRPCRequest) {
	// Use official A2A pattern: interface{} → re-marshal → unmarshal to struct
	paramBytes, err := json.Marshal(req.Params)
	if err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Invalid params")
		return
	}
	
	var params struct {
		AgentCard AgentCard `json:"agentCard"`
	}
	
	if err := json.Unmarshal(paramBytes, &params); err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Invalid params")
		return
	}

	if params.AgentCard.Name == "" {
		g.sendError(w, req, ErrorInvalidParams, "Missing agent name")
		return
	}

	log.Printf("🤖 Registering agent %s", params.AgentCard.Name)

	// Call Agent Registry microservice
	// Registry expects agent data wrapped in "agentCard" field
	reqBody := map[string]interface{}{
		"agentCard": params.AgentCard,
	}
	
	
	respBody, err := g.callAgentRegistry("POST", "/agents/register", reqBody)
	if err != nil {
		log.Printf("⚠️ Failed to register agent: %v", err)
		g.sendError(w, req, ErrorAgentRegistrationFailed, fmt.Sprintf("Failed to register agent: %v", err))
		return
	}
	
	var response RegisterAgentResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		log.Printf("⚠️ Failed to parse agent registry response: %v", err)
		g.sendError(w, req, ErrorAgentRegistryConnection, "Failed to parse registration response")
		return
	}
	
	g.sendResult(w, req, map[string]interface{}{
		"agentId": response.AgentID,
	})
}

func (g *Gateway) handleDiscoverAgents(w http.ResponseWriter, req *JSONRPCRequest) {
	var params DiscoverAgentsParams
	paramBytes, err := json.Marshal(req.Params)
	if err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Failed to marshal request params")
		return
	}
	if err := json.Unmarshal(paramBytes, &params); err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Invalid params")
		return
	}

	log.Printf("🔍 Discovering agents (capability: %s, keyword: %s, limit: %d)", params.Capability, params.Keyword, params.Limit)

	// Build query parameters
	queryParams := ""
	if params.Capability != "" {
		queryParams += fmt.Sprintf("?capability=%s", params.Capability)
	}
	if params.Keyword != "" {
		if queryParams == "" {
			queryParams += "?"
		} else {
			queryParams += "&"
		}
		queryParams += fmt.Sprintf("keyword=%s", params.Keyword)
	}
	if params.Limit > 0 {
		if queryParams == "" {
			queryParams += "?"
		} else {
			queryParams += "&"
		}
		queryParams += fmt.Sprintf("limit=%d", params.Limit)
	}
	
	// Call Agent Registry microservice
	respBody, err := g.callAgentRegistry("GET", "/agents/discover"+queryParams, nil)
	if err != nil {
		log.Printf("⚠️ Failed to discover agents: %v", err)
		// Fallback to built-in agents on error
		g.sendResult(w, req, map[string]interface{}{
			"agents": []interface{}{},
		})
		return
	}
	
	// The registry returns Google SDK-compatible AgentCard objects
	// Just pass through the response as-is since it's already in the correct format
	log.Printf("🔄 Using new pass-through discovery logic")
	var registryResponse map[string]interface{}
	if err := json.Unmarshal(respBody, &registryResponse); err != nil {
		log.Printf("⚠️ NEW: Failed to parse agent discovery response: %v", err)
		log.Printf("⚠️ Raw response body: %s", string(respBody))
		g.sendError(w, req, ErrorAgentRegistryConnection, "Failed to parse discovery response")
		return
	}
	
	log.Printf("✅ Successfully parsed registry response, passing through")
	// Pass through the registry response directly
	g.sendResult(w, req, registryResponse)
}

func (g *Gateway) handleListAgents(w http.ResponseWriter, req *JSONRPCRequest) {
	agents := []AgentInfo{
		{ID: "echo-agent", Name: "Echo Agent"},
		{ID: "langchain-agent", Name: "LangChain Agent"},
		{ID: "crewai-agent", Name: "CrewAI Agent"},
		{ID: "autogen-agent", Name: "AutoGen Agent"},
	}

	g.sendResult(w, req, map[string]interface{}{
		"agents": agents,
	})
}

func (g *Gateway) handleCreateTask(w http.ResponseWriter, req *JSONRPCRequest) {
	var params CreateTaskParams
	paramBytes, err := json.Marshal(req.Params)
	if err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Failed to marshal request params")
		return
	}
	if err := json.Unmarshal(paramBytes, &params); err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Invalid params")
		return
	}

	if params.AgentID == "" {
		g.sendError(w, req, ErrorInvalidParams, "Missing required parameter: agentId")
		return
	}

	taskID := uuid.New().String()
	log.Printf("🔄 Creating task %s for agent %s", taskID, params.AgentID)
	
	// Record task creation metrics
	ctx := context.Background()
	RecordTaskCreated(ctx, params.AgentID)

	// Get workflow and task queue for agent
	workflowType, ok := agentWorkflows[params.AgentID]
	if !ok {
		g.sendError(w, req, ErrorAgentNotFound, fmt.Sprintf("Unknown agent: %s", params.AgentID))
		return
	}

	taskQueue, ok := agentTaskQueues[params.AgentID]
	if !ok {
		g.sendError(w, req, ErrorAgentNotFound, fmt.Sprintf("Unknown agent: %s", params.AgentID))
		return
	}

	// Convert input to A2A message format if needed
	normalizedInput := g.normalizeTaskInput(params.Input)
	
	// Create workflow input with metadata
	workflowInput, ok := normalizedInput.(map[string]interface{})
	if !ok {
		log.Printf("❌ Failed to convert normalized input to map[string]interface{}")
		g.sendError(w, req, ErrorTaskCreationFailed, "Failed to process task input")
		return
	}
	
	if params.Metadata != nil {
		workflowInput["metadata"] = params.Metadata
	}
	
	// Start Temporal workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        taskID,
		TaskQueue: taskQueue,
	}
	workflowRun, err := g.temporalClient.ExecuteWorkflow(ctx, workflowOptions, workflowType, workflowInput)
	if err != nil {
		log.Printf("❌ Failed to start workflow: %v", err)
		g.sendError(w, req, ErrorTaskCreationFailed, fmt.Sprintf("Failed to create task: %v", err))
		return
	}

	// Create A2A compliant task structure
	createdAt := newISO8601Timestamp()
	
	// Generate contextId from metadata or create one
	contextId := fmt.Sprintf("ctx-%s", taskID[:8])
	if params.Metadata != nil {
		if ctxId, exists := params.Metadata["contextId"]; exists {
			if ctxIdStr, ok := ctxId.(string); ok {
				contextId = ctxIdStr
			}
		}
	}

	// Create A2A spec compliant submitted status
	taskStatus := TaskStatus{
		State:     "submitted",   // A2A spec compliant: initial 'submitted' state before execution
		Timestamp: createdAt,     // A2A spec compliant: use 'timestamp'
	}

	// Store task metadata
	taskData := &StoredTask{
		A2ATask: A2ATask{
			ID:        taskID,
			ContextID: contextId,
			Status:    taskStatus,
			Kind:      "task",           // A2A v0.2.5 required field
			AgentID:   params.AgentID,
			Input:     normalizedInput,
			Metadata:  params.Metadata,
			CreatedAt: createdAt,
		},
		WorkflowID: taskID, // Use taskID as workflowID for simplicity
	}
	
	// Store in Redis
	err = g.storeTaskInRedis(taskData)
	if err != nil {
		log.Printf("⚠️ Failed to store task in Redis: %v", err)
		// Continue - task will still be tracked via Temporal
	}

	// Transition task from "submitted" to "working" now that workflow has started
	err = g.updateTaskStatusInRedis(taskID, "working", nil, "")
	if err != nil {
		log.Printf("⚠️ Failed to update task status to working: %v", err)
		// Continue - task will still be tracked via Temporal
	}

	log.Printf("✅ Started workflow %s on queue %s", taskID, taskQueue)

	// Start background goroutine to monitor workflow completion
	go g.monitorWorkflow(taskID, workflowRun)

	agentName := params.AgentID
	if agentName == "echo-agent" {
		agentName = "Echo Agent"
	}

	g.sendDeprecatedResult(w, req, &TaskResult{
		TaskID: taskID,
		Status: "running",
		Agent: AgentInfo{
			ID:   params.AgentID,
			Name: agentName,
		},
		Created: taskData.A2ATask.CreatedAt,
	})
}

func (g *Gateway) handleGetTask(w http.ResponseWriter, req *JSONRPCRequest) {
	var params GetTaskParams
	paramBytes, err := json.Marshal(req.Params)
	if err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Failed to marshal request params")
		return
	}
	if err := json.Unmarshal(paramBytes, &params); err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Invalid params")
		return
	}

	if params.ID == "" {
		g.sendError(w, req, ErrorInvalidParams, "Missing id parameter")
		return
	}

	log.Printf("📊 Getting task %s", params.ID)

	// Get task from Redis
	storedTask, err := g.getTaskFromRedis(params.ID)
	if err != nil {
		g.sendError(w, req, ErrorTaskNotFound, fmt.Sprintf("Task not found: %s", params.ID))
		return
	}

	agentName := storedTask.A2ATask.AgentID
	if agentName == "echo-agent" {
		agentName = "Echo Agent"
	}

	// Return the embedded A2A task
	a2aTask := g.getA2ATask(storedTask)

	g.sendDeprecatedResult(w, req, a2aTask)
}

func (g *Gateway) monitorWorkflow(taskID string, workflowRun client.WorkflowRun) {
	ctx := context.Background()
	
	// Wait for workflow completion with timeout
	var result interface{}
	err := workflowRun.Get(ctx, &result)
	
	if err != nil {
		log.Printf("❌ Workflow %s failed: %v", taskID, err)
		// Update task status in Redis
		updateErr := g.updateTaskStatusInRedis(taskID, "failed", nil, err.Error())
		if updateErr != nil {
			log.Printf("⚠️ Failed to update failed task status in Redis: %v", updateErr)
		}
	} else {
		log.Printf("✅ Workflow %s completed successfully", taskID)
		// Update task status and result in Redis
		updateErr := g.updateTaskStatusInRedis(taskID, "completed", result, "")
		if updateErr != nil {
			log.Printf("⚠️ Failed to update completed task status in Redis: %v", updateErr)
		}
	}
}

func (g *Gateway) handleGetTasksByMetadata(w http.ResponseWriter, req *JSONRPCRequest) {
	var params GetTasksByMetadataParams
	paramBytes, err := json.Marshal(req.Params)
	if err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Failed to marshal request params")
		return
	}
	if err := json.Unmarshal(paramBytes, &params); err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Invalid params")
		return
	}

	if params.MetadataKey == "" {
		g.sendError(w, req, ErrorInvalidParams, "metadataKey is required")
		return
	}

	log.Printf("🔍 Getting tasks by metadata: %s = %v", params.MetadataKey, params.MetadataValue)

	ctx := context.Background()
	valueStr := fmt.Sprintf("%v", params.MetadataValue)
	setKey := fmt.Sprintf("tasks:by_metadata:%s:%s", params.MetadataKey, valueStr)
	
	// Get task IDs from metadata index
	taskIDs, err := g.redisClient.SMembers(ctx, setKey).Result()
	if err != nil {
		log.Printf("❌ Failed to get tasks from metadata index: %v", err)
		g.sendError(w, req, ErrorInternalError, "Failed to query tasks by metadata")
		return
	}

	// Apply limit if specified
	if params.Limit > 0 && len(taskIDs) > params.Limit {
		taskIDs = taskIDs[:params.Limit]
	}

	// Retrieve task details
	var tasks []*StoredTask
	for _, taskID := range taskIDs {
		task, err := g.getTaskFromRedis(taskID)
		if err != nil {
			log.Printf("⚠️ Failed to get task %s: %v", taskID, err)
			continue // Skip missing tasks but continue processing
		}
		tasks = append(tasks, task)
	}

	log.Printf("✅ Found %d tasks with metadata %s = %v", len(tasks), params.MetadataKey, params.MetadataValue)

	g.sendResult(w, req, map[string]interface{}{
		"tasks":       tasks,
		"count":       len(tasks),
		"metadataKey": params.MetadataKey,
		"metadataValue": params.MetadataValue,
	})
}

// Standard A2A Protocol Methods Implementation
func (g *Gateway) handleA2AMessageSend(w http.ResponseWriter, req *JSONRPCRequest) {
	var params MessageSendParams
	paramBytes, err := json.Marshal(req.Params)
	if err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Failed to marshal request params")
		return
	}
	if err := json.Unmarshal(paramBytes, &params); err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Invalid params")
		return
	}

	if params.AgentID == "" {
		g.sendError(w, req, ErrorInvalidParams, "Missing required parameter: agentId")
		return
	}

	log.Printf("📨 A2A message/send for agent %s", params.AgentID)

	// Generate unique task ID
	taskID := uuid.New().String()
	
	// Create A2A-compliant task directly without delegation
	ctx := context.Background()
	taskQueue := fmt.Sprintf("%s-tasks", params.AgentID)
	workflowType := fmt.Sprintf("%sWorkflow", params.AgentID)
	
	// Temporal workflow options
	workflowOptions := client.StartWorkflowOptions{
		ID:        taskID,
		TaskQueue: taskQueue,
	}
	
	// Start Temporal workflow
	workflowRun, err := g.temporalClient.ExecuteWorkflow(ctx, workflowOptions, workflowType, params.Message)
	if err != nil {
		log.Printf("❌ Failed to start workflow: %v", err)
		g.sendError(w, req, ErrorTaskCreationFailed, fmt.Sprintf("Failed to create task: %v", err))
		return
	}
	
	// Create A2A-compliant task
	currentTime := newISO8601Timestamp()
	contextId := fmt.Sprintf("ctx-%s", taskID[:8])
	if params.Metadata != nil {
		if ctxId, exists := params.Metadata["contextId"]; exists {
			if ctxIdStr, ok := ctxId.(string); ok {
				contextId = ctxIdStr
			}
		}
	}
	
	// Create task with working state (A2A spec: message/send is immediate execution)
	taskStatus := TaskStatus{
		State:     "working",
		Timestamp: currentTime,
	}
	
	a2aTask := A2ATask{
		ID:        taskID,
		ContextID: contextId,
		Status:    taskStatus,
		Kind:      "task",
		AgentID:   params.AgentID,
		Input:     params.Message,
		Result:    nil,  // A2A spec: initially null until completion
		Error:     nil,  // A2A spec: initially null unless error occurs
		Metadata: map[string]interface{}{
			"source":     "a2a-gateway",
			"method":     "message/send",
			"workflowId": workflowRun.GetID(),
			"timestamp":  currentTime,
		},
		CreatedAt: currentTime,
	}
	
	// Store in Redis with working state (A2A spec: message/send starts as working)
	storedTask := &StoredTask{A2ATask: a2aTask}
	err = g.storeTaskInRedis(storedTask)
	if err != nil {
		log.Printf("⚠️ Failed to store task in Redis: %v", err)
	}
	
	// Start monitoring
	go g.monitorWorkflow(taskID, workflowRun)
	
	log.Printf("✅ A2A task %s created successfully", taskID)
	
	// Return A2A-compliant task response (not wrapped TaskResult)
	g.sendResult(w, req, a2aTask)
}

func (g *Gateway) handleA2ATasksGet(w http.ResponseWriter, req *JSONRPCRequest) {
	var params TasksGetParams
	paramBytes, err := json.Marshal(req.Params)
	if err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Failed to marshal request params")
		return
	}
	if err := json.Unmarshal(paramBytes, &params); err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Invalid params")
		return
	}

	log.Printf("📊 A2A tasks/get for task %s", params.ID)

	// Convert to getTask format
	getTaskParams := GetTaskParams{
		ID: params.ID,
	}

	// Create new request for getTask
	getTaskReq := &JSONRPCRequest{
		Jsonrpc: req.Jsonrpc,
		Method:  "a2a.getTask",
		Params:  getTaskParams,
		ID:      req.ID,
	}

	// Delegate to existing getTask implementation
	g.handleGetTask(w, getTaskReq)
}

func (g *Gateway) handleA2ATasksCancel(w http.ResponseWriter, req *JSONRPCRequest) {
	var params TasksCancelParams
	paramBytes, err := json.Marshal(req.Params)
	if err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Failed to marshal request params")
		return
	}
	if err := json.Unmarshal(paramBytes, &params); err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Invalid params")
		return
	}

	log.Printf("🚫 A2A tasks/cancel for task %s", params.ID)

	// Convert to cancelTask format
	cancelTaskParams := CancelTaskParams{
		ID: params.ID,
	}

	// Create new request for cancelTask
	cancelTaskReq := &JSONRPCRequest{
		Jsonrpc: req.Jsonrpc,
		Method:  "a2a.cancelTask",
		Params:  cancelTaskParams,
		ID:      req.ID,
	}

	// Delegate to existing cancelTask implementation
	g.handleCancelTask(w, cancelTaskReq)
}

func (g *Gateway) handleErrorCodes(w http.ResponseWriter, r *http.Request) {
	// Return all A2A error codes with their metadata
	errorCodes := []ErrorCodeInfo{}
	
	// Add all defined error codes
	allCodes := []int{
		// JSON-RPC Standard
		ErrorParseError, ErrorInvalidRequest, ErrorMethodNotFound, ErrorInvalidParams, ErrorInternalError,
		// Task Management
		ErrorTaskNotFound, ErrorTaskStateInvalid, ErrorTaskCreationFailed, ErrorTaskUpdateFailed, 
		ErrorTaskCancelFailed, ErrorTaskTimeout, ErrorTaskQuotaExceeded,
		// Agent Management
		ErrorAgentNotFound, ErrorAgentUnavailable, ErrorAgentIncompatible, ErrorAgentRegistrationFailed,
		ErrorAgentCapabilityMismatch, ErrorAgentQuotaExceeded,
		// Authentication & Authorization
		ErrorUnauthorized, ErrorForbidden, ErrorInvalidAPIKey, ErrorAPIKeyExpired, 
		ErrorRateLimitExceeded, ErrorQuotaExceeded,
		// Service Integration
		ErrorTemporalConnection, ErrorRedisConnection, ErrorDatabaseConnection, 
		ErrorAgentRegistryConnection, ErrorExternalServiceTimeout, ErrorExternalServiceError,
		// Validation
		ErrorValidationFailed, ErrorInvalidMessageFormat, ErrorInvalidConfiguration, 
		ErrorEnvironmentInvalid, ErrorSchemaValidation,
	}
	
	for _, code := range allCodes {
		errorCodes = append(errorCodes, GetErrorInfo(code))
	}
	
	response := map[string]interface{}{
		"title": "A2A Error Codes",
		"description": "Standardized error codes for Agent-to-Agent protocol",
		"version": "1.0.0",
		"errors": errorCodes,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (g *Gateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check Redis health
	redisHealthy := false
	if g.redisClient != nil {
		_, err := g.redisClient.Ping(context.Background()).Result()
		redisHealthy = (err == nil)
	}
	
	// Check Agent Registry health
	agentRegistryHealthy := false
	resp, err := http.Get(g.agentRegistryURL + "/health")
	if err == nil {
		resp.Body.Close()
		agentRegistryHealthy = (resp.StatusCode == 200)
	}
	
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": newISO8601Timestamp(),
		"version":   "0.4.0-go",
		"service":   "a2a-gateway-go",
		"temporal": map[string]interface{}{
			"connected": g.temporalClient != nil,
		},
		"redis": map[string]interface{}{
			"connected": redisHealthy,
		},
		"agentRegistry": map[string]interface{}{
			"connected": agentRegistryHealthy,
			"url": g.agentRegistryURL,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (g *Gateway) sendResult(w http.ResponseWriter, req *JSONRPCRequest, result interface{}) {
	response := &JSONRPCResponse{
		Jsonrpc: "2.0",
		Result:  result,
		ID:      req.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// sendDeprecatedResult sends a JSON-RPC response with deprecation flag for legacy methods
func (g *Gateway) sendDeprecatedResult(w http.ResponseWriter, req *JSONRPCRequest, result interface{}) {
	// Wrap result with deprecation metadata
	deprecatedResult := map[string]interface{}{
		"deprecated": true,
		"migration_info": map[string]interface{}{
			"sunset_date": "2024-10-03T00:00:00Z",
			"replacement_methods": map[string]string{
				"a2a.createTask":  "message/send",
				"a2a.getTask":     "tasks/get", 
				"a2a.cancelTask":  "tasks/cancel",
				"a2a.sendMessage": "message/send",
			},
		},
		"result": result,
	}
	
	response := &JSONRPCResponse{
		Jsonrpc: "2.0",
		Result:  deprecatedResult,
		ID:      req.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (g *Gateway) sendError(w http.ResponseWriter, req *JSONRPCRequest, code int, message string) {
	g.sendA2AError(w, req, NewA2AError(code, message, nil))
}

// Metrics middleware to track requests
func (g *Gateway) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Record request in flight
		ctx := r.Context()
		if metrics != nil {
			metrics.RequestsInFlight.Add(ctx, 1)
		}
		
		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
		
		// Process request
		next.ServeHTTP(wrapped, r)
		
		// Record metrics
		duration := time.Since(start)
		RecordRequest(ctx, r.Method, wrapped.statusCode, duration)
		
		if metrics != nil {
			metrics.RequestsInFlight.Add(ctx, -1)
		}
	})
}

// Response writer wrapper to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (g *Gateway) sendA2AError(w http.ResponseWriter, req *JSONRPCRequest, a2aError *A2AError) {
	response := &JSONRPCResponse{
		Jsonrpc: "2.0",
		Error: &RPCError{
			Code:    a2aError.Code,
			Message: a2aError.Message,
			Data:    a2aError.Data,
		},
		ID: req.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC errors still return 200
	
	// Log error with category for monitoring
	errorInfo := GetErrorInfo(a2aError.Code)
	log.Printf("❌ A2A Error [%s]: %s (%d) - %s", errorInfo.Category, errorInfo.Title, a2aError.Code, a2aError.Message)
	
	// Record error metrics
	ctx := context.Background()
	RecordA2AError(ctx, a2aError.Code, errorInfo.Category)
	
	json.NewEncoder(w).Encode(response)
}

func (g *Gateway) Start() error {
	// Initialize OpenTelemetry
	_, cleanup, err := initTelemetry()
	if err != nil {
		return fmt.Errorf("failed to initialize telemetry: %w", err)
	}
	defer cleanup()

	r := mux.NewRouter()
	
	// Add request metrics middleware
	r.Use(g.metricsMiddleware)
	
	r.HandleFunc("/health", g.handleHealth).Methods("GET")
	r.HandleFunc("/errors", g.handleErrorCodes).Methods("GET")
	r.HandleFunc("/a2a", g.handleTasks).Methods("POST")
	r.Handle("/metrics", CreateMetricsHandler()).Methods("GET")
	
	// Google A2A SDK Compatibility Routes
	r.HandleFunc("/agents/{agentId}/.well-known/agent.json", g.handleAgentCard).Methods("GET")
	r.HandleFunc("/agents/{agentId}/a2a", g.handleAgentProxy).Methods("POST")

	// Start Redis cleanup scheduler
	g.startRedisCleanupScheduler()

	log.Printf("🚀 A2A Gateway (Go) listening on port %s", g.port)
	log.Printf("📋 Phase 1 Week 2: Temporal integration ready")
	log.Printf("🤖 Available agents: %v", getAgentIDs())
	log.Printf("📊 Metrics available at /metrics")

	return http.ListenAndServe(":"+g.port, r)
}

func getAgentIDs() []string {
	var ids []string
	for id := range agentTaskQueues {
		ids = append(ids, id)
	}
	return ids
}

// getA2ATask returns the embedded A2A task from StoredTask
func (g *Gateway) getA2ATask(stored *StoredTask) *A2ATask {
	// Return A2A specification v0.2.5 compliant task
	return &stored.A2ATask
}

// normalizeTaskInput converts various input formats to A2A message format
func (g *Gateway) normalizeTaskInput(input interface{}) interface{} {
	if input == nil {
		return map[string]interface{}{
			"messages": []interface{}{},
		}
	}
	
	// If input is already a map with messages, return as-is
	if inputMap, ok := input.(map[string]interface{}); ok {
		if _, hasMessages := inputMap["messages"]; hasMessages {
			return input
		}
		
		// If input has "message" field, convert to A2A format
		if message, hasMessage := inputMap["message"]; hasMessage {
			if messageStr, ok := message.(string); ok && messageStr != "" {
				return map[string]interface{}{
					"messages": []interface{}{
						map[string]interface{}{
							"role": "user",
							"parts": []interface{}{
								map[string]interface{}{
									"type":    "text",
									"content": messageStr,
								},
							},
							"timestamp": newISO8601Timestamp(),
						},
					},
				}
			}
		}
		
		// If input has "text" field, convert to A2A format
		if text, hasText := inputMap["text"]; hasText {
			if textStr, ok := text.(string); ok && textStr != "" {
				return map[string]interface{}{
					"messages": []interface{}{
						map[string]interface{}{
							"role": "user",
							"parts": []interface{}{
								map[string]interface{}{
									"type":    "text",
									"content": textStr,
								},
							},
							"timestamp": newISO8601Timestamp(),
						},
					},
				}
			}
		}
	}
	
	// If input is a string, convert to A2A format
	if inputStr, ok := input.(string); ok && inputStr != "" {
		return map[string]interface{}{
			"messages": []interface{}{
				map[string]interface{}{
					"role": "user",
					"parts": []interface{}{
						map[string]interface{}{
							"type":    "text",
							"content": inputStr,
						},
					},
					"timestamp": newISO8601Timestamp(),
				},
			},
		}
	}
	
	// Default: return empty messages
	return map[string]interface{}{
		"messages": []interface{}{},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Google A2A SDK Compatibility Handlers

func (g *Gateway) handleAgentCard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentId := vars["agentId"]
	
	// Get agent info from registry
	respBody, err := g.callAgentRegistry("GET", fmt.Sprintf("/agents/%s", agentId), nil)
	if err != nil {
		log.Printf("⚠️ Failed to get agent from registry: %v", err)
		http.NotFound(w, r)
		return
	}
	
	// Parse registry response to AgentCard
	var agentCard AgentCard
	if err := json.Unmarshal(respBody, &agentCard); err != nil {
		log.Printf("⚠️ Failed to parse agent card: %v", err)
		http.Error(w, "Invalid agent card format", http.StatusInternalServerError)
		return
	}
	
	// Update URL to point to our proxy endpoint
	agentCard.URL = fmt.Sprintf("http://localhost:8080/agents/%s", agentId)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agentCard)
}

func (g *Gateway) handleAgentProxy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentId := vars["agentId"]
	
	// Parse incoming A2A request
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.sendError(w, &req, ErrorInvalidRequest, "Invalid JSON-RPC request")
		return
	}
	
	// Handle A2A specification compliant methods
	switch req.Method {
	case "message/send":
		// A2A spec compliant method - proper message/send handler
		g.handleMessageSend(w, &req, agentId)
	case "tasks/get":
		// A2A spec compliant method - pass through to existing handler
		g.handleGetTask(w, &req)
	case "tasks/cancel":
		// A2A spec compliant method - pass through to existing handler
		g.handleCancelTask(w, &req)
	// Keep backward compatibility with old method names
	case "a2a.sendMessage":
		addDeprecationWarnings(w, "a2a.sendMessage")
		g.handleSendMessageProxy(w, &req, agentId)
	case "a2a.getTask":
		addDeprecationWarnings(w, "a2a.getTask")
		g.handleGetTask(w, &req)
	case "a2a.cancelTask":
		addDeprecationWarnings(w, "a2a.cancelTask")
		g.handleCancelTask(w, &req)
	default:
		g.sendError(w, &req, ErrorMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (g *Gateway) handleSendMessageProxy(w http.ResponseWriter, req *JSONRPCRequest, agentId string) {
	// Parse sendMessage params
	paramBytes, err := json.Marshal(req.Params)
	if err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Failed to marshal request params")
		return
	}
	
	var messageParams map[string]interface{}
	if err := json.Unmarshal(paramBytes, &messageParams); err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Invalid params")
		return
	}
	
	// Convert to createTask format
	createTaskParams := CreateTaskParams{
		AgentID: agentId,
		Input:   messageParams,
		Metadata: map[string]interface{}{
			"source":    "google-a2a-sdk",
			"method":    "sendMessage",
			"timestamp": time.Now().Unix(),
		},
	}
	
	// Update request to createTask format
	newReq := JSONRPCRequest{
		Jsonrpc: req.Jsonrpc,
		Method:  "a2a.createTask",
		Params:  createTaskParams,
		ID:      req.ID,
	}
	
	// Call existing createTask handler
	g.handleCreateTask(w, &newReq)
}

// A2AMessageSendParams represents A2A specification message/send parameters (without agentId)
type A2AMessageSendParams struct {
	Message       interface{}            `json:"message"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

func (g *Gateway) handleMessageSend(w http.ResponseWriter, req *JSONRPCRequest, agentId string) {
	log.Printf("📨 A2A message/send for agent %s", agentId)
	
	// Parse message/send params according to A2A specification
	paramBytes, err := json.Marshal(req.Params)
	if err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Failed to marshal request params")
		return
	}
	
	var params A2AMessageSendParams
	if err := json.Unmarshal(paramBytes, &params); err != nil {
		g.sendError(w, req, ErrorInvalidParams, "Invalid message/send params")
		return
	}
	
	// Validate required message field
	if params.Message == nil {
		g.sendError(w, req, ErrorInvalidParams, "Missing required parameter: message")
		return
	}
	
	// Generate task ID and context ID for A2A compliance
	taskID := uuid.New().String()
	contextID := taskID // Use same ID for context as per A2A examples
	
	// Create task input for Temporal workflow
	taskInput := map[string]interface{}{
		"message":   params.Message,
		"metadata":  params.Metadata,
		"agentId":   agentId,
		"contextId": contextID,
	}
	
	log.Printf("🔄 Creating task %s for agent %s", taskID, agentId)
	
	// Get workflow and task queue for agent from configuration
	workflowType, ok := agentWorkflows[agentId]
	if !ok {
		log.Printf("❌ Unknown agent: %s", agentId)
		g.sendError(w, req, ErrorAgentNotFound, fmt.Sprintf("Unknown agent: %s", agentId))
		return
	}

	taskQueue, ok := agentTaskQueues[agentId]
	if !ok {
		log.Printf("❌ No task queue configured for agent: %s", agentId)
		g.sendError(w, req, ErrorAgentNotFound, fmt.Sprintf("No task queue configured for agent: %s", agentId))
		return
	}

	// Start Temporal workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        taskID,
		TaskQueue: taskQueue,
	}
	
	we, err := g.temporalClient.ExecuteWorkflow(context.Background(), workflowOptions, workflowType, taskInput)
	if err != nil {
		log.Printf("❌ Failed to start workflow: %v", err)
		g.sendError(w, req, ErrorInternalError, "Failed to start task processing")
		return
	}
	
	log.Printf("✅ Started workflow %s (%s) on queue %s", taskID, workflowType, taskQueue)
	
	// Store task in Redis using the correct A2A-compliant format
	ctx := context.Background()
	taskKey := fmt.Sprintf("task:%s", taskID)
	currentTime := newISO8601Timestamp()
	
	// Create A2A v0.2.5 compliant TaskStatus
	taskStatus := TaskStatus{
		State:     "submitted",   // A2A spec compliant: initial 'submitted' state before execution
		Timestamp: currentTime,   // A2A spec compliant: use 'timestamp'
	}
	
	// Create A2A-compliant StoredTask
	storedTask := &StoredTask{
		A2ATask: A2ATask{
			ID:        taskID,
			ContextID: contextID,
			Status:    taskStatus,
			Kind:      "task",           // A2A v0.2.5 required field
			AgentID:   agentId,
			Input:     taskInput,
			Metadata:  params.Metadata,
			CreatedAt: currentTime,
		},
		WorkflowID: we.GetID(),
	}
	
	// Store in Redis using hash format (original working approach)
	taskData := map[string]interface{}{
		"id":         taskID,
		"contextId":  contextID,
		"status":     "submitted", // Initial submitted state before execution
		"agentId":    agentId,
		"createdAt":  currentTime,
		"updatedAt":  currentTime,
		"workflowId": we.GetID(),
	}
	
	// Store input and metadata as JSON strings
	if inputJSON, err := json.Marshal(taskInput); err == nil {
		taskData["input"] = string(inputJSON)
	}
	if metadataJSON, err := json.Marshal(map[string]interface{}{
		"source": "a2a-gateway",
		"method": "message/send",
	}); err == nil {
		taskData["metadata"] = string(metadataJSON)
	}
	
	// Use Redis pipeline for atomic operations
	pipe := g.redisClient.Pipeline()
	pipe.HSet(ctx, taskKey, taskData)
	
	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("❌ Failed to store task in Redis: %v", err)
		g.sendError(w, req, ErrorInternalError, "Failed to store task")
		return
	}
	
	// Transition task from "submitted" to "working" now that workflow has started
	err = g.updateTaskStatusInRedis(taskID, "working", nil, "")
	if err != nil {
		log.Printf("⚠️ Failed to update task status to working: %v", err)
		// Continue - task will still be tracked via Temporal
	}
	
	log.Printf("✅ Task %s stored successfully", taskID)
	
	// Start monitoring workflow to update task status when complete
	go g.monitorWorkflow(taskID, we)
	
	// Return A2A-compliant task response
	response := &JSONRPCResponse{
		Jsonrpc: "2.0",
		Result:  &storedTask.A2ATask,
		ID:      req.ID,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	gateway, err := NewGateway()
	if err != nil {
		log.Fatalf("❌ Failed to initialize gateway: %v", err)
	}

	log.Fatal(gateway.Start())
}