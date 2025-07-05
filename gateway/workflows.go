package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// ProgressUpdate represents a progress signal from agent workflow
type ProgressUpdate struct {
	TaskID    string                 `json:"taskId"`
	Status    string                 `json:"status"`
	Progress  float64                `json:"progress"`
	Artifact  map[string]interface{} `json:"artifact,omitempty"`
	Append    bool                   `json:"append"`
	LastChunk bool                   `json:"lastChunk"`
	Timestamp string                 `json:"timestamp"`
	Error     string                 `json:"error,omitempty"`
}

// GatewayStreamingWorkflow receives signals from agent workflows and streams to client
func GatewayStreamingWorkflow(ctx workflow.Context, streamID string, sseChannelID string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting gateway streaming workflow", "streamID", streamID)

	// Channel to receive progress signals from agent workflow
	progressChannel := workflow.GetSignalChannel(ctx, "progress_update")

	// Track streaming state
	lastSentPosition := 0
	artifactID := ""

	// Process signals until workflow is cancelled
	for {
		var update ProgressUpdate
		more := progressChannel.ReceiveAsync(&update)
		if !more {
			// No signal available, continue
			workflow.Sleep(ctx, 50*time.Millisecond)
			continue
		}

		logger.Info("Received progress update", 
			"taskId", update.TaskID, 
			"status", update.Status)

		// Convert to A2A streaming events
		events := convertProgressToStreamingEvents(update, &lastSentPosition, &artifactID)

		// Send events via activity
		activityOptions := workflow.ActivityOptions{
			StartToCloseTimeout: 5 * time.Second,
		}
		activityCtx := workflow.WithActivityOptions(ctx, activityOptions)

		for _, event := range events {
			err := workflow.ExecuteActivity(activityCtx, PushEventToSSE, sseChannelID, event).Get(ctx, nil)
			if err != nil {
				logger.Error("Failed to push event to SSE", "error", err)
			}
		}

		// Check if streaming is complete
		if update.Status == "completed" || update.Status == "failed" {
			logger.Info("Streaming complete", "status", update.Status)
			return nil
		}
	}
}

// convertProgressToStreamingEvents converts progress updates to A2A streaming events
func convertProgressToStreamingEvents(update ProgressUpdate, lastSentPosition *int, artifactID *string) []interface{} {
	events := []interface{}{}
	contextID := fmt.Sprintf("ctx-%s", update.TaskID[:8])

	// Always send status update
	events = append(events, TaskStatusUpdateEvent{
		TaskID:    update.TaskID,
		ContextID: contextID,
		Kind:      "status-update",
		Status: map[string]interface{}{
			"state":     update.Status,
			"timestamp": update.Timestamp,
		},
		Final: update.Status == "completed" || update.Status == "failed",
	})

	// Handle artifact updates for progressive streaming
	if update.Artifact != nil {
		// Extract artifact content for incremental sending
		if parts, ok := update.Artifact["parts"].([]interface{}); ok && len(parts) > 0 {
			if part, ok := parts[0].(map[string]interface{}); ok {
				if text, ok := part["text"].(string); ok {
					// For progressive streaming, track position
					contentLen := len(text)
					if contentLen > *lastSentPosition {
						// Extract only new content
						newContent := text[*lastSentPosition:]
						
						// Set artifact ID if first chunk
						if *artifactID == "" {
							// Use artifact ID from the update if provided, otherwise generate one
							if providedID, ok := update.Artifact["artifactId"].(string); ok && providedID != "" {
								*artifactID = providedID
							} else {
								*artifactID = fmt.Sprintf("artifact-%s", update.TaskID[:8])
							}
						}

						artifactEvent := TaskArtifactUpdateEvent{
							TaskID:    update.TaskID,
							ContextID: contextID,
							Kind:      "artifact-update",
							Artifact: map[string]interface{}{
								"artifactId": *artifactID,
								"name":       "Echo Response",
								"parts": []map[string]interface{}{
									{
										"kind": "text",
										"text": newContent,
									},
								},
							},
							Append:    *lastSentPosition > 0,
							LastChunk: update.LastChunk,
							Timestamp: update.Timestamp,
						}
						events = append(events, artifactEvent)
						*lastSentPosition = contentLen
					}
				}
			}
		}
	}

	return events
}

// PushEventToSSE activity pushes events to SSE stream
func PushEventToSSE(ctx context.Context, channelID string, event interface{}) error {
	// Get the gateway instance from activity context
	// In production, this would use a proper service locator
	logger := activity.GetLogger(ctx)
	
	// Marshal event to JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	logger.Info("Pushing event to SSE", 
		"channelID", channelID, 
		"event", string(eventData))

	// In real implementation, this would push to the SSE channel
	// For now, we'll store in a global channel map
	gatewayInstance.sseChannelsMutex.RLock()
	if ch, exists := gatewayInstance.sseChannels[channelID]; exists {
		select {
		case ch <- event:
			logger.Info("Event pushed to SSE channel")
		default:
			logger.Warn("SSE channel buffer full")
		}
	}
	gatewayInstance.sseChannelsMutex.RUnlock()

	return nil
}

// Global gateway instance for activities
var gatewayInstance *Gateway

// SetGatewayInstance sets the global gateway instance for activities
func SetGatewayInstance(g *Gateway) {
	gatewayInstance = g
}