#!/usr/bin/env python3
"""
Example client using Temporal A2A SDK
Shows Google A2A SDK compatibility with direct Temporal integration
"""
import asyncio
from temporal.a2a import A2AClient, A2ATask


def extract_text_from_result(result: dict) -> str:
    """Extract human-readable text from A2A result"""
    if not result or "artifacts" not in result:
        return "(no result)"
    
    artifacts = result["artifacts"]
    if not artifacts:
        return "(no artifacts)"
    
    # Get text from first artifact
    artifact = artifacts[0]
    parts = artifact.get("parts", [])
    if not parts:
        return "(no parts in artifact)"
    
    # Find text part
    for part in parts:
        if part.get("kind") == "text":
            return part.get("text", "(empty text)")
    
    return "(no text parts found)"


async def main():
    """Example client usage"""
    # Create client - connects directly to Temporal
    client = A2AClient(temporal_host="localhost:7233", namespace="default")
    
    try:
        print("📤 Sending message to echo agent...")
        
        # Send message to agent (simple string interface)
        task = await client.send_message("echo-agent", "Hello from Temporal Agent")
        print(f"✅ Task created: {task.task_id}")
        
        # Wait for completion
        print("⏳ Waiting for response...")
        max_retries = 10
        for i in range(max_retries):
            await asyncio.sleep(1)
            task_status = await client.get_task(task.task_id)
            print(f"   Status: {task_status.status}")
            
            if task_status.status == "completed":
                print("✅ Task completed!")
                
                # Extract and display human-readable response
                response_text = extract_text_from_result(task_status.result)
                print(f"📝 Response: {response_text}")
                break
            elif task_status.status in ["canceled", "failed"]:
                print(f"❌ Task failed: {task_status.status}")
                break
        else:
            print("⏰ Task taking too long, cancelling...")
            await client.cancel_task(task.task_id)
            
    finally:
        # Clean up
        client.close()


async def streaming_example():
    """Example of streaming client usage"""
    async with A2AClient(temporal_host="localhost:7233") as client:
        print("📡 Starting streaming request...")
        
        # Stream responses with timeout
        timeout = 30  # seconds
        start_time = asyncio.get_event_loop().time()
        
        # Stream responses (simple string interface)
        async for event in client.stream_message("streaming-echo-agent", "Hello from Temporal Agent - streaming!"):
            if event.get("kind") == "artifact-update":
                artifact = event.get("artifact", {})
                parts = artifact.get("parts", [])
                if parts:
                    text = parts[0].get("text", "")
                    print(f"   Chunk: '{text}'")
            elif event.get("kind") == "status-update":
                status = event.get("status", {})
                state = status.get('state')
                print(f"   Status: {state}")
                
                # Break when task is completed
                if state in ["completed", "failed", "canceled"]:
                    break
            
            # Timeout check
            if asyncio.get_event_loop().time() - start_time > timeout:
                print("⏰ Timeout - stopping stream")
                break
                

if __name__ == "__main__":
    print("=== Regular Message Example ===")
    asyncio.run(main())
    
    print("\n=== Streaming Example ===")
    asyncio.run(streaming_example())