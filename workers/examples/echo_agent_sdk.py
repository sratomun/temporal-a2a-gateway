#!/usr/bin/env python3
"""
SDK-based Echo Agent - Clean implementation
Agent 1 Sprint 4 - Step 4: Clean SDK interface
"""
import asyncio
import sys
from typing import List

# Add SDK to path
sys.path.insert(0, '/app/python-sdk')
from temporal_a2a_sdk import Agent, A2AMessage, A2AResponse, message_handler

# Import pure business logic
from echo_logic import EchoLogic


class EchoAgentSDK(Agent):
    """Echo agent with clean SDK interface"""
    
    def __init__(self):
        super().__init__(
            agent_id="echo-agent",
            name="Echo Agent (SDK)",
            capabilities={"streaming": False},
            metadata={
                "description": "Simple echo agent using Temporal A2A SDK",
                "version": "2.0.0"  # Full SDK version
            }
        )
    
    async def handle_message(self, message: A2AMessage) -> A2AResponse:
        """Handle incoming messages with pure logic"""
        text = message.get_text()
        echo_response = EchoLogic.process_message(text)
        return A2AResponse.text(echo_response, name="Echo Response")


class StreamingEchoAgentSDK(Agent):
    """Streaming echo agent with clean SDK interface"""
    
    def __init__(self):
        super().__init__(
            agent_id="streaming-echo-agent",
            name="Streaming Echo Agent (SDK)",
            capabilities={"streaming": True},
            metadata={
                "description": "Streaming echo agent using Temporal A2A SDK",
                "version": "2.0.0"  # Full SDK version
            }
        )
    
    async def handle_streaming_message(self, message: A2AMessage) -> List[str]:
        """Handle streaming messages with pure logic"""
        text = message.get_text()
        return EchoLogic.process_streaming_message(text)


# Simple main function - all Temporal complexity hidden
if __name__ == "__main__":
    import os
    import logging
    
    logging.basicConfig(level=logging.INFO)
    
    # Determine which agent to run based on environment
    agent_type = os.getenv('AGENT_TYPE', 'basic')
    
    if agent_type == 'streaming':
        agent = StreamingEchoAgentSDK()
    else:
        agent = EchoAgentSDK()
    
    # Run the agent - SDK handles all Temporal complexity
    asyncio.run(agent.run())