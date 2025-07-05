"""
Decorator framework for Temporal A2A SDK
Agent 1 Sprint 4 - Step 3: Handler Decorators
"""
import functools
from typing import Callable, Any, Dict


def message_handler(func: Callable) -> Callable:
    """Decorator for basic message handlers"""
    func._a2a_handler_type = "message"
    func._a2a_handler_config = {}
    return func


def streaming_handler(func: Callable) -> Callable:
    """Decorator for streaming message handlers"""
    func._a2a_handler_type = "streaming"
    func._a2a_handler_config = {"supports_streaming": True}
    return func


def context_aware(func: Callable) -> Callable:
    """Decorator for handlers that need conversation context"""
    func._a2a_context_aware = True
    return func


def rate_limited(requests_per_minute: int) -> Callable:
    """Decorator for rate limiting"""
    def decorator(func: Callable) -> Callable:
        func._a2a_rate_limit = requests_per_minute
        return func
    return decorator


def capability_required(*capabilities: str) -> Callable:
    """Decorator that specifies required agent capabilities"""
    def decorator(func: Callable) -> Callable:
        func._a2a_required_capabilities = list(capabilities)
        return func
    return decorator