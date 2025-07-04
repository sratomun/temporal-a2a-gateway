#!/usr/bin/env python3
"""
Test if the workflow routing fix works by checking regular message/send
"""
import requests
import json
import time

def test_workflow_routing():
    url = "http://localhost:8080/a2a"
    
    # Test regular message/send to verify workflow routing is working
    request_data = {
        "jsonrpc": "2.0",
        "method": "message/send",
        "params": {
            "agentId": "echo-agent",
            "message": {
                "parts": [{"text": "Test workflow routing fix"}]
            }
        },
        "id": "workflow-test-001"
    }
    
    print("🔄 Testing workflow routing fix...")
    print(f"Request: {json.dumps(request_data, indent=2)}")
    
    try:
        response = requests.post(
            url,
            json=request_data,
            headers={"Content-Type": "application/json"},
            timeout=10
        )
        
        print(f"Response status: {response.status_code}")
        
        if response.status_code == 200:
            result = response.json()
            print(f"✅ Workflow started successfully!")
            print(f"Response: {json.dumps(result, indent=2)}")
            
            # Get task ID from response
            if 'result' in result and 'id' in result['result']:
                task_id = result['result']['id']
                print(f"📋 Task ID: {task_id}")
                
                # Wait a bit and check task status
                time.sleep(3)
                
                # Check task status
                status_request = {
                    "jsonrpc": "2.0",
                    "method": "tasks/get",
                    "params": {"id": task_id},
                    "id": "status-check-001"
                }
                
                status_response = requests.post(url, json=status_request)
                if status_response.status_code == 200:
                    status_result = status_response.json()
                    print(f"📊 Task Status: {json.dumps(status_result, indent=2)}")
                else:
                    print(f"⚠️ Failed to get task status: {status_response.status_code}")
            
        else:
            print(f"❌ Request failed: {response.status_code}")
            print(f"Response: {response.text}")
            
    except Exception as e:
        print(f"❌ Error testing workflow: {e}")

if __name__ == "__main__":
    test_workflow_routing()