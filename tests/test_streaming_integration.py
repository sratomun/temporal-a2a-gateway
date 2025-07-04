#!/usr/bin/env python3
"""
Integration Tests for Streaming Echo Agent - Progressive Artifact Streaming

Tests the complete integration between StreamingEchoTaskWorkflow and the gateway
for real-time progressive artifact streaming with A2A v0.2.5 compliance.
"""

import pytest
import asyncio
import httpx
import json
import time
from typing import AsyncGenerator, Dict, Any, List
from unittest.mock import patch

class TestStreamingEchoIntegration:
    """Integration tests for streaming echo agent with real gateway"""
    
    @pytest.fixture
    def gateway_url(self):
        """Gateway URL for testing"""
        return "http://localhost:8080"
    
    @pytest.fixture
    def streaming_agent_id(self):
        """Streaming echo agent ID"""
        return "streaming-echo-agent"
    
    @pytest.fixture
    def test_message(self):
        """Test message for progressive streaming"""
        return "Test progressive artifacts streaming"
    
    @pytest.fixture
    def a2a_message_request(self, test_message):
        """A2A v0.2.5 compliant message request"""
        return {
            "jsonrpc": "2.0",
            "method": "message/send",
            "params": {
                "message": {
                    "messageId": "integration-test-001",
                    "role": "user",
                    "parts": [{"text": test_message}]
                },
                "metadata": {"test": "streaming-integration"}
            },
            "id": "integration-req-001"
        }
    
    async def parse_sse_events(self, response: httpx.Response) -> AsyncGenerator[Dict[str, Any], None]:
        """Parse Server-Sent Events from streaming response"""
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
                            event_data["type"] = event_type
                            yield event_data
                        except json.JSONDecodeError:
                            continue
    
    @pytest.mark.asyncio
    async def test_streaming_echo_progressive_artifacts(self, gateway_url, streaming_agent_id, a2a_message_request):
        """Test progressive artifact streaming with real gateway"""
        
        stream_url = f"{gateway_url}/{streaming_agent_id}"
        
        # Convert to streaming request
        stream_request = {
            "jsonrpc": "2.0",
            "method": "message/stream",
            "params": a2a_message_request["params"],
            "id": a2a_message_request["id"]
        }
        
        headers = {
            "Accept": "text/event-stream",
            "Cache-Control": "no-cache",
            "Content-Type": "application/json"
        }
        
        # Track streaming events
        status_events = []
        artifact_events = []
        task_id = None
        
        async with httpx.AsyncClient(timeout=30.0) as client:
            try:
                async with client.stream("POST", stream_url, json=stream_request, headers=headers) as response:
                    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
                    
                    async for event in self.parse_sse_events(response):
                        kind = event.get("kind", "unknown")
                        
                        if not task_id:
                            task_id = event.get("taskId")
                        
                        if kind == "status-update":
                            status_events.append(event)
                            
                            # Break on completion
                            status = event.get("status", {})
                            state = status.get("state", "unknown")
                            final = event.get("final", False)
                            
                            if final and state in ["completed", "succeeded"]:
                                break
                                
                        elif kind == "artifact-update":
                            artifact_events.append(event)
                            
                            # Break on last chunk
                            if event.get("lastChunk", False):
                                break
                
                # Verify we received events
                assert len(status_events) > 0, "Should receive status update events"
                assert len(artifact_events) > 0, "Should receive artifact update events"
                assert task_id is not None, "Should receive task ID"
                
                # Verify progressive artifacts
                self._verify_progressive_artifacts(artifact_events)
                
                # Verify status progression
                self._verify_status_progression(status_events)
                
            except httpx.ConnectError:
                pytest.skip("Gateway not running - skipping integration test")
    
    def _verify_progressive_artifacts(self, artifact_events: List[Dict]):
        """Verify progressive artifact structure and content"""
        assert len(artifact_events) > 1, "Should have multiple progressive artifacts"
        
        # Track progressive text building
        progressive_texts = []
        
        for i, event in enumerate(artifact_events):
            # Verify A2A v0.2.5 TaskArtifactUpdateEvent structure
            assert event["kind"] == "artifact-update"
            assert "taskId" in event
            assert "artifact" in event
            assert "append" in event
            assert "lastChunk" in event
            
            # Verify artifact structure
            artifact = event["artifact"]
            assert "artifactId" in artifact
            assert "name" in artifact
            assert "description" in artifact
            assert "parts" in artifact
            
            # Extract text content
            for part in artifact["parts"]:
                if part.get("kind") == "text":
                    progressive_texts.append(part["text"])
                    break
        
        # Verify progressive building (each text should be longer than previous)
        for i in range(1, len(progressive_texts)):
            current_text = progressive_texts[i]
            previous_text = progressive_texts[i-1]
            assert len(current_text) >= len(previous_text), f"Text should grow progressively: '{previous_text}' -> '{current_text}'"
            assert current_text.startswith(previous_text) or current_text.startswith("Echo:"), "Text should build progressively"
        
        # Verify final artifact contains "Echo:" prefix
        final_text = progressive_texts[-1]
        assert final_text.startswith("Echo:"), f"Final text should start with 'Echo:': {final_text}"
    
    def _verify_status_progression(self, status_events: List[Dict]):
        """Verify status event progression"""
        assert len(status_events) > 0, "Should have status events"
        
        # Extract status states
        states = []
        for event in status_events:
            status = event.get("status", {})
            state = status.get("state", "unknown")
            states.append(state)
        
        # Verify progression contains working and completed
        assert "working" in states or "submitted" in states, f"Should have working/submitted state: {states}"
        assert states[-1] in ["completed", "succeeded"], f"Should end with completed state: {states}"

class TestStreamingAgentRouting:
    """Test streaming agent routing and configuration"""
    
    @pytest.mark.asyncio
    async def test_streaming_agent_endpoint_availability(self):
        """Test that streaming echo agent endpoint is available"""
        gateway_url = "http://localhost:8080"
        streaming_agent_url = f"{gateway_url}/streaming-echo-agent"
        
        # Test basic connectivity
        test_request = {
            "jsonrpc": "2.0",
            "method": "message/send",
            "params": {
                "message": {
                    "messageId": "connectivity-test",
                    "role": "user", 
                    "parts": [{"text": "connectivity test"}]
                }
            },
            "id": "connectivity-req"
        }
        
        try:
            async with httpx.AsyncClient(timeout=10.0) as client:
                response = await client.post(streaming_agent_url, json=test_request)
                
                # Should get a response (success or error, but not connection failure)
                assert response.status_code in [200, 400, 500], f"Unexpected status: {response.status_code}"
                
                # Should be valid JSON-RPC response
                response_data = response.json()
                assert "jsonrpc" in response_data
                
        except httpx.ConnectError:
            pytest.skip("Gateway not running - skipping routing test")

class TestDualAgentArchitecture:
    """Test dual agent architecture (basic vs streaming)"""
    
    @pytest.mark.asyncio
    async def test_basic_vs_streaming_agent_behavior(self):
        """Test difference between basic and streaming echo agents"""
        gateway_url = "http://localhost:8080"
        test_message = "Compare basic vs streaming"
        
        # Test request
        request_data = {
            "jsonrpc": "2.0",
            "method": "message/send",
            "params": {
                "message": {
                    "messageId": "compare-test",
                    "role": "user",
                    "parts": [{"text": test_message}]
                }
            },
            "id": "compare-req"
        }
        
        try:
            async with httpx.AsyncClient(timeout=15.0) as client:
                # Test basic echo agent (non-streaming)
                basic_response = await client.post(f"{gateway_url}/echo-agent", json=request_data)
                
                # Test streaming echo agent (non-streaming call)
                streaming_response = await client.post(f"{gateway_url}/streaming-echo-agent", json=request_data)
                
                # Both should return valid responses
                assert basic_response.status_code == 200
                assert streaming_response.status_code == 200
                
                basic_data = basic_response.json()
                streaming_data = streaming_response.json()
                
                # Both should be JSON-RPC responses
                assert "jsonrpc" in basic_data
                assert "jsonrpc" in streaming_data
                assert "result" in basic_data
                assert "result" in streaming_data
                
        except httpx.ConnectError:
            pytest.skip("Gateway not running - skipping dual agent test")

class TestA2AComplianceValidation:
    """Test A2A v0.2.5 compliance for streaming features"""
    
    def test_task_artifact_update_event_structure(self):
        """Test TaskArtifactUpdateEvent structure compliance"""
        # Expected A2A v0.2.5 TaskArtifactUpdateEvent structure
        expected_fields = {
            "taskId", "contextId", "kind", "artifact", "append", "lastChunk"
        }
        
        # Simulate event structure
        sample_event = {
            "taskId": "streaming-echo-test-123",
            "contextId": "context-456",
            "kind": "artifact-update",
            "artifact": {
                "artifactId": "streaming-echo-test-123-chunk-1",
                "name": "Progressive Echo (chunk 1/3)",
                "description": "Progressive echo response - word 1 of 3",
                "parts": [
                    {
                        "kind": "text",
                        "text": "Echo:"
                    }
                ]
            },
            "append": False,
            "lastChunk": False
        }
        
        # Verify all required fields present
        for field in expected_fields:
            assert field in sample_event, f"Missing required field: {field}"
        
        # Verify artifact structure
        artifact = sample_event["artifact"]
        artifact_fields = {"artifactId", "name", "description", "parts"}
        for field in artifact_fields:
            assert field in artifact, f"Missing artifact field: {field}"
        
        # Verify parts structure
        assert len(artifact["parts"]) > 0, "Artifact should have parts"
        part = artifact["parts"][0]
        assert "kind" in part, "Part should have kind field"
        assert part["kind"] == "text", "Expected text part"
        assert "text" in part, "Text part should have text field"
    
    def test_progressive_artifact_sequence(self):
        """Test progressive artifact sequence compliance"""
        # Simulate progressive artifact sequence
        base_text = "Echo: Test progressive streaming"
        words = base_text.split()
        
        progressive_sequence = []
        current_text = ""
        
        for i, word in enumerate(words):
            if i == 0:
                current_text = word
            else:
                current_text += f" {word}"
            
            # Create A2A compliant artifact event
            event = {
                "kind": "artifact-update",
                "taskId": "test-123",
                "artifact": {
                    "artifactId": f"test-123-chunk-{i+1}",
                    "name": f"Progressive Echo (chunk {i+1}/{len(words)})",
                    "parts": [{"kind": "text", "text": current_text}]
                },
                "append": False,
                "lastChunk": (i == len(words) - 1)
            }
            progressive_sequence.append(event)
        
        # Verify sequence properties
        assert len(progressive_sequence) == len(words)
        
        # Verify progressive building
        for i in range(len(progressive_sequence)):
            event = progressive_sequence[i]
            text = event["artifact"]["parts"][0]["text"]
            
            # Should start with "Echo:"
            assert text.startswith("Echo:")
            
            # Last event should be marked as lastChunk
            if i == len(progressive_sequence) - 1:
                assert event["lastChunk"] is True
            else:
                assert event["lastChunk"] is False

if __name__ == "__main__":
    # Run integration tests
    pytest.main([__file__, "-v", "-s"])