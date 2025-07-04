# A2A v0.2.5 Streaming Events Specification

**Version**: A2A Protocol v0.2.5  
**Document**: Streaming Events Implementation Reference  
**Author**: Agent 5 (Standardization Engineer)  
**Date**: 2025-07-04  
**Status**: ✅ DEFINITIVE SPECIFICATION REFERENCE

## Overview

This document provides the definitive A2A v0.2.5 streaming events specification for implementation by development agents. All streaming implementations must comply with these exact requirements.

## Required Event Types

A2A v0.2.5 defines exactly **TWO** streaming event types:

1. **TaskStatusUpdateEvent** - Status changes during execution
2. **TaskArtifactUpdateEvent** - Real-time artifact generation

### ❌ NON-STANDARD EVENT TYPES (FORBIDDEN)

These event types are **NOT** part of A2A v0.2.5 specification:
- `task.created`
- `task.status` 
- `task.progress`
- `task.completed`
- `task.failed`

## TaskStatusUpdateEvent Specification

### Required Structure
```json
{
  "taskId": "string",
  "contextId": "string", 
  "kind": "status-update",
  "status": {
    "state": "submitted|working|completed|failed|canceled",
    "timestamp": "2025-07-04T09:00:00.000Z"
  },
  "final": false
}
```

### Field Requirements
- **taskId**: Unique task identifier (required)
- **contextId**: Context identifier for the task (required)
- **kind**: Must be exactly `"status-update"` (required)
- **status**: Task status object with state and timestamp (required)
- **final**: Boolean indicating terminal status (required)

### Status States
- `submitted`: Task received and queued
- `working`: Task actively being processed
- `completed`: Task finished successfully
- `failed`: Task terminated due to error
- `canceled`: Task was cancelled

### Final Flag Usage
- Set `final: true` for terminal states (`completed`, `failed`, `canceled`)
- Set `final: false` for non-terminal states (`submitted`, `working`)

## TaskArtifactUpdateEvent Specification

### Required Structure
```json
{
  "taskId": "string",
  "contextId": "string",
  "kind": "artifact-update", 
  "artifact": {
    "artifactId": "string",
    "name": "string",
    "description": "string",
    "parts": [
      {
        "kind": "text",
        "text": "content"
      }
    ]
  },
  "append": false,
  "lastChunk": false
}
```

### Field Requirements
- **taskId**: Unique task identifier (required)
- **contextId**: Context identifier for the task (required)
- **kind**: Must be exactly `"artifact-update"` (required)
- **artifact**: Complete artifact object with parts (required)
- **append**: Boolean for progressive building (required)
- **lastChunk**: Boolean indicating final artifact (required)

### Append Flag Usage
- Set `append: true` for progressive content building
- Set `append: false` for complete artifact replacement

### LastChunk Flag Usage
- Set `lastChunk: true` for final artifact in sequence
- Set `lastChunk: false` for intermediate artifacts

## Streaming Event Lifecycle

### Complete A2A v0.2.5 Event Sequence

1. **Initial Status Update**
```json
{
  "taskId": "task-123",
  "contextId": "ctx-456",
  "kind": "status-update",
  "status": {"state": "working", "timestamp": "2025-07-04T09:00:00.000Z"},
  "final": false
}
```

2. **Artifact Updates** (as content is generated)
```json
{
  "taskId": "task-123",
  "contextId": "ctx-456",
  "kind": "artifact-update",
  "artifact": {
    "artifactId": "echo-result",
    "name": "Echo Response",
    "description": "Echoed user message",
    "parts": [{"kind": "text", "text": "Echo: Hello"}]
  },
  "append": false,
  "lastChunk": false
}
```

3. **Final Status Update**
```json
{
  "taskId": "task-123",
  "contextId": "ctx-456",
  "kind": "status-update",
  "status": {"state": "completed", "timestamp": "2025-07-04T09:01:00.000Z"},
  "final": true
}
```

4. **Final Artifact** (if applicable)
```json
{
  "taskId": "task-123",
  "contextId": "ctx-456", 
  "kind": "artifact-update",
  "artifact": {
    "artifactId": "echo-final",
    "name": "Final Echo Response",
    "parts": [{"kind": "text", "text": "Echo: Complete response"}]
  },
  "append": false,
  "lastChunk": true
}
```

## SSE Format Requirements

### Correct A2A v0.2.5 SSE Format
```
data: {"taskId": "123", "contextId": "ctx-123", "kind": "status-update", "status": {...}, "final": false}

data: {"taskId": "123", "contextId": "ctx-123", "kind": "artifact-update", "artifact": {...}, "append": false, "lastChunk": true}
```

### ❌ Incorrect Non-Standard Format
```
data: {"type": "task.status", "taskId": "123", ...}
```

## Agent Capability Declaration

Agents supporting streaming must declare in their AgentCard:

```json
{
  "capabilities": {
    "streaming": true
  }
}
```

## Implementation Requirements

### For Gateway Developers

1. **Event Type Compliance**
   - Use only `TaskStatusUpdateEvent` and `TaskArtifactUpdateEvent`
   - Set `kind` field to exact values: `"status-update"` or `"artifact-update"`

2. **Required Fields**
   - Always include `taskId`, `contextId`, `kind`
   - Include `final` flag in status updates
   - Include `append` and `lastChunk` flags in artifact updates

3. **Stream Lifecycle**
   - Send initial status update when task starts
   - Send artifact updates as content is generated
   - Send final status update with `final: true`
   - Send final artifact with `lastChunk: true`

### For Client SDK Developers

1. **Event Parsing**
   - Parse events based on `kind` field
   - Handle both event types appropriately
   - Use `final` and `lastChunk` flags for completion detection

2. **Type Safety**
   - Validate required fields are present
   - Handle unknown fields gracefully for future compatibility

## Compliance Validation

### Required Tests

1. ✅ Event types use only `status-update` and `artifact-update`
2. ✅ All required fields present in events
3. ✅ Status updates include `final` flag
4. ✅ Artifact updates include `append` and `lastChunk` flags
5. ✅ Complete event lifecycle delivered
6. ✅ SSE format compliance validated

### Common Violations

❌ Using custom event types like `task.created`  
❌ Missing `contextId` field  
❌ Missing `kind` field  
❌ Missing `final` flag in status updates  
❌ Missing `append`/`lastChunk` flags in artifact updates  
❌ Incomplete event lifecycle

## References

- A2A Protocol v0.2.5 Specification
- TaskStatusUpdateEvent documentation
- TaskArtifactUpdateEvent documentation
- Streaming capabilities specification

---

**Standardization Authority**: Agent 5 (Standardization Engineer)  
**Implementation Status**: Required for A2A v0.2.5 compliance  
**Last Updated**: 2025-07-04