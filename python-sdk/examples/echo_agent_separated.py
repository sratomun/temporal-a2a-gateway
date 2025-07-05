#!/usr/bin/env python3
"""
Echo Agent using separated SDK packages
Shows the clean separation between temporal.agent and temporal.a2a
"""
import asyncio
import logging

# For building agents
from temporal.agent import Agent, agent_activity

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class EchoAgent(Agent):
    """Simple echo agent using temporal.agent SDK"""
    
    def __init__(self):
        super().__init__(
            agent_id="echo-agent",
            name="Echo Agent",
            capabilities={"streaming": False}
        )
    
    @agent_activity
    async def process_message_activity(self, text: str) -> str:
        """Activity that processes messages"""
        # Pure business logic
        return f"Echo: {text}"


async def main():
    """Run the echo agent"""
    agent = EchoAgent()
    logger.info(f"Starting {agent.name} with separated SDK...")
    
    # Run the agent - SDK handles all Temporal complexity
    await agent.run()


if __name__ == "__main__":
    asyncio.run(main())