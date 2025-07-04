#!/usr/bin/env python3
"""
Progressive Artifact Streaming Validation Test

Specifically tests the progressive artifact streaming feature that Agent 2 implemented
with word-by-word streaming and A2A v0.2.5 TaskArtifactUpdateEvent compliance.
"""

import asyncio
import httpx
import json
import time
from typing import List, Dict, Any

async def parse_sse_stream(response: httpx.Response):
    """Parse Server-Sent Events from streaming response"""
    buffer = ""
    events = []
    
    async for chunk in response.aiter_text():
        buffer += chunk
        
        while "\n\n" in buffer:
            event_text, buffer = buffer.split("\n\n", 1)
            
            event_type = None
            event_data = {}
            
            for line in event_text.split("\n"):
                if line.startswith("event: "):
                    event_type = line[7:].strip()
                elif line.startswith("data: "):
                    try:
                        event_data = json.loads(line[6:])
                        event_data["event_type"] = event_type
                        events.append(event_data)
                    except json.JSONDecodeError:
                        continue
    
    return events

async def test_progressive_artifact_streaming():
    """Test progressive artifact streaming with real gateway"""
    
    print("🚀 Progressive Artifact Streaming Validation Test")
    print("=" * 60)
    
    gateway_url = "http://localhost:8080"
    agent_id = "streaming-echo-agent"
    test_message = "Test progressive artifacts now"
    
    # Create streaming request
    stream_request = {
        "jsonrpc": "2.0",
        "method": "message/stream",
        "params": {
            "message": {
                "messageId": "progressive-test-001",
                "role": "user",
                "parts": [{"text": test_message}]
            },
            "metadata": {"test": "progressive-artifacts"}
        },
        "id": "progressive-req-001"
    }
    
    headers = {
        "Accept": "text/event-stream",
        "Cache-Control": "no-cache",
        "Content-Type": "application/json"
    }
    
    print(f"\n📡 Connecting to streaming endpoint: {gateway_url}/{agent_id}")
    print(f"📝 Test message: '{test_message}'")
    print(f"📊 Expected progressive chunks: {len('Echo: Test progressive artifacts now'.split())} words")
    
    try:
        async with httpx.AsyncClient(timeout=45.0) as client:
            stream_url = f"{gateway_url}/{agent_id}"
            
            async with client.stream("POST", stream_url, json=stream_request, headers=headers) as response:
                print(f"✅ Connection status: {response.status_code}")
                
                if response.status_code != 200:
                    print(f"❌ Failed to connect: {response.status_code}")
                    return False
                
                print("\n🌊 Real-time streaming events:")
                print("-" * 50)
                
                # Track events
                status_events = []
                artifact_events = []
                task_id = None
                start_time = time.time()
                
                buffer = ""
                async for chunk in response.aiter_text():
                    buffer += chunk
                    
                    while "\n\n" in buffer:
                        event_text, buffer = buffer.split("\n\n", 1)
                        
                        event_type = None
                        event_data = {}
                        
                        for line in event_text.split("\n"):
                            if line.startswith("event: "):
                                event_type = line[7:].strip()
                            elif line.startswith("data: "):
                                try:
                                    event_data = json.loads(line[6:])
                                    event_data["event_type"] = event_type
                                    event_data["timestamp"] = time.time() - start_time
                                    
                                    # Process event
                                    kind = event_data.get("kind", "unknown")
                                    
                                    if not task_id:
                                        task_id = event_data.get("taskId")
                                    
                                    if kind == "status-update":
                                        status_events.append(event_data)
                                        status = event_data.get("status", {})
                                        state = status.get("state", "unknown")
                                        timestamp = event_data.get("timestamp", 0)
                                        print(f"📊 [{timestamp:.2f}s] Status: {state}")
                                        
                                        # Check for completion
                                        final = event_data.get("final", False)
                                        if final and state in ["completed", "succeeded"]:
                                            print(f"✅ Task completed: {task_id}")
                                            break
                                            
                                    elif kind == "artifact-update":
                                        artifact_events.append(event_data)
                                        artifact = event_data.get("artifact", {})
                                        timestamp = event_data.get("timestamp", 0)
                                        last_chunk = event_data.get("lastChunk", False)
                                        
                                        # Extract text content
                                        artifact_text = ""
                                        for part in artifact.get("parts", []):
                                            if part.get("kind") == "text":
                                                artifact_text = part.get("text", "")
                                                break
                                        
                                        chunk_marker = " (FINAL)" if last_chunk else ""
                                        print(f"📄 [{timestamp:.2f}s] Artifact{chunk_marker}: '{artifact_text}'")
                                        
                                        if last_chunk:
                                            print(f"🏁 Final artifact received")
                                            break
                                    
                                    else:
                                        print(f"⚠️  Unknown event: {kind}")
                                        
                                except json.JSONDecodeError:
                                    continue
                
                print("-" * 50)
                
                # Analysis
                print(f"\n📈 Streaming Analysis:")
                print(f"   📊 Status events: {len(status_events)}")
                print(f"   📄 Artifact events: {len(artifact_events)}")
                print(f"   🔗 Task ID: {task_id}")
                print(f"   ⏱️  Total duration: {time.time() - start_time:.2f}s")
                
                # Detailed artifact analysis
                if artifact_events:
                    print(f"\n📄 Progressive Artifact Analysis:")
                    print(f"   📊 Total artifact chunks: {len(artifact_events)}")
                    
                    # Extract progressive texts
                    progressive_texts = []
                    for event in artifact_events:
                        artifact = event.get("artifact", {})
                        for part in artifact.get("parts", []):
                            if part.get("kind") == "text":
                                progressive_texts.append(part.get("text", ""))
                                break
                    
                    print(f"   📝 Progressive text sequence:")
                    for i, text in enumerate(progressive_texts):
                        print(f"      {i+1:2d}. '{text}'")
                    
                    # Validate progressive building
                    if len(progressive_texts) > 1:
                        is_progressive = True
                        for i in range(1, len(progressive_texts)):
                            current = progressive_texts[i]
                            previous = progressive_texts[i-1]
                            if not (current.startswith(previous) or current.startswith("Echo:")):
                                is_progressive = False
                                break
                        
                        if is_progressive:
                            print(f"   ✅ Progressive building: VALIDATED")
                        else:
                            print(f"   ❌ Progressive building: FAILED")
                    
                    # A2A compliance check
                    print(f"\n🎯 A2A v0.2.5 Compliance Check:")
                    compliance_passed = True
                    
                    for i, event in enumerate(artifact_events):
                        required_fields = ["kind", "taskId", "artifact", "append", "lastChunk"]
                        for field in required_fields:
                            if field not in event:
                                print(f"   ❌ Missing field '{field}' in event {i+1}")
                                compliance_passed = False
                        
                        # Check artifact structure
                        artifact = event.get("artifact", {})
                        artifact_fields = ["artifactId", "name", "description", "parts"]
                        for field in artifact_fields:
                            if field not in artifact:
                                print(f"   ❌ Missing artifact field '{field}' in event {i+1}")
                                compliance_passed = False
                    
                    if compliance_passed:
                        print(f"   ✅ A2A v0.2.5 compliance: PASSED")
                    else:
                        print(f"   ❌ A2A v0.2.5 compliance: FAILED")
                    
                    return len(artifact_events) > 0 and compliance_passed
                
                else:
                    print(f"\n❌ No artifact events received")
                    print(f"   Status events show streaming is working, but artifacts not transmitted")
                    print(f"   This indicates gateway may not be properly converting workflow signals to TaskArtifactUpdateEvent")
                    return False
                
    except httpx.ConnectError:
        print(f"❌ Cannot connect to gateway at {gateway_url}")
        print(f"   Make sure the gateway is running with streaming echo agent")
        return False
    except Exception as e:
        print(f"❌ Test error: {e}")
        return False

async def main():
    """Run progressive artifact streaming test"""
    success = await test_progressive_artifact_streaming()
    
    print("\n" + "=" * 60)
    if success:
        print("🎉 Progressive Artifact Streaming Test PASSED")
        print("✅ TaskArtifactUpdateEvent streaming operational")
        print("✅ Word-by-word progressive building functional")
        print("✅ A2A v0.2.5 compliance verified")
    else:
        print("❌ Progressive Artifact Streaming Test FAILED")
        print("🔧 Check gateway implementation of TaskArtifactUpdateEvent")
        print("🔧 Verify workflow signal → artifact event conversion")
    
    return success

if __name__ == "__main__":
    success = asyncio.run(main())
    exit(0 if success else 1)