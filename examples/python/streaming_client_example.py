#!/usr/bin/env python3

"""
A2A Protocol v0.2.5 Google SDK Streaming Example
===============================================

This example demonstrates how to use the Google A2A SDK with streaming capabilities
to receive real-time updates from agents via Server-Sent Events (SSE).

Usage:
    python streaming_client_example.py

Requirements:
    pip install google-generativeai
"""

import asyncio
import os
from typing import Optional

import google.generativeai as genai
from google.generativeai.types import A2AClient, A2AMessage, A2ATaskStatus


class GoogleA2AStreamingClient:
    """Pure Google A2A SDK streaming client"""
    
    def __init__(self, gateway_url: str = "http://localhost:8080", api_key: str = None):
        if not api_key:
            raise ValueError("Google API key is required for A2A SDK")
        
        self.gateway_url = gateway_url
        
        # Configure Google SDK for A2A
        genai.configure(api_key=api_key)
        self.sdk_client = A2AClient(base_url=gateway_url)
        
        print(f"🔗 Connected to A2A Gateway: {gateway_url}")
        print("✅ Google A2A SDK initialized")
    
    async def discover_agents(self) -> list:
        """Discover available agents using Google SDK"""
        try:
            agents = await self.sdk_client.discover_agents()
            print("🔍 Available agents:")
            for agent in agents:
                print(f"  - {agent.name}: {agent.description}")
            return agents
        except Exception as e:
            print(f"❌ Agent discovery failed: {e}")
            return []
    
    async def send_streaming_message(self, agent_id: str, message: str) -> None:
        """
        Send streaming message using Google A2A SDK
        """
        print(f"🔄 Starting streaming message to agent: {agent_id}")
        
        try:
            # Create A2A message using SDK
            a2a_message = A2AMessage(
                parts=[{"text": message}]
            )
            
            # Send with streaming enabled using message/stream method
            task = await self.sdk_client.send_message_async(
                agent_id=agent_id,
                message=a2a_message,
                stream=True,  # Enable streaming mode
                method="message/stream"  # Use streaming endpoint
            )
            
            print(f"📡 SDK streaming started for task: {task.id}")
            
            # Process streaming updates
            async for status_update in task.stream_status():
                self._handle_status_update(task.id, status_update)
                
                # Break on completion or failure
                if status_update.state in ["completed", "failed"]:
                    break
                    
        except Exception as e:
            print(f"❌ SDK streaming failed: {e}")
    
    def _handle_status_update(self, task_id: str, status: A2ATaskStatus) -> None:
        """Handle status updates from Google SDK"""
        print(f"📋 Task {task_id}: {status.state}")
        
        if status.state == "submitted":
            print(f"🚀 Task {task_id} has been submitted to agent")
        elif status.state == "working":
            print(f"🔄 Task {task_id} is being processed by agent")
            if hasattr(status, 'progress') and status.progress:
                print(f"📈 Progress: {status.progress:.1%}")
        elif status.state == "completed" and status.result:
            print(f"✅ Task {task_id} completed successfully!")
            print(f"📄 Result: {status.result}")
        elif status.state == "failed" and status.error:
            print(f"❌ Task {task_id} failed: {status.error}")
        
        # Show timestamp if available
        if hasattr(status, 'timestamp') and status.timestamp:
            print(f"🕐 Timestamp: {status.timestamp}")
    
    async def send_multiple_streaming_messages(self, agent_id: str, messages: list[str]) -> None:
        """
        Demonstrate multiple concurrent streaming messages
        """
        print(f"🔄 Starting {len(messages)} concurrent streaming messages to {agent_id}")
        
        tasks = []
        for i, message in enumerate(messages):
            print(f"\n📨 Message {i+1}: {message}")
            task = asyncio.create_task(self.send_streaming_message(agent_id, message))
            tasks.append(task)
        
        # Wait for all streaming tasks to complete
        await asyncio.gather(*tasks)


async def demo_echo_streaming():
    """Demonstrate basic streaming with Google A2A SDK"""
    # Get API key from environment or user input
    api_key = os.getenv('GOOGLE_API_KEY') or input("Enter Google API key: ").strip()
    
    if not api_key:
        print("❌ Google API key is required")
        return
    
    client = GoogleA2AStreamingClient(api_key=api_key)
    
    print("=" * 60)
    print("Google A2A SDK Streaming Demo")
    print("=" * 60)
    
    # Discover agents first
    agents = await client.discover_agents()
    
    # Send streaming message to echo agent
    test_message = "Hello from Google A2A SDK streaming client! This is a test of real-time communication."
    
    await client.send_streaming_message("echo-agent", test_message)
    
    print("🏁 Echo streaming demo completed")


async def demo_multiple_streaming():
    """Demonstrate multiple concurrent streaming messages"""
    api_key = os.getenv('GOOGLE_API_KEY') or input("Enter Google API key: ").strip()
    
    if not api_key:
        print("❌ Google API key is required")
        return
    
    client = GoogleA2AStreamingClient(api_key=api_key)
    
    print("=" * 60)
    print("Multiple Concurrent Streaming Demo")
    print("=" * 60)
    
    # Multiple test messages
    messages = [
        "First streaming message - testing concurrent processing",
        "Second streaming message - real-time updates",
        "Third streaming message - Google SDK integration"
    ]
    
    await client.send_multiple_streaming_messages("echo-agent", messages)
    
    print("🏁 Multiple streaming demo completed")


async def demo_custom_agent_streaming():
    """Demonstrate streaming with user-specified agent"""
    api_key = os.getenv('GOOGLE_API_KEY') or input("Enter Google API key: ").strip()
    
    if not api_key:
        print("❌ Google API key is required")
        return
    
    client = GoogleA2AStreamingClient(api_key=api_key)
    
    print("=" * 60)
    print("Custom Agent Streaming Demo")
    print("=" * 60)
    
    # Discover available agents
    agents = await client.discover_agents()
    
    # Get agent ID from user
    agent_id = input("Enter agent ID (or press Enter for 'echo-agent'): ").strip()
    if not agent_id:
        agent_id = "echo-agent"
    
    # Get message from user
    message_text = input("Enter message (or press Enter for default): ").strip()
    if not message_text:
        message_text = "Hello from custom streaming demo with Google A2A SDK!"
    
    await client.send_streaming_message(agent_id, message_text)
    
    print("🏁 Custom agent streaming demo completed")


async def main():
    """Main demo selector"""
    print("🌟 Google A2A SDK Streaming Examples 🌟")
    print("=" * 50)
    print("Pure Google SDK implementation - no fallbacks!")
    print("=" * 50)
    print("1. Basic Echo Streaming Demo")
    print("2. Multiple Concurrent Streaming Demo")
    print("3. Custom Agent Streaming Demo")
    print("=" * 50)
    
    choice = input("Select demo (1, 2, 3, or press Enter for basic demo): ").strip()
    
    if choice == "2":
        await demo_multiple_streaming()
    elif choice == "3":
        await demo_custom_agent_streaming()
    else:
        await demo_echo_streaming()


if __name__ == "__main__":
    print("🚀 A2A Protocol v0.2.5 - Google SDK Streaming Client")
    print("Requires: pip install google-generativeai")
    print("Set GOOGLE_API_KEY environment variable or enter when prompted")
    print()
    
    asyncio.run(main())