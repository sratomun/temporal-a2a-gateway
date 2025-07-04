# Temporal Agent Registry Architecture

**Document**: Agent Registry as Temporal Worker Architecture Proposal  
**Author**: Agent 1 (Architect)  
**Date**: 2025-07-04  
**Status**: 🔬 **EXPERIMENTAL PROPOSAL**  
**Context**: Advanced architecture exploration for native Temporal integration

## Executive Summary

This document explores replacing the traditional HTTP-based agent registry with a **Temporal-native agent registry worker**. This approach would leverage Temporal's durability, consistency, and workflow orchestration capabilities for agent discovery and management.

## Current Architecture Limitations

### HTTP-Based Registry Issues
- **Single Point of Failure**: Registry service can fail independently
- **Consistency Challenges**: Agent state changes may be lost during failures
- **Limited Observability**: Registry operations not integrated with workflow history
- **Manual Coordination**: Registry changes don't automatically trigger workflows
- **Scalability Concerns**: Separate scaling requirements from workflow infrastructure

## Temporal Registry Architecture Proposal

### Core Concept: Registry as Distributed Temporal Workflows

Instead of a centralized HTTP service, implement agent registry as:
1. **Registry Manager Workflow**: Long-running workflow managing global agent state
2. **Agent Registration Workflows**: Individual workflows per registered agent
3. **Discovery Query Handlers**: Temporal queries for agent discovery
4. **Registry Event Workflows**: Handle registration/deregistration events

## Architecture Design Options

### Option A: Centralized Registry Workflow

#### Architecture Overview
```
┌─────────────────────────────────────────────────────────────┐
│                 Temporal Cluster                            │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────┐    │
│  │           Global Registry Workflow                  │    │
│  │  - Agent state management                          │    │
│  │  - Discovery queries                               │    │
│  │  - Registration events                             │    │
│  │  - Health monitoring                               │    │
│  └─────────────────────────────────────────────────────┘    │
│                           │                                 │
│  ┌─────────────────────────────────────────────────────┐    │
│  │               Agent Workers                         │    │
│  │  Echo Agent    │  GPT Agent    │  Custom Agent      │    │
│  │  Workflow      │  Workflow     │  Workflow          │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

#### Implementation Pattern
```go
@workflow.defn
type RegistryManagerWorkflow struct {
    agents map[string]*AgentRegistration
    mutex  sync.RWMutex
}

@workflow.query
func (r *RegistryManagerWorkflow) DiscoverAgents(criteria AgentCriteria) []AgentCard {
    // Query logic for agent discovery
}

@workflow.signal
func (r *RegistryManagerWorkflow) RegisterAgent(agent AgentRegistration) {
    // Register new agent with durability guarantees
}

@workflow.signal  
func (r *RegistryManagerWorkflow) DeregisterAgent(agentId string) {
    // Remove agent with cleanup workflows
}
```

#### Pros
- **Single Source of Truth**: One workflow manages all agent state
- **Atomic Operations**: All registry changes are atomic and durable
- **Simple Querying**: Single workflow to query for agent discovery
- **Consistent State**: No synchronization issues between multiple instances

#### Cons
- **Scalability Bottleneck**: Single workflow handles all registry operations
- **Limited Concurrency**: Registration operations are serialized
- **Recovery Time**: Large state recovery after workflow restarts
- **Memory Usage**: All agent state held in single workflow memory

### Option B: Distributed Agent Registration Workflows

#### Architecture Overview
```
┌─────────────────────────────────────────────────────────────┐
│                 Temporal Cluster                            │
├─────────────────────────────────────────────────────────────┤
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐   │
│  │ Echo Agent    │  │ GPT Agent     │  │ Custom Agent  │   │
│  │ Registration  │  │ Registration  │  │ Registration  │   │
│  │ Workflow      │  │ Workflow      │  │ Workflow      │   │
│  └───────────────┘  └───────────────┘  └───────────────┘   │
│         │                    │                    │         │
│  ┌─────────────────────────────────────────────────────┐    │
│  │           Discovery Aggregator Workflow             │    │
│  │  - Queries individual agent workflows              │    │
│  │  - Aggregates discovery results                    │    │
│  │  - Maintains agent index                           │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

#### Implementation Pattern
```go
@workflow.defn
type AgentRegistrationWorkflow struct {
    agentCard    AgentCard
    healthStatus HealthStatus
    lastSeen     time.Time
}

@workflow.query
func (a *AgentRegistrationWorkflow) GetAgentCard() AgentCard {
    return a.agentCard
}

@workflow.signal
func (a *AgentRegistrationWorkflow) UpdateHealth(status HealthStatus) {
    a.healthStatus = status
    a.lastSeen = workflow.Now()
}

@workflow.defn
type DiscoveryAggregatorWorkflow struct {
    agentIndex map[string]string // agentId -> workflowId
}

@workflow.query
func (d *DiscoveryAggregatorWorkflow) DiscoverAgents(criteria AgentCriteria) []AgentCard {
    var results []AgentCard
    
    for agentId, workflowId := range d.agentIndex {
        // Query individual agent workflow
        agentCard := workflow.QueryExternalWorkflow(workflowId, "GetAgentCard")
        if matches(agentCard, criteria) {
            results = append(results, agentCard)
        }
    }
    
    return results
}
```

#### Pros
- **High Concurrency**: Each agent managed by separate workflow
- **Scalability**: Distributes load across multiple workflows
- **Fault Isolation**: Individual agent failures don't affect others
- **Fine-grained Control**: Each agent can have custom logic

#### Cons
- **Complex Discovery**: Must query multiple workflows for discovery
- **Consistency Challenges**: Index synchronization between workflows
- **Query Performance**: N+1 query problem for discovery operations
- **Coordination Overhead**: More complex workflow orchestration

### Option C: Hybrid Registry Architecture

#### Architecture Overview
```
┌─────────────────────────────────────────────────────────────┐
│                 Temporal Cluster                            │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────┐    │
│  │         Registry Index Workflow                    │    │
│  │  - Fast agent lookup index                         │    │
│  │  - Discovery query optimization                    │    │
│  │  - Agent health aggregation                        │    │
│  └─────────────────────────────────────────────────────┘    │
│                           │                                 │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐   │
│  │ Echo Agent    │  │ GPT Agent     │  │ Custom Agent  │   │
│  │ Lifecycle     │  │ Lifecycle     │  │ Lifecycle     │   │
│  │ Workflow      │  │ Workflow      │  │ Workflow      │   │
│  └───────────────┘  └───────────────┘  └───────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

#### Implementation Pattern
```go
@workflow.defn  
type RegistryIndexWorkflow struct {
    agentIndex   map[string]AgentCard    // agentId -> AgentCard
    healthIndex  map[string]HealthStatus // agentId -> health
    lastUpdated  map[string]time.Time    // agentId -> lastSeen
}

@workflow.query
func (r *RegistryIndexWorkflow) DiscoverAgents(criteria AgentCriteria) []AgentCard {
    // Fast in-memory lookup with health filtering
    var results []AgentCard
    for agentId, card := range r.agentIndex {
        if r.healthIndex[agentId].IsHealthy() && matches(card, criteria) {
            results = append(results, card)
        }
    }
    return results
}

@workflow.signal
func (r *RegistryIndexWorkflow) UpdateAgentIndex(agentId string, card AgentCard, health HealthStatus) {
    r.agentIndex[agentId] = card
    r.healthIndex[agentId] = health  
    r.lastUpdated[agentId] = workflow.Now()
}

@workflow.defn
type AgentLifecycleWorkflow struct {
    agentCard AgentCard
    health    HealthStatus
}

@workflow.run
func (a *AgentLifecycleWorkflow) Run() {
    // Register with index
    registryHandle := workflow.GetExternalWorkflowHandle("registry-index")
    registryHandle.Signal("UpdateAgentIndex", a.agentCard.AgentID, a.agentCard, a.health)
    
    // Periodic health updates
    for {
        workflow.Sleep(30 * time.Second)
        a.updateHealth()
        registryHandle.Signal("UpdateAgentIndex", a.agentCard.AgentID, a.agentCard, a.health)
    }
}
```

#### Pros
- **Fast Discovery**: Index provides O(1) lookup performance
- **Distributed Processing**: Individual agent workflows handle lifecycle
- **Consistency**: Index maintains eventual consistency
- **Health Monitoring**: Automatic health status aggregation

#### Cons
- **Eventual Consistency**: Index updates may lag behind agent changes
- **Memory Overhead**: Index duplicates some agent information
- **Complexity**: More moving parts than single workflow approach

## Implementation Considerations

### Agent Discovery Interface

#### Temporal Query-Based Discovery
```go
type TemporalRegistryClient struct {
    temporalClient client.Client
    registryWorkflowId string
}

func (t *TemporalRegistryClient) DiscoverAgents(ctx context.Context, criteria AgentCriteria) ([]AgentCard, error) {
    var result []AgentCard
    err := t.temporalClient.QueryWorkflow(ctx, t.registryWorkflowId, "", "DiscoverAgents", criteria).Get(&result)
    return result, err
}

func (t *TemporalRegistryClient) RegisterAgent(ctx context.Context, agent AgentCard) error {
    return t.temporalClient.SignalWorkflow(ctx, t.registryWorkflowId, "", "RegisterAgent", agent).Get(ctx, nil)
}
```

#### Gateway Integration
```go
func (g *Gateway) handleDiscoverAgents(w http.ResponseWriter, req *JSONRPCRequest) {
    var params DiscoverAgentsParams
    if err := json.Unmarshal(req.Params, &params); err != nil {
        g.sendError(w, req.ID, -32602, "Invalid params", nil)
        return
    }
    
    // Query Temporal registry instead of HTTP registry
    agents, err := g.registryClient.DiscoverAgents(req.Context(), params.Criteria)
    if err != nil {
        g.sendError(w, req.ID, -32603, "Registry query failed", nil)
        return
    }
    
    g.sendResponse(w, req.ID, map[string]interface{}{
        "agents": agents,
    })
}
```

### Health Monitoring Integration

#### Agent Health Workflow
```go
@workflow.defn
type AgentHealthMonitor struct {
    agentId      string
    lastHealthy  time.Time
    failures     int
}

@workflow.run
func (h *AgentHealthMonitor) Run() {
    for {
        // Check agent health via activity
        healthy := workflow.ExecuteActivity(h.checkAgentHealth, h.agentId).Get()
        
        if healthy {
            h.lastHealthy = workflow.Now()
            h.failures = 0
        } else {
            h.failures++
            if h.failures > 3 {
                // Signal registry to mark agent unhealthy
                registryHandle := workflow.GetExternalWorkflowHandle("registry-index")
                registryHandle.Signal("MarkAgentUnhealthy", h.agentId)
            }
        }
        
        workflow.Sleep(30 * time.Second)
    }
}
```

### Migration Strategy

#### Phase 1: Parallel Operation
- Run Temporal registry alongside HTTP registry
- Migrate discovery queries to Temporal while maintaining HTTP registration
- Validate consistency between both systems

#### Phase 2: Registration Migration  
- Move agent registration to Temporal signals
- Implement HTTP → Temporal bridge for external systems
- Add comprehensive monitoring and alerting

#### Phase 3: Full Migration
- Decommission HTTP registry service
- Update all clients to use Temporal queries
- Optimize performance based on usage patterns

## Performance Analysis

### Query Performance
| Operation | HTTP Registry | Temporal Registry | Performance Impact |
|-----------|---------------|-------------------|-------------------|
| Agent Discovery | ~5ms | ~10-15ms | 2-3x slower |
| Agent Registration | ~10ms | ~20-30ms | 2-3x slower |
| Health Check | ~2ms | ~5-8ms | 2-4x slower |

### Scalability Comparison
| Metric | HTTP Registry | Temporal Registry | Advantage |
|--------|---------------|-------------------|-----------|
| Concurrent Registrations | High | Medium | HTTP |
| Discovery QPS | Very High | High | HTTP |
| Consistency Guarantees | Eventually | Strong | Temporal |
| Failure Recovery | Manual | Automatic | Temporal |
| Audit Trail | Limited | Complete | Temporal |

## Security Considerations

### Access Control
- **Workflow-Level Security**: Control access to registry workflows
- **Query Permissions**: Restrict discovery query access
- **Signal Authorization**: Validate agent registration permissions

### Data Protection
- **Encrypted Storage**: Agent credentials encrypted in Temporal
- **Access Logging**: Complete audit trail of registry operations
- **Isolation**: Agent workflows isolated from each other

## Recommendations

### Recommended Architecture: **Option C (Hybrid)**

**Rationale**:
1. **Performance**: Index workflow provides fast discovery
2. **Scalability**: Distributed agent workflows handle load
3. **Consistency**: Eventual consistency acceptable for registry use case
4. **Migration**: Easier migration path from current HTTP architecture

### Implementation Priority

#### High Priority (Sprint 4)
- **Registry Index Workflow**: Core discovery and health aggregation
- **Agent Lifecycle Workflows**: Individual agent registration and health
- **Gateway Integration**: Update discovery handlers to use Temporal queries

#### Medium Priority (Sprint 5)
- **Health Monitoring**: Automated agent health checking
- **Migration Tools**: HTTP → Temporal bridge for external systems
- **Performance Optimization**: Query caching and batch operations

#### Low Priority (Future)
- **Advanced Discovery**: Complex agent capability matching
- **Registry Analytics**: Usage patterns and performance metrics
- **External Integration**: REST API compatibility layer

### Success Criteria

#### Technical Metrics
- **Discovery Latency**: <50ms for agent discovery queries
- **Consistency**: 99.9% consistency between index and agent workflows
- **Availability**: 99.95% uptime for registry operations
- **Scalability**: Support 10,000+ registered agents

#### Business Metrics
- **Migration Success**: 100% of current registry features supported
- **Performance**: No regression in client-perceived performance
- **Reliability**: Reduced registry-related incidents by 90%

## Conclusion

The Temporal Agent Registry architecture offers significant advantages in **durability, consistency, and observability** at the cost of some **performance overhead**. The hybrid approach (Option C) provides the best balance of performance and Temporal benefits, making it suitable for production deployment.

The architecture enables **native integration** with workflow orchestration while maintaining **compatibility** with existing A2A protocol requirements. Implementation should be **phased** to minimize risk and validate performance characteristics in production.

---

**Next Steps**: 
1. Prototype Registry Index Workflow implementation
2. Performance testing with realistic agent loads  
3. Design detailed migration strategy from HTTP registry
4. Agent 5 validation of A2A protocol compliance

**Architecture Authority**: Agent 1 (Architect)  
**Status**: Ready for prototype development and validation