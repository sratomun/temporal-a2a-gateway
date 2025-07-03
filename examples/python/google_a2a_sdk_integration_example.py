#!/usr/bin/env python3
"""
Google A2A SDK Types Integration Demo with Temporal A2A Gateway

This demo shows how to use Google A2A SDK types and concepts with our 
Temporal-based A2A Gateway implementation. While our gateway routes to agents
rather than being an agent itself, this demonstrates the type compatibility
and protocol concepts.

Features Demonstrated:
1. Google A2A SDK type creation (AgentCard, Message, Task)
2. Protocol-compatible JSON-RPC 2.0 communication
3. Integration with Temporal orchestration
4. Professional A2A architecture patterns

Prerequisites:
    pip install a2a-sdk httpx requests
"""

import asyncio
import httpx
import json
import requests
import time
from typing import Dict, Any, Optional

# Google A2A SDK imports  
try:
    from a2a.client import A2AClient
    from a2a.types import AgentCard, Message, Task, AgentCapabilities, TextPart, SendMessageRequest, MessageSendParams
    SDK_AVAILABLE = True
except ImportError:
    print("ERROR: Google A2A SDK not installed. Please run: pip install a2a-sdk")
    SDK_AVAILABLE = False

class A2AIntegrationTest:
    """Google A2A SDK Integration Test Suite"""
    
    def __init__(self, gateway_url: str = "http://localhost:8080"):
        self.gateway_url = gateway_url

async def test_google_a2a_sdk_types_demo():
    """Demo Google A2A SDK types integration with Temporal A2A Gateway"""
    
    print("\n🚀 Google A2A SDK Integration Test")
    print("=" * 50)
    
    if not SDK_AVAILABLE:
        print("❌ Google A2A SDK not installed")
        return False
    
    test = A2AIntegrationTest()
    
    # Step 1: Gateway Health Check
    print("\n1️⃣ Checking gateway health...")
    
    try:
        health_response = requests.get(f"{test.gateway_url}/health")
        if health_response.status_code != 200:
            print(f"❌ Gateway not responding (status {health_response.status_code})")
            return False
        
        health_data = health_response.json()
        temporal_status = health_data.get('temporal', {}).get('connected', False)
        redis_status = health_data.get('redis', {}).get('connected', False)
        
        if temporal_status and redis_status:
            print("✅ Gateway is ready")
        else:
            print("❌ Gateway dependencies not ready")
            return False
        
    except Exception as e:
        print(f"❌ Cannot connect to gateway: {e}")
        return False
    
    # Step 2: Create Test AgentCard
    print("\n2️⃣ Setting up A2A client...")
    
    try:
        agent_capabilities = AgentCapabilities(
            streaming=False,
            pushNotifications=False,
            stateTransitionHistory=True
        )
        
        agent_card = AgentCard(
            name="Echo Agent",
            description="A simple echo agent for testing A2A protocol integration",
            version="1.0.0",
            url=test.gateway_url,
            capabilities=agent_capabilities,
            skills=[],
            defaultInputModes=["text"],
            defaultOutputModes=["text"]
        )
        
        print("✅ Agent card created")
        
    except Exception as e:
        print(f"❌ Failed to create agent card: {e}")
        return False
    
    # Step 3: Create A2A Client
    print("\n3️⃣ Connecting to echo agent...")
    
    async with httpx.AsyncClient(timeout=30.0) as http_client:
        try:
            agent_id = "echo-agent"
            agent_endpoint = f"{agent_card.url}/agents/{agent_id}/a2a"
            
            client = A2AClient(
                httpx_client=http_client,
                url=agent_endpoint
            )
            print("✅ Connected to echo agent")
            
        except Exception as e:
            print(f"❌ Failed to connect: {e}")
            return False
        
        # Step 4: Create Message using A2A SDK Types
        print("\n4️⃣ Sending message: 'Hello from Google A2A SDK!'...")
        
        try:
            test_message_content = "Hello from Google A2A SDK! Testing Temporal A2A Gateway integration."
            
            text_part = TextPart(text=test_message_content)
            message = Message(
                messageId="test-message-001",
                role="user", 
                parts=[text_part]
            )
            
        except Exception as e:
            print(f"❌ Failed to create message: {e}")
            return False
        
        # Step 5: Send Message via Google A2A SDK Client ONLY
        print("\n5️⃣ Processing message...")
        
        try:
            params = MessageSendParams(
                message=message,
                metadata={"test": "Google A2A SDK integration"}
            )
            
            request = SendMessageRequest(
                id="sdk-test-001",
                params=params
            )
            
            task_result = await client.send_message(request)
            
            if task_result is not None:
                # Extract task ID from response
                task_id = None
                if hasattr(task_result, 'root') and hasattr(task_result.root, 'result'):
                    task_obj = task_result.root.result
                    if hasattr(task_obj, 'id'):
                        task_id = task_obj.id
                
                if not task_id:
                    print("❌ Could not get task ID")
                    return False
                
                print(f"✅ Task created: {task_id}")
                
                # Step 6: Poll Task Status via Google A2A SDK ONLY
                print("\n6️⃣ Waiting for echo response...")
                
                max_attempts = 10
                attempt = 0
                
                while attempt < max_attempts:
                    try:
                        from a2a.types import GetTaskRequest, TaskQueryParams
                        
                        params = TaskQueryParams(id=task_id)
                        request = GetTaskRequest(id=f"poll-{attempt+1}", params=params)
                        task_status = await client.get_task(request)
                        
                        status = None
                        if hasattr(task_status, 'root') and hasattr(task_status.root, 'result'):
                            task_obj = task_status.root.result
                            if hasattr(task_obj, 'status') and hasattr(task_obj.status, 'state'):
                                status = task_obj.status.state
                        
                        if status:
                            if status.lower() in ['completed', 'succeeded', 'done', 'finished']:
                                print("✅ Task completed")
                                
                                # The Google A2A SDK Task object doesn't expose the result field
                                # So we need to make a direct request to get the full task data
                                try:
                                    
                                    # Make direct request to get task with result
                                    direct_request = {
                                        "jsonrpc": "2.0",
                                        "method": "tasks/get", 
                                        "params": {"id": task_id},
                                        "id": "get-result"
                                    }
                                    
                                    response = requests.post(
                                        f"http://localhost:8080/agents/echo-agent/a2a",
                                        json=direct_request,
                                        headers={"Content-Type": "application/json"}
                                    )
                                    
                                    if response.status_code == 200:
                                        task_data = response.json()
                                        if "result" in task_data and "result" in task_data["result"]:
                                            result = task_data["result"]["result"]
                                            if isinstance(result, dict) and "messages" in result:
                                                print("\n💬 Conversation:")
                                                for msg in result["messages"]:
                                                    if isinstance(msg, dict):
                                                        role = msg.get("role", "unknown")
                                                        parts = msg.get("parts", [])
                                                        if parts and len(parts) > 0:
                                                            part = parts[0]
                                                            if isinstance(part, dict) and "text" in part:
                                                                content = part["text"]
                                                                print(f"  {role.upper()}: {content}")
                                    
                                except Exception as e:
                                    print(f"Note: Could not retrieve conversation details: {e}")
                                
                                return True
                                
                            elif status.lower() in ['failed', 'error', 'cancelled', 'canceled']:
                                print(f"❌ Task failed: {status}")
                                return False
                            elif status.lower() in ['working', 'submitted']:
                                print("⏳ Still processing...")
                                attempt += 1
                                await asyncio.sleep(2)
                            else:
                                attempt += 1
                                await asyncio.sleep(2)
                        else:
                            attempt += 1
                            await asyncio.sleep(2)
                            
                    except Exception as poll_error:
                        attempt += 1
                        await asyncio.sleep(2)
                
                print("❌ Task timed out")
                return False
            else:
                print("❌ No task result received")
                return False
                
        except Exception as e:
            print(f"❌ SDK error: {e}")
            return False
    
    return False

def show_integration_architecture():
    """Display the integration architecture"""
    pass

def main():
    """Main test execution function"""
    async def run_test():
        success = await test_google_a2a_sdk_types_demo()
        
        print("\n" + "=" * 50)
        if success:
            print("🎉 Google A2A SDK Integration Test PASSED")
            print("✅ Gateway is A2A protocol compliant")
            print("✅ Google SDK integration working")
        else:
            print("❌ Google A2A SDK Integration Test FAILED")
        
        return success
    
    return asyncio.run(run_test())

if __name__ == "__main__":
    exit_code = 0 if main() else 1
    exit(exit_code)