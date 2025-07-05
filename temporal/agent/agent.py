"""
Agent base class and activity decorator - Clean interface for developers
"""
import functools
import inspect
import logging
from typing import Dict, Any, Optional, Callable
from temporalio import activity

logger = logging.getLogger(__name__)


def agent_activity(func):
    """
    Decorator to mark a method as an agent activity.
    This is the only Temporal construct exposed to agent developers.
    
    Simple message example:
        @agent_activity
        async def process_message_activity(self, text: str) -> str:
            return f"Processed: {text}"
    
    Streaming example:
        @agent_activity
        async def process_streaming_activity(self, text: str, stream) -> None:
            async for chunk in generate_chunks(text):
                await stream.send_chunk(chunk)
    """
    @functools.wraps(func)
    async def wrapper(self, message_data: Dict[str, Any]) -> Dict[str, Any]:
        # Import A2A protocol types (hidden from developer)
        from temporal.a2a import A2AMessage, A2AResponse
        from .streaming import streaming_context
        
        # Extract text from A2A message
        message = A2AMessage.from_dict(message_data)
        text = message.get_text()
        
        # Check if this is a streaming activity
        sig = inspect.signature(func)
        params = list(sig.parameters.keys())
        
        if len(params) >= 3 and params[2] == 'stream':
            # Streaming activity
            stream = streaming_context(message_data)
            await func(self, text, stream)
            return {"is_streaming": True, "status": "completed"}
        else:
            # Simple activity
            result = await func(self, text)
            if isinstance(result, str):
                return A2AResponse.text(result).to_dict()
            else:
                raise ValueError(f"Activity must return str, got {type(result)}")
    
    # Wrap with Temporal's activity decorator
    wrapped = activity.defn(wrapper)
    wrapped._is_agent_activity = True
    
    # Set predictable names for SDK discovery
    if func.__name__ == "process_message_activity":
        wrapped.__name__ = "_agent_task_activity"
    elif func.__name__ == "process_streaming_activity":
        wrapped.__name__ = "_agent_streaming_activity"
    
    return wrapped


class Agent:
    """Base class for agents - hides all Temporal and A2A complexity"""
    
    def __init__(self, agent_id: str, name: str, 
                 capabilities: Optional[Dict[str, Any]] = None,
                 metadata: Optional[Dict[str, Any]] = None):
        self.agent_id = agent_id
        self.name = name
        self.capabilities = capabilities or {}
        self.metadata = metadata or {}
        
    async def run(self):
        """Run the agent with all complexity hidden"""
        # Import runner (which uses temporal.a2a internally)
        from .runner import AgentRunner
        runner = AgentRunner(self)
        await runner.run()