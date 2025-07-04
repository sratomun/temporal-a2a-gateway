#!/usr/bin/env python3
"""
Unit Tests for StreamingEchoTaskWorkflow - Progressive Artifact Streaming

Tests the progressive word-by-word artifact streaming implementation that 
Agent 2 created for A2A v0.2.5 compliance.
"""

import pytest
import asyncio
from unittest.mock import Mock, patch
from datetime import datetime, timedelta
import json

# Import the workflow classes from echo_worker
import sys
import os
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from workers.echo_worker import (
    StreamingEchoTaskWorkflow, 
    WorkflowProgressSignal,
    Artifact,
    ArtifactPart,
    TaskResult
)

# Mock temporal workflow functions
class MockWorkflowInfo:
    def __init__(self, workflow_id="test-streaming-echo-123"):
        self.workflow_id = workflow_id

class MockWorkflow:
    @staticmethod
    def info():
        return MockWorkflowInfo()
    
    @staticmethod
    def now():
        return datetime.now()

# Mock workflow module for testing
@pytest.fixture
def mock_workflow():
    with patch('workers.echo_worker.workflow', MockWorkflow):
        yield MockWorkflow

class TestStreamingEchoTaskWorkflow:
    """Test suite for StreamingEchoTaskWorkflow progressive streaming"""
    
    def setup_method(self):
        """Set up test workflow instance"""
        self.workflow = StreamingEchoTaskWorkflow()
    
    def test_workflow_initialization(self):
        """Test workflow initializes with empty progress signals"""
        assert self.workflow.progress_signals == []
    
    def test_get_progress_signals_empty(self):
        """Test progress signals query returns empty list initially"""
        signals = self.workflow.get_progress_signals()
        assert signals == []
        assert isinstance(signals, list)
    
    @patch('workers.echo_worker.workflow', MockWorkflow)
    def test_add_progress_signal(self):
        """Test adding progress signals with proper structure"""
        # Add a progress signal
        self.workflow.add_progress_signal("working", 0.5, {"test": "data"}, "")
        
        # Verify signal was added
        signals = self.workflow.get_progress_signals()
        assert len(signals) == 1
        
        signal = signals[0]
        assert signal["taskId"] == "test-streaming-echo-123"
        assert signal["status"] == "working"
        assert signal["progress"] == 0.5
        assert signal["result"] == {"test": "data"}
        assert signal["error"] == ""
        assert "timestamp" in signal
    
    @patch('workers.echo_worker.workflow', MockWorkflow)
    def test_multiple_progress_signals(self):
        """Test adding multiple progress signals maintains order"""
        # Add multiple signals
        self.workflow.add_progress_signal("working", 0.1)
        self.workflow.add_progress_signal("working", 0.5)
        self.workflow.add_progress_signal("completed", 1.0)
        
        signals = self.workflow.get_progress_signals()
        assert len(signals) == 3
        
        # Verify order and progression
        assert signals[0]["status"] == "working"
        assert signals[0]["progress"] == 0.1
        assert signals[1]["status"] == "working" 
        assert signals[1]["progress"] == 0.5
        assert signals[2]["status"] == "completed"
        assert signals[2]["progress"] == 1.0

class TestProgressiveArtifactGeneration:
    """Test progressive artifact generation logic"""
    
    def test_artifact_creation(self):
        """Test creating A2A v0.2.5 compliant artifacts"""
        artifact = Artifact(
            artifact_id="test-chunk-1",
            name="Progressive Echo (chunk 1/3)",
            description="Progressive echo response - word 1 of 3"
        )
        artifact.add_text_part("Echo:")
        
        artifact_dict = artifact.to_dict()
        
        # Verify A2A v0.2.5 structure
        assert artifact_dict["artifactId"] == "test-chunk-1"
        assert artifact_dict["name"] == "Progressive Echo (chunk 1/3)"
        assert artifact_dict["description"] == "Progressive echo response - word 1 of 3"
        assert len(artifact_dict["parts"]) == 1
        assert artifact_dict["parts"][0]["kind"] == "text"
        assert artifact_dict["parts"][0]["text"] == "Echo:"
    
    def test_progressive_text_building(self):
        """Test progressive text building for word-by-word streaming"""
        test_message = "Hello world test"
        words = test_message.split()
        progressive_texts = []
        
        # Simulate progressive building
        current_text = ""
        for i, word in enumerate(words):
            if i == 0:
                current_text = word
            else:
                current_text += f" {word}"
            progressive_texts.append(current_text)
        
        # Verify progressive building
        assert progressive_texts == ["Hello", "Hello world", "Hello world test"]
    
    def test_task_result_with_artifacts(self):
        """Test TaskResult structure with progressive artifacts"""
        artifact = Artifact("test-artifact", "Test")
        artifact.add_text_part("Test content")
        
        task_result = TaskResult()
        task_result.add_artifact(artifact)
        
        result_dict = task_result.to_dict()
        
        # Verify A2A v0.2.5 task result structure
        assert "artifacts" in result_dict
        assert len(result_dict["artifacts"]) == 1
        assert result_dict["artifacts"][0]["artifactId"] == "test-artifact"

class TestMessageExtraction:
    """Test message extraction from different input formats"""
    
    def test_extract_message_from_parts(self):
        """Test extracting message text from A2A message parts"""
        message_data = {
            "messageId": "test-msg-001",
            "role": "user",
            "parts": [
                {"kind": "text", "text": "Hello streaming test"}
            ]
        }
        
        # Simulate message extraction logic
        user_message = ""
        if isinstance(message_data, dict) and "parts" in message_data:
            for part in message_data["parts"]:
                if isinstance(part, dict) and "text" in part:
                    user_message = part["text"]
                    break
        
        assert user_message == "Hello streaming test"
    
    def test_extract_message_fallback(self):
        """Test fallback when no text parts found"""
        message_data = {
            "messageId": "test-msg-002",
            "role": "user",
            "parts": []
        }
        
        # Simulate message extraction with fallback
        user_message = ""
        if isinstance(message_data, dict) and "parts" in message_data:
            for part in message_data["parts"]:
                if isinstance(part, dict) and "text" in part:
                    user_message = part["text"]
                    break
        
        # Apply fallback
        if not user_message:
            user_message = "Hello"
        
        assert user_message == "Hello"

class TestProgressiveStreamingFlow:
    """Test complete progressive streaming flow simulation"""
    
    def test_progressive_streaming_simulation(self):
        """Test simulated progressive streaming flow"""
        input_message = "Test progressive streaming"
        echo_response = f"Echo: {input_message}"
        words = echo_response.split()
        
        # Simulate progressive artifacts
        progressive_artifacts = []
        current_text = ""
        
        for i, word in enumerate(words):
            if i == 0:
                current_text = word
            else:
                current_text += f" {word}"
            
            # Create progressive artifact
            artifact = Artifact(
                artifact_id=f"streaming-test-chunk-{i+1}",
                name=f"Progressive Echo (chunk {i+1}/{len(words)})",
                description=f"Progressive echo response - word {i+1} of {len(words)}"
            )
            artifact.add_text_part(current_text)
            progressive_artifacts.append(artifact.to_dict())
        
        # Verify progressive streaming
        assert len(progressive_artifacts) == 4  # "Echo:", "Echo: Test", "Echo: Test progressive", "Echo: Test progressive streaming"
        
        # Verify first chunk
        first_chunk = progressive_artifacts[0]
        assert first_chunk["name"] == "Progressive Echo (chunk 1/4)"
        assert first_chunk["parts"][0]["text"] == "Echo:"
        
        # Verify last chunk
        last_chunk = progressive_artifacts[-1]
        assert last_chunk["name"] == "Progressive Echo (chunk 4/4)"
        assert last_chunk["parts"][0]["text"] == "Echo: Test progressive streaming"
    
    def test_a2a_compliance_structure(self):
        """Test A2A v0.2.5 compliance for TaskArtifactUpdateEvent structure"""
        # Simulate TaskArtifactUpdateEvent structure that should be generated
        task_id = "streaming-echo-test-123"
        artifact = Artifact(
            artifact_id=f"streaming-echo-{task_id}-chunk-1",
            name="Progressive Echo (chunk 1/3)",
            description="Progressive echo response - word 1 of 3"
        )
        artifact.add_text_part("Echo:")
        
        # Simulate the event structure that gateway should generate
        task_artifact_update_event = {
            "taskId": task_id,
            "contextId": "context-123",
            "kind": "artifact-update",
            "artifact": artifact.to_dict(),
            "append": False,
            "lastChunk": False
        }
        
        # Verify A2A v0.2.5 TaskArtifactUpdateEvent structure
        assert task_artifact_update_event["kind"] == "artifact-update"
        assert task_artifact_update_event["taskId"] == task_id
        assert "contextId" in task_artifact_update_event
        assert "artifact" in task_artifact_update_event
        assert "append" in task_artifact_update_event
        assert "lastChunk" in task_artifact_update_event
        
        # Verify artifact structure
        artifact_data = task_artifact_update_event["artifact"]
        assert "artifactId" in artifact_data
        assert "name" in artifact_data
        assert "description" in artifact_data
        assert "parts" in artifact_data
        assert len(artifact_data["parts"]) == 1
        assert artifact_data["parts"][0]["kind"] == "text"

# Test timing and performance aspects
class TestStreamingPerformance:
    """Test streaming performance and timing aspects"""
    
    def test_progress_calculation(self):
        """Test progress calculation for streaming workflow"""
        words = ["Echo:", "Test", "progressive", "streaming", "now"]
        
        for i in range(len(words)):
            # Simulate progress calculation (30% to 90%)
            progress = 0.3 + (0.6 * (i + 1) / len(words))
            
            # Verify progress bounds
            assert 0.3 <= progress <= 0.9
        
        # Verify final progress
        final_progress = 0.3 + (0.6 * len(words) / len(words))
        assert final_progress == 0.9

if __name__ == "__main__":
    # Run tests
    pytest.main([__file__, "-v"])