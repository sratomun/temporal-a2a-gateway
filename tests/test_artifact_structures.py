#!/usr/bin/env python3
"""
A2A v0.2.5 Artifact Structure Tests

Tests the A2A artifact structures and progressive streaming logic without
requiring full Temporal environment.
"""

import pytest
import json
from datetime import datetime

class ArtifactPart:
    """A2A v0.2.5 compliant artifact part"""
    def __init__(self, kind: str, text: str = None, file: dict = None, data: dict = None):
        self.kind = kind
        self.text = text
        self.file = file
        self.data = data
    
    def to_dict(self):
        result = {"kind": self.kind}
        if self.text is not None:
            result["text"] = self.text
        if self.file is not None:
            result["file"] = self.file
        if self.data is not None:
            result["data"] = self.data
        return result

class Artifact:
    """A2A v0.2.5 compliant artifact structure"""
    def __init__(self, artifact_id: str, name: str, description: str = None, parts: list = None):
        self.artifact_id = artifact_id
        self.name = name
        self.description = description
        self.parts = parts or []
    
    def add_text_part(self, text: str):
        self.parts.append(ArtifactPart(kind="text", text=text))
    
    def to_dict(self):
        return {
            "artifactId": self.artifact_id,
            "name": self.name,
            "description": self.description,
            "parts": [part.to_dict() for part in self.parts]
        }

class TaskResult:
    """A2A v0.2.5 compliant task result structure"""
    def __init__(self, artifacts: list = None, error: str = None):
        self.artifacts = artifacts or []
        self.error = error
    
    def add_artifact(self, artifact: Artifact):
        self.artifacts.append(artifact)
    
    def to_dict(self):
        result = {
            "artifacts": [artifact.to_dict() for artifact in self.artifacts]
        }
        if self.error:
            result["error"] = self.error
        return result

class TestArtifactStructures:
    """Test A2A v0.2.5 artifact structures"""
    
    def test_artifact_part_text(self):
        """Test text artifact part creation"""
        part = ArtifactPart(kind="text", text="Hello world")
        part_dict = part.to_dict()
        
        assert part_dict["kind"] == "text"
        assert part_dict["text"] == "Hello world"
        assert "file" not in part_dict
        assert "data" not in part_dict
    
    def test_artifact_creation(self):
        """Test artifact creation with proper A2A structure"""
        artifact = Artifact(
            artifact_id="test-artifact-001",
            name="Test Artifact",
            description="Test artifact for validation"
        )
        artifact.add_text_part("Test content")
        
        artifact_dict = artifact.to_dict()
        
        # Verify required A2A fields
        assert artifact_dict["artifactId"] == "test-artifact-001"
        assert artifact_dict["name"] == "Test Artifact"
        assert artifact_dict["description"] == "Test artifact for validation"
        assert len(artifact_dict["parts"]) == 1
        assert artifact_dict["parts"][0]["kind"] == "text"
        assert artifact_dict["parts"][0]["text"] == "Test content"
    
    def test_task_result_with_artifacts(self):
        """Test task result structure with artifacts array"""
        artifact = Artifact("test-001", "Test")
        artifact.add_text_part("Content")
        
        task_result = TaskResult()
        task_result.add_artifact(artifact)
        
        result_dict = task_result.to_dict()
        
        # Verify A2A v0.2.5 structure
        assert "artifacts" in result_dict
        assert isinstance(result_dict["artifacts"], list)
        assert len(result_dict["artifacts"]) == 1
        assert result_dict["artifacts"][0]["artifactId"] == "test-001"

class TestProgressiveStreamingLogic:
    """Test progressive streaming logic"""
    
    def test_progressive_text_building(self):
        """Test word-by-word progressive text building"""
        input_text = "Echo: Progressive streaming test"
        words = input_text.split()
        
        progressive_artifacts = []
        current_text = ""
        
        for i, word in enumerate(words):
            if i == 0:
                current_text = word
            else:
                current_text += f" {word}"
            
            artifact = Artifact(
                artifact_id=f"progressive-{i+1}",
                name=f"Progressive Text (chunk {i+1}/{len(words)})",
                description=f"Progressive text - word {i+1} of {len(words)}"
            )
            artifact.add_text_part(current_text)
            progressive_artifacts.append(artifact.to_dict())
        
        # Verify progressive building
        assert len(progressive_artifacts) == len(words)
        
        # Check first artifact
        first = progressive_artifacts[0]
        assert first["parts"][0]["text"] == "Echo:"
        
        # Check final artifact
        final = progressive_artifacts[-1]
        assert final["parts"][0]["text"] == input_text
        
        # Verify each step builds on the previous
        for i in range(1, len(progressive_artifacts)):
            current_text = progressive_artifacts[i]["parts"][0]["text"]
            previous_text = progressive_artifacts[i-1]["parts"][0]["text"]
            assert current_text.startswith(previous_text)
    
    def test_task_artifact_update_event_structure(self):
        """Test TaskArtifactUpdateEvent structure for A2A v0.2.5"""
        artifact = Artifact(
            artifact_id="streaming-echo-task-123-chunk-1",
            name="Progressive Echo (chunk 1/4)",
            description="Progressive echo response - word 1 of 4"
        )
        artifact.add_text_part("Echo:")
        
        # Simulate TaskArtifactUpdateEvent structure
        event = {
            "taskId": "streaming-echo-task-123",
            "contextId": "context-456",
            "kind": "artifact-update",
            "artifact": artifact.to_dict(),
            "append": False,
            "lastChunk": False
        }
        
        # Verify A2A v0.2.5 TaskArtifactUpdateEvent structure
        required_fields = ["taskId", "contextId", "kind", "artifact", "append", "lastChunk"]
        for field in required_fields:
            assert field in event, f"Missing required field: {field}"
        
        assert event["kind"] == "artifact-update"
        assert event["append"] is False
        assert event["lastChunk"] is False
        
        # Verify artifact structure
        artifact_data = event["artifact"]
        artifact_fields = ["artifactId", "name", "description", "parts"]
        for field in artifact_fields:
            assert field in artifact_data, f"Missing artifact field: {field}"
    
    def test_progressive_streaming_sequence(self):
        """Test complete progressive streaming sequence"""
        test_message = "Test progressive streaming"
        echo_response = f"Echo: {test_message}"
        words = echo_response.split()
        
        streaming_events = []
        current_text = ""
        
        for i, word in enumerate(words):
            if i == 0:
                current_text = word
            else:
                current_text += f" {word}"
            
            artifact = Artifact(
                artifact_id=f"echo-task-123-chunk-{i+1}",
                name=f"Progressive Echo (chunk {i+1}/{len(words)})",
                description=f"Progressive echo response - word {i+1} of {len(words)}"
            )
            artifact.add_text_part(current_text)
            
            event = {
                "taskId": "echo-task-123",
                "contextId": "context-123",
                "kind": "artifact-update",
                "artifact": artifact.to_dict(),
                "append": False,
                "lastChunk": (i == len(words) - 1)
            }
            streaming_events.append(event)
        
        # Verify sequence properties
        assert len(streaming_events) == len(words)
        
        # Verify progressive lastChunk flags
        for i, event in enumerate(streaming_events):
            if i == len(streaming_events) - 1:
                assert event["lastChunk"] is True, "Final event should have lastChunk=True"
            else:
                assert event["lastChunk"] is False, f"Event {i} should have lastChunk=False"
        
        # Verify progressive text content
        texts = [event["artifact"]["parts"][0]["text"] for event in streaming_events]
        for i in range(1, len(texts)):
            assert texts[i].startswith(texts[i-1]), f"Text should build progressively: '{texts[i-1]}' -> '{texts[i]}'"

class TestMessageExtraction:
    """Test message extraction from A2A message formats"""
    
    def test_extract_text_from_a2a_message(self):
        """Test extracting text from A2A message parts"""
        message = {
            "messageId": "test-msg-001",
            "role": "user",
            "parts": [
                {"kind": "text", "text": "Hello streaming test"}
            ]
        }
        
        # Extract text (simulate workflow logic)
        extracted_text = ""
        if "parts" in message:
            for part in message["parts"]:
                if part.get("kind") == "text":
                    extracted_text = part.get("text", "")
                    break
        
        assert extracted_text == "Hello streaming test"
    
    def test_extract_text_with_multiple_parts(self):
        """Test text extraction with multiple parts"""
        message = {
            "messageId": "test-msg-002",
            "role": "user",
            "parts": [
                {"kind": "file", "file": {"name": "test.txt"}},
                {"kind": "text", "text": "Text content here"},
                {"kind": "data", "data": {"key": "value"}}
            ]
        }
        
        # Extract first text part
        extracted_text = ""
        if "parts" in message:
            for part in message["parts"]:
                if part.get("kind") == "text":
                    extracted_text = part.get("text", "")
                    break
        
        assert extracted_text == "Text content here"
    
    def test_extract_text_fallback(self):
        """Test fallback when no text parts found"""
        message = {
            "messageId": "test-msg-003",
            "role": "user",
            "parts": [
                {"kind": "file", "file": {"name": "test.txt"}}
            ]
        }
        
        # Extract with fallback
        extracted_text = ""
        if "parts" in message:
            for part in message["parts"]:
                if part.get("kind") == "text":
                    extracted_text = part.get("text", "")
                    break
        
        # Apply fallback
        if not extracted_text:
            extracted_text = "Hello"
        
        assert extracted_text == "Hello"

if __name__ == "__main__":
    pytest.main([__file__, "-v"])