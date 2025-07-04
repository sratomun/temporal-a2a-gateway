# A2A v0.2.5 Task Artifacts Specification

**Version**: A2A Protocol v0.2.5  
**Document**: Task Artifacts Implementation Reference  
**Author**: Agent 5 (Standardization Engineer)  
**Date**: 2025-07-04  
**Status**: ✅ DEFINITIVE SPECIFICATION REFERENCE

## Overview

This document provides the definitive A2A v0.2.5 task artifacts specification. All implementations must use the `artifacts` array structure, not custom `result` fields, to achieve specification compliance.

## Critical Specification Requirement

### ✅ CORRECT: Task Results in Artifacts Array

A2A v0.2.5 task results **MUST** be represented in the `artifacts` array:

```json
{
  "id": "task-123",
  "contextId": "ctx-456",
  "status": {"state": "completed", "timestamp": "2025-07-04T09:00:00.000Z"},
  "kind": "task",
  "agentId": "echo-agent",
  "input": {...},
  "artifacts": [
    {
      "artifactId": "echo-result-1",
      "name": "Echo Response",
      "description": "Echoed user message",
      "parts": [
        {
          "kind": "text",
          "text": "Echo: Hello from A2A!"
        }
      ]
    }
  ],
  "error": null,
  "metadata": {},
  "createdAt": "2025-07-04T09:00:00.000Z"
}
```

### ❌ INCORRECT: Custom Result Fields

Custom `result` fields are **NOT** part of the A2A v0.2.5 specification:

```json
{
  "id": "task-123",
  "result": {                    // ❌ NON-STANDARD FIELD
    "messages": [...],
    "status": "completed"
  }
}
```

## Artifact Structure Specification

### Required Artifact Fields

Every artifact **MUST** include these fields:

- **artifactId**: Unique identifier (string, required)
- **name**: Human-readable name (string, required)
- **description**: Brief description (string, optional but recommended)
- **parts**: Array of part objects (required)

### Artifact Example

```json
{
  "artifactId": "response-123",
  "name": "Agent Response",
  "description": "Primary response from the agent",
  "parts": [
    {
      "kind": "text",
      "text": "Response content here"
    }
  ]
}
```

## Part Object Types

### Text Parts

For text content (most common for echo agents):

```json
{
  "kind": "text",
  "text": "Echo: Hello from Google A2A SDK!"
}
```

### File Parts

For file attachments:

```json
{
  "kind": "file", 
  "file": {
    "name": "report.pdf",
    "uri": "https://example.com/files/report.pdf"
  }
}
```

### Data Parts

For structured data:

```json
{
  "kind": "data",
  "data": {
    "key1": "value1",
    "key2": 42,
    "nested": {
      "property": "value"
    }
  }
}
```

## Implementation Requirements

### For Gateway Developers

#### Remove Custom Result Fields

```go
// ❌ REMOVE THIS - Non-standard result field
taskResult := map[string]interface{}{
    "result": map[string]interface{}{
        "messages": messages,
        "status": "completed",
    },
}
```

#### Implement A2A Compliant Artifacts

```go
// ✅ A2A v0.2.5 compliant artifacts structure
taskArtifacts := []map[string]interface{}{
    {
        "artifactId": fmt.Sprintf("%s-result", taskID),
        "name": "Agent Response",
        "description": fmt.Sprintf("Response from %s agent", agentID),
        "parts": []map[string]interface{}{
            {
                "kind": "text",
                "text": extractTextFromMessages(messages),
            },
        },
    },
}

// Include artifacts in A2A Task object
taskResponse := map[string]interface{}{
    "id": taskID,
    "contextId": contextID,
    "status": map[string]interface{}{
        "state": "completed",
        "timestamp": newISO8601Timestamp(),
    },
    "kind": "task",
    "agentId": agentID,
    "input": originalInput,
    "artifacts": taskArtifacts,  // ✅ A2A STANDARD
    "error": nil,
    "metadata": metadata,
    "createdAt": createdAt,
}
```

### For Worker Developers

#### Echo Worker Artifact Generation

```python
# ✅ A2A v0.2.5 compliant artifact structure
def echo_activity(task_input):
    # Extract input text
    input_text = extract_text_from_input(task_input)
    echo_response = f"Echo: {input_text}"
    
    # Return A2A compliant artifact
    return {
        "artifacts": [
            {
                "artifactId": f"echo-{int(time.time())}",
                "name": "Echo Response", 
                "description": "Echoed user message",
                "parts": [
                    {
                        "kind": "text",
                        "text": echo_response
                    }
                ]
            }
        ]
    }
```

#### Remove Custom Result Structures

```python
# ❌ REMOVE THIS - Custom result structure
return {
    "messages": [{"role": "assistant", "parts": [{"text": f"Echo: {input_text}"}]}],
    "status": "completed"
}
```

### For Storage Systems

#### Redis Task Storage

```go
// ✅ A2A compliant task storage
func (g *Gateway) updateTaskStatusInRedis(taskID, status string, artifacts []interface{}, errorMsg string) error {
    task := map[string]interface{}{
        "id": taskID,
        "status": map[string]interface{}{
            "state": status,
            "timestamp": newISO8601Timestamp(),
        },
        "artifacts": artifacts,  // ✅ Store artifacts, not custom result
        "error": errorMsg,
    }
    
    // Store in Redis
    return g.redisClient.Set(ctx, taskID, task, 0).Err()
}
```

## Streaming Artifact Updates

### TaskArtifactUpdateEvent Structure

For progressive artifact streaming:

```go
type TaskArtifactUpdateEvent struct {
    TaskID    string      `json:"taskId"`
    ContextID string      `json:"contextId"`
    Kind      string      `json:"kind"`        // "artifact-update"
    Artifact  interface{} `json:"artifact"`    // Single artifact object
    Append    bool        `json:"append,omitempty"`
    LastChunk bool        `json:"lastChunk,omitempty"`
    Timestamp string      `json:"timestamp"`
}
```

### Progressive Artifact Example

```go
// Stream artifact updates during task execution
artifactEvent := TaskArtifactUpdateEvent{
    TaskID:    taskID,
    ContextID: contextID,
    Kind:      "artifact-update",
    Artifact: map[string]interface{}{
        "artifactId": "echo-progress",
        "name": "Echo Progress",
        "description": "Progressive echo response",
        "parts": []map[string]interface{}{
            {
                "kind": "text", 
                "text": partialEchoResponse,
            },
        },
    },
    Append:    false,  // Complete artifact each time for echo
    LastChunk: isComplete,
    Timestamp: newISO8601Timestamp(),
}
```

## Validation Requirements

### Implementation Checklist

**Phase 1: Core Structure Fix**
- [ ] Remove all custom `result` fields from task objects
- [ ] Implement `artifacts` array in all task responses
- [ ] Update worker to return artifacts structure
- [ ] Modify storage to use artifacts

**Phase 2: Response Handler Updates**
- [ ] Fix all message handlers to return A2A compliant tasks
- [ ] Update task retrieval to return artifacts structure
- [ ] Update streaming events to use TaskArtifactUpdateEvent
- [ ] Ensure all JSON-RPC responses have single result field

**Phase 3: Validation**
- [ ] Test Google SDK integration with artifact structure
- [ ] Validate streaming works with artifact events
- [ ] Confirm all endpoints return A2A compliant responses
- [ ] Run comprehensive compliance testing

### Compliance Tests

1. ✅ Task objects contain `artifacts` array, not custom `result` field
2. ✅ All artifacts have required fields: `artifactId`, `name`, `parts`
3. ✅ All parts have valid `kind` field ("text", "file", or "data")
4. ✅ Text parts contain `text` field with string content
5. ✅ Streaming uses `TaskArtifactUpdateEvent` with correct structure

### Common Violations

❌ Using custom `result` field instead of `artifacts`  
❌ Missing required artifact fields (`artifactId`, `name`, `parts`)  
❌ Missing `kind` field in parts  
❌ Invalid part types (not "text", "file", or "data")  
❌ Non-standard artifact structure  

## Google SDK Compatibility

### SDK Integration Example

```python
# ✅ Google A2A SDK can access artifacts correctly
task_result = client.get_task(task_id).model_dump().get('result', {})
artifacts = task_result.get('artifacts', [])

for artifact in artifacts:
    artifact_name = artifact.get('name', 'Unknown')
    parts = artifact.get('parts', [])
    
    for part in parts:
        if part.get('kind') == 'text':
            text_content = part.get('text')
            print(f"Agent Response: {text_content}")
```

### Echo Agent Conversation Flow

```
USER: Hello from Google A2A SDK! Testing integration.
AGENT (via artifacts): Echo: Hello from Google A2A SDK! Testing integration.
```

## References

- A2A Protocol v0.2.5 Specification
- Task object documentation
- Artifact structure requirements
- Google A2A SDK integration examples

---

**Standardization Authority**: Agent 5 (Standardization Engineer)  
**Implementation Status**: Required for A2A v0.2.5 compliance  
**Last Updated**: 2025-07-04