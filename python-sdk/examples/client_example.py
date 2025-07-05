#!/usr/bin/env python3
"""
Example client using Temporal A2A SDK
Shows Google A2A SDK compatibility
"""
import asyncio
from temporal_a2a_sdk import A2AClient, A2AMessage


async def main():
    """Example client usage"""
    # Create client - just like Google A2A SDK
    client = A2AClient(gateway_url="http://localhost:8080")
    
    try:
        # Create a message
        message = A2AMessage(
            role="user",
            parts=[{"type": "text", "text": "Hello from SDK client!"}]
        )
        
        print("📤 Sending message to echo agent...")
        
        # Send message to agent
        task = await client.send_message("sdk-echo-agent", message)
        print(f"✅ Task created: {task.id}")
        
        # Wait for completion
        print("⏳ Waiting for response...")
        while task.is_running:
            await asyncio.sleep(1)
            task = await client.get_task(task.id)
            print(f"   Status: {task.status.get('state')}")
            
        # Get result
        if task.is_completed:
            print("✅ Task completed!")
            artifacts = task.get_artifacts()
            if artifacts:
                for artifact in artifacts:
                    print(f"📝 Response: {artifact}")
        elif task.is_failed:
            print(f"❌ Task failed: {task.get_error()}")
            
    finally:
        # Clean up
        client.close()


async def streaming_example():
    """Example of streaming client usage"""
    async with A2AClient() as client:
        message = A2AMessage(
            role="user",
            parts=[{"type": "text", "text": "Stream this message!"}]
        )
        
        print("📡 Starting streaming request...")
        
        # Stream responses
        async for event in client.stream_message("sdk-streaming-echo-agent", message):
            if event.get("kind") == "artifact-update":
                artifact = event.get("artifact", {})
                parts = artifact.get("parts", [])
                if parts:
                    text = parts[0].get("text", "")
                    print(f"   Chunk: '{text}'")
            elif event.get("kind") == "status-update":
                status = event.get("status", {})
                print(f"   Status: {status.get('state')}")
                

if __name__ == "__main__":
    print("=== Regular Message Example ===")
    asyncio.run(main())
    
    print("\n=== Streaming Example ===")
    asyncio.run(streaming_example())