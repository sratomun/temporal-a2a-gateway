package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/qdrant/go-client/qdrant"
)

// Agent Registry Service with Qdrant
// Provides semantic agent discovery backed by Qdrant vector database

// A2A Protocol v0.2.5 compliant ISO 8601 timestamp generator
// Returns current time in UTC with millisecond precision: 2024-07-03T14:30:00.000Z
func newISO8601Timestamp() string {
	return time.Now().UTC().Format(time.RFC3339Nano)[:23] + "Z"
}

type AgentRegistry struct {
	qdrantClient     *qdrant.Client
	embeddingService EmbeddingService
	port             string
	collectionName   string
}

// Google SDK Compatible AgentCard
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

// Google SDK Compatible AgentSkill
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

// Request/Response Types
type RegisterAgentRequest struct {
	AgentCard AgentCard `json:"agentCard"`
}

type RegisterAgentResponse struct {
	AgentID string `json:"agentId"`
}

type DiscoverAgentsRequest struct {
	Capability string `json:"capability,omitempty"`
	Keyword    string `json:"keyword,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type DiscoverAgentsResponse struct {
	Agents []AgentInfo `json:"agents"`
}

type AgentInfo struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Version     string       `json:"version"`
	URL         string       `json:"url"`
	Skills      []AgentSkill `json:"skills"`
	Score       float32      `json:"score,omitempty"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
	Qdrant    string `json:"qdrant"`
}

func NewAgentRegistry() (*AgentRegistry, error) {
	registryPort := getEnv("PORT", "8001")
	qdrantURL := getEnv("QDRANT_URL", "http://localhost:6333")
	collectionName := "agent_registry"

	// Initialize embedding service
	embeddingService, err := NewEmbeddingService()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embedding service: %v", err)
	}
	
	log.Printf("🧠 Embedding service initialized: %s", embeddingService.GetProviderName())

	// Parse Qdrant URL to get host and port
	var host string
	var qdrantPort int
	if strings.HasPrefix(qdrantURL, "http://") {
		hostPort := strings.TrimPrefix(qdrantURL, "http://")
		parts := strings.Split(hostPort, ":")
		host = parts[0]
		if len(parts) > 1 {
			if p, err := strconv.Atoi(parts[1]); err == nil {
				qdrantPort = p
			} else {
				qdrantPort = 6334
			}
		} else {
			qdrantPort = 6334
		}
	} else {
		host = "localhost"
		qdrantPort = 6334
	}

	log.Printf("Connecting to Qdrant at %s:%d", host, qdrantPort)
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: qdrantPort,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %v", err)
	}

	registry := &AgentRegistry{
		qdrantClient:     client,
		embeddingService: embeddingService,
		port:             registryPort,
		collectionName:   collectionName,
	}

	// Initialize collection and seed data
	err = registry.initializeQdrant()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Qdrant: %v", err)
	}

	return registry, nil
}

func (ar *AgentRegistry) initializeQdrant() error {
	ctx := context.Background()

	// Check if collection exists
	collections, err := ar.qdrantClient.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list collections: %v", err)
	}

	collectionExists := false
	for _, collectionName := range collections {
		if collectionName == ar.collectionName {
			collectionExists = true
			break
		}
	}

	if !collectionExists {
		// Create collection with dimensions matching embedding service
		dimensions := ar.embeddingService.GetDimensions()
		err = ar.qdrantClient.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: ar.collectionName,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     uint64(dimensions),
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			return fmt.Errorf("failed to create collection: %v", err)
		}
		log.Printf("✅ Created Qdrant collection: %s (%d dimensions)", ar.collectionName, dimensions)
	}

	// Seed with built-in agents
	err = ar.seedBuiltInAgents()
	if err != nil {
		log.Printf("⚠️ Failed to seed built-in agents: %v", err)
	}

	log.Printf("✅ Qdrant initialized successfully")
	return nil
}

func (ar *AgentRegistry) seedBuiltInAgents() error {
	// Helper function to create Google SDK compatible capabilities
	createSDKCapabilities := func() map[string]interface{} {
		return map[string]interface{}{
			"streaming":              false,
			"pushNotifications":      false,
			"stateTransitionHistory": true,
			"extensions":             nil,
		}
	}

	// Seed Echo Agent - Google SDK Compatible
	echoAgent := AgentCard{
		Name:        "Echo Agent",
		Description: "Simple echo agent for testing A2A protocol functionality",
		Version:     "1.0.0",
		URL:         "http://a2a-gateway:3000",
		Capabilities: createSDKCapabilities(),
		Skills: []AgentSkill{
			{
				ID:          "echo",
				Name:        "Echo Messages",
				Description: "Echoes back user messages with a prefix to test A2A communication",
				InputModes:  []string{"text", "json"},
				OutputModes: []string{"text", "json"},
				Tags:        []string{"testing", "echo", "communication"},
				Examples: []string{
					"Input: Hello World -> Output: Echo: Hello World",
					"Input: Test message -> Output: Echo: Test message",
				},
			},
		},
		DefaultInputModes:  []string{"text", "json"},
		DefaultOutputModes: []string{"text", "json"},
		Security: []map[string][]string{
			{"apiKey": []string{}},
		},
		SecuritySchemes: map[string]interface{}{
			"apiKey": map[string]interface{}{
				"type": "apiKey",
				"name": "X-API-Key",
				"in":   "header",
			},
		},
	}

	// Python Security Expert Agent - Google SDK Compatible
	pythonSecurityAgent := AgentCard{
		Name:        "Python Security Expert",
		Description: "Specialist in Python security analysis and vulnerability detection",
		Version:     "2.1.0",
		URL:         "http://a2a-gateway:3000",
		Capabilities: createSDKCapabilities(),
		Skills: []AgentSkill{
			{
				ID:          "vulnerability_scan",
				Name:        "Vulnerability Scanner",
				Description: "Scans Python code for security vulnerabilities",
				InputModes:  []string{"text", "json"},
				OutputModes: []string{"text", "json"},
				Tags:        []string{"security", "python", "vulnerability"},
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"code": map[string]interface{}{
							"type":        "string",
							"description": "Python code to analyze",
						},
						"severity": map[string]interface{}{
							"type": "string",
							"enum": []string{"low", "medium", "high", "critical"},
						},
					},
					"required": []string{"code"},
				},
			},
			{
				ID:          "secure_code_review",
				Name:        "Secure Code Review",
				Description: "Provides security recommendations for Python code",
				InputModes:  []string{"text", "json"},
				OutputModes: []string{"text", "json", "markdown"},
				Tags:        []string{"security", "python", "review"},
			},
		},
		DefaultInputModes:  []string{"text", "json"},
		DefaultOutputModes: []string{"text", "json", "markdown"},
		Security: []map[string][]string{
			{"oauth2": []string{"read", "write"}},
		},
		SecuritySchemes: map[string]interface{}{
			"oauth2": map[string]interface{}{
				"type": "oauth2",
				"flows": map[string]interface{}{
					"clientCredentials": map[string]interface{}{
						"tokenUrl": "https://auth.example.com/oauth/token",
						"scopes": map[string]string{
							"read":  "Read agent capabilities",
							"write": "Execute agent tasks",
						},
					},
				},
			},
		},
	}

	// Register both agents
	err := ar.addAgentToQdrant("echo-agent", echoAgent)
	if err != nil {
		return err
	}
	
	return ar.addAgentToQdrant("python-security-expert", pythonSecurityAgent)
}

func (ar *AgentRegistry) addAgentToQdrant(agentID string, card AgentCard) error {
	ctx := context.Background()

	// Create text content for embedding
	content := fmt.Sprintf("%s %s", card.Name, card.Description)
	skillNames := make([]string, len(card.Skills))
	for i, skill := range card.Skills {
		content += fmt.Sprintf(" %s %s", skill.Name, skill.Description)
		skillNames[i] = skill.Name
	}

	// Create embedding using configured embedding service
	vector, embeddingErr := ar.embeddingService.CreateEmbedding(ctx, content)
	if embeddingErr != nil {
		return fmt.Errorf("failed to create embedding: %v", embeddingErr)
	}

	// Prepare payload with agent data
	fullCardJSON, _ := json.Marshal(card)
	payload := map[string]interface{}{
		"agent_id":       agentID,
		"name":           card.Name,
		"description":    card.Description,
		"version":        card.Version,
		"url":            card.URL,
		"skills":         strings.Join(skillNames, ","),
		"full_card_json": string(fullCardJSON),
		"content":        content,
	}

	// Create point ID from agent ID
	pointID := ar.createPointID(agentID)

	// Upsert point (insert or update)
	points := []*qdrant.PointStruct{
		{
			Id:      qdrant.NewIDNum(pointID),
			Vectors: qdrant.NewVectors(vector...),
			Payload: qdrant.NewValueMap(payload),
		},
	}

	_, err := ar.qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: ar.collectionName,
		Points:         points,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert agent: %v", err)
	}

	log.Printf("✅ Added/updated agent: %s", agentID)
	return nil
}

func (ar *AgentRegistry) queryAgentsFromQdrant(keyword, capability string, limit int) ([]AgentInfo, error) {
	ctx := context.Background()

	// Build search query
	searchQuery := ""
	if keyword != "" {
		searchQuery += keyword + " "
	}
	if capability != "" {
		searchQuery += capability + " "
	}
	if searchQuery == "" {
		searchQuery = "agent capabilities" // Default broad query
	}

	// Create embedding for search query
	queryVector, embeddingErr := ar.embeddingService.CreateEmbedding(ctx, strings.TrimSpace(searchQuery))
	if embeddingErr != nil {
		return nil, fmt.Errorf("failed to create query embedding: %v", embeddingErr)
	}

	// Search similar vectors  
	searchResult, err := ar.qdrantClient.Query(ctx, &qdrant.QueryPoints{
		CollectionName: ar.collectionName,
		Query:          qdrant.NewQuery(queryVector...),
		Limit:          qdrant.PtrOf(uint64(limit)),
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search agents: %v", err)
	}

	// Convert results to AgentInfo
	var agents []AgentInfo
	for _, point := range searchResult {
		if point.Payload == nil {
			continue
		}

		agent := AgentInfo{
			ID:          getStringFromPayload(point.Payload, "agent_id"),
			Name:        getStringFromPayload(point.Payload, "name"),
			Description: getStringFromPayload(point.Payload, "description"),
			Version:     getStringFromPayload(point.Payload, "version"),
			URL:         getStringFromPayload(point.Payload, "url"),
			Score:       point.Score,
		}

		// Parse full card from JSON to get skills
		if fullCardJSON := getStringFromPayload(point.Payload, "full_card_json"); fullCardJSON != "" {
			var fullCard AgentCard
			if err := json.Unmarshal([]byte(fullCardJSON), &fullCard); err == nil {
				agent.Skills = fullCard.Skills
			}
		}

		agents = append(agents, agent)
	}

	return agents, nil
}

// createSimpleEmbedding is deprecated - use embeddingService.CreateEmbedding instead
// Kept for backward compatibility only
func (ar *AgentRegistry) createSimpleEmbedding(text string) []float32 {
	// Fallback to embedding service
	ctx := context.Background()
	vector, embeddingErr := ar.embeddingService.CreateEmbedding(ctx, text)
	if embeddingErr != nil {
		// Last resort: create simple hash-based embedding
		hash := md5.Sum([]byte(text))
		fallbackVector := make([]float32, ar.embeddingService.GetDimensions())
		
		for i := 0; i < len(fallbackVector); i++ {
			byteIndex := i % 16
			fallbackVector[i] = (float32(hash[byteIndex]) - 128.0) / 128.0
		}
		return fallbackVector
	}
	return vector
}

func (ar *AgentRegistry) createPointID(agentID string) uint64 {
	hash := md5.Sum([]byte(agentID))
	// Convert first 8 bytes to uint64
	var id uint64
	for i := 0; i < 8; i++ {
		id = (id << 8) | uint64(hash[i])
	}
	return id
}

func getStringFromPayload(payload map[string]*qdrant.Value, key string) string {
	if val, ok := payload[key]; ok && val.GetStringValue() != "" {
		return val.GetStringValue()
	}
	return ""
}

func (ar *AgentRegistry) getAgentPayload(agentID string) map[string]*qdrant.Value {
	ctx := context.Background()
	pointID := ar.createPointID(agentID)
	
	result, err := ar.qdrantClient.Get(ctx, &qdrant.GetPoints{
		CollectionName: ar.collectionName,
		Ids:            []*qdrant.PointId{qdrant.NewIDNum(pointID)},
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil || len(result) == 0 {
		return make(map[string]*qdrant.Value)
	}
	
	return result[0].Payload
}

func (ar *AgentRegistry) healthCheck() error {
	ctx := context.Background()
	_, err := ar.qdrantClient.HealthCheck(ctx)
	return err
}

// HTTP Handlers
func (ar *AgentRegistry) handleRegisterAgent(w http.ResponseWriter, r *http.Request) {
	var req RegisterAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.AgentCard.Name) == "" {
		log.Printf("❌ Rejecting agent registration: empty name")
		http.Error(w, "Agent name is required and cannot be empty", http.StatusBadRequest)
		return
	}
	
	if strings.TrimSpace(req.AgentCard.Description) == "" {
		log.Printf("❌ Rejecting agent registration: empty description for agent '%s'", req.AgentCard.Name)
		http.Error(w, "Agent description is required and cannot be empty", http.StatusBadRequest)
		return
	}

	// Generate agent ID
	agentID := fmt.Sprintf("%s-%s", strings.ToLower(strings.ReplaceAll(req.AgentCard.Name, " ", "-")), uuid.New().String()[:8])

	log.Printf("🤖 Registering agent: %s (%s)", req.AgentCard.Name, agentID)

	// Store in Qdrant
	err := ar.addAgentToQdrant(agentID, req.AgentCard)
	if err != nil {
		log.Printf("❌ Failed to register agent: %v", err)
		http.Error(w, fmt.Sprintf("Failed to register agent: %v", err), http.StatusInternalServerError)
		return
	}

	resp := RegisterAgentResponse{
		AgentID: agentID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
	log.Printf("✅ Agent registered: %s", agentID)
}

func (ar *AgentRegistry) handleDiscoverAgents(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	capability := r.URL.Query().Get("capability")
	keyword := r.URL.Query().Get("keyword")
	limitStr := r.URL.Query().Get("limit")
	minScoreStr := r.URL.Query().Get("min_score")

	limit := 10 // Default limit
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	minScore := 0.8 // Default minimum score threshold (80% similarity)
	if minScoreStr != "" {
		if parsed, err := strconv.ParseFloat(minScoreStr, 32); err == nil && parsed >= 0.0 && parsed <= 1.0 {
			minScore = parsed
		}
	}

	log.Printf("🔍 Discovering agents (capability: %s, keyword: %s, limit: %d, min_score: %.2f)", capability, keyword, limit, minScore)

	// Query Qdrant directly and return full AgentCards
	ctx := context.Background()

	// Build search query
	searchQuery := ""
	if keyword != "" {
		searchQuery += keyword + " "
	}
	if capability != "" {
		searchQuery += capability + " "
	}
	if searchQuery == "" {
		searchQuery = "agent capabilities" // Default broad query
	}

	// Create embedding for search query
	queryVector, embeddingErr := ar.embeddingService.CreateEmbedding(ctx, strings.TrimSpace(searchQuery))
	if embeddingErr != nil {
		log.Printf("❌ Failed to create query embedding: %v", embeddingErr)
		http.Error(w, fmt.Sprintf("Failed to discover agents: %v", embeddingErr), http.StatusInternalServerError)
		return
	}

	// Search similar vectors  
	searchResult, err := ar.qdrantClient.Query(ctx, &qdrant.QueryPoints{
		CollectionName: ar.collectionName,
		Query:          qdrant.NewQuery(queryVector...),
		Limit:          qdrant.PtrOf(uint64(limit)),
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		log.Printf("❌ Failed to search agents: %v", err)
		http.Error(w, fmt.Sprintf("Failed to discover agents: %v", err), http.StatusInternalServerError)
		return
	}

	// Return Google SDK compatible AgentCards directly, filtered by score
	sdkAgents := make([]AgentCard, 0, len(searchResult))
	filteredCount := 0
	for _, point := range searchResult {
		if point.Payload == nil {
			continue
		}

		// Apply score-based filtering
		if point.Score < float32(minScore) {
			filteredCount++
			log.Printf("🔽 Filtering out agent with low score: %.3f < %.2f", point.Score, minScore)
			continue
		}

		// Get full card from JSON payload
		if fullCardJSON := getStringFromPayload(point.Payload, "full_card_json"); fullCardJSON != "" {
			var fullCard AgentCard
			if err := json.Unmarshal([]byte(fullCardJSON), &fullCard); err == nil {
				// Filter out agents with empty names or descriptions
				if strings.TrimSpace(fullCard.Name) == "" || strings.TrimSpace(fullCard.Description) == "" {
					log.Printf("🗑️ Filtering out agent with empty name/description")
					continue
				}
				log.Printf("✅ Including agent '%s' with score: %.3f", fullCard.Name, point.Score)
				sdkAgents = append(sdkAgents, fullCard)
				continue
			}
		}
		
		// Skip agents without full card data (old format)
		log.Printf("⚠️ Skipping agent without full card data: %s", getStringFromPayload(point.Payload, "agent_id"))
	}
	
	log.Printf("📊 Search results: %d total, %d filtered out by score, %d returned", len(searchResult), filteredCount, len(sdkAgents))

	resp := map[string]interface{}{
		"agents": sdkAgents,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
	log.Printf("✅ Found %d Google SDK compatible agents", len(sdkAgents))
}

func (ar *AgentRegistry) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentId := vars["agentId"]
	
	if agentId == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}
	
	log.Printf("🔍 Looking up agent: %s", agentId)
	
	// Get agent payload from Qdrant
	payload := ar.getAgentPayload(agentId)
	
	// Check if agent exists
	if len(payload) == 0 {
		log.Printf("❌ Agent not found: %s", agentId)
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}
	
	// Get full card from JSON payload
	fullCardJSON := getStringFromPayload(payload, "full_card_json")
	if fullCardJSON == "" {
		log.Printf("❌ Agent card data not found for: %s", agentId)
		http.Error(w, "Agent card data not found", http.StatusNotFound)
		return
	}
	
	var agentCard AgentCard
	if err := json.Unmarshal([]byte(fullCardJSON), &agentCard); err != nil {
		log.Printf("❌ Failed to parse agent card for %s: %v", agentId, err)
		http.Error(w, "Failed to parse agent data", http.StatusInternalServerError)
		return
	}
	
	log.Printf("✅ Found agent: %s", agentCard.Name)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agentCard)
}

func (ar *AgentRegistry) handleHealth(w http.ResponseWriter, r *http.Request) {
	qdrantStatus := "healthy"
	if err := ar.healthCheck(); err != nil {
		qdrantStatus = fmt.Sprintf("unhealthy: %v", err)
	}

	health := HealthResponse{
		Status:    "healthy",
		Service:   "agent-registry",
		Version:   "1.0.0",
		Timestamp: newISO8601Timestamp(),
		Qdrant:    qdrantStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (ar *AgentRegistry) Start() error {
	r := mux.NewRouter()

	// API endpoints
	r.HandleFunc("/health", ar.handleHealth).Methods("GET")
	r.HandleFunc("/agents/register", ar.handleRegisterAgent).Methods("POST")
	r.HandleFunc("/agents/discover", ar.handleDiscoverAgents).Methods("GET")
	r.HandleFunc("/agents/{agentId}", ar.handleGetAgent).Methods("GET")

	log.Printf("🚀 Agent Registry Service listening on port %s", ar.port)
	log.Printf("📋 Endpoints:")
	log.Printf("   GET  /health")
	log.Printf("   POST /agents/register")
	log.Printf("   GET  /agents/discover")
	log.Printf("   GET  /agents/{agentId}")

	return http.ListenAndServe(":"+ar.port, r)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	registry, err := NewAgentRegistry()
	if err != nil {
		log.Fatalf("❌ Failed to initialize Agent Registry: %v", err)
	}

	if err := registry.Start(); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}