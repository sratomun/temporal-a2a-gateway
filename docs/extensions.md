# Implementation Extensions

This document describes the specific extensions and enhancements that this implementation adds to the base A2A protocol specification.

## Agent Routing Configuration

### Overview
While the A2A specification defines agent discovery mechanisms, this implementation adds YAML-based static routing configuration for production deployments where agent locations are known in advance.

### Configuration Format
```yaml
version: "1.0"
routing:
  "Agent Name":
    taskQueue: "queue-name"
    workflowType: "WorkflowType"

workflowCategories:
  LLMAgentWorkflow:
    description: "For agents that use LLM for reasoning and generation"
    examples: ["Product Manager", "Developer", "Architect"]
```

### Routing Process
1. Client specifies agent by name in `agentId` field
2. Gateway looks up agent in routing configuration
3. Task is routed to corresponding Temporal task queue
4. Appropriate workflow type is invoked

### Benefits
- Predictable agent routing in production environments
- Type safety through workflow categorization
- Easy configuration management through version control

## Task Metadata System

### Enhanced Task Creation
The implementation extends the standard `a2a.createTask` method with optional metadata support:

```json
{
  "jsonrpc": "2.0",
  "method": "a2a.createTask",
  "params": {
    "agentId": "product-manager-agent",
    "input": {...},
    "metadata": {
      "project": "mobile-app",
      "priority": "high",
      "requestor": "user@example.com"
    }
  },
  "id": 1
}
```

### Metadata Indexing
Tasks are automatically indexed by metadata key-value pairs, enabling efficient querying:

- Redis key pattern: `tasks:by_metadata:{key}:{value}`
- Supports arbitrary metadata fields
- Maintains referential integrity with task lifecycle

### Metadata Query Method
Non-standard method `x-a2a.getTasksByMetadata` enables metadata-based task discovery:

```json
{
  "jsonrpc": "2.0",
  "method": "x-a2a.getTasksByMetadata",
  "params": {
    "metadataKey": "project",
    "metadataValue": "mobile-app",
    "limit": 10
  },
  "id": 1
}
```

## Enhanced Error Classification

### Error Code Extensions
The implementation extends the A2A specification with detailed error codes organized by functional area:

#### Task Management (40000-40099)
- `40001`: Task not found
- `40002`: Invalid task state for requested operation
- `40003`: Task creation failed
- `40004`: Task update failed
- `40005`: Task cancellation failed

#### Agent Management (40100-40199)
- `40101`: Agent not found in routing configuration
- `40102`: Agent unavailable or offline
- `40103`: Agent capability mismatch
- `40104`: Agent registration failed

#### Service Integration (40300-40399)
- `40301`: Temporal connection failure
- `40302`: Redis connection failure
- `40303`: Agent registry connection failure

### Error Metadata
Each error response includes additional metadata for debugging:

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": 40001,
    "message": "Task not found",
    "data": {
      "category": "task_management",
      "taskId": "abc-123",
      "timestamp": "2024-01-01T12:00:00Z"
    }
  },
  "id": 1
}
```

## Message Format Normalization

### Input Normalization
The gateway automatically normalizes various input formats to the A2A message structure:

#### String Input
```json
"Hello, agent!"
```
Normalized to:
```json
{
  "messages": [{
    "role": "user",
    "parts": [{
      "type": "text",
      "content": "Hello, agent!"
    }],
    "timestamp": "2024-01-01T12:00:00Z"
  }]
}
```

#### Legacy Message Format
```json
{
  "message": "Hello, agent!"
}
```
Automatically converted to standard A2A message format.

### Benefits
- Backward compatibility with simple message formats
- Consistent internal message handling
- Simplified client integration

## Task Lifecycle Extensions

### Automatic Cleanup
The implementation includes automatic task cleanup mechanisms:

#### TTL-Based Cleanup
- Completed tasks: 24-hour retention
- Active tasks: 7-day maximum lifetime
- Failed tasks: 24-hour retention for debugging

#### Scheduled Cleanup
- Hourly cleanup process removes expired tasks
- Index maintenance ensures query performance
- Configurable retention policies per task type

### Task State Synchronization
- Redis serves as fast lookup cache
- Temporal maintains authoritative state
- Automatic synchronization on state changes

## Health and Monitoring Extensions

### Enhanced Health Endpoint
The `/health` endpoint provides detailed service status:

```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z",
  "version": "0.4.0-go",
  "service": "temporal-a2a-gateway",
  "temporal": {
    "connected": true
  },
  "redis": {
    "connected": true
  },
  "agentRegistry": {
    "connected": true,
    "url": "http://registry:8001"
  }
}
```

### Metrics Endpoint
OpenTelemetry metrics exposed at `/metrics` include:

- Request latency percentiles
- Task creation and completion rates
- Error rates by category and agent
- Active task counts by status

## Environment Validation

### Startup Validation
The gateway performs comprehensive environment validation on startup:

#### Required Variables
- `TEMPORAL_HOST`: Temporal server hostname
- `TEMPORAL_PORT`: Temporal server port
- `TEMPORAL_NAMESPACE`: Temporal namespace
- `A2A_PORT`: Gateway listening port
- `REDIS_URL`: Redis connection URL
- `AGENT_REGISTRY_URL`: Agent registry service URL

#### Optional Variables
- `JWT_SECRET`: JWT signing secret
- `LOG_LEVEL`: Logging verbosity level
- `DATABASE_URL`: PostgreSQL connection URL

### Validation Rules
- Port values must be numeric
- URLs must be properly formatted
- JWT secrets are checked for strength
- Missing required variables prevent startup

## Agent Capability Extensions

### Enhanced Agent Registration
The implementation extends agent registration with additional capability metadata:

```json
{
  "agentCard": {
    "name": "Product Manager Agent",
    "description": "Generates product requirements and specifications",
    "version": "1.0.0",
    "capabilities": {
      "streaming": true,
      "pushNotifications": false,
      "stateTransitionHistory": true
    }
  }
}
```

### Capability Matching
Agent discovery can filter by specific capabilities, enabling intelligent agent selection based on client requirements.

## Backward Compatibility

All extensions are designed to maintain compatibility with the base A2A protocol:

- Standard A2A methods work without modification
- Extensions use `x-a2a.*` method prefix for non-standard methods
- Optional fields in standard methods remain optional
- Error codes extend rather than replace standard codes

Clients implementing only the base A2A specification can interact with this gateway without utilizing extension features.