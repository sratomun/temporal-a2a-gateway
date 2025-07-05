#!/usr/bin/env python3
"""
Quick agent trigger script - test the A2A Client interface
"""
import asyncio
from temporal.a2a import A2AClient


async def trigger_echo_agent():
    """Test the echo agent"""
    client = A2AClient(temporal_host="localhost:7233", namespace="default")
    
    print("🚀 Triggering echo agent...")
    task = await client.send_message("echo-agent", "Hello from Temporal Agent")
    print(f"✅ Task ID: {task.task_id}")
    
    # Poll for result
    for i in range(10):
        await asyncio.sleep(1)
        status = await client.get_task(task.task_id)
        print(f"Status: {status.status}")
        
        if status.status == "completed":
            print(f"🎉 Result: {status.result}")
            break
    
    client.close()


async def trigger_streaming_agent():
    """Test the streaming agent"""
    client = A2AClient(temporal_host="localhost:7233", namespace="default")
    
    print("🌊 Triggering streaming agent...")
    async for event in client.stream_message("streaming-echo-agent", "Hello from Temporal Agent - streaming!"):
        print(f"📡 Event: {event}")
    
    client.close()


if __name__ == "__main__":
    import sys
    
    if len(sys.argv) > 1 and sys.argv[1] == "stream":
        asyncio.run(trigger_streaming_agent())
    else:
        asyncio.run(trigger_echo_agent())