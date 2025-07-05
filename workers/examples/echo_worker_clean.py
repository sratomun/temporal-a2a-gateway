#!/usr/bin/env python3
"""
Echo Worker - Clean SDK Implementation
Zero Temporal complexity - just business logic
"""
import asyncio
import os
import sys
import logging
from typing import List

# Add SDK to path
sys.path.insert(0, '/app/python-sdk')
from temporal_a2a_sdk import Agent, A2AMessage, A2AResponse

# Import pure business logic
from echo_logic import EchoLogic

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class EchoAgent(Agent):
    """Simple echo agent using SDK"""
    
    def __init__(self):
        super().__init__(
            agent_id="echo-agent",
            name="Echo Agent",
            capabilities={"streaming": False}
        )
    
    async def handle_message(self, message: A2AMessage) -> A2AResponse:
        """Echo the message back using pure logic"""
        text = message.get_text()
        echo_response = EchoLogic.process_message(text)
        return A2AResponse.text(echo_response, name="Echo Response")


class StreamingEchoAgent(Agent):
    """Streaming echo agent using SDK"""
    
    def __init__(self):
        super().__init__(
            agent_id="streaming-echo-agent",
            name="Streaming Echo Agent",
            capabilities={"streaming": True}
        )
    
    async def handle_streaming_message(self, message: A2AMessage) -> List[str]:
        """Stream the echo response using pure logic"""
        text = message.get_text()
        return EchoLogic.process_streaming_message(text)


async def main():
    """Run both echo agents concurrently"""
    # Create agents
    echo_agent = EchoAgent()
    streaming_agent = StreamingEchoAgent()
    
    logger.info("Starting Echo Workers with Temporal A2A SDK...")
    
    # Run both agents - SDK handles all Temporal complexity
    await asyncio.gather(
        echo_agent.run(),
        streaming_agent.run()
    )


if __name__ == "__main__":
    asyncio.run(main())