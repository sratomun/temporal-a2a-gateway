"""
Temporal A2A SDK - Developer-friendly SDK for A2A protocol with Temporal
Sprint 4 Prototype Implementation
"""

from .agent import Agent, agent, agent_activity
from .messages import (
    A2AMessage, 
    A2AResponse, 
    A2AStreamingResponse, 
    A2ATask,
    A2AArtifact,
    A2AProgressUpdate
)
from .client import A2AClient
from .exceptions import A2AException, TaskNotFoundException
from .decorators import (
    message_handler,
    streaming_handler,
    context_aware,
    rate_limited,
    capability_required
)
from .workflow_bridge import SDKWorkflowBridge
from .workflow_generator import WorkflowGenerator
from .streaming import streaming_context

__version__ = "0.1.0"
__all__ = [
    "Agent",
    "agent",
    "agent_activity",
    "A2AMessage",
    "A2AResponse",
    "A2AStreamingResponse",
    "A2AArtifact",
    "A2AProgressUpdate",
    "A2ATask",
    "A2AClient",
    "A2AException",
    "TaskNotFoundException",
    "message_handler",
    "streaming_handler",
    "context_aware",
    "rate_limited",
    "capability_required",
    "streaming_context",
]