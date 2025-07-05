#!/usr/bin/env python3
"""
Streaming Echo Worker - Progressive Response Implementation
Streaming echo agent that yields progressive word-by-word responses.
"""
import asyncio
import os
import sys
import logging
from typing import Dict, Any, List

# Add SDK to path
sys.path.insert(0, '/app/python-sdk')
from temporal.agent import Agent, agent_activity

# Import pure business logic
from echo_logic import EchoLogic

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class StreamingEchoAgent(Agent):
    """Streaming echo agent using SDK"""
    
    def __init__(self):
        super().__init__(
            agent_id="streaming-echo-agent",
            name="Streaming Echo Agent",
            capabilities={"streaming": True}
        )
    
    @agent_activity
    async def process_streaming_activity(self, text: str, stream) -> None:
        """Activity that processes streaming messages with real-time StreamContext"""
        # Import everything needed here - activities run in separate processes
        from echo_logic import EchoLogic
        
        # Stream chunks in real-time using StreamContext
        async for chunk in EchoLogic.process_streaming_message(text):
            await stream.send_chunk(chunk)
        
        # Finish streaming
        await stream.finish()
        
        # Streaming activities return None - chunks sent via StreamContext


async def main():
    """Run the streaming echo agent"""
    agent = StreamingEchoAgent()
    logger.info(f"Starting {agent.name} with Temporal A2A SDK...")
    
    # Run the agent - SDK handles all Temporal complexity
    await agent.run()


if __name__ == "__main__":
    asyncio.run(main())