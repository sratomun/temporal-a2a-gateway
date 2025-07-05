#!/usr/bin/env python3
"""
Echo Agent using Temporal A2A SDK
Demonstrates zero Temporal knowledge required
"""
from temporal_a2a_sdk import Agent, A2AMessage, A2AResponse


class EchoAgent(Agent):
    """Simple echo agent that echoes back messages"""
    
    def __init__(self):
        super().__init__(
            agent_id="sdk-echo-agent",
            name="SDK Echo Agent",
            capabilities=["text_processing"],
            metadata={
                "description": "Echo agent built with Temporal A2A SDK",
                "version": "1.0.0"
            }
        )
        
    @Agent.message_handler
    def handle_message(self, message: A2AMessage) -> A2AResponse:
        """Handle incoming messages - no Temporal code needed!"""
        # Get text from message
        text = message.get_text()
        
        # Create response
        return A2AResponse.text(
            content=f"Echo: {text}",
            name="Echo Response"
        )


def main():
    """Run the echo agent"""
    # Create agent instance
    agent = EchoAgent()
    
    # Run the agent - it handles all Temporal complexity
    print(f"🚀 Starting {agent.name} with SDK...")
    print("✨ No Temporal knowledge required!")
    print(f"📡 Agent ID: {agent.agent_id}")
    
    # Run the agent (blocks until stopped)
    agent.run_sync(
        temporal_host="temporal:7233",
        namespace="default"
    )


if __name__ == "__main__":
    main()