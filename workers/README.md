# A2A Workers Directory

This directory contains A2A agent workers that integrate with the Temporal A2A Gateway.

## Structure

### Core Files
- `echo_worker.py` - Non-streaming echo agent using Temporal A2A SDK
- `streaming_echo_worker.py` - Streaming echo agent with progressive responses
- `echo_logic.py` - Pure business logic (no framework dependencies)
- `echo_agent_card.json` - Echo agent capability card
- `streaming_echo_agent_card.json` - Streaming echo agent capability card
- `Dockerfile` - Container for non-streaming echo worker
- `Dockerfile.streaming` - Container for streaming echo worker
- `requirements.txt` - Python dependencies

### Examples Directory
- `examples/` - Example implementations and tests
  - `echo_worker_clean.py` - Minimal SDK example
  - `echo_agent_sdk.py` - Alternative SDK patterns
  - `test_sdk_bridge.py` - SDK bridge testing

## Building an Agent

Agents are built using the Temporal A2A SDK which hides all Temporal complexity:

```python
from temporal_a2a_sdk import Agent, A2AMessage, A2AResponse

class MyAgent(Agent):
    def __init__(self):
        super().__init__(
            agent_id="my-agent",
            name="My Agent",
            capabilities={"streaming": False}
        )
    
    async def handle_message(self, message: A2AMessage) -> A2AResponse:
        # Your business logic here
        return A2AResponse.text("Response")
```

## Running Workers

### Local Development

**Non-streaming echo worker:**
```bash
python echo_worker.py
```

**Streaming echo worker:**
```bash
python streaming_echo_worker.py
```

### Docker

**Non-streaming worker:**
```bash
docker build -t echo-worker -f Dockerfile .
docker run echo-worker
```

**Streaming worker:**
```bash
docker build -t streaming-echo-worker -f Dockerfile.streaming .
docker run streaming-echo-worker
```

### Docker Compose
Both workers are automatically started as part of the gateway stack:
```bash
cd ../examples
docker-compose up
```

This will start:
- `echo-worker` - Handles `echo-agent` (non-streaming)
- `streaming-echo-worker` - Handles `streaming-echo-agent` (progressive responses)

## Adding New Workers

1. Create your agent class extending `Agent`
2. Implement message handlers
3. Add agent card JSON
4. Update `docker-compose.yml` if needed

See the [SDK Quickstart Guide](../docs/sdk-quickstart.md) for detailed instructions.