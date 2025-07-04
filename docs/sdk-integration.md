# Google A2A SDK Integration Guide

This document explains the A2A v0.2.5 specification-compliant patterns used in our Google A2A SDK integration, providing guidance for implementing proper A2A clients.

## A2A v0.2.5 Client Implementation

Our Google A2A SDK integration demonstrates complete A2A v0.2.5 specification compliance. The implementation patterns follow the protocol's design philosophy of JSON-first communication with maximum flexibility.

## A2A Protocol Design Philosophy

### JSON-First Architecture

The A2A v0.2.5 specification is fundamentally designed as a **JSON-based communication protocol**:

- **Data Format**: All agent interactions use JSON structures
- **Transport**: Standard HTTP with JSON-RPC 2.0 envelope  
- **Processing**: Direct JSON manipulation is the **intended** interaction model
- **Flexibility**: JSON structures allow infinite extensibility without breaking compatibility

### SDK Responsibility Scope

A2A client SDKs are designed as **transport layers only**:

**✅ SDK Responsibilities:**
- HTTP communication and network handling
- Authentication and security management
- JSON-RPC 2.0 protocol compliance
- Request serialization and validation

**❌ NOT SDK Responsibilities:**
- Data abstraction or object mapping
- Workflow management or polling control
- Content interpretation or parsing
- State management or caching

## A2A v0.2.5 Specification Patterns

### 1. JSON-RPC Response Parsing with SDK Types

The A2A v0.2.5 specification supports both direct JSON parsing and SDK type objects:

```python
# A2A v0.2.5 compliant response parsing with SDK types
response_data = task_response.model_dump()
task_data = response_data.get('result', {})

# Parse as Task object using SDK types
task = Task(**task_data)
task_id = task.id
```

**Specification Requirements:**
- **JSON-RPC 2.0**: Clients extract the `result` field from JSON-RPC responses
- **Task Structure**: A2A defines tasks as JSON objects with standardized fields
- **SDK Types**: SDKs may provide type objects for structured access
- **Flexibility**: Both direct JSON and typed access are specification-compliant

### 2. Task Status Monitoring with SDK Types

A2A v0.2.5 task status can be accessed through SDK type objects:

```python
# A2A specification-compliant status access using SDK types
task_status: TaskStatus = task.status
state = task_status.state

# State comparison using SDK enums
if state in [TaskState.completed]:
    # Handle completion
```

**Specification Design:**
- **Status Object**: Task status structure with `state` and `timestamp` fields
- **SDK Types**: SDKs may provide `TaskStatus` and `TaskState` objects
- **State Values**: Specification defines standard states as enum values
- **Type Safety**: SDK types provide compile-time validation while maintaining flexibility

### 3. Client-Controlled Polling

The A2A v0.2.5 specification designates polling as the standard completion detection mechanism:

```python
# A2A specification-required polling pattern
while state not in ['completed', 'failed', 'canceled']:
    time.sleep(poll_interval)
    task_response = client.get_task(GetTaskRequest(id=task_id, params=TaskQueryParams(id=task_id)))
```

**Protocol Requirements:**
- **Polling Standard**: A2A v0.2.5 specifies polling for task completion detection
- **Client Control**: Applications manage polling frequency and termination logic
- **Agent Diversity**: Task completion times vary significantly across agent types
- **Use Case Flexibility**: Different applications require different polling strategies

### 4. Artifact Processing with SDK Types

A2A v0.2.5 artifacts can be accessed through SDK type objects while maintaining flexibility:

```python
# A2A specification-compliant artifact processing with SDK types
if task.artifacts:
    for artifact in task.artifacts:
        artifact_name = artifact.name
        for part in artifact.parts:
            # SDK types provide structured access while preserving flexibility
            if hasattr(part.root, 'text'):
                text_content = part.root.text
```

**Specification Design:**
- **Artifact Schema**: Flexible structures accommodate diverse content types
- **SDK Types**: Artifact and Part objects provide structured access
- **Part Types**: Extensible design supports text, files, data, and future types
- **Future Compatibility**: SDK types gracefully handle unknown part types

## A2A Protocol Design Rationale

### JSON-First Protocol Design

The A2A v0.2.5 specification follows established patterns for JSON-based protocols:

**JSON-RPC 2.0 Foundation**: Built on standard JSON-RPC 2.0 for maximum compatibility
**Direct Field Access**: JSON structures accessed directly for flexibility
**Universal Parsing**: JSON parsing is consistent across programming languages
**Extensible Schema**: Unknown fields and types handled gracefully

### Specification Benefits

The A2A v0.2.5 design prioritizes flexibility to support diverse AI agent ecosystems:

1. **Agent Diversity**: Accommodates vastly different agent response structures
2. **Evolution Support**: New capabilities don't break existing clients
3. **Performance**: Direct JSON access avoids object mapping overhead
4. **Debugging**: Raw JSON responses provide clear visibility
5. **Interoperability**: Works across all A2A-compliant implementations

## A2A Specification Compliance

The A2A v0.2.5 specification defines clear boundaries for compliant implementations:

**✅ Specification-Compliant Patterns:**
- Using standard JSON-RPC 2.0 methods and formats
- Direct JSON field access for task data
- Client-controlled polling for completion detection
- Flexible artifact parsing for unknown content types

**❌ Specification Violations:**
- Non-standard JSON-RPC request formats
- Proprietary extensions to A2A methods
- Agent-specific hardcoded logic
- Breaking compatibility with unknown artifact types

## Implementation Examples

### Complete A2A v0.2.5 Compliant Client

```python
import asyncio
import httpx
from a2a.client import A2AClient
from a2a.types import (
    Message, TextPart, SendMessageRequest, MessageSendParams,
    GetTaskRequest, Task, TaskStatus, TaskState
)

async def a2a_compliant_interaction():
    """Demonstrates A2A v0.2.5 specification compliance with SDK types"""
    
    async with httpx.AsyncClient() as http_client:
        # ✅ SDK handles transport layer
        client = A2AClient(
            httpx_client=http_client,
            url="http://localhost:8080/agents/echo-agent/a2a"
        )
        
        # ✅ SDK provides type safety for requests
        message = Message(
            messageId="example-001",
            role="user", 
            parts=[TextPart(text="Hello A2A Protocol!")]
        )
        
        params = MessageSendParams(message=message)
        request = SendMessageRequest(id="req-001", params=params)
        
        # ✅ SDK handles JSON-RPC communication
        task_response = await client.send_message(request)
        
        # ✅ SPECIFICATION-COMPLIANT: JSON-RPC result extraction with SDK types
        response_data = task_response.model_dump()
        task_data = response_data.get('result', {})
        task = Task(**task_data)
        task_id = task.id
        
        # ✅ SPECIFICATION-COMPLIANT: Client-controlled polling
        max_attempts = 30
        poll_interval = 1.0
        
        for attempt in range(max_attempts):
            # ✅ SDK handles transport, client handles workflow
            get_request = GetTaskRequest(
                id=f"poll-{attempt}",
                params={"id": task_id}
            )
            status_response = await client.get_task(get_request)
            
            # ✅ SPECIFICATION-COMPLIANT: SDK types for structured access
            response_data = status_response.model_dump()
            task_data = response_data.get('result', {})
            task = Task(**task_data)
            
            # ✅ SDK types provide type-safe status access
            task_status: TaskStatus = task.status
            state = task_status.state
            
            if state in [TaskState.completed, TaskState.failed, TaskState.canceled]:
                break
                
            await asyncio.sleep(poll_interval)
        
        # ✅ SPECIFICATION-COMPLIANT: SDK types for artifact processing
        if task.artifacts:
            for artifact in task.artifacts:
                artifact_name = artifact.name
                
                for part in artifact.parts:
                    # ✅ SDK types with graceful fallback for unknown types
                    if hasattr(part.root, 'text'):
                        text_content = part.root.text
                        print(f"📄 {artifact_name}: {text_content}")
                    elif hasattr(part.root, 'file'):
                        file_info = part.root.file
                        print(f"📁 {artifact_name}: {file_info.name}")
                    else:
                        # ✅ Graceful handling of unknown part types
                        print(f"❓ {artifact_name}: Unknown part type")

# Usage
asyncio.run(a2a_compliant_interaction())
```

### Enhanced Error Handling

```python
async def robust_a2a_client():
    """Demonstrates production-ready A2A error handling"""
    
    try:
        # ... task creation ...
        
        # ✅ SPECIFICATION-COMPLIANT: Robust polling with exponential backoff
        base_delay = 1.0
        max_delay = 30.0
        max_attempts = 50
        
        for attempt in range(max_attempts):
            try:
                status_response = await client.get_task(get_request)
                task_result = status_response.model_dump().get('result', {})
                
                # ✅ Handle A2A-compliant error states
                state = task_result.get('status', {}).get('state', '').lower()
                error_message = task_result.get('error')
                
                if state == 'completed':
                    artifacts = task_result.get('artifacts', [])
                    return process_artifacts(artifacts)
                    
                elif state == 'failed':
                    raise Exception(f"Task failed: {error_message}")
                    
                elif state == 'canceled':
                    raise Exception("Task was canceled")
                    
                # ✅ Exponential backoff for active tasks
                delay = min(base_delay * (2 ** attempt), max_delay)
                await asyncio.sleep(delay)
                
            except Exception as poll_error:
                # ✅ Graceful handling of transient errors
                if attempt == max_attempts - 1:
                    raise Exception(f"Polling failed after {max_attempts} attempts: {poll_error}")
                await asyncio.sleep(base_delay)
                
    except Exception as client_error:
        # ✅ Comprehensive error reporting
        print(f"A2A client error: {client_error}")
        raise

def process_artifacts(artifacts):
    """✅ SPECIFICATION-COMPLIANT: Flexible artifact processing"""
    results = []
    
    for artifact in artifacts:
        artifact_data = {
            'id': artifact.get('artifactId'),
            'name': artifact.get('name', 'Unnamed'),
            'description': artifact.get('description', ''),
            'content': []
        }
        
        # ✅ Handle any part type gracefully
        for part in artifact.get('parts', []):
            part_kind = part.get('kind', 'unknown')
            
            if part_kind == 'text':
                artifact_data['content'].append({
                    'type': 'text',
                    'data': part.get('text', '')
                })
            elif part_kind == 'file':
                file_info = part.get('file', {})
                artifact_data['content'].append({
                    'type': 'file',
                    'name': file_info.get('name'),
                    'uri': file_info.get('uri'),
                    'mimeType': file_info.get('mimeType')
                })
            elif part_kind == 'data':
                artifact_data['content'].append({
                    'type': 'data',
                    'data': part.get('data', {})
                })
            else:
                # ✅ Future-proof: Preserve unknown part types
                artifact_data['content'].append({
                    'type': 'unknown',
                    'kind': part_kind,
                    'raw': part
                })
        
        results.append(artifact_data)
    
    return results
```

## FAQ: Addressing Developer Concerns

### Q: Why use direct JSON parsing instead of SDK convenience methods?

**A**: The A2A v0.2.5 specification defines SDKs as transport layers, not abstraction layers:
- SDKs handle JSON-RPC communication and serialization
- Clients handle JSON parsing for maximum flexibility
- Direct parsing supports diverse agent implementations
- Specification prioritizes interoperability over convenience

### Q: Is client-controlled polling required by the specification?

**A**: Yes, A2A v0.2.5 specifies polling as the standard completion detection mechanism:
- Agents have varying execution times (seconds to hours)
- Applications control retry logic and timeout handling
- Different use cases require different polling strategies
- Protocol design emphasizes client workflow control

### Q: Why parse artifact structures directly?

**A**: The A2A v0.2.5 artifact schema is designed for maximum extensibility:
- Supports unlimited artifact and part types
- Handles unknown future content formats gracefully
- Provides debugging visibility into agent responses
- Ensures forward compatibility with protocol evolution

### Q: How does this approach ensure A2A compliance?

**A**: These patterns achieve complete A2A v0.2.5 specification compliance:
- Follow JSON-RPC 2.0 standard exactly
- Use only standardized A2A methods and fields
- Handle all defined task states and error conditions
- Support the complete artifact schema as specified
- Maintain interoperability with any A2A-compliant agent

## Anti-Patterns to Avoid

### ❌ Over-Abstraction

```python
# WRONG: Adding unnecessary abstraction layers
class TaskWrapper:
    def is_completed(self):
        # This breaks with unknown task states
        return self.state == 'completed'
    
    def get_text_content(self):
        # This assumes specific artifact structure
        return self.artifacts[0]['parts'][0]['text']
```

### ❌ Agent-Specific Logic

```python
# WRONG: Hardcoding agent-specific behavior
if agent_id == 'echo-agent':
    # Special handling breaks interoperability
    result = parse_echo_response(task_result)
elif agent_id == 'llm-agent':
    result = parse_llm_response(task_result)
```

### ❌ Protocol Extensions

```python
# WRONG: Adding non-standard fields
request_data = {
    "jsonrpc": "2.0",
    "method": "message/send", 
    "params": {...},
    "custom_timeout": 30,  # Not part of A2A spec
    "proprietary_flag": True  # Breaks compatibility
}
```

## A2A v0.2.5 Compliance Verification

Our implementation demonstrates complete A2A v0.2.5 specification compliance:

- ✅ **Google SDK Compatibility**: Full integration working correctly
- ✅ **JSON-RPC 2.0 Compliance**: All requests follow standard format
- ✅ **Protocol Method Support**: Complete implementation of required A2A methods
- ✅ **Artifact Schema Compliance**: Proper handling of A2A artifact structures
- ✅ **Error Handling**: Correct processing of A2A error responses
- ✅ **Task State Management**: Support for all specified task states

## Conclusion

The Google A2A SDK integration example demonstrates complete A2A v0.2.5 specification compliance through patterns that:

1. **Follow JSON-first protocol design** as specified in A2A v0.2.5
2. **Leverage SDK transport capabilities** while maintaining specification flexibility  
3. **Support maximum agent diversity** through compliant JSON parsing
4. **Future-proof applications** against protocol evolution
5. **Implement established patterns** for JSON-RPC 2.0 protocols

These patterns serve as a **reference implementation** for A2A v0.2.5 client development, demonstrating the specification's design philosophy of prioritizing flexibility and interoperability over convenience abstractions.

---

**Related Documentation:**
- [API Reference](./api.md) - Complete A2A Protocol methods
- [Streaming Guide](./streaming.md) - Real-time streaming capabilities  
- [Implementation Guide](./implementation.md) - Architecture details

**Specification References:**
- [A2A Protocol v0.2.5 Specification](https://a2aproject.github.io/A2A/v0.2.5/specification/)
- JSON-RPC 2.0 Specification
- Google A2A SDK Documentation