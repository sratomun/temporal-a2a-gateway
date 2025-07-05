#!/usr/bin/env python3
"""
Echo Worker - Non-Streaming Implementation
Simple echo agent that responds to messages without streaming.
"""
import asyncio
import os
import sys
import logging
from typing import Dict, Any

# Add SDK to path
sys.path.insert(0, '/app')
from temporal.agent import Agent, agent_activity

# Import pure business logic
from echo_logic import EchoLogic

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class EchoAgent(Agent):
    """Simple echo agent using SDK - no streaming"""
    
    def __init__(self):
        super().__init__(
            agent_id="echo-agent",
            name="Echo Agent",
            capabilities={"streaming": False}
        )
    
    @agent_activity
    async def process_message_activity(self, text: str) -> str:
        """Activity that processes messages - runs in separate process"""
        # Import everything needed here - activities run in separate processes
        from echo_logic import EchoLogic
        
        # Process using pure business logic
        return EchoLogic.process_message(text)


async def main():
    """Run the echo agent"""
    agent = EchoAgent()
    logger.info(f"Starting {agent.name} with Temporal A2A SDK...")
    
    # Run the agent - SDK handles all Temporal complexity
    await agent.run()


if __name__ == "__main__":
    asyncio.run(main())