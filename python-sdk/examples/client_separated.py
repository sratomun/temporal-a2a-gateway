#!/usr/bin/env python3
"""
Client example using separated SDK packages
Shows the clean separation - only imports from temporal.a2a
"""
import asyncio

# For calling agents - no agent building imports needed!
from temporal.a2a import A2AClient


async def main():
    """Example client usage with separated SDK"""
    # Create client - only needs temporal.a2a
    client = A2AClient(temporal_host="localhost:7233")
    
    try:
        print("📤 Sending message to echo agent...")
        
        # Send message to agent
        task = await client.send_message("echo-agent", "Hello from separated SDK!")
        print(f"✅ Task created: {task.task_id}")
        
        # Wait for completion
        print("⏳ Waiting for response...")
        for i in range(10):
            await asyncio.sleep(1)
            task_status = await client.get_task(task.task_id)
            print(f"   Status: {task_status.status}")
            
            if task_status.status == "completed":
                print("✅ Task completed!")
                print(f"📝 Response: {task_status.result}")
                break
                
    finally:
        client.close()


if __name__ == "__main__":
    asyncio.run(main())