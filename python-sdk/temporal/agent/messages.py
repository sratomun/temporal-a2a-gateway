"""
Message types re-exported from a2a package for agent use
This allows activities to import messages without cross-package dependencies
"""
# Import from the a2a package
from temporal.a2a.messages import (
    A2AMessage,
    A2AResponse,
    A2AStreamingResponse,
    A2AArtifact,
    A2AProgressUpdate,
    A2ATask,
    A2APart,
    A2AStatus,
    A2AMetadata,
    TaskState
)

# Re-export everything
__all__ = [
    'A2AMessage',
    'A2AResponse', 
    'A2AStreamingResponse',
    'A2AArtifact',
    'A2AProgressUpdate',
    'A2ATask',
    'A2APart',
    'A2AStatus',
    'A2AMetadata',
    'TaskState'
]