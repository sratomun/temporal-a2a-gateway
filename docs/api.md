# API Reference

This document provides complete API reference for the Temporal A2A Gateway implementation following the official A2A Protocol v0.2.5 specification.

## Protocol Compliance

This gateway fully implements the [A2A Protocol v0.2.5 specification](https://a2aproject.github.io/A2A/v0.2.5/specification/) and is compatible with the Google A2A SDK.

The A2A Protocol is an open standard for agent-to-agent communication that facilitates:
- Agent discovery and collaboration
- Secure, flexible interactions  
- Asynchronous, long-running tasks
- Standardized communication framework

## Base URL

All API requests are made to agent-specific endpoints using HTTP POST with JSON-RPC 2.0 format.

**Agent endpoint pattern:** `http://localhost:8080/agents/{agentId}/a2a`

**Example:** `http://localhost:8080/agents/echo-agent/a2a`

## Transport & Communication

- **Protocol**: HTTP(S) with JSON-RPC 2.0
- **Content-Type**: `application/json`
- **Method**: POST
- **Streaming**: Server-Sent Events (SSE) support planned
- **Security**: HTTPS required for production environments
- **Timestamps**: All timestamps use ISO 8601 format with millisecond precision in UTC (`YYYY-MM-DDTHH:mm:ss.sssZ`)

## Authentication

Currently operates without authentication for development. Production deployments should implement:
- JWT-based authentication using `JWT_SECRET` environment variable
- Standard web security practices
- Task-level and server-level authentication
- Various authentication schemes as per A2A spec

## Request Format

All requests follow JSON-RPC 2.0 specification:

```json
{
  "jsonrpc": "2.0",
  "method": "method_name",
  "params": {...},
  "id": "unique-request-id"
}
```

## Response Format

**Successful responses:**
```json
{
  "jsonrpc": "2.0",
  "result": {...},
  "id": "unique-request-id"
}
```

**Error responses:**
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32601,
    "message": "Method not found",
    "data": {...}
  },
  "id": "unique-request-id"
}
```

## Core A2A Protocol Methods

### message/send

Sends a synchronous message to an agent and creates a new task.

**Endpoint:** `/agents/{agentId}/a2a`

**Method:** `message/send`

**Parameters:**
- `message` (Message object, required): Message to send to the agent
- `metadata` (object, optional): Additional task metadata

**Message Object Structure:**
```typescript
interface Message {
  messageId: string;
  role: "user" | "agent";
  parts: MessagePart[];
}

interface MessagePart {
  type: "text" | "file" | "data";
  text?: string;           // For text parts
  file?: FileData;         // For file parts  
  data?: any;             // For data parts
}

interface FileData {
  name: string;
  mimeType: string;
  data: string;           // Base64 encoded
}
```

**Example Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "message/send",
  "params": {
    "message": {
      "messageId": "msg-001",
      "role": "user",
      "parts": [
        {
          "type": "text",
          "text": "Hello from A2A Protocol! Please process this request."
        }
      ]
    },
    "metadata": {
      "priority": "normal",
      "source": "api-test"
    }
  },
  "id": "send-001"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "taskId": "47df4688-5266-435e-9700-54773d6e2c81",
    "status": "working",
    "agent": {
      "id": "echo-agent",
      "name": "Echo Agent"
    },
    "created": "2024-07-03T14:30:00.000Z"
  },
  "id": "send-001"
}
```

### tasks/get

Retrieves the current status and results of a task.

**Endpoint:** `/agents/{agentId}/a2a`

**Method:** `tasks/get`

**Parameters:**
- `taskId` (string, required): Task identifier

**Example Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "tasks/get",
  "params": {
    "taskId": "47df4688-5266-435e-9700-54773d6e2c81"
  },
  "id": "get-001"
}
```

**Response (Working):**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "taskId": "47df4688-5266-435e-9700-54773d6e2c81",
    "status": "working",
    "agent": {
      "id": "echo-agent",
      "name": "Echo Agent"
    },
    "created": "2024-07-03T14:30:00.000Z",
    "lastUpdated": "2024-07-03T14:30:05.123Z"
  },
  "id": "get-001"
}
```

**Response (Completed):**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "taskId": "47df4688-5266-435e-9700-54773d6e2c81",
    "status": "completed", 
    "agent": {
      "id": "echo-agent",
      "name": "Echo Agent"
    },
    "created": "2024-07-03T14:30:00.000Z",
    "completed": "2024-07-03T14:30:10.456Z",
    "messages": [
      {
        "messageId": "msg-001",
        "role": "user",
        "parts": [
          {
            "type": "text",
            "text": "Hello from A2A Protocol! Please process this request."
          }
        ]
      },
      {
        "messageId": "response-001",
        "role": "agent",
        "parts": [
          {
            "type": "text", 
            "text": "Echo: Hello from A2A Protocol! Please process this request."
          }
        ]
      }
    ],
    "artifacts": []
  },
  "id": "get-001"
}
```

### tasks/cancel

Cancels a running task according to A2A specification.

**Endpoint:** `/agents/{agentId}/a2a`

**Method:** `tasks/cancel`

**Parameters:**
- `taskId` (string, required): Task identifier to cancel

**Example Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "tasks/cancel",
  "params": {
    "taskId": "47df4688-5266-435e-9700-54773d6e2c81"
  },
  "id": "cancel-001"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "taskId": "47df4688-5266-435e-9700-54773d6e2c81",
    "status": "canceled",
    "canceledAt": "2024-07-03T14:30:15.789Z"
  },
  "id": "cancel-001"
}
```

### message/stream

Real-time task updates via Server-Sent Events (SSE) using Temporal workflow signals.

**Endpoint:** `/agents/{agentId}/a2a`

**Method:** `message/stream`

**Implementation:** Pure Temporal signals with 100ms query intervals for optimal performance.

**Parameters:**
- `message` (Message object, required): Message to send to the agent
- `streamConfig` (object, optional): Streaming configuration options

**Stream Configuration:**
```typescript
interface StreamConfig {
  heartbeat?: number;     // Heartbeat interval in seconds (default: 30)
  bufferSize?: number;    // Stream buffer size (default: 1024)
  timeout?: number;       // Stream timeout in seconds (default: 300)
}
```

**Example Request:**
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
          "text": "Generate a response with real-time progress updates"
        }
      ]
    },
    "streamConfig": {
      "heartbeat": 30,
      "bufferSize": 1024
    }
  },
  "id": "stream-001"
}
```

**Response:** SSE stream with real-time task progress events

**SSE Event Types:**
- `task.created`: Task started execution
- `task.status`: Status update during execution  
- `task.progress`: Partial results or progress updates
- `task.completed`: Task finished successfully
- `task.error`: Task failed with error
- `heartbeat`: Keep-alive signal

## Legacy Methods (Deprecated)

### a2a.createTask (DEPRECATED)

⚠️ **DEPRECATED as of v0.4.1** - Use `message/send` instead

**Deprecation Timeline:**
- **Notice Date**: July 2024
- **Sunset Date**: October 3, 2024 (3 months)
- **Migration Required**: All clients should migrate to `message/send`

**Deprecation Indicators:**
- HTTP Header: `Deprecation: true`
- HTTP Header: `Sunset: 2024-10-03T00:00:00.000Z`
- Response includes `deprecated: true` flag
- Server logs deprecation usage for monitoring

**Migration Guide:**
```json
// OLD (deprecated)
{
  "method": "a2a.createTask",
  "params": {
    "message": "Hello world"
  }
}

// NEW (recommended)
{
  "method": "message/send", 
  "params": {
    "message": {
      "messageId": "msg-001",
      "role": "user",
      "parts": [{"type": "text", "text": "Hello world"}]
    }
  }
}
```

## Task Lifecycle States

Tasks progress through states as defined in A2A Protocol v0.2.5:

- **`submitted`**: Task has been received and queued
- **`working`**: Task is actively being processed  
- **`input_required`**: Task needs additional input from user (planned)
- **`completed`**: Task finished successfully
- **`canceled`**: Task was cancelled before completion
- **`failed`**: Task failed due to an error
- **`rejected`**: Task was rejected by the agent (planned)
- **`auth_required`**: Task requires authentication (planned)

### State Transitions
```
submitted → working → {completed|failed|canceled}
```

**Streaming Updates**: Real-time state transitions are available via `message/stream` endpoint using Temporal workflow signals for immediate notification of status changes.

## Agent Card Support

Agent Cards describe agent capabilities and metadata. Currently supported agents:

### Echo Agent (`echo-agent`)
```json
{
  "id": "echo-agent",
  "name": "Echo Agent",
  "description": "Enhanced echo agent with real-time streaming capabilities",
  "version": "2.0.0",
  "capabilities": {
    "streaming": true,
    "progressiveArtifacts": true,
    "pushNotifications": false,
    "stateTransitionHistory": true
  },
  "inputModes": ["text"],
  "outputModes": ["text", "stream"]
}
```

### Custom Agents
Additional agents can be configured via `agent-routing.yaml` configuration file following A2A agent card specification.

## Google A2A SDK Integration

This gateway is fully compatible with the Google A2A SDK and demonstrates complete A2A v0.2.5 specification compliance. 

> **📖 Reference Implementation**: See [SDK Integration Guide](./sdk-integration.md) for detailed A2A v0.2.5 client patterns and specification compliance.

The integration follows the A2A protocol's JSON-first design philosophy for maximum compatibility and flexibility.

```python
import asyncio
import httpx
from a2a.client import A2AClient
from a2a.types import (
    Message, TextPart, SendMessageRequest, MessageSendParams,
    GetTaskRequest, TaskQueryParams
)

async def example_a2a_interaction():
    async with httpx.AsyncClient() as http_client:
        # Create A2A client
        client = A2AClient(
            httpx_client=http_client,
            url="http://localhost:8080/agents/echo-agent/a2a"
        )
        
        # Create message following A2A spec
        message = Message(
            messageId="test-001",
            role="user",
            parts=[TextPart(text="Hello A2A Protocol!")]
        )
        
        # Send message
        params = MessageSendParams(message=message)
        request = SendMessageRequest(id="req-001", params=params)
        task_result = await client.send_message(request)
        
        # Get task ID
        task_id = task_result.root.result.id
        
        # Poll for completion
        while True:
            task_params = TaskQueryParams(id=task_id)
            get_request = GetTaskRequest(id="poll-001", params=task_params)
            task_status = await client.get_task(get_request)
            
            status = task_status.root.result.status.state
            if status in ['completed', 'failed', 'canceled']:
                break
                
            await asyncio.sleep(1)
```

## Error Handling

### Standard JSON-RPC Errors
- `-32700`: Parse error - Invalid JSON
- `-32600`: Invalid request - Malformed request object
- `-32601`: Method not found - Unknown method  
- `-32602`: Invalid params - Invalid method parameters
- `-32603`: Internal error - Server-side error

### A2A Protocol-Specific Errors

**Task Management (4000-4099):**
- `4001`: Task not found
- `4002`: Invalid task state transition
- `4003`: Task creation failed
- `4004`: Task execution timeout
- `4005`: Task quota exceeded

**Agent Management (4100-4199):**
- `4101`: Agent not found or unavailable
- `4102`: Agent configuration error
- `4103`: Agent capability mismatch
- `4104`: Agent authentication required

**Message Validation (4200-4299):**
- `4201`: Invalid message format
- `4202`: Unsupported message part type
- `4203`: Message size limit exceeded
- `4204`: Invalid message role

**Infrastructure (4300-4399):**
- `4301`: Temporal workflow engine error
- `4302`: Redis cache unavailable
- `4303`: Agent registry connection failure
- `4304`: Database connection error

## Health and Monitoring

### GET /health

Returns gateway health status following A2A best practices:

```json
{
  "status": "healthy",
  "timestamp": "2025-07-03T00:06:46.123Z",
  "version": "0.4.0-go",
  "service": "a2a-gateway-temporal",
  "protocol": {
    "name": "A2A",
    "version": "0.2.5"
  },
  "dependencies": {
    "temporal": {"connected": true},
    "redis": {"connected": true},
    "agentRegistry": {
      "connected": true,
      "url": "http://agent-registry:8001"
    }
  }
}
```

### GET /metrics

OpenTelemetry metrics in Prometheus format for A2A protocol monitoring.

## Security & Production Considerations

### A2A Security Requirements
- **HTTPS**: Required for production per A2A spec
- **Authentication**: Implement JWT or OAuth2 as per A2A guidelines
- **Rate Limiting**: Prevent abuse of agent endpoints
- **Input Validation**: Sanitize all message content
- **Audit Logging**: Log all agent interactions

### Production Checklist
- [ ] Enable HTTPS/TLS for all communications
- [ ] Implement authentication per A2A specification
- [ ] Configure rate limiting and DDoS protection
- [ ] Enable comprehensive audit logging
- [ ] Implement message content validation
- [ ] Set up monitoring and alerting
- [ ] Configure proper CORS policies

## Protocol Compliance Status

✅ **Fully Compliant with A2A Protocol v0.2.5:**
- JSON-RPC 2.0 transport layer
- Standard message/send method
- Proper task lifecycle management  
- A2A-compliant data structures
- Google A2A SDK compatibility
- Task status and retrieval methods
- Error handling per specification

🚧 **Planned A2A Features:**
- Advanced authentication schemes
- Agent capability negotiation  
- File and data message parts
- Enhanced artifact management
- Push notification configuration

## Versioning

- **A2A Protocol**: v0.2.5 (fully compliant)
- **Gateway Version**: 0.4.0-go
- **JSON-RPC**: 2.0
- **Temporal Integration**: Custom enterprise extension

This implementation prioritizes full A2A Protocol compliance while adding enterprise-grade reliability through Temporal workflow orchestration.