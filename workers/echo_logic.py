#!/usr/bin/env python3
"""
Pure echo business logic with zero Temporal dependencies
Agent 1 Sprint 4 - Step 1: Extract Pure Business Logic
"""
from typing import List


class EchoLogic:
    """Pure echo processing logic - no framework dependencies"""
    
    @staticmethod
    def process_message(message_text: str) -> str:
        """Pure echo logic - no Temporal dependencies"""
        return f"Echo: {message_text or 'Hello'}"
    
    @staticmethod 
    async def process_streaming_message(message_text: str):
        """Pure streaming logic - yields chunks as generator"""
        import asyncio
        full_response = f"Echo: {message_text or 'Hello'}"
        words = full_response.split()
        
        # Yield progressive chunks
        for i in range(len(words)):
            chunk = " ".join(words[:i+1])
            yield chunk
            await asyncio.sleep(0.5)  # Simulate processing delay


# Unit tests to verify pure logic
if __name__ == "__main__":
    import asyncio
    
    # Test basic message processing
    assert EchoLogic.process_message("test") == "Echo: test"
    assert EchoLogic.process_message("") == "Echo: Hello"
    assert EchoLogic.process_message(None) == "Echo: Hello"
    
    # Test streaming message processing
    async def test_streaming():
        chunks = []
        async for chunk in EchoLogic.process_streaming_message("Hello World"):
            chunks.append(chunk)
        assert chunks == ["Echo:", "Echo: Hello", "Echo: Hello World"]
        
        chunks_empty = []
        async for chunk in EchoLogic.process_streaming_message(""):
            chunks_empty.append(chunk)
        assert chunks_empty == ["Echo:", "Echo: Hello"]
        
        print("✅ All pure logic tests passed!")
    
    asyncio.run(test_streaming())