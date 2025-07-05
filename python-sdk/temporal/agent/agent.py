"""
Agent base class - Abstracts all Temporal complexity
"""
import asyncio
import functools
import logging
import inspect
from typing import Dict, Any, Callable, Optional, List
from temporalio import workflow, activity
from temporalio.client import Client
from temporalio.worker import Worker

from temporal.a2a.messages import A2AMessage, A2AResponse
from .runner import AgentRunner
from .decorators import message_handler as _message_handler_decorator
from .decorators import streaming_handler as _streaming_handler_decorator

logger = logging.getLogger(__name__)

# Global agent instance for decorator access
_agent_instance = None


def agent_activity(func):
    """
    Decorator to mark a method as an agent activity.
    This is the only Temporal construct exposed to agent developers.
    Activities run in separate processes and must handle their own imports.
    
    The decorator handles all A2A protocol complexity. Activities work with
    simple Python types (str, List[str], etc) instead of protocol objects.
    
    Example for simple message:
        @agent_activity
        async def process_message_activity(self, text: str) -> str:
            # Import everything needed here - activities run in separate processes
            from my_logic import MyLogic
            return MyLogic.process(text)
    
    Example for streaming:
        @agent_activity
        async def process_streaming_activity(self, text: str, stream) -> List[str]:
            chunks = []
            async for chunk in generate_chunks(text):
                chunks.append(chunk)
                await stream.send_chunk(chunk)  # Real-time streaming
            return chunks
    """
    import functools
    import inspect
    
    @functools.wraps(func)
    async def wrapper(self, message_data: Dict[str, Any]) -> Dict[str, Any]:
        # Import A2A types in the activity process
        from temporal.a2a.messages import A2AMessage, A2AResponse, A2AStreamingResponse
        from .streaming import streaming_context
        
        # Convert message data to A2AMessage
        message = A2AMessage.from_dict(message_data)
        text = message.get_text()
        
        # Check if this is a streaming activity (has 'stream' parameter)
        sig = inspect.signature(func)
        params = list(sig.parameters.keys())
        
        if len(params) >= 3 and params[2] == 'stream':
            # Streaming activity - create stream context and call with text + stream
            stream = streaming_context(message_data)
            result = await func(self, text, stream)
            
            # For streaming, activity returns None, but stream context has chunks
            if result is None:
                chunks = stream.get_chunks()
                return {
                    "is_streaming": True,
                    "chunks": chunks
                }
            else:
                raise ValueError(f"Streaming activity should return None (chunks collected via stream), got {type(result)}")
        else:
            # Simple activity - just pass text
            result = await func(self, text)
            
            # Check if this is a streaming result
            if isinstance(result, dict) and result.get("is_streaming"):
                return result
            elif isinstance(result, str):
                return A2AResponse.text(result, name="Response").to_dict()
            else:
                raise ValueError(f"Activity must return str or streaming dict, got {type(result)}")
    
    # Wrap with Temporal's activity decorator
    wrapped = activity.defn(wrapper)
    # Mark it so the SDK can discover it
    wrapped._is_agent_activity = True
    wrapped._original_name = func.__name__
    
    # Mark streaming activities
    if 'streaming' in func.__name__.lower():
        wrapped._is_streaming = True
    else:
        wrapped._is_streaming = False
    
    return wrapped


class Agent:
    """Base class for A2A agents - hides all Temporal complexity"""
    
    def __init__(self, agent_id: str, name: str, 
                 capabilities: Optional[Dict[str, Any]] = None,
                 metadata: Optional[Dict[str, Any]] = None):
        """Initialize an A2A agent
        
        Args:
            agent_id: Unique identifier for the agent
            name: Human-readable name for the agent
            capabilities: Dict of capabilities (e.g., {"streaming": True, "artifacts": True})
            metadata: Additional agent metadata
        """
        self.agent_id = agent_id
        self.name = name
        self.capabilities = capabilities or {}
        self.metadata = metadata or {}
        self._handlers = {}  # All handlers by type
        self._temporal_client = None
        self._worker = None
        
        # Set global instance for decorator access
        global _agent_instance
        _agent_instance = self
        
        # Auto-discover decorated methods
        self._discover_handlers()
        
    def _get_activities(self):
        """Get all activities decorated with @agent_activity"""
        activities = []
        for attr_name in dir(self):
            if attr_name.startswith('_'):
                continue
            attr = getattr(self, attr_name)
            if callable(attr) and hasattr(attr, '_is_agent_activity'):
                activities.append(attr)
        return activities
        
    def _discover_handlers(self):
        """Auto-discover decorated handler methods"""
        for attr_name in dir(self):
            if attr_name.startswith('_'):
                continue
            attr = getattr(self, attr_name)
            if callable(attr) and hasattr(attr, '_a2a_handler_type'):
                handler_type = attr._a2a_handler_type
                if handler_type not in self._handlers:
                    self._handlers[handler_type] = {}
                self._handlers[handler_type][attr_name] = attr
                logger.info(f"📌 Discovered {handler_type} handler: {attr_name}")
        
    def message_handler(self, func: Callable) -> Callable:
        """Decorator for message handlers"""
        # Apply the decorator
        decorated = _message_handler_decorator(func)
        # Register the handler
        if 'message' not in self._handlers:
            self._handlers['message'] = {}
        self._handlers['message'][func.__name__] = decorated
        return decorated
        
    def streaming_handler(self, func: Callable) -> Callable:
        """Decorator for streaming handlers"""
        # Apply the decorator
        decorated = _streaming_handler_decorator(func)
        # Register the handler
        if 'streaming' not in self._handlers:
            self._handlers['streaming'] = {}
        self._handlers['streaming'][func.__name__] = decorated
        return decorated
        
    def get_handler(self, handler_type: str) -> Optional[Callable]:
        """Get a handler by type"""
        handlers = self._handlers.get(handler_type, {})
        if handlers:
            # Return the first handler of this type
            return list(handlers.values())[0]
        return None
        
    async def _process_message(self, message_data: Dict[str, Any]) -> Dict[str, Any]:
        """Internal method to process messages - called by Temporal workflow"""
        # Convert to A2AMessage
        message = A2AMessage.from_dict(message_data)
        
        # Determine handler type based on capabilities and request
        handler_type = "message"  # Default
        if self.capabilities.get("streaming", False):
            # Check if this is a streaming request (could be in message metadata)
            handler_type = "streaming"
            
        # Get appropriate handler
        handler_func = self.get_handler(handler_type)
        if not handler_func:
            # Fallback to message handler
            handler_func = self.get_handler("message")
            
        if not handler_func:
            raise ValueError(f"No handler found for type: {handler_type}")
            
        # Call the handler
        if asyncio.iscoroutinefunction(handler_func):
            response = await handler_func(message)
        else:
            response = handler_func(message)
            
        # Convert response to dict
        return response.to_dict()
        
    async def run(self, temporal_host: Optional[str] = None,
                  namespace: Optional[str] = None):
        """Run the agent - all Temporal complexity hidden"""
        # Create runner and delegate all complexity
        runner = AgentRunner(self)
        
        # Setup client if parameters provided
        if temporal_host or namespace:
            await runner.setup_temporal_client(temporal_host, namespace)
        
        # Run the agent
        await runner.run()
        
    def run_sync(self, **kwargs):
        """Synchronous version of run() for simpler usage"""
        asyncio.run(self.run(**kwargs))


# Module-level agent instance for decorator syntax
agent = Agent.__new__(Agent)


def message_handler(func: Callable) -> Callable:
    """Module-level decorator for message handlers
    
    Can be used as @message_handler or @agent.message_handler
    """
    if _agent_instance is None:
        raise RuntimeError("No Agent instance created yet")
    return _agent_instance.message_handler(func)