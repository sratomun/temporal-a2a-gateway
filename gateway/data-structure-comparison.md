# Data Structure Comparison: Python SDK vs A2A Gateway

## 1. A2AMessage Structure

### Python SDK (temporal_a2a_sdk/messages.py)
```python
class A2AMessage:
    def __init__(self, role: str, parts: List[Dict[str, Any]], 
                 timestamp: Optional[str] = None):
        self.role = role
        self.parts = parts
        self.timestamp = timestamp or datetime.utcnow().isoformat() + "Z"
```

### Google A2A SDK (from integration example)
```python
Message(
    messageId="test-msg-001",
    role="user",
    parts=[TextPart(text=test_message)]
)
```

### Gateway Expectation (from echo_worker.py)
The gateway passes the message directly to the workflow. The worker expects:
```python
# Message structure expected:
{
    "parts": [
        {"text": "message content"}
    ]
}
```

### Gateway Handling (from main.go)
```go
type AgentMessageParams struct {
    Message  interface{}            `json:"message"`
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

**MISMATCH**: The Google A2A SDK includes `messageId` field, but the gateway and worker don't expect or handle this field.

## 2. A2AResponse/TaskResult Structure

### Python SDK (temporal_a2a_sdk/messages.py)
```python
class A2AResponse:
    def to_dict(self) -> Dict[str, Any]:
        result = {"artifacts": self.artifacts}
        if self.error:
            result["error"] = self.error
        return result

# Artifact structure:
{
    "artifactId": str,
    "name": str,
    "parts": [{"kind": "text", "text": content}]
}
```

### Worker Implementation (echo_worker.py)
```python
class Artifact:
    # Returns:
    {
        "artifactId": str,
        "name": str,
        "description": str,  # Optional
        "parts": [{"kind": "text", "text": content}]
    }

class TaskResult:
    # Returns:
    {
        "artifacts": [artifact.to_dict()],
        "error": str  # Optional
    }
```

### Gateway Expectation (from main.go)
```go
type A2ATask struct {
    Artifacts interface{} `json:"artifacts,omitempty"`
    Error     *string    `json:"error,omitempty"`
}
```

**MATCH**: The response structures match between SDK and gateway.

## 3. Task Input/Output Formats

### Gateway to Worker
- **Basic Echo**: The gateway sends the message object directly as workflow input
- **Streaming Echo**: The gateway wraps the message with additional metadata:
  ```python
  {
      "gateway_workflow_id": str,  # For streaming
      "message": message_object
  }
  ```

### Worker Response
Both workflows return the same A2A v0.2.5 compliant structure:
```python
{
    "artifacts": [...],
    "error": str  # Optional
}
```

## 4. Workflow Names and Task Queue Names

### From echo_worker.py
- **Basic Echo Worker**:
  - Workflow: `EchoTaskWorkflow`
  - Task Queue: `echo-agent-tasks`
  - Activity: `echo_activity`
  
- **Streaming Echo Worker**:
  - Workflow: `StreamingEchoTaskWorkflow`
  - Task Queue: `streaming-echo-agent-tasks`
  - Activity: `echo_activity` (same as basic)

### From gateway configuration (agent-routing.yaml)
```yaml
"echo-agent":
  taskQueue: "echo-agent-tasks"
  workflowType: "EchoTaskWorkflow"

"streaming-echo-agent":
  taskQueue: "streaming-echo-agent-tasks"
  workflowType: "StreamingEchoTaskWorkflow"
```

**MATCH**: Workflow names and task queues match perfectly between worker and gateway configuration.

## 5. Key Differences and Issues

### Issue 1: Message ID Field
- Google A2A SDK includes `messageId` in Message object
- Gateway and workers don't expect or use this field
- **Impact**: Field is ignored but doesn't break functionality

### Issue 2: Part Structure
- Python SDK uses `"kind"` for part type
- Google SDK uses proper typed parts (TextPart, etc.)
- Both serialize to the same structure, so this is compatible

### Issue 3: Timestamp Handling
- Python SDK auto-generates timestamp if not provided
- Gateway doesn't pass timestamp to workers
- Workers generate their own timestamps using `workflow.now()`

### Issue 4: Error Field Type
- Python SDK: `error` is a string
- Gateway: `error` is a pointer to string (`*string`)
- This is compatible as JSON serialization handles both

## Recommendations

1. **Message ID**: The `messageId` field from Google SDK is safely ignored
2. **Timestamps**: Are handled correctly by workers using workflow time
3. **Part Types**: Both SDKs produce compatible JSON structures
4. **Error Handling**: Compatible between implementations

The data structures are largely compatible with only minor differences that don't affect functionality.

## Summary

### ✅ Compatible Elements
1. **Artifact Structure**: Both SDKs and the gateway use the same artifact format
2. **Task Result Format**: Consistent `{artifacts: [...], error: "..."}` structure
3. **Workflow/Queue Names**: Perfect match between worker and gateway config
4. **Part Types**: Both use `"kind": "text"` for text parts
5. **Activity Names**: Consistent activity naming (`echo_activity`)

### ⚠️ Minor Differences (Non-Breaking)
1. **Message ID**: Google SDK includes `messageId`, gateway/worker ignores it
2. **Timestamp Generation**: SDK auto-generates, workers use workflow time
3. **Error Type**: String vs pointer-to-string (compatible in JSON)
4. **Description Field**: Worker's Artifact class includes optional description

### 🔧 Integration Points
1. **Message Passing**: Gateway passes message object directly to workflow
2. **Streaming Support**: Gateway adds `gateway_workflow_id` for streaming workflows
3. **Progress Signals**: Use `WorkflowProgressSignal` for streaming updates
4. **Query Handler**: Workers expose `get_progress_signals` for progress queries

The integration between the Python SDK and A2A Gateway is functional with no critical mismatches.