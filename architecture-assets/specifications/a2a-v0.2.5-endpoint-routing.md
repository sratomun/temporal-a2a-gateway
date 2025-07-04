# A2A v0.2.5 Agent Endpoint Routing Specification

**Version**: A2A Protocol v0.2.5  
**Document**: Agent Endpoint URL Format and Routing Requirements  
**Author**: Agent 5 (Standardization Engineer)  
**Date**: 2025-07-04  
**Status**: ✅ DEFINITIVE SPECIFICATION REFERENCE

## Overview

This document provides the definitive A2A v0.2.5 agent endpoint routing specification. All gateway implementations must follow these URL format requirements to achieve specification compliance and Google SDK compatibility.

## Critical Specification Requirement

### ✅ CORRECT: Agent-Specific Base URLs

A2A v0.2.5 requires each agent to have its own **dedicated endpoint URL**:

```
✅ COMPLIANT: https://gateway.example.com/echo-agent     (for echo-agent)
✅ COMPLIANT: https://gateway.example.com/math-agent     (for math-agent)  
✅ COMPLIANT: https://gateway.example.com/code-agent     (for code-agent)
```

### ❌ INCORRECT: Parameter-Based Routing

These URL patterns violate A2A v0.2.5 specification:

```
❌ NON-COMPLIANT: /a2a?agentId=echo-agent
❌ NON-COMPLIANT: /agents/{agentId}/a2a
❌ NON-COMPLIANT: /{agentId}/a2a
```

## A2A v0.2.5 Endpoint Requirements

### 1. Dedicated Agent URLs

**Specification Requirement**: Each agent has its own complete base URL

- **Example**: `"url": "https://georoute-agent.example.com/a2a/v1"`
- **Structure**: Complete HTTPS URL specified in agent's `AgentCard`
- **Discovery**: Optional well-known URI at `https://{server_domain}/.well-known/agent.json`

### 2. No AgentID in URL Path

**Specification Requirement**: Agent identity is implicit in the endpoint URL

- **A2A Compliant**: Each agent has separate base URL (no agentId routing)
- **NOT A2A Compliant**: Using agentId in URL path or parameters
- **Rationale**: Dedicated endpoints ensure clean agent discovery and routing

### 3. Request Parameter Handling

**Specification Requirement**: AgentID is NOT passed as a parameter

- Agent identity is implicit in the endpoint URL
- JSON-RPC 2.0 methods sent via HTTP POST to agent's base URL
- `Content-Type: application/json` required

## Implementation Requirements

### For Gateway Developers

#### Remove Parameter-Based Routing

```go
// ❌ REMOVE THESE - Non-A2A compliant patterns
router.HandleFunc("/a2a", handler)                    // with agentId parameter
router.HandleFunc("/agents/{agentId}/a2a", handler)   // with agentId in path
```

#### Implement Agent-Specific Endpoints

```go
// ✅ A2A v0.2.5 compliant agent-specific routing
router.HandleFunc("/{agentId}", g.handleAgentSpecificEndpoint).Methods("POST")

func (g *Gateway) handleAgentSpecificEndpoint(w http.ResponseWriter, r *http.Request) {
    // Extract agentId from URL path
    vars := mux.Vars(r)
    agentID := vars["agentId"]
    
    // Route to agent-specific handler
    g.routeToAgent(w, r, agentID)
}
```

#### Agent-Specific Endpoint Examples

```go
// Dynamic routing for all agents
func (g *Gateway) setupAgentRoutes() {
    // Single route pattern handles all agents
    g.router.HandleFunc("/{agentId}", g.handleAgentSpecificEndpoint).Methods("POST")
    
    // Health endpoint separate from agent routing
    g.router.HandleFunc("/health", g.handleHealth).Methods("GET")
}
```

### Agent Card URL Updates

#### Correct Agent Card Declaration

```json
{
  "name": "Echo Agent",
  "description": "Simple echo agent for testing",
  "version": "1.0.0",
  "url": "https://gateway.example.com/echo-agent",
  "capabilities": {
    "streaming": true
  }
}
```

#### URL Format Requirements

- **Base URL**: Complete HTTPS URL for the specific agent
- **Path**: Agent identifier as path component
- **No Parameters**: No agentId query parameters required
- **Dedicated**: Each agent gets its own unique URL

### Request Format Changes

#### A2A Compliant Request Format

```bash
# ✅ COMPLIANT: Agent identity from URL, no agentId parameter
curl -X POST https://gateway.example.com/echo-agent \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
      "message": {
        "parts": [{"text": "Hello"}]
      }
    },
    "id": "test"
  }'
```

#### Non-Compliant Legacy Format

```bash
# ❌ NON-COMPLIANT: Using agentId parameter (deprecated)
curl -X POST https://gateway.example.com/a2a \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send", 
    "params": {
      "agentId": "echo-agent",
      "message": {...}
    }
  }'
```

## Google SDK Compatibility

### SDK Configuration

```python
# ✅ Google A2A SDK with agent-specific URL
from google.ai.a2a import A2AClient

# Configure client with dedicated agent endpoint
client = A2AClient(base_url="http://localhost:8080/echo-agent")

# No agentId parameter needed in requests
response = client.send_message(SendMessageRequest(
    params=MessageSendParams(message=message)
))
```

### SDK Integration Benefits

1. **No Parameter Confusion**: SDK doesn't need to manage agentId parameters
2. **Clean URLs**: Each agent has clear, dedicated endpoint
3. **Standard Compliance**: Follows A2A v0.2.5 specification exactly
4. **Future-Proof**: Compatible with agent discovery mechanisms

## Migration Strategy

### Phase 1: Add Agent-Specific Endpoints

```go
// Add new A2A compliant routing alongside existing
func (g *Gateway) setupRoutes() {
    // New A2A compliant endpoints
    g.router.HandleFunc("/{agentId}", g.handleAgentSpecificEndpoint).Methods("POST")
    
    // Legacy endpoints (deprecated)
    g.router.HandleFunc("/a2a", g.handleLegacyEndpoint).Methods("POST")
}
```

### Phase 2: Update Agent Cards

```json
{
  "url": "https://gateway.example.com/echo-agent"  // Updated to agent-specific URL
}
```

### Phase 3: Deprecate Legacy Endpoints

```go
func (g *Gateway) handleLegacyEndpoint(w http.ResponseWriter, r *http.Request) {
    // Add deprecation headers
    w.Header().Set("Deprecation", "true")
    w.Header().Set("Sunset", "2024-10-03T00:00:00Z")
    
    // Process request with deprecation warnings
    g.processLegacyRequest(w, r)
}
```

## Validation Requirements

### Implementation Checklist

**Phase 1: Agent-Specific Routing**
- [ ] Implement `/{agentId}` route pattern
- [ ] Extract agentId from URL path, not parameters
- [ ] Route requests to appropriate agent handlers
- [ ] Remove agentId from request parameters

**Phase 2: Agent Card Updates**  
- [ ] Update agent card URLs to dedicated endpoints
- [ ] Remove agentId parameter dependencies
- [ ] Test agent discovery with new URLs
- [ ] Validate SDK integration works

**Phase 3: Legacy Deprecation**
- [ ] Add deprecation headers to legacy endpoints
- [ ] Provide migration guidance in responses
- [ ] Plan sunset timeline for legacy routing
- [ ] Monitor usage of deprecated endpoints

### Compliance Tests

1. ✅ Agent-specific URLs work without agentId parameters
2. ✅ Agent cards reference correct dedicated endpoints  
3. ✅ Google SDK integration successful with new URLs
4. ✅ Legacy endpoints include deprecation warnings
5. ✅ All A2A methods work on agent-specific endpoints

### Common Violations

❌ Using agentId query parameters  
❌ Generic `/a2a` endpoint requiring agentId  
❌ Agent cards with non-dedicated URLs  
❌ Missing agent-specific routing logic  
❌ Breaking Google SDK compatibility  

## Agent Discovery

### Well-Known URI Pattern (Optional)

```
https://gateway.example.com/.well-known/agent.json
```

### Agent Registry Response

```json
{
  "agents": [
    {
      "id": "echo-agent",
      "name": "Echo Agent", 
      "url": "https://gateway.example.com/echo-agent"
    },
    {
      "id": "math-agent",
      "name": "Math Agent",
      "url": "https://gateway.example.com/math-agent"
    }
  ]
}
```

## References

- A2A Protocol v0.2.5 Specification
- Agent Card documentation
- Google A2A SDK integration guide
- JSON-RPC 2.0 specification

---

**Standardization Authority**: Agent 5 (Standardization Engineer)  
**Implementation Status**: Required for A2A v0.2.5 compliance  
**Last Updated**: 2025-07-04