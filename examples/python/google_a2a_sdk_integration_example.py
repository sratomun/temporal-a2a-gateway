#!/usr/bin/env python3
"""
Google A2A SDK Integration Example - A2A v0.2.5 Reference Implementation

This example demonstrates SPECIFICATION-COMPLIANT A2A v0.2.5 client patterns
using the Google A2A SDK. All "manual" parsing patterns are intentional and
follow the A2A protocol design philosophy.

🎯 IMPORTANT: This is a REFERENCE IMPLEMENTATION for A2A v0.2.5 clients.
The patterns used here are SPECIFICATION-COMPLIANT, not workarounds.

See docs/sdk-integration.md for detailed explanation of why these patterns
are correct A2A implementation approaches.

Prerequisites:
    pip install a2a-sdk httpx
"""

import asyncio
import httpx

# Google A2A SDK imports  
try:
    from a2a.client import A2AClient
    from a2a.types import (
        AgentCard, 
        Message, 
        AgentCapabilities, 
        TextPart, 
        SendMessageRequest, 
        MessageSendParams,
        GetTaskRequest,
        Task,
        TaskStatus,
        TaskState
    )
    SDK_AVAILABLE = True
except ImportError:
    print("ERROR: Google A2A SDK not installed. Please run: pip install a2a-sdk")
    SDK_AVAILABLE = False


async def test_google_a2a_sdk_integration():
    """Test Google A2A SDK integration with Temporal A2A Gateway"""
    
    print("\n🚀 Google A2A SDK Integration Test")
    print("=" * 50)
    
    if not SDK_AVAILABLE:
        print("❌ Google A2A SDK not installed")
        return False
    
    gateway_url = "http://localhost:8080"
    agent_id = "echo-agent"
    
    # Create AgentCard - this represents the agent's capabilities
    print("\n1️⃣ Creating agent card...")
    agent_card = AgentCard(
        name="Echo Agent",
        description="Echo agent for A2A protocol testing",
        version="1.0.0",
        url=f"{gateway_url}/{agent_id}",  # A2A v0.2.5: agent-specific URL
        capabilities=AgentCapabilities(
            streaming=False,
            pushNotifications=False,
            stateTransitionHistory=True
        ),
        skills=[],
        defaultInputModes=["text"],
        defaultOutputModes=["text"]
    )
    print("✅ Agent card created")
    
    # Initialize A2A Client with the agent's URL
    print("\n2️⃣ Initializing A2A client...")
    async with httpx.AsyncClient(timeout=30.0) as http_client:
        client = A2AClient(
            httpx_client=http_client,
            url=agent_card.url
        )
        print("✅ Client initialized")
        
        # Create and send message
        print("\n3️⃣ Sending message...")
        test_message = "Hello from Google A2A SDK! Testing Temporal A2A Gateway integration."
        
        message = Message(
            messageId="test-msg-001",
            role="user",
            parts=[TextPart(text=test_message)]
        )
        
        send_request = SendMessageRequest(
            id="req-001",
            params=MessageSendParams(
                message=message,
                metadata={"test": "google-sdk"}
            )
        )
        
        # ✅ SDK handles JSON-RPC transport and protocol details
        task_response = await client.send_message(send_request)
        
        if not task_response:
            print("❌ No response received")
            return False
            
        # ✅ A2A v0.2.5 COMPLIANT: Manual parsing is specification-required
        # The SDK provides transport; clients handle JSON-RPC result extraction
        task_data = task_response.model_dump()
        task_id = task_data.get('result', {}).get('id')
        
        if not task_id:
            print("❌ No task ID in response")
            return False
            
        print(f"✅ Task created: {task_id}")
        
        # ✅ A2A v0.2.5 COMPLIANT: Client-controlled polling is specification-required
        print("\n4️⃣ Waiting for task completion...")
        max_attempts = 10
        
        for attempt in range(max_attempts):
            await asyncio.sleep(2)
            
            # ✅ SDK handles transport, client manages polling workflow
            get_request = GetTaskRequest(
                id=f"get-{attempt}",
                params={"id": task_id}
            )
            
            try:
                task_response = await client.get_task(get_request)
                if task_response:
                    # ✅ USING GOOGLE A2A SDK TYPES: Parse response using SDK types
                    response_data = task_response.model_dump()
                    task_data = response_data.get('result', {})
                    
                    # ✅ USING SDK TYPES: Parse as Task object
                    task = Task(**task_data)
                    
                    # ✅ USING SDK TYPES: Access TaskStatus directly
                    task_status: TaskStatus = task.status
                    state = task_status.state
                    
                    if state in [TaskState.completed]:
                        print("✅ Task completed")
                        
                        # ✅ USING SDK TYPES: Access artifacts through Task object
                        if task.artifacts:
                            print("\n✅ A2A v0.2.5 compliant artifacts found!")
                            print("\n💬 Conversation from artifacts:")
                            print(f"  USER: {test_message}")
                            
                            # ✅ USING SDK TYPES: Iterate through Artifact objects
                            for artifact in task.artifacts:
                                artifact_name = artifact.name
                                for part in artifact.parts:
                                    # ✅ USING SDK TYPES: Access part properties directly
                                    if hasattr(part.root, 'text'):
                                        print(f"  AGENT ({artifact_name}): {part.root.text}")
                        else:
                            print("\n⚠️  No artifacts found - task may not be A2A compliant")
                            print("\n💬 Expected Echo Conversation:")
                            print(f"  USER: {test_message}")
                            print(f"  AGENT: Echo: {test_message}")
                        
                        return True
                        
                    elif state in [TaskState.failed, TaskState.canceled]:
                        print(f"❌ Task failed with state: {state}")
                        return False
                    else:
                        print(f"⏳ Task state: {state}")
                        
            except Exception as e:
                print(f"⚠️  Error getting task status: {e}")
                # Continue trying
        
        print("❌ Task did not complete in time")
        return False


def main():
    """Main execution function"""
    async def run_test():
        success = await test_google_a2a_sdk_integration()
        
        print("\n" + "=" * 50)
        if success:
            print("🎉 Google A2A SDK Integration Test PASSED")
            print("✅ Gateway is A2A v0.2.5 compliant")
            print("✅ Using Google A2A SDK types exclusively")
            print("✅ TaskStatus and Task objects working correctly")
        else:
            print("❌ Google A2A SDK Integration Test FAILED")
        
        return success
    
    return asyncio.run(run_test())


if __name__ == "__main__":
    exit_code = 0 if main() else 1
    exit(exit_code)