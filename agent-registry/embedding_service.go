package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// EmbeddingService interface for different embedding providers
type EmbeddingService interface {
	CreateEmbedding(ctx context.Context, text string) ([]float32, error)
	GetDimensions() int
	GetProviderName() string
}

// EmbeddingConfig holds configuration for embedding services
type EmbeddingConfig struct {
	Provider    string `json:"provider"`    // "openai", "local", "huggingface"
	Dimensions  int    `json:"dimensions"`  // 384, 1536, etc.
	ModelName   string `json:"model_name"`  // "text-embedding-ada-002", "all-MiniLM-L6-v2"
	APIKey      string `json:"api_key"`     // For external APIs
	ServiceURL  string `json:"service_url"` // For local microservice
	Timeout     int    `json:"timeout"`     // Request timeout in seconds
}

// OpenAIEmbeddingService implements EmbeddingService for OpenAI
type OpenAIEmbeddingService struct {
	apiKey     string
	model      string
	dimensions int
	httpClient *http.Client
}

// LocalEmbeddingService implements EmbeddingService for local microservice
type LocalEmbeddingService struct {
	serviceURL string
	model      string
	dimensions int
	httpClient *http.Client
}

// Removed FallbackEmbeddingService - only OpenAI and local microservice supported

// OpenAI API types
type OpenAIEmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type OpenAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// Local microservice API types
type LocalEmbeddingRequest struct {
	Text  string `json:"text"`
	Model string `json:"model"`
}

type LocalEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
	Model     string    `json:"model"`
	Provider  string    `json:"provider"`
}

// FallbackEmbeddingService implements simple hash-based embeddings for testing
type FallbackEmbeddingService struct {
	dimensions int
}

func NewFallbackEmbeddingService(dimensions int) *FallbackEmbeddingService {
	if dimensions <= 0 {
		dimensions = 384 // Default to reasonable size
	}
	return &FallbackEmbeddingService{
		dimensions: dimensions,
	}
}

// NewEmbeddingService creates an embedding service based on configuration
func NewEmbeddingService() (EmbeddingService, error) {
	config := getEmbeddingConfig()
	
	switch config.Provider {
	case "openai":
		return NewOpenAIEmbeddingService(config)
	case "local":
		return NewLocalEmbeddingService(config)
	default:
		// Fail fast - require explicit configuration
		return nil, fmt.Errorf("invalid embedding provider '%s'. Must be 'openai' or 'local'", config.Provider)
	}
}

func getEmbeddingConfig() EmbeddingConfig {
	provider := os.Getenv("EMBEDDING_PROVIDER")
	if provider == "" {
		provider = "openai"
	}
	
	dimensionsStr := os.Getenv("EMBEDDING_DIMENSIONS")
	if dimensionsStr == "" {
		dimensionsStr = "1536"
	}
	dimensions, _ := strconv.Atoi(dimensionsStr)
	
	timeoutStr := os.Getenv("EMBEDDING_TIMEOUT")
	if timeoutStr == "" {
		timeoutStr = "30"
	}
	timeout, _ := strconv.Atoi(timeoutStr)

	modelName := os.Getenv("EMBEDDING_MODEL")
	if modelName == "" {
		modelName = "text-embedding-ada-002"
	}
	
	serviceURL := os.Getenv("EMBEDDING_SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://embedding-service:8080"
	}

	return EmbeddingConfig{
		Provider:   provider,
		Dimensions: dimensions,
		ModelName:  modelName,
		APIKey:     os.Getenv("OPENAI_API_KEY"),
		ServiceURL: serviceURL,
		Timeout:    timeout,
	}
}

// NewOpenAIEmbeddingService creates OpenAI embedding service
func NewOpenAIEmbeddingService(config EmbeddingConfig) (*OpenAIEmbeddingService, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable required for OpenAI embedding service")
	}
	
	if config.ModelName == "" {
		return nil, fmt.Errorf("EMBEDDING_MODEL environment variable required for OpenAI embedding service")
	}

	return &OpenAIEmbeddingService{
		apiKey:     config.APIKey,
		model:      config.ModelName,
		dimensions: config.Dimensions,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}, nil
}

func (s *OpenAIEmbeddingService) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Truncate text if too long (OpenAI has token limits)
	if len(text) > 8000 {
		text = text[:8000]
	}

	reqBody := OpenAIEmbeddingRequest{
		Input: text,
		Model: s.model,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	var response OpenAIEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}

	embedding := response.Data[0].Embedding
	
	// Log usage for monitoring
	fmt.Printf("📊 OpenAI Embedding: %d tokens, model: %s\n", response.Usage.TotalTokens, response.Model)
	
	return embedding, nil
}

func (s *OpenAIEmbeddingService) GetDimensions() int {
	return s.dimensions
}

func (s *OpenAIEmbeddingService) GetProviderName() string {
	return fmt.Sprintf("openai:%s", s.model)
}

// NewLocalEmbeddingService creates local microservice embedding service
func NewLocalEmbeddingService(config EmbeddingConfig) (*LocalEmbeddingService, error) {
	return &LocalEmbeddingService{
		serviceURL: config.ServiceURL,
		model:      config.ModelName,
		dimensions: config.Dimensions,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}, nil
}

func (s *LocalEmbeddingService) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	reqBody := LocalEmbeddingRequest{
		Text:  text,
		Model: s.model,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.serviceURL+"/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call local embedding service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("local embedding service error (status %d): %s", resp.StatusCode, string(body))
	}

	var response LocalEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Printf("📊 Local Embedding: model: %s, provider: %s\n", response.Model, response.Provider)
	
	return response.Embedding, nil
}

func (s *LocalEmbeddingService) GetDimensions() int {
	return s.dimensions
}

func (s *LocalEmbeddingService) GetProviderName() string {
	return fmt.Sprintf("local:%s", s.model)
}

// Removed all fallback embedding code - only OpenAI and local microservice supported