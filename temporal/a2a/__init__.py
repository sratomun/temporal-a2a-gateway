"""
temporal.a2a - A2A protocol types and internals

This package contains all A2A protocol types and handling.
Agent developers don't need to use this directly.
"""

from .messages import (
    A2AMessage,
    A2AResponse,
    A2AStreamingResponse,
    A2AArtifact,
    A2AProgressUpdate,
    A2ATask
)

from .client import A2AClient
from .exceptions import A2AException, TaskNotFoundException

__all__ = [
    'A2AMessage',
    'A2AResponse', 
    'A2AStreamingResponse',
    'A2AArtifact',
    'A2AProgressUpdate',
    'A2ATask',
    'A2AClient',
    'A2AException',
    'TaskNotFoundException'
]