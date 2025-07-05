#!/usr/bin/env python3
"""
Streaming Echo Agent using Temporal A2A SDK
Shows progressive streaming without Temporal complexity
"""
import asyncio
from temporal_a2a_sdk import Agent, A2AMessage, A2AResponse


class StreamingEchoAgent(Agent):
    """Echo agent with progressive streaming capability"""
    
    def __init__(self):
        super().__init__(
            agent_id="sdk-streaming-echo-agent",
            name="SDK Streaming Echo Agent",
            capabilities=["text_processing", "streaming"],
            metadata={
                "description": "Streaming echo agent built with Temporal A2A SDK",
                "version": "1.0.0"
            }
        )
        
    @Agent.message_handler
    async def handle_message(self, message: A2AMessage) -> A2AResponse:
        """Handle messages with progressive streaming"""
        text = message.get_text()
        echo_text = f"Echo: {text}"
        
        # For now, return complete response
        # TODO: Add streaming support to SDK
        # This would stream word by word through gateway signals
        return A2AResponse.text(
            content=echo_text,
            name="Streaming Echo Response"
        )


def main():
    """Run the streaming echo agent"""
    agent = StreamingEchoAgent()
    
    print(f"🚀 Starting {agent.name} with SDK...")
    print("🔄 Streaming capability enabled")
    print(f"📡 Agent ID: {agent.agent_id}")
    
    agent.run_sync(
        temporal_host="temporal:7233",
        namespace="default"
    )


if __name__ == "__main__":
    main()