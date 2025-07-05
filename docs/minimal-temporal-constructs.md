# Minimal Temporal Constructs for A2A SDK

## Current SDK Architecture

The SDK currently uses these Temporal constructs:

1. **@workflow.defn** - Defines workflow classes
2. **@activity.defn** - Defines activity functions  
3. **Worker** - Runs workflows and activities
4. **Client** - Connects to Temporal cluster

## Minimal Required Constructs

After analyzing the implementation, the absolute minimum Temporal constructs needed are:

### Option 1: Activity-Only Pattern (Current)
- **@activity.defn** - For business logic execution
- **Pre-defined workflows** - Hidden in SDK
- **Worker** - Hidden in runner
- **Client** - Hidden in runner

**Pros:**
- Business logic runs in isolated processes (Temporal's activity model)
- No workflow decorators needed in user code
- Clean separation between SDK and user code

**Cons:**
- Requires importing business logic in activities
- Limited by activity execution model

### Option 2: Direct Workflow Pattern
Would require exposing:
- **@workflow.defn** - User must decorate agent class
- **@workflow.run** - User must decorate handler method

**Pros:**
- More direct integration
- No activity bridge needed

**Cons:**
- Exposes Temporal concepts to users
- Workflow limitations (determinism, no I/O)
- Cannot dynamically create workflows

### Option 3: Hybrid Pattern
- **Single decorator** - Custom `@a2a_agent` that wraps Temporal
- Everything else hidden

**Pros:**
- Single touchpoint for users
- Can handle both workflows and activities internally

**Cons:**
- Complex implementation
- Still subject to Temporal's limitations

## Recommendation

The current **Activity-Only Pattern** is the best approach because:

1. **Zero Temporal exposure** - Users never see Temporal constructs
2. **Pure Python classes** - Agents are just Python classes with methods
3. **Flexible execution** - Activities can do I/O, use async, etc.
4. **Process isolation** - Each activity runs in its own process

## Implementation Requirements

For the current pattern to work reliably:

1. **Business logic separation** - Pure logic modules (like `echo_logic.py`)
2. **No state in agents** - Agents are just interfaces to business logic
3. **Activity imports** - Activities must be able to import business logic
4. **Path management** - Proper PYTHONPATH configuration in containers

## Example: Minimal Agent Code

```python
# Pure business logic (no Temporal)
class MyLogic:
    @staticmethod
    async def process(text: str) -> str:
        return f"Processed: {text}"

# Agent using SDK (no Temporal visible)
class MyAgent(Agent):
    def __init__(self):
        super().__init__(
            agent_id="my-agent",
            name="My Agent"
        )
    
    async def handle_message(self, message: A2AMessage) -> A2AResponse:
        result = await MyLogic.process(message.get_text())
        return A2AResponse.text(result)

# Run it (no Temporal visible)
if __name__ == "__main__":
    agent = MyAgent()
    asyncio.run(agent.run())
```

This achieves the goal of hiding all Temporal constructs while maintaining full functionality.