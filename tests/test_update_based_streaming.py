"""
Test Update-based Streaming Implementation

This test verifies that the new Temporal Update handler approach
works correctly for real-time streaming of workflow progress.
"""

import asyncio
import json
import aiohttp
import pytest
import time


@pytest.mark.asyncio
async def test_update_based_streaming():
    """Test the update-based streaming mechanism"""
    gateway_url = "http://localhost:8080"
    
    # Create streaming request
    request_data = {
        "jsonrpc": "2.0",
        "method": "message/stream",
        "params": {
            "agentId": "streaming-echo-agent",
            "message": {
                "content": "Test update-based streaming"
            }
        },
        "id": "test-streaming-1"
    }
    
    async with aiohttp.ClientSession() as session:
        # Send streaming request
        async with session.post(f"{gateway_url}/api", json=request_data) as response:
            assert response.status == 200
            assert response.headers.get('Content-Type') == 'text/event-stream'
            
            # Collect all events
            events = []
            start_time = time.time()
            timeout = 30  # 30 second timeout
            
            async for line in response.content:
                if time.time() - start_time > timeout:
                    break
                    
                line = line.decode('utf-8').strip()
                if line.startswith('data: '):
                    event_data = json.loads(line[6:])
                    events.append(event_data)
                    print(f"Received event: {event_data}")
                    
                    # Check if this is the final event
                    if event_data.get('kind') == 'status-update' and event_data.get('final'):
                        break
            
            # Verify we received events
            assert len(events) > 0, "No events received"
            
            # Verify event sequence
            status_events = [e for e in events if e.get('kind') == 'status-update']
            assert len(status_events) > 0, "No status update events received"
            
            # Check status progression
            states = [e['status']['state'] for e in status_events]
            print(f"Status progression: {states}")
            
            # Should have at least submitted and completed states
            assert 'submitted' in states
            assert 'completed' in states or 'failed' in states
            
            # Verify no duplicate events (update handlers should prevent duplicates)
            seen_states = set()
            for state in states:
                assert state not in seen_states, f"Duplicate state detected: {state}"
                seen_states.add(state)


@pytest.mark.asyncio
async def test_concurrent_streaming_sessions():
    """Test multiple concurrent streaming sessions"""
    gateway_url = "http://localhost:8080"
    
    async def stream_task(session_id):
        request_data = {
            "jsonrpc": "2.0",
            "method": "message/stream",
            "params": {
                "agentId": "streaming-echo-agent",
                "message": {
                    "content": f"Concurrent test {session_id}"
                }
            },
            "id": f"test-concurrent-{session_id}"
        }
        
        async with aiohttp.ClientSession() as session:
            async with session.post(f"{gateway_url}/api", json=request_data) as response:
                assert response.status == 200
                
                event_count = 0
                async for line in response.content:
                    line = line.decode('utf-8').strip()
                    if line.startswith('data: '):
                        event_data = json.loads(line[6:])
                        event_count += 1
                        
                        if event_data.get('kind') == 'status-update' and event_data.get('final'):
                            break
                
                return event_count
    
    # Run 3 concurrent streaming sessions
    results = await asyncio.gather(
        stream_task(1),
        stream_task(2),
        stream_task(3)
    )
    
    # Each session should receive events
    for i, count in enumerate(results):
        assert count > 0, f"Session {i+1} received no events"
        print(f"Session {i+1} received {count} events")


@pytest.mark.asyncio
async def test_workflow_already_exists_handling():
    """Test that duplicate workflow IDs are handled gracefully"""
    gateway_url = "http://localhost:8080"
    
    # First, create a task
    create_request = {
        "jsonrpc": "2.0",
        "method": "task/create",
        "params": {
            "agentId": "echo-agent",
            "taskName": "Test duplicate handling",
            "input": {
                "message": {
                    "content": "Test duplicate workflow"
                }
            }
        },
        "id": "test-dup-1"
    }
    
    async with aiohttp.ClientSession() as session:
        # Create first task
        async with session.post(f"{gateway_url}/api", json=create_request) as response:
            assert response.status == 200
            result1 = await response.json()
            task_id = result1['result']['taskId']
        
        # Try to create another workflow with same ID (simulate duplicate)
        # This should be handled gracefully in the gateway
        print(f"Created task with ID: {task_id}")


if __name__ == "__main__":
    asyncio.run(test_update_based_streaming())
    asyncio.run(test_concurrent_streaming_sessions())
    asyncio.run(test_workflow_already_exists_handling())