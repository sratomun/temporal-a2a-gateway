#!/usr/bin/env python3
"""
Test script for SDK workflow bridge
"""
import asyncio
import sys
import logging

# Add SDK to path
sys.path.insert(0, '/app/python-sdk')
from temporal_a2a_sdk import Agent, A2AMessage, A2AResponse

# Import pure business logic
from echo_logic import EchoLogic

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class TestEchoAgent(Agent):
    """Test echo agent for workflow bridge"""
    
    def __init__(self):
        super().__init__(
            agent_id="test-echo-agent",
            name="Test Echo Agent",
            capabilities={"streaming": False},
            metadata={
                "description": "Test agent for SDK workflow bridge",
                "version": "test"
            }
        )
    
    async def handle_message(self, message: A2AMessage) -> A2AResponse:
        """Handle messages using pure logic"""
        text = message.get_text()
        logger.info(f"TestEchoAgent handling message: '{text}'")
        echo_response = EchoLogic.process_message(text)
        return A2AResponse.text(echo_response, name="Test Echo Response")


async def main():
    """Test the SDK workflow bridge"""
    agent = TestEchoAgent()
    
    # Test message handling directly
    test_message = A2AMessage.from_dict({
        "role": "user",
        "parts": [{"kind": "text", "text": "Hello from SDK bridge test!"}]
    })
    
    response = await agent.handle_message(test_message)
    logger.info(f"Direct handler test result: {response.to_dict()}")
    
    # Now run the agent with Temporal
    logger.info("Starting agent with SDK workflow bridge...")
    await agent.run()


if __name__ == "__main__":
    asyncio.run(main())