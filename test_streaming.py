#!/usr/bin/env python3
"""
Quick test of SSE streaming endpoint
"""
import requests
import json
import time

def test_sse_streaming():
    url = "http://localhost:8080/a2a"
    
    # Create the JSON-RPC request for streaming
    request_data = {
        "jsonrpc": "2.0",
        "method": "message/stream",
        "params": {
            "agentId": "echo-agent",
            "message": {"text": "Test SSE streaming functionality"}
        },
        "id": "sse-test-001"
    }
    
    print("🔄 Testing SSE streaming endpoint...")
    print(f"Request: {json.dumps(request_data, indent=2)}")
    
    try:
        # Make streaming request
        response = requests.post(
            url,
            json=request_data,
            headers={
                "Content-Type": "application/json",
                "Accept": "text/event-stream"
            },
            stream=True  # Important for SSE
        )
        
        print(f"Response status: {response.status_code}")
        print(f"Response headers: {dict(response.headers)}")
        
        if response.status_code == 200:
            print("✅ SSE stream started, reading events...")
            
            # Read SSE events
            for line in response.iter_lines(decode_unicode=True):
                if line:
                    print(f"SSE: {line}")
                    
                    # Stop after a few seconds to avoid hanging
                    if "task.completed" in line or "task.error" in line:
                        break
        else:
            print(f"❌ Streaming failed: {response.status_code}")
            print(f"Response: {response.text}")
            
    except Exception as e:
        print(f"❌ Error testing streaming: {e}")

if __name__ == "__main__":
    test_sse_streaming()