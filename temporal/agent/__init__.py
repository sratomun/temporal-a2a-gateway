"""
temporal.agent - Simple interface for building Temporal A2A agents

This is what agent developers import. All A2A protocol complexity is hidden.
"""

from .agent import Agent, agent_activity
from .streaming import streaming_context

__all__ = ['Agent', 'agent_activity', 'streaming_context']