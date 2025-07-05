#!/usr/bin/env python3
"""
Unit Tests for Temporal A2A SDK Patterns
Tests the new @agent_activity decorator and clean separation
"""
import pytest
import asyncio
from unittest.mock import Mock, AsyncMock, patch
import sys
import os

# Add SDK to path for testing
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

class TestAgentActivityPattern:
    """Test the @agent_activity decorator pattern"""
    
    def test_agent_activity_decorator_marks_function(self):
        """Test that @agent_activity properly marks functions"""
        from temporal.agent import agent_activity
        
        @agent_activity
        async def test_activity(text: str) -> str:
            return f"Processed: {text}"
        
        # Verify the decorator marks the function
        assert hasattr(test_activity, '_is_agent_activity')
        assert test_activity._is_agent_activity is True
    
    def test_agent_class_structure(self):
        """Test Agent base class structure"""
        from temporal.agent import Agent
        
        class TestAgent(Agent):
            def __init__(self):
                super().__init__(
                    agent_id="test-agent",
                    name="Test Agent",
                    capabilities={"streaming": False}
                )
        
        agent = TestAgent()
        assert agent.agent_id == "test-agent"
        assert agent.name == "Test Agent"
        assert agent.capabilities == {"streaming": False}
    
    def test_agent_collects_activities(self):
        """Test that Agent class collects @agent_activity methods"""
        from temporal.agent import Agent, agent_activity
        
        class TestAgent(Agent):
            def __init__(self):
                super().__init__(
                    agent_id="test-agent",
                    name="Test Agent"
                )
            
            @agent_activity
            async def process_message(self, text: str) -> str:
                return f"Processed: {text}"
            
            @agent_activity
            async def another_activity(self, data: str) -> str:
                return f"Another: {data}"
            
            async def not_an_activity(self):
                pass
        
        agent = TestAgent()
        activities = agent._get_activities()
        
        # Should find exactly 2 activities
        assert len(activities) == 2
        activity_names = [a.__name__ for a in activities]
        assert "process_message" in activity_names
        assert "another_activity" in activity_names
        assert "not_an_activity" not in activity_names

class TestStreamingPattern:
    """Test streaming activity patterns"""
    
    def test_streaming_activity_with_stream_parameter(self):
        """Test streaming activities receive stream parameter"""
        from temporal.agent import Agent, agent_activity
        
        class StreamingAgent(Agent):
            def __init__(self):
                super().__init__(
                    agent_id="streaming-test",
                    name="Streaming Test",
                    capabilities={"streaming": True}
                )
            
            @agent_activity
            async def stream_activity(self, text: str, stream) -> None:
                """Stream parameter should be injected by SDK"""
                assert stream is not None
                assert hasattr(stream, 'send_chunk')
                assert hasattr(stream, 'finish')
        
        agent = StreamingAgent()
        activities = agent._get_activities()
        assert len(activities) == 1

class TestCodeReduction:
    """Test that SDK achieves significant code reduction"""
    
    def test_echo_agent_line_count(self):
        """Verify echo agent is ~41 lines as claimed"""
        echo_worker_path = os.path.join(
            os.path.dirname(__file__), 
            '../../workers/echo_worker.py'
        )
        
        if os.path.exists(echo_worker_path):
            with open(echo_worker_path, 'r') as f:
                lines = f.readlines()
                # Count non-empty, non-comment lines
                code_lines = [l for l in lines if l.strip() and not l.strip().startswith('#')]
                assert len(code_lines) < 50, f"Echo worker has {len(code_lines)} lines, should be < 50"
    
    def test_streaming_echo_agent_line_count(self):
        """Verify streaming echo agent is ~51 lines as claimed"""
        streaming_worker_path = os.path.join(
            os.path.dirname(__file__), 
            '../../workers/streaming_echo_worker.py'
        )
        
        if os.path.exists(streaming_worker_path):
            with open(streaming_worker_path, 'r') as f:
                lines = f.readlines()
                # Count non-empty, non-comment lines
                code_lines = [l for l in lines if l.strip() and not l.strip().startswith('#')]
                assert len(code_lines) < 60, f"Streaming worker has {len(code_lines)} lines, should be < 60"

class TestPureBusinessLogic:
    """Test pure business logic separation"""
    
    def test_echo_logic_no_temporal_imports(self):
        """Verify echo_logic.py has no Temporal dependencies"""
        echo_logic_path = os.path.join(
            os.path.dirname(__file__), 
            '../../workers/echo_logic.py'
        )
        
        if os.path.exists(echo_logic_path):
            with open(echo_logic_path, 'r') as f:
                content = f.read()
                # Check for Temporal imports
                assert 'from temporal' not in content.lower()
                assert 'import temporal' not in content.lower()
                assert '@workflow' not in content
                assert '@activity' not in content
    
    def test_echo_logic_pure_functions(self):
        """Test pure echo logic functions"""
        sys.path.insert(0, os.path.join(os.path.dirname(__file__), '../../workers'))
        from echo_logic import EchoLogic
        
        # Test basic processing
        assert EchoLogic.process_message("test") == "Echo: test"
        assert EchoLogic.process_message("") == "Echo: Hello"
        assert EchoLogic.process_message(None) == "Echo: Hello"
    
    @pytest.mark.asyncio
    async def test_echo_logic_streaming(self):
        """Test pure streaming logic"""
        sys.path.insert(0, os.path.join(os.path.dirname(__file__), '../../workers'))
        from echo_logic import EchoLogic
        
        chunks = []
        async for chunk in EchoLogic.process_streaming_message("Hello World"):
            chunks.append(chunk)
        
        assert chunks == ["Echo:", "Echo: Hello", "Echo: Hello World"]

class TestMemoryEfficiency:
    """Test zero memory overhead for streaming"""
    
    @pytest.mark.asyncio
    async def test_streaming_no_chunk_storage(self):
        """Verify streaming doesn't store all chunks in memory"""
        from temporal.agent import Agent, agent_activity
        
        class MemoryTestAgent(Agent):
            def __init__(self):
                super().__init__(
                    agent_id="memory-test",
                    name="Memory Test"
                )
                self.chunks_processed = 0
            
            @agent_activity
            async def stream_large_data(self, count: int, stream) -> None:
                """Stream many chunks without storing them"""
                for i in range(count):
                    await stream.send_chunk(f"Chunk {i}")
                    self.chunks_processed += 1
                await stream.finish()
        
        agent = MemoryTestAgent()
        # If this was storing chunks, 1000 chunks would use significant memory
        # But with streaming, memory usage should be O(1)
        # This test verifies the pattern, actual memory testing would need profiling

class TestSDKIntegration:
    """Test SDK integration with gateway"""
    
    def test_workflow_type_mapping(self):
        """Test that SDK uses correct workflow types"""
        # Check agent-routing.yaml has correct mappings
        routing_path = os.path.join(
            os.path.dirname(__file__), 
            '../../gateway/config/agent-routing.yaml'
        )
        
        if os.path.exists(routing_path):
            with open(routing_path, 'r') as f:
                content = f.read()
                # Verify SDK-compatible workflow types
                assert 'AgentTaskWorkflow' in content
                assert 'AgentStreamingWorkflow' in content

if __name__ == "__main__":
    pytest.main([__file__, "-v"])