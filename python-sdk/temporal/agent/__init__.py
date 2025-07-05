"""
temporal.agent - SDK for building A2A agents with Temporal
"""

from .agent import Agent, agent_activity
from .decorators import message_handler, streaming_handler
from .runner import AgentRunner
from .streaming import StreamingContext, streaming_context

__version__ = "0.1.0"
__all__ = [
    "Agent",
    "agent_activity", 
    "message_handler",
    "streaming_handler",
    "AgentRunner",
    "StreamingContext",
    "streaming_context",
]