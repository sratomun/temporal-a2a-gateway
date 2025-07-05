# Temporal A2A SDK - Clean Package Separation

## Overview

The Temporal A2A SDK is split into two clean, separate packages:

### `temporal.agent` - For Building Agents
```python
from temporal.agent import Agent, agent_activity
```
Used by developers who want to build A2A-compliant agents.

### `temporal.a2a` - For Calling Agents  
```python
from temporal.a2a import A2AClient
```
Used by developers who want to call existing agents.

## Installation

```bash
# For agent developers
pip install temporal-agent

# For client developers  
pip install temporal-a2a
```

## Clean Separation Benefits

1. **No Cross-Dependencies**: Client users don't need agent-building code
2. **Clear Responsibilities**: Each package has a single purpose
3. **Smaller Footprint**: Install only what you need
4. **Type Safety**: Each package exports only relevant types

## Building Agents (`temporal.agent`)

```python
from temporal.agent import Agent, agent_activity

class MyAgent(Agent):
    def __init__(self):
        super().__init__(
            agent_id="my-agent",
            name="My Agent"
        )
    
    @agent_activity
    async def process_task(self, text: str) -> str:
        # Your business logic here
        return f"Processed: {text}"

# Run the agent
agent = MyAgent()
await agent.run()
```

## Calling Agents (`temporal.a2a`)

```python
from temporal.a2a import A2AClient

# No agent imports needed!
client = A2AClient(temporal_host="localhost:7233")

# Send message
task = await client.send_message("my-agent", "Hello")

# Get result
result = await client.get_task(task.task_id)
print(result)
```

## Package Contents

### temporal.agent
- `Agent` - Base class for building agents
- `@agent_activity` - Decorator for agent activities
- `AgentRunner` - Handles Temporal complexity
- `StreamingContext` - For streaming responses

### temporal.a2a  
- `A2AClient` - Client for calling agents
- `A2ATask` - Task representation
- `A2AMessage` - Message types
- `A2AException` - Client exceptions

## Migration from Single Package

```python
# Old (single package)
from temporal_a2a_sdk import Agent, agent_activity, A2AClient

# New (separated packages)
from temporal.agent import Agent, agent_activity  # For agents
from temporal.a2a import A2AClient              # For clients
```

## Architecture

```
temporal/
├── agent/          # Agent building SDK
│   ├── __init__.py
│   ├── agent.py    # Agent base class
│   ├── runner.py   # Temporal runner
│   └── workflows.py # Hidden complexity
│
└── a2a/            # Client SDK
    ├── __init__.py
    ├── client.py   # A2A client
    ├── messages.py # Protocol types
    └── exceptions.py
```

This clean separation ensures that:
- Agent developers only import `temporal.agent`
- Client developers only import `temporal.a2a`
- No mixing of concerns
- Clear, focused APIs