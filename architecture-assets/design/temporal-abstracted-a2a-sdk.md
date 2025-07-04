# Temporal-Abstracted A2A SDK Design

**Document**: Temporal A2A SDK - Hiding Workflow Complexity  
**Author**: Agent 1 (Architect)  
**Date**: 2025-07-04  
**Status**: 🎯 **STRATEGIC DESIGN**  
**Context**: Developer-friendly SDK abstracting Temporal internals while leveraging workflow benefits

## Executive Summary

Design a **Temporal A2A SDK** that provides the **same developer experience as Google A2A SDK** for **ALL A2A protocol operations** (message/send, tasks/get, tasks/cancel, message/stream, agent discovery) while transparently leveraging Temporal's durability, observability, and reliability benefits. Developers interact with the complete A2A protocol without knowing about workflows, signals, or Temporal concepts.

## Core Design Philosophy

### Developer Experience Goals
1. **Zero Temporal Knowledge Required**: Developers focus on agent logic, not infrastructure
2. **Complete A2A Protocol Support**: All A2A operations (message/send, tasks/get, tasks/cancel, message/stream, discovery)
3. **Google A2A SDK Compatibility**: Same APIs and patterns as existing A2A SDKs
4. **Transparent Benefits**: Gain Temporal advantages without complexity
5. **Progressive Enhancement**: Advanced Temporal features available for power users

### Abstraction Principles
```python
# What developers want to write (simple)
@agent.message_handler
def handle_user_message(message: A2AMessage) -> A2AResponse:
    return f"Echo: {message.get_text()}"

# Not this (complex Temporal workflow)
@workflow.defn
class AgentWorkflow:
    @workflow.signal
    def receive_message(self, msg):
        # Complex workflow orchestration logic
```

## SDK Architecture Overview

### Layer Architecture
```
┌─────────────────────────────────────────────────────────────┐
│                Developer Agent Code                         │
│  @agent.message_handler                                     │
│  def handle_message(msg): return response                   │
└─────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────┐
│              Temporal A2A SDK (This Design)                 │
│  - Agent class abstraction                                  │
│  - Message routing and handling                             │
│  - Automatic workflow generation                            │
│  - State management                                         │
└─────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────┐
│             Temporal SDK (Hidden)                           │
│  - Workflow execution                                       │
│  - Signal/Update handlers                                   │
│  - Activity execution                                       │
│  - State persistence                                        │
└─────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────┐
│               Temporal Cluster                              │
└─────────────────────────────────────────────────────────────┘
```

## SDK Design Components

### 1. Agent Class Abstraction

#### Simple Agent Interface
```python
from temporal_a2a_sdk import Agent, A2AMessage, A2AResponse

class EchoAgent(Agent):
    def __init__(self):
        super().__init__(
            agent_id="echo-agent",
            name="Echo Agent",
            description="Echoes user messages"
        )
    
    @agent.message_handler
    def handle_message(self, message: A2AMessage) -> A2AResponse:
        """Handle incoming messages - completely unaware of Temporal"""
        user_text = message.get_text()
        response_text = f"Echo: {user_text}"
        
        return A2AResponse.text(response_text)
    
    @agent.capability("greeting")
    def handle_greeting(self, message: A2AMessage) -> A2AResponse:
        """Capability-specific handler"""
        return A2AResponse.text("Hello! I'm an echo agent.")
    
    # Optional: State management (automatically persisted by Temporal)
    def get_conversation_state(self, context_id: str) -> dict:
        return self.state.get(context_id, {})
    
    def update_conversation_state(self, context_id: str, updates: dict):
        self.state[context_id].update(updates)

# Run agent (SDK handles all Temporal complexity)
if __name__ == "__main__":
    agent = EchoAgent()
    agent.run()  # Automatically creates and runs Temporal workflows
```

#### Advanced Agent with Streaming
```python
class StreamingEchoAgent(Agent):
    def __init__(self):
        super().__init__(
            agent_id="streaming-echo-agent",
            name="Streaming Echo Agent",
            capabilities={"streaming": True, "progressive_artifacts": True}
        )
    
    @agent.streaming_handler
    async def handle_streaming_message(self, message: A2AMessage) -> AsyncGenerator[A2AResponse, None]:
        """Streaming handler - SDK manages TaskArtifactUpdateEvent complexity"""
        user_text = message.get_text()
        words = f"Echo: {user_text}".split()
        
        # Developer just yields responses - SDK handles all Temporal streaming
        for i, word in enumerate(words):
            current_text = " ".join(words[:i+1])
            yield A2AResponse.partial_text(current_text, is_final=(i == len(words)-1))
            await asyncio.sleep(0.5)  # Simulate processing time
```

### 2. Message and Response Abstractions

#### A2A Message Wrapper
```python
class A2AMessage:
    """Developer-friendly message abstraction"""
    
    def __init__(self, raw_message: dict):
        self._raw = raw_message
        self.message_id = raw_message.get("messageId")
        self.context_id = raw_message.get("contextId")
        self.role = raw_message.get("role", "user")
        self.parts = raw_message.get("parts", [])
        self.timestamp = raw_message.get("timestamp")
    
    def get_text(self) -> str:
        """Extract text content from message parts"""
        for part in self.parts:
            if part.get("kind") == "text":
                return part.get("text", "")
        return ""
    
    def get_files(self) -> List[dict]:
        """Extract file attachments"""
        return [part.get("file") for part in self.parts if part.get("kind") == "file"]
    
    def get_data(self) -> List[dict]:
        """Extract structured data"""
        return [part.get("data") for part in self.parts if part.get("kind") == "data"]
    
    def to_dict(self) -> dict:
        """Access raw A2A message if needed"""
        return self._raw

class A2AResponse:
    """Developer-friendly response builder"""
    
    @staticmethod
    def text(content: str, artifact_name: str = "Response") -> 'A2AResponse':
        """Create text response"""
        return A2AResponse({
            "artifacts": [{
                "artifactId": f"response-{uuid.uuid4()}",
                "name": artifact_name,
                "description": "Agent response",
                "parts": [{"kind": "text", "text": content}]
            }]
        })
    
    @staticmethod
    def partial_text(content: str, is_final: bool = False, artifact_name: str = "Streaming Response") -> 'A2AResponse':
        """Create partial text response for streaming"""
        response = A2AResponse.text(content, artifact_name)
        response._is_partial = not is_final
        response._is_final = is_final
        return response
    
    @staticmethod
    def file(file_path: str, name: str = None) -> 'A2AResponse':
        """Create file response"""
        return A2AResponse({
            "artifacts": [{
                "artifactId": f"file-{uuid.uuid4()}",
                "name": name or os.path.basename(file_path),
                "description": "File attachment",
                "parts": [{"kind": "file", "file": {"name": name, "uri": file_path}}]
            }]
        })
    
    @staticmethod
    def data(data: dict, name: str = "Data Response") -> 'A2AResponse':
        """Create structured data response"""
        return A2AResponse({
            "artifacts": [{
                "artifactId": f"data-{uuid.uuid4()}",
                "name": name,
                "description": "Structured data response",
                "parts": [{"kind": "data", "data": data}]
            }]
        })
```

### 3. Agent Decorator Framework

#### Handler Decorators
```python
def message_handler(func):
    """Decorator for basic message handlers"""
    func._a2a_handler_type = "message"
    func._a2a_handler_config = {}
    return func

def streaming_handler(func):
    """Decorator for streaming message handlers"""
    func._a2a_handler_type = "streaming"
    func._a2a_handler_config = {"supports_streaming": True}
    return func

def capability(capability_name: str):
    """Decorator for capability-specific handlers"""
    def decorator(func):
        func._a2a_handler_type = "capability"
        func._a2a_handler_config = {"capability": capability_name}
        return func
    return decorator

def context_aware(func):
    """Decorator for handlers that need conversation context"""
    func._a2a_context_aware = True
    return func

def rate_limited(requests_per_minute: int):
    """Decorator for rate limiting"""
    def decorator(func):
        func._a2a_rate_limit = requests_per_minute
        return func
    return decorator
```

### 4. Complete A2A Client Interface

#### Full A2A Protocol Client
```python
from temporal_a2a_sdk import A2AClient

class TemporalA2AClient:
    """Complete A2A protocol client backed by Temporal workflows"""
    
    def __init__(self, temporal_host="localhost:7233", namespace="default"):
        self.temporal_client = TemporalClient.connect(temporal_host, namespace)
    
    # A2A message/send
    def send_message(self, agent_id: str, message: A2AMessage) -> A2ATaskResult:
        """Send message to agent - creates task workflow"""
        agent_workflow_id = self._discover_agent_workflow(agent_id)
        
        # Signal agent workflow to create task
        return self.temporal_client.signal_workflow(
            agent_workflow_id, "SendMessage", message
        )
    
    # A2A tasks/get  
    def get_task(self, task_id: str) -> A2ATask:
        """Get task status - queries task workflow"""
        task_workflow_id = self._get_task_workflow_id(task_id)
        
        return self.temporal_client.query_workflow(
            task_workflow_id, "GetTaskStatus"
        )
    
    # A2A tasks/cancel
    def cancel_task(self, task_id: str) -> bool:
        """Cancel task - signals task workflow"""
        task_workflow_id = self._get_task_workflow_id(task_id)
        
        self.temporal_client.signal_workflow(
            task_workflow_id, "CancelTask"
        )
        return True
    
    # A2A message/stream
    def stream_message(self, agent_id: str, message: A2AMessage) -> Iterator[A2ATaskUpdate]:
        """Stream message response - uses update handlers"""
        task_result = self.send_message(agent_id, message)
        task_workflow_id = self._get_task_workflow_id(task_result.id)
        
        # Stream updates using Temporal update handlers
        while True:
            update = self.temporal_client.update_workflow(
                task_workflow_id, "GetProgressUpdate"
            )
            
            if update:
                yield A2ATaskUpdate.from_dict(update)
                if update.get("status") in ["completed", "failed", "cancelled"]:
                    break
            
            time.sleep(0.1)  # 100ms polling
    
    # A2A discovery
    def discover_agents(self, criteria: dict = None) -> List[A2AAgentCard]:
        """Discover agents - queries registry workflow"""
        registry_workflow_id = "agent-registry"
        
        return self.temporal_client.query_workflow(
            registry_workflow_id, "DiscoverAgents", criteria or {}
        )

# Usage - identical to Google A2A SDK
client = TemporalA2AClient()

# Send message (creates durable task)
task = client.send_message("echo-agent", A2AMessage.text("Hello"))

# Check task status (queries workflow state)  
task_status = client.get_task(task.id)

# Stream response (real-time updates)
for update in client.stream_message("echo-agent", A2AMessage.text("Hello")):
    print(f"Progress: {update.progress}%, Status: {update.status}")

# Cancel task (signals workflow)
client.cancel_task(task.id)

# Discover agents (queries registry)
agents = client.discover_agents({"capabilities": ["greeting"]})
```

### 5. Automatic Workflow Generation

#### SDK Internal: Workflow Factory
```python
class TemporalWorkflowFactory:
    """Generates Temporal workflows from Agent classes (hidden from developers)"""
    
    @staticmethod
    def create_workflow_class(agent_class: Type[Agent]) -> Type:
        """Dynamically create Temporal workflow from Agent class"""
        
        class GeneratedAgentWorkflow:
            def __init__(self):
                self.agent_instance = agent_class()
                self.conversation_states = {}
                self.progress_signals = []
            
            @workflow.query
            def get_progress_signals(self) -> List[Dict[str, Any]]:
                return self.progress_signals
            
            @workflow.signal  
            def receive_message(self, message_data: dict):
                """Convert Temporal signal to Agent.handle_message call"""
                try:
                    # Wrap in developer-friendly A2AMessage
                    message = A2AMessage(message_data)
                    
                    # Find appropriate handler
                    handler = self._find_handler(message)
                    
                    # Execute handler (developer code)
                    response = handler(message)
                    
                    # Convert response to A2A artifacts
                    artifacts = self._convert_response(response)
                    
                    # Update progress signals for streaming
                    self._add_progress_signal("completed", 1.0, artifacts)
                    
                except Exception as e:
                    self._add_progress_signal("failed", 0.0, error=str(e))
            
            def _find_handler(self, message: A2AMessage):
                """Find appropriate message handler based on decorators"""
                # Check capability-specific handlers first
                for method_name in dir(self.agent_instance):
                    method = getattr(self.agent_instance, method_name)
                    if hasattr(method, '_a2a_handler_type'):
                        if method._a2a_handler_type == "capability":
                            # Match capability logic
                            pass
                
                # Fall back to default message handler
                return self.agent_instance.handle_message
        
        return GeneratedAgentWorkflow

class Agent:
    """Base agent class that developers inherit from"""
    
    def __init__(self, agent_id: str, name: str, description: str = "", capabilities: dict = None):
        self.agent_id = agent_id
        self.name = name
        self.description = description
        self.capabilities = capabilities or {}
        self.state = PersistentState()  # Automatically managed by Temporal
    
    def run(self):
        """Start the agent (creates and runs Temporal workflows automatically)"""
        # SDK handles all Temporal complexity
        workflow_class = TemporalWorkflowFactory.create_workflow_class(type(self))
        
        # Create Temporal worker with generated workflow
        temporal_worker = TemporalWorker(
            workflows=[workflow_class],
            task_queue=f"{self.agent_id}-tasks"
        )
        
        # Register agent with registry
        self._register_with_registry()
        
        # Start worker (hidden from developer)
        temporal_worker.run()
    
    def send_message(self, target_agent_id: str, message: A2AMessage) -> A2AResponse:
        """Send message to another agent (abstracts Temporal communication)"""
        # SDK handles agent discovery and Temporal signal sending
        target_workflow_id = self._discover_agent_workflow(target_agent_id)
        
        # Send via Temporal signal (hidden from developer)
        return self._send_temporal_message(target_workflow_id, message)
```

### 5. State Management Abstraction

#### Persistent State Helper
```python
class PersistentState:
    """Automatically persisted state using Temporal workflow state"""
    
    def __init__(self):
        self._data = {}  # Backed by Temporal workflow state
    
    def get(self, key: str, default=None):
        """Get state value"""
        return self._data.get(key, default)
    
    def set(self, key: str, value):
        """Set state value (automatically persisted)"""
        self._data[key] = value
        # SDK automatically persists via Temporal workflow state
    
    def update(self, updates: dict):
        """Update multiple state values"""
        self._data.update(updates)
    
    def clear(self):
        """Clear all state"""
        self._data.clear()
    
    def __getitem__(self, key):
        return self._data[key]
    
    def __setitem__(self, key, value):
        self.set(key, value)
    
    def __contains__(self, key):
        return key in self._data

# Usage in agent
class StatefulAgent(Agent):
    @agent.message_handler
    def handle_message(self, message: A2AMessage) -> A2AResponse:
        # Automatically persisted state
        visit_count = self.state.get(message.context_id, 0) + 1
        self.state[message.context_id] = visit_count
        
        return A2AResponse.text(f"Visit #{visit_count}: Echo: {message.get_text()}")
```

## Advanced Features (Optional)

### 1. Multi-Agent Conversations

#### Conversation Management
```python
class MultiAgentConversation:
    """High-level conversation abstraction"""
    
    def __init__(self, conversation_id: str):
        self.conversation_id = conversation_id
        self.participants = []
    
    def add_participant(self, agent_id: str):
        """Add agent to conversation"""
        self.participants.append(agent_id)
    
    def broadcast_message(self, message: A2AMessage):
        """Send message to all participants"""
        for agent_id in self.participants:
            self.send_to_agent(agent_id, message)
    
    def send_to_agent(self, agent_id: str, message: A2AMessage):
        """Send message to specific agent"""
        # SDK handles Temporal communication

class CollaborativeAgent(Agent):
    @agent.message_handler
    def handle_message(self, message: A2AMessage) -> A2AResponse:
        # Can easily communicate with other agents
        if "translate" in message.get_text():
            translator_response = self.send_message("translator-agent", message)
            return A2AResponse.text(f"Translated: {translator_response.get_text()}")
        
        return A2AResponse.text(f"Echo: {message.get_text()}")
```

### 2. Testing and Development Tools

#### Agent Testing Framework
```python
from temporal_a2a_sdk.testing import AgentTester

class TestEchoAgent:
    def setup_method(self):
        self.agent = EchoAgent()
        self.tester = AgentTester(self.agent)
    
    def test_basic_echo(self):
        """Test agent without any Temporal complexity"""
        message = A2AMessage.text("Hello")
        response = self.tester.send_message(message)
        
        assert response.get_text() == "Echo: Hello"
    
    def test_streaming_echo(self):
        """Test streaming functionality"""
        message = A2AMessage.text("Hello World")
        responses = list(self.tester.send_streaming_message(message))
        
        assert len(responses) == 2  # "Echo: Hello", "Echo: Hello World"
        assert responses[-1].is_final
```

### 3. Monitoring and Observability

#### Built-in Metrics
```python
class Agent:
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.metrics = AgentMetrics(self.agent_id)  # Automatic Temporal metrics
    
    @agent.message_handler
    def handle_message(self, message: A2AMessage) -> A2AResponse:
        # Automatic metrics collection (hidden from developer)
        with self.metrics.track_message_processing():
            # Developer code here
            pass

# Automatic dashboards in Temporal UI showing:
# - Message processing times
# - Success/failure rates  
# - Agent utilization
# - Conversation flows
```

## SDK Installation and Setup

### Installation
```bash
pip install temporal-a2a-sdk
```

### Configuration
```python
# config.py
TEMPORAL_A2A_CONFIG = {
    "temporal_host": "localhost:7233",
    "temporal_namespace": "default",
    "agent_registry_workflow_id": "agent-registry",
    "task_queue_prefix": "a2a-agents"
}
```

### Quick Start
```python
from temporal_a2a_sdk import Agent, A2AMessage, A2AResponse

class MyAgent(Agent):
    def __init__(self):
        super().__init__(
            agent_id="my-agent",
            name="My First Agent"
        )
    
    @agent.message_handler
    def handle_message(self, message: A2AMessage) -> A2AResponse:
        return A2AResponse.text(f"You said: {message.get_text()}")

if __name__ == "__main__":
    agent = MyAgent()
    agent.run()  # That's it! Temporal complexity hidden
```

## Migration Path from Google A2A SDK

### Compatibility Layer
```python
# Drop-in replacement for Google A2A SDK
from temporal_a2a_sdk.compat import A2AClient as GoogleA2AClient

# Existing Google SDK code works unchanged
client = GoogleA2AClient()  # Now backed by Temporal
response = client.send_message(agent_id="echo", message=message)
```

### Side-by-Side Comparison: Complete A2A Operations

#### Google A2A SDK (HTTP-based)
```python
from google_a2a_sdk import A2AClient, Message

client = A2AClient(base_url="https://agent.example.com")

# Send message
message = Message(parts=[{"text": "Hello"}])
task = client.send_message(agent_id="echo", message=message)

# Get task status
task_status = client.get_task(task.id)

# Cancel task
client.cancel_task(task.id)

# Stream message (if supported)
for update in client.stream_message(agent_id="echo", message=message):
    print(update)

# Discover agents
agents = client.discover_agents()
```

#### Temporal A2A SDK (Workflow-based, identical API)
```python
from temporal_a2a_sdk.compat import A2AClient, Message

client = A2AClient()  # Automatically connects to Temporal

# Send message (creates durable task workflow)
message = Message(parts=[{"text": "Hello"}])
task = client.send_message(agent_id="echo", message=message)

# Get task status (queries workflow state)
task_status = client.get_task(task.id)

# Cancel task (signals workflow to cancel)
client.cancel_task(task.id)

# Stream message (real-time workflow updates)
for update in client.stream_message(agent_id="echo", message=message):
    print(update)

# Discover agents (queries agent registry workflow)
agents = client.discover_agents()
```

## Implementation Roadmap

### Phase 1: Core SDK (Sprint 5-6)
- **Complete A2A Client Interface** (send_message, get_task, cancel_task, discover_agents)
- **Agent base class and decorators**
- **Message/Response abstractions**  
- **Automatic workflow generation**
- **Basic state management**

### Phase 2: Advanced Features (Sprint 7-8)
- **Full streaming support** (message/stream with TaskArtifactUpdateEvent)
- **Multi-agent conversation management**
- **Testing framework for all A2A operations**
- **Migration compatibility layer**

### Phase 3: Production Features (Sprint 9+)
- **Performance optimization** for all A2A operations
- **Advanced observability** (complete A2A operation tracking)
- **Production deployment tools**
- **Enterprise features** (security, multi-tenancy)

## Benefits Summary

### For Developers
1. **Familiar API**: Same experience as Google A2A SDK
2. **Zero Learning Curve**: No Temporal knowledge required
3. **Better Testing**: Local testing without Temporal complexity
4. **Rapid Development**: Focus on agent logic, not infrastructure

### For Operations
1. **Durability**: All agent state persisted automatically
2. **Observability**: Complete conversation history in Temporal UI
3. **Reliability**: Automatic retry and failure recovery
4. **Scalability**: Temporal's proven scaling capabilities

### For Business
1. **Faster Development**: Reduced time-to-market for new agents
2. **Lower Maintenance**: Less infrastructure complexity
3. **Better Reliability**: Reduced downtime and data loss
4. **Future-Proof**: Built on proven Temporal foundation

## Conclusion

The **Temporal A2A SDK** provides the **best of both worlds**: the **developer simplicity** of traditional A2A SDKs with the **operational benefits** of Temporal workflows for **ALL A2A protocol operations**. Developers can use the complete A2A protocol (message/send, tasks/get, tasks/cancel, message/stream, discovery) without learning Temporal, while operations teams gain unprecedented reliability and observability.

This abstraction makes **complete Temporal-native A2A implementations accessible** to any developer familiar with the A2A protocol, dramatically lowering the barrier to adoption while providing significant technical advantages for every aspect of agent communication.

---

**Next Steps**:
1. Prototype core Agent class and decorator framework
2. Implement automatic workflow generation
3. Build compatibility layer with Google A2A SDK
4. Performance testing and optimization

**Architecture Authority**: Agent 1 (Architect)  
**Status**: Ready for implementation planning and prototyping