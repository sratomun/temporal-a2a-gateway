"""
A2A SDK Exceptions
"""


class A2AException(Exception):
    """Base exception for A2A SDK"""
    pass


class TaskNotFoundException(A2AException):
    """Raised when a task is not found"""
    pass


class AgentNotFoundException(A2AException):
    """Raised when an agent is not found"""
    pass


class StreamingNotSupportedException(A2AException):
    """Raised when streaming is requested but not supported"""
    pass