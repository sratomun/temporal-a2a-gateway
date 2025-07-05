# Sprint 4: Temporal A2A SDK Prototype

**Document**: Sprint 4 Implementation Plan - Temporal A2A SDK Prototype  
**Author**: Agent 1 (Architect)  
**Date**: 2025-07-04  
**Sprint**: Sprint 4 (Week 4-5)  
**Status**: 🎯 **FOCUSED IMPLEMENTATION PLAN**  
**PM Priority**: **#1 Sprint 4 Objective**

## Executive Summary

Following Agent 6 (Project Manager) directive, Sprint 4 will focus exclusively on **prototyping the Temporal A2A SDK abstraction layer**. This will create a developer-friendly SDK that hides all Temporal complexity while providing the complete A2A protocol interface with the revolutionary benefits achieved in Sprint 3.

## Sprint 4 Focus: SDK Prototype Only

### PM-Specified Sprint 4 Priorities
1. ✅ **Temporal A2A SDK Prototype** ← **Agent 1 Focus**
2. Agent Registry Enhancement ← Agent 2 Focus  
3. Performance Optimization ← Agent 3 Focus

### Agent 1 Singular Mission
**Create a working prototype SDK** that demonstrates:
- **Zero Temporal Knowledge Required** for developers
- **Complete A2A Protocol Support** (all operations)
- **Google A2A SDK Compatibility** (drop-in replacement)
- **Leverages Sprint 3 Achievements** (workflow-to-workflow signals)

## Prototype Scope Definition

### Core Prototype Components

#### 1. Echo Worker SDK Wrapper (Building on Existing Code)
```python
# Target Developer Experience (Sprint 4 Goal) - Wraps existing echo_worker.py
from temporal_a2a_sdk import Agent, A2AMessage, A2AResponse

class EchoAgent(Agent):
    def __init__(self):
        super().__init__(
            agent_id="echo-agent", 
            name="Echo Agent",
            capabilities={"streaming": False}
        )
    
    @agent.message_handler
    def handle_message(self, message: A2AMessage) -> A2AResponse:
        # Developer writes simple code
        return A2AResponse.text(f"Echo: {message.get_text()}")
        # SDK automatically maps this to existing EchoTaskWorkflow

class StreamingEchoAgent(Agent):
    def __init__(self):
        super().__init__(
            agent_id="streaming-echo-agent",
            name="Streaming Echo Agent", 
            capabilities={"streaming": True, "progressive_artifacts": True}
        )
    
    @agent.streaming_handler
    async def handle_streaming_message(self, message: A2AMessage) -> AsyncGenerator[A2AResponse, None]:
        # Developer writes simple streaming code
        words = f"Echo: {message.get_text()}".split()
        for i, word in enumerate(words):
            current_text = " ".join(words[:i+1])
            yield A2AResponse.partial_text(current_text, is_final=(i == len(words)-1))
            await asyncio.sleep(0.5)
        # SDK automatically maps this to existing StreamingEchoTaskWorkflow

# SDK reuses proven echo_worker.py workflows internally
if __name__ == "__main__":
    echo_agent = EchoAgent()
    streaming_agent = StreamingEchoAgent()
    
    # Both agents run using existing echo_worker workflows
    await asyncio.gather(
        echo_agent.run(),  # Uses EchoTaskWorkflow
        streaming_agent.run()  # Uses StreamingEchoTaskWorkflow
    )
```

#### 2. Complete A2A Client Interface
```python
# Google A2A SDK Compatibility (Sprint 4 Goal)
from temporal_a2a_sdk import A2AClient

client = A2AClient()  # Connects to Temporal, not HTTP

# All A2A operations work identically to Google SDK
task = client.send_message("echo-agent", A2AMessage.text("Hello"))
status = client.get_task(task.id)
client.cancel_task(task.id)

# Progressive streaming (leveraging Sprint 3 achievements)
for update in client.stream_message("echo-agent", A2AMessage.text("Stream this")):
    print(f"Progress: {update}")
```

#### 3. Message/Response Abstractions
```python
# Developer-friendly wrappers hiding A2A complexity
class A2AMessage:
    def get_text(self) -> str: ...
    def get_files(self) -> List[dict]: ...
    
class A2AResponse:
    @staticmethod
    def text(content: str) -> 'A2AResponse': ...
    
    @staticmethod  
    def streaming_text(content: str, is_final: bool = False) -> 'A2AResponse': ...
```

## Prototype Implementation Plan

### Week 4 (Days 1-5): Echo Worker SDK Migration Priority

#### Day 1-2: Echo Worker SDK Wrapper
- **Analyze existing echo_worker.py** (already has A2A artifacts, streaming patterns)
- **Create SDK wrapper** for existing EchoTaskWorkflow and StreamingEchoTaskWorkflow  
- **Agent class that generates** these proven workflows automatically

#### Day 3-4: SDK Interface for Echo Worker
- **EchoAgent SDK class** that hides workflow complexity
- **Automatic workflow registration** using existing echo_worker patterns
- **Message/Response abstractions** compatible with current artifact structures

#### Day 5: Echo Agent SDK Demo
- **Working Echo agent via SDK** (`@agent.message_handler` → EchoTaskWorkflow)
- **Streaming echo agent via SDK** (`@agent.streaming_handler` → StreamingEchoTaskWorkflow)  
- **Full A2A compatibility** leveraging existing proven implementation

### Week 5 (Days 6-10): Prototype Refinement

#### Day 6-7: Advanced Features
- **Streaming support** leveraging Sprint 3 workflow signals
- **State management** (persistent agent memory)
- **Error handling** and edge cases

#### Day 8-9: Developer Experience
- **Documentation and examples**
- **Testing framework** for SDK users
- **Installation and setup** process

#### Day 10: Sprint 4 Demo
- **Working prototype demonstration**
- **Performance validation**
- **Handoff to Sprint 5** planning

## Technical Implementation Details

### SDK Architecture (Prototype)

#### Core Components
```
┌─────────────────────────────────────────────────────────────┐
│                Developer Agent Code                         │
│  @agent.message_handler                                     │
│  def handle_message(msg): return response                   │
└─────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────┐
│              Temporal A2A SDK (Prototype)                   │
│  - Agent class with decorators                              │
│  - A2AClient (all operations)                               │
│  - Automatic workflow generation                            │
│  - Message/Response wrappers                                │
└─────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────┐
│             Existing Temporal Infrastructure                │
│  - Sprint 3 workflow signals                                │
│  - Gateway streaming workflows                              │
│  - Agent workflows                                          │
└─────────────────────────────────────────────────────────────┘
```

#### Integration with Sprint 3 Achievements
- **Leverage workflow-to-workflow signals** for streaming
- **Use existing gateway streaming workflows**
- **Build on proven A2A v0.2.5 compliance**

### Prototype Validation Criteria

#### Technical Success Metrics (Echo Worker Focus)
- ✅ **Echo Agent SDK**: Existing echo_worker.py accessible via simple SDK interface
- ✅ **Streaming Echo SDK**: Existing StreamingEchoTaskWorkflow accessible via SDK
- ✅ **Zero Temporal Knowledge**: Developers write simple handlers, SDK maps to proven workflows
- ✅ **Proven Foundation**: Building on battle-tested echo_worker.py implementation

#### Developer Experience Metrics (Echo Focus)
- ✅ **Simple Echo Agent**: `@agent.message_handler` → working echo via existing workflow
- ✅ **Simple Streaming**: `@agent.streaming_handler` → working progressive streaming
- ✅ **Drop-in Replacement**: Same experience as Google SDK for echo operations
- ✅ **Immediate Value**: Leverage existing Sprint 3 streaming achievements

## Integration with Existing Sprint 3 Work

### Leveraging Sprint 3 Achievements
1. **Workflow Signals Architecture**: SDK will use existing signal infrastructure
2. **Progressive Streaming**: SDK will expose Sprint 3 streaming via simple APIs  
3. **A2A Compliance**: SDK will maintain full v0.2.5 specification compliance
4. **Gateway Integration**: SDK will work with existing gateway workflows

### No Disruption to Current System
- **Existing functionality unchanged**
- **SDK as optional layer** on top of current architecture
- **Backward compatibility maintained**
- **Production system unaffected**

## Risk Mitigation

### Technical Risks
- **Temporal Abstraction Complexity**: Mitigated by starting with simple use cases
- **Performance Overhead**: Prototype will validate performance impact
- **Integration Challenges**: Build on proven Sprint 3 foundation

### Timeline Risks
- **Scope Creep**: Strict focus on prototype only (per PM directive)
- **Technical Unknowns**: Prototype approach allows early validation
- **Resource Allocation**: Clear single-focus mandate from PM

## Success Criteria for Sprint 4

### Prototype Demonstration Goals
1. **Working Echo Agent**: Built with SDK, zero Temporal knowledge required
2. **Complete A2A Operations**: Send, get, cancel, stream all functional
3. **Google SDK Compatibility**: Same API experience demonstrated
4. **Progressive Streaming**: Sprint 3 achievements accessible via SDK

### Handoff to Sprint 5
- **SDK Foundation**: Core architecture validated and working
- **Technical Documentation**: Implementation patterns documented
- **Developer Experience**: Validated with real usage scenarios
- **Production Roadmap**: Clear path to full SDK implementation

## Deliverables Summary

### Week 4 Deliverables
- **Core SDK Framework**: Agent class, decorators, automatic workflow generation
- **A2A Client Interface**: Complete protocol operations
- **Basic Integration**: Working with Sprint 3 infrastructure

### Week 5 Deliverables  
- **Refined Prototype**: Advanced features, streaming, state management
- **Developer Tools**: Testing framework, documentation, examples
- **Sprint 4 Demo**: Complete prototype demonstration

### Sprint 4 Success Metrics
- **Technical Validation**: SDK prototype fully functional
- **Developer Experience**: Zero Temporal knowledge required achieved
- **Sprint 3 Integration**: Leveraging workflow signals successfully
- **PM Objective Met**: Focused delivery on #1 Sprint 4 priority

## Conclusion

Sprint 4 will deliver a **focused, working prototype** of the Temporal A2A SDK that demonstrates the revolutionary potential of combining Sprint 3's technical achievements with developer-friendly abstractions. This prototype will validate the approach and provide a clear foundation for full SDK implementation in future sprints.

**Agent 1 Commitment**: Singular focus on Temporal A2A SDK prototype as specified by Agent 6 (Project Manager) for Sprint 4 success.

---

**Next Steps**:
1. Begin core SDK framework implementation
2. Integrate with Sprint 3 workflow signal architecture  
3. Validate developer experience with working prototype
4. Prepare Sprint 5 handoff with proven foundation

**Sprint Focus**: ✅ **Temporal A2A SDK Prototype Only** (PM Priority #1)  
**Timeline**: Week 4-5 (Days 1-10)  
**Success Metric**: Working prototype demonstrating zero Temporal knowledge required