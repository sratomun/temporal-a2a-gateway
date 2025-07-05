# Temporal A2A Python SDK

Build reliable, scalable AI agents with the A2A (Agent-to-Agent) protocol, powered by Temporal's enterprise-grade orchestration platform.

## Overview

The Temporal A2A SDK enables Python developers to create production-ready AI agents without dealing with the complexities of distributed systems. Your agents automatically benefit from:

- **Guaranteed message delivery** - Never lose a message, even during failures
- **Automatic retries and error handling** - Built-in resilience for production environments  
- **Durable execution** - Agent conversations survive server restarts
- **Real-time streaming** - Progressive response streaming for enhanced user experience
- **Enterprise scalability** - Handle thousands of concurrent conversations

## Installation

```bash
pip install temporal-a2a-sdk
```

## Quick Start

### 1. Create Your First Agent

```python
from temporal_a2a_sdk import Agent, A2AMessage, A2AResponse

class EchoAgent(Agent):
    """A simple agent that echoes messages back to the user."""
    
    def __init__(self):
        super().__init__(
            agent_id="echo-agent",
            name="Echo Agent",
            description="Echoes your messages back to you"
        )
    
    async def handle_message(self, message: A2AMessage) -> A2AResponse:
        """Process incoming messages and return responses."""
        user_text = message.get_text()
        return A2AResponse.text(f"Echo: {user_text}")

# Run the agent
if __name__ == "__main__":
    agent = EchoAgent()
    agent.run()
```

### 2. Send Messages to Your Agent

```python
from temporal_a2a_sdk import A2AClient, A2AMessage

async def main():
    # Initialize the client
    client = A2AClient()
    
    # Create a message
    message = A2AMessage(
        role="user",
        parts=[{"type": "text", "text": "Hello, Echo Agent!"}]
    )
    
    # Send the message and get a task handle
    task = await client.send_message("echo-agent", message)
    print(f"Task created: {task.id}")
    
    # Wait for completion and get the response
    result = await client.wait_for_completion(task.id)
    for artifact in result.artifacts:
        print(f"Agent response: {artifact.get_text()}")
```

## Advanced Features

### Progressive Streaming

Enable real-time streaming responses for a ChatGPT-like experience:

```python
class StreamingAssistant(Agent):
    def __init__(self):
        super().__init__(
            agent_id="streaming-assistant",
            name="Streaming Assistant",
            capabilities=AgentCapabilities(streaming=True)
        )
    
    async def handle_message(self, message: A2AMessage) -> A2AResponse:
        """Stream responses word by word."""
        response_text = "Here is a thoughtful response to your query..."
        
        # Stream each word progressively
        for word in response_text.split():
            await self.stream_partial(word + " ")
            await asyncio.sleep(0.1)  # Simulate thinking
        
        return A2AResponse.complete()
```

### Multi-Modal Support

Handle various content types beyond text:

```python
async def handle_message(self, message: A2AMessage) -> A2AResponse:
    # Process different content types
    for part in message.parts:
        if part.type == "text":
            text_content = part.text
        elif part.type == "image":
            image_url = part.image_url
            # Process image...
        elif part.type == "file":
            file_data = part.file_data
            # Process file...
    
    # Return multi-part response
    return A2AResponse.multi_part([
        {"type": "text", "text": "I analyzed your content"},
        {"type": "data", "data": {"analysis": "results"}}
    ])
```

### Error Handling

Built-in resilience with automatic retries:

```python
async def handle_message(self, message: A2AMessage) -> A2AResponse:
    try:
        # Your agent logic here
        result = await self.process_with_ai(message)
        return A2AResponse.text(result)
    except RateLimitError:
        # SDK automatically retries with exponential backoff
        raise
    except Exception as e:
        # Return error response to user
        return A2AResponse.error(f"I encountered an error: {str(e)}")
```

## Architecture

The SDK provides a clean abstraction over Temporal's powerful workflow engine:

```
Your Agent Code (Simple Python)
      ↓
Temporal A2A SDK (Handles Complexity)
      ↓
Temporal Platform (Enterprise-Grade Infrastructure)
```

### What You Write
- Simple Python classes with message handlers
- Business logic and AI integration
- Response formatting

### What the SDK Handles
- Workflow orchestration and state management
- Message routing and delivery guarantees
- Streaming infrastructure
- Error handling and retries
- Scalability and performance

## Client Compatibility

The SDK client is compatible with the Google A2A SDK, making migration seamless:

```python
# Works with existing Google A2A code
from temporal_a2a_sdk import A2AClient  # Drop-in replacement

client = A2AClient()
# Use exactly like Google's A2A client
```

## Best Practices

### 1. Agent Design
- Keep agents focused on a single responsibility
- Use descriptive agent IDs and names
- Document capabilities clearly

### 2. Message Handling
- Always validate input messages
- Return appropriate error responses
- Use streaming for long responses

### 3. Production Deployment
- Set up proper Temporal namespace isolation
- Configure appropriate task queues
- Monitor agent performance metrics

## API Reference

### Core Classes

#### `Agent`
Base class for all agents. Override `handle_message()` to implement your logic.

#### `A2AMessage`
Represents an incoming message with role and content parts.

#### `A2AResponse`
Factory for creating agent responses:
- `A2AResponse.text(content)` - Simple text response
- `A2AResponse.streaming()` - Enable progressive streaming
- `A2AResponse.error(message)` - Error response
- `A2AResponse.multi_part(parts)` - Multi-modal response

#### `A2AClient`
Client for interacting with agents:
- `send_message(agent_id, message)` - Send a message
- `get_task(task_id)` - Check task status
- `wait_for_completion(task_id)` - Wait for response
- `stream_task(task_id)` - Stream progressive updates

## Requirements

- Python 3.8+
- Temporal server (local or cloud)
- Network access to Temporal cluster

## Support

- **Documentation**: [https://docs.temporal.io/a2a](https://docs.temporal.io/a2a)
- **Examples**: See the `examples/` directory
- **Issues**: [GitHub Issues](https://github.com/temporal/a2a-sdk/issues)
- **Community**: [Temporal Community Slack](https://temporal.io/slack)

## License

MIT License - see LICENSE file for details.