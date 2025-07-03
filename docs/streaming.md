# Streaming API Reference

> **🚀 Sprint 2 Implementation**: Real-time streaming with Server-Sent Events (SSE)
> Based on architectural design: `architecture-assets/design/sprint-2-streaming-architecture.md`

## Overview

The A2A Gateway implements real-time streaming capabilities via Server-Sent Events (SSE) as specified in A2A Protocol v0.2.5.

## message/stream Endpoint

### Request Format

**Method**: `message/stream`
**Transport**: HTTP with Server-Sent Events (SSE)
**Content-Type**: `text/event-stream`

```json
{
  "jsonrpc": "2.0",
  "method": "message/stream",
  "params": {
    "message": {
      "messageId": "stream-msg-001",
      "role": "user", 
      "parts": [
        {
          "type": "text",
          "text": "Generate a long response with streaming updates"
        }
      ]
    },
    "streamConfig": {
      "heartbeat": 30,
      "bufferSize": 1024
    }
  },
  "id": "stream-req-001"
}
```

### Response Format

#### Initial Response
```http
HTTP/1.1 200 OK
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
Access-Control-Allow-Origin: *

```

#### Stream Events

**Task Created Event:**
```
data: {"type": "task.created", "task": {"id": "task-123", "status": {"state": "working"}, "agent": {"id": "echo-agent", "name": "Echo Agent"}, "created": "2024-07-03T14:30:00.000Z"}}

```

**Status Update Event:**
```
data: {"type": "task.status", "taskId": "task-123", "status": {"state": "working", "timestamp": "2024-07-03T14:30:01.234Z"}}

```

**Progress Update Event (Partial Results):**
```
data: {"type": "task.progress", "taskId": "task-123", "partialResult": "Here is the beginning of the response..."}

```

**Task Completed Event:**
```
data: {"type": "task.completed", "task": {"id": "task-123", "status": {"state": "completed"}, "result": {"messages": [{"messageId": "response-001", "role": "agent", "parts": [{"type": "text", "text": "Complete response here"}]}]}, "completed": "2024-07-03T14:30:10.890Z"}}

```

**Error Event:**
```
data: {"type": "task.error", "taskId": "task-123", "error": "Workflow execution failed: timeout"}

```

**Heartbeat Event:**
```
data: {"type": "heartbeat", "timestamp": "2024-07-03T14:30:30.000Z"}

```

### Error Handling

**Stream Error Event:**
```
event: error
data: {"code": 4301, "message": "Temporal workflow error", "taskId": "task-123", "timestamp": "2024-07-03T14:30:05.000Z"}

```

### Client Implementation Examples

#### JavaScript/Browser (EventSource)

**Basic Usage:**
```javascript
// Connect to streaming endpoint
const eventSource = new EventSource('http://localhost:8080/agents/echo-agent/a2a/stream');

// Handle different event types
eventSource.onmessage = function(event) {
    const data = JSON.parse(event.data);
    
    switch(data.type) {
        case 'task.created':
            console.log('Task started:', data.task.id);
            updateUI('Task created', data.task);
            break;
            
        case 'task.status':
            console.log('Status update:', data.status.state);
            updateProgressBar(data.status);
            break;
            
        case 'task.progress':
            console.log('Progress:', data.partialResult);
            appendToOutput(data.partialResult);
            break;
            
        case 'task.completed':
            console.log('Task completed:', data.task.result);
            showFinalResult(data.task.result);
            eventSource.close(); // Clean disconnect
            break;
            
        case 'task.error':
            console.error('Task failed:', data.error);
            showError(data.error);
            eventSource.close();
            break;
    }
};

// Handle connection errors
eventSource.onerror = function(event) {
    console.error('Stream connection error:', event);
    // Implement reconnection logic with exponential backoff
    setTimeout(() => reconnectWithBackoff(), 1000);
};

// Reconnection with exponential backoff
let reconnectAttempts = 0;
function reconnectWithBackoff() {
    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
    reconnectAttempts++;
    
    setTimeout(() => {
        console.log(`Reconnecting attempt ${reconnectAttempts}...`);
        // Recreate EventSource connection
    }, delay);
}
```

**Advanced Usage with Error Handling:**
```javascript
class A2AStreamingClient {
    constructor(agentId, baseUrl = 'http://localhost:8080') {
        this.agentId = agentId;
        this.baseUrl = baseUrl;
        this.eventSource = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
    }
    
    async sendStreamingMessage(message) {
        const streamUrl = `${this.baseUrl}/agents/${this.agentId}/a2a`;
        
        // Send initial JSON-RPC request to start streaming
        const response = await fetch(streamUrl, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'text/event-stream'
            },
            body: JSON.stringify({
                jsonrpc: '2.0',
                method: 'message/stream',
                params: {
                    message: {
                        messageId: `msg-${Date.now()}`,
                        role: 'user',
                        parts: [{ type: 'text', text: message }]
                    },
                    streamConfig: {
                        heartbeat: 30,
                        bufferSize: 1024
                    }
                },
                id: `req-${Date.now()}`
            })
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        // Start SSE stream
        this.startEventStream(response.headers.get('X-Task-ID'));
    }
    
    startEventStream(taskId) {
        const streamUrl = `${this.baseUrl}/agents/${this.agentId}/a2a/stream?taskId=${taskId}`;
        this.eventSource = new EventSource(streamUrl);
        
        this.eventSource.onmessage = (event) => this.handleStreamEvent(event);
        this.eventSource.onerror = (event) => this.handleStreamError(event);
    }
    
    handleStreamEvent(event) {
        try {
            const data = JSON.parse(event.data);
            this.onStreamEvent(data);
            
            // Reset reconnection counter on successful event
            this.reconnectAttempts = 0;
        } catch (error) {
            console.error('Failed to parse stream event:', error);
        }
    }
    
    handleStreamError(event) {
        console.error('Stream error:', event);
        
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectWithBackoff();
        } else {
            this.onStreamFailed('Max reconnection attempts exceeded');
        }
    }
    
    reconnectWithBackoff() {
        const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
        this.reconnectAttempts++;
        
        setTimeout(() => {
            console.log(`Reconnecting attempt ${this.reconnectAttempts}...`);
            this.startEventStream(this.currentTaskId);
        }, delay);
    }
    
    // Override these methods in your implementation
    onStreamEvent(data) { /* Handle stream events */ }
    onStreamFailed(reason) { /* Handle stream failure */ }
    
    disconnect() {
        if (this.eventSource) {
            this.eventSource.close();
            this.eventSource = null;
        }
    }
}

// Usage example
const client = new A2AStreamingClient('echo-agent');

client.onStreamEvent = function(data) {
    switch(data.type) {
        case 'task.created':
            console.log('Task started:', data.task.id);
            break;
        case 'task.completed':
            console.log('Task finished:', data.task.result);
            client.disconnect();
            break;
    }
};

client.sendStreamingMessage('Generate a long response with streaming updates');
```

#### Python with httpx and asyncio

```python
import asyncio
import json
import httpx
from typing import AsyncGenerator

class A2AStreamingClient:
    def __init__(self, agent_id: str, base_url: str = "http://localhost:8080"):
        self.agent_id = agent_id
        self.base_url = base_url
        
    async def send_streaming_message(self, message: str) -> AsyncGenerator[dict, None]:
        """Send message and yield streaming events"""
        
        url = f"{self.base_url}/agents/{self.agent_id}/a2a"
        
        request_data = {
            "jsonrpc": "2.0",
            "method": "message/stream",
            "params": {
                "message": {
                    "messageId": f"msg-{asyncio.get_event_loop().time()}",
                    "role": "user",
                    "parts": [{"type": "text", "text": message}]
                },
                "streamConfig": {
                    "heartbeat": 30,
                    "bufferSize": 1024
                }
            },
            "id": f"req-{asyncio.get_event_loop().time()}"
        }
        
        async with httpx.AsyncClient() as client:
            async with client.stream(
                "POST", 
                url,
                json=request_data,
                headers={"Accept": "text/event-stream"}
            ) as response:
                response.raise_for_status()
                
                async for line in response.aiter_lines():
                    if line.startswith("data: "):
                        try:
                            event_data = json.loads(line[6:])  # Remove "data: " prefix
                            yield event_data
                            
                            # Break on completion
                            if event_data.get("type") in ["task.completed", "task.error"]:
                                break
                                
                        except json.JSONDecodeError:
                            continue  # Skip malformed events

# Usage example
async def example_streaming():
    client = A2AStreamingClient('echo-agent')
    
    async for event in client.send_streaming_message("Generate streaming response"):
        event_type = event.get("type")
        
        if event_type == "task.created":
            print(f"Task started: {event['task']['id']}")
            
        elif event_type == "task.status":
            print(f"Status: {event['status']['state']}")
            
        elif event_type == "task.progress":
            print(f"Progress: {event.get('partialResult', '')}", end="", flush=True)
            
        elif event_type == "task.completed":
            print(f"\nCompleted: {event['task']['result']}")
            
        elif event_type == "task.error":
            print(f"Error: {event['error']}")

# Run the example
asyncio.run(example_streaming())
```

#### Go Client with http.Client

```go
package main

import (
    "bufio"
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"
)

type A2AStreamingClient struct {
    AgentID string
    BaseURL string
    Client  *http.Client
}

type StreamEvent struct {
    Type string      `json:"type"`
    Task *A2ATask    `json:"task,omitempty"`
    Status *TaskStatus `json:"status,omitempty"`
    PartialResult interface{} `json:"partialResult,omitempty"`
    Error string     `json:"error,omitempty"`
}

func NewA2AStreamingClient(agentID, baseURL string) *A2AStreamingClient {
    return &A2AStreamingClient{
        AgentID: agentID,
        BaseURL: baseURL,
        Client:  &http.Client{Timeout: 0}, // No timeout for streaming
    }
}

func (c *A2AStreamingClient) SendStreamingMessage(ctx context.Context, message string) (<-chan StreamEvent, error) {
    url := fmt.Sprintf("%s/agents/%s/a2a", c.BaseURL, c.AgentID)
    
    requestData := map[string]interface{}{
        "jsonrpc": "2.0",
        "method":  "message/stream",
        "params": map[string]interface{}{
            "message": map[string]interface{}{
                "messageId": fmt.Sprintf("msg-%d", time.Now().UnixNano()),
                "role":      "user",
                "parts": []map[string]interface{}{
                    {"type": "text", "text": message},
                },
            },
            "streamConfig": map[string]interface{}{
                "heartbeat":  30,
                "bufferSize": 1024,
            },
        },
        "id": fmt.Sprintf("req-%d", time.Now().UnixNano()),
    }
    
    jsonData, err := json.Marshal(requestData)
    if err != nil {
        return nil, err
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "text/event-stream")
    
    resp, err := c.Client.Do(req)
    if err != nil {
        return nil, err
    }
    
    eventChan := make(chan StreamEvent, 10)
    
    go func() {
        defer resp.Body.Close()
        defer close(eventChan)
        
        scanner := bufio.NewScanner(resp.Body)
        for scanner.Scan() {
            line := scanner.Text()
            
            if strings.HasPrefix(line, "data: ") {
                eventData := strings.TrimPrefix(line, "data: ")
                
                var event StreamEvent
                if err := json.Unmarshal([]byte(eventData), &event); err == nil {
                    select {
                    case eventChan <- event:
                        // Event sent successfully
                        if event.Type == "task.completed" || event.Type == "task.error" {
                            return // End stream
                        }
                    case <-ctx.Done():
                        return // Context cancelled
                    }
                }
            }
        }
    }()
    
    return eventChan, nil
}

// Usage example
func main() {
    client := NewA2AStreamingClient("echo-agent", "http://localhost:8080")
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    eventChan, err := client.SendStreamingMessage(ctx, "Generate streaming response")
    if err != nil {
        fmt.Printf("Error starting stream: %v\n", err)
        return
    }
    
    for event := range eventChan {
        switch event.Type {
        case "task.created":
            fmt.Printf("Task started: %s\n", event.Task.ID)
            
        case "task.status":
            fmt.Printf("Status: %s\n", event.Status.State)
            
        case "task.progress":
            fmt.Printf("Progress: %v", event.PartialResult)
            
        case "task.completed":
            fmt.Printf("\nCompleted: %v\n", event.Task.Result)
            
        case "task.error":
            fmt.Printf("Error: %s\n", event.Error)
        }
    }
}
```

### Stream Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `heartbeat` | integer | 30 | Heartbeat interval in seconds |
| `bufferSize` | integer | 1024 | Stream buffer size in bytes |
| `timeout` | integer | 300 | Stream timeout in seconds |

### Event Types Reference

| Event | Description | Data Format |
|-------|-------------|-------------|
| `task.started` | Task execution began | `{taskId, status, timestamp}` |
| `task.progress` | Progress update | `{taskId, progress, message, timestamp}` |
| `message.partial` | Partial response chunk | `{taskId, messageId, partialText, timestamp}` |
| `task.completed` | Task finished successfully | `{taskId, status, result, timestamp}` |
| `task.failed` | Task failed with error | `{taskId, status, error, timestamp}` |
| `task.canceled` | Task was canceled | `{taskId, status, timestamp}` |
| `heartbeat` | Keep-alive signal | `{timestamp}` |
| `error` | Stream or task error | `{code, message, taskId?, timestamp}` |

## Technical Implementation Notes

### Architecture Considerations
- **Temporal Integration**: Stream events generated from Temporal workflow progress
- **Redis Pub/Sub**: Event broadcasting across multiple gateway instances
- **Connection Management**: Handle client disconnections gracefully
- **Backpressure**: Implement flow control for high-volume streams

### Performance Characteristics
- **Latency**: Sub-100ms for event delivery
- **Throughput**: Support 1000+ concurrent streams
- **Reliability**: Automatic reconnection with event replay capability

### Security Considerations
- **Authentication**: JWT validation for stream endpoints
- **Rate Limiting**: Per-client stream connection limits
- **Resource Limits**: Maximum stream duration and event count

---

**Sprint 2 Implementation Checklist:**
- [ ] SSE endpoint implementation
- [ ] Temporal workflow event integration
- [ ] Redis pub/sub for multi-instance support
- [ ] Client SDK examples
- [ ] Load testing and performance validation
- [ ] Security implementation
- [ ] Final documentation review

**Related Documentation:**
- [API Reference](./api.md) - Core A2A Protocol methods
- [Implementation Guide](./implementation.md) - Architecture details
- [Testing Guide](./testing.md) - Streaming test scenarios