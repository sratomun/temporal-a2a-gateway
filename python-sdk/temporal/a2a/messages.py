"""
A2A Message abstractions - Python-specific implementation
"""
from typing import Dict, Any, List, Optional, Union
from datetime import datetime
from enum import Enum


# Core A2A Types per v0.2.5 specification

class TaskState(Enum):
    """Task state enumeration per A2A v0.2.5"""
    SUBMITTED = "submitted"
    WORKING = "working"
    INPUT_REQUIRED = "input-required"
    COMPLETED = "completed"
    CANCELED = "canceled"
    FAILED = "failed"
    REJECTED = "rejected"
    AUTH_REQUIRED = "auth-required"
    UNKNOWN = "unknown"


class A2APart:
    """Base class for A2A parts - union of TextPart, FilePart, DataPart"""
    
    def __init__(self, kind: str, metadata: Optional[Dict[str, Any]] = None):
        self.kind = kind
        self.metadata = metadata or {}
    
    @staticmethod
    def text(text: str, metadata: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Create a text part"""
        part = {"kind": "text", "text": text}
        if metadata:
            part["metadata"] = metadata
        return part
    
    @staticmethod
    def file(file_data: Dict[str, Any], metadata: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Create a file part"""
        part = {"kind": "file", "file": file_data}
        if metadata:
            part["metadata"] = metadata
        return part
    
    @staticmethod
    def data(data: Dict[str, Any], metadata: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Create a data part"""
        part = {"kind": "data", "data": data}
        if metadata:
            part["metadata"] = metadata
        return part


class A2AStatus:
    """Task status per A2A v0.2.5 specification"""
    
    def __init__(self, state: Union[str, TaskState], 
                 message: Optional[Dict[str, Any]] = None,
                 timestamp: Optional[str] = None):
        if isinstance(state, TaskState):
            self.state = state.value
        else:
            self.state = state
        self.message = message
        self.timestamp = timestamp or datetime.utcnow().isoformat() + "Z"
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        result = {"state": self.state}
        if self.message:
            result["message"] = self.message
        if self.timestamp:
            result["timestamp"] = self.timestamp
        return result
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "A2AStatus":
        """Create from dictionary"""
        return cls(
            state=data.get("state", "unknown"),
            message=data.get("message"),
            timestamp=data.get("timestamp")
        )


# Type alias for metadata
A2AMetadata = Optional[Dict[str, Any]]


class A2AMessage:
    """Represents an incoming A2A message"""
    
    def __init__(self, role: str, parts: List[Dict[str, Any]], 
                 timestamp: Optional[str] = None):
        self.role = role
        self.parts = parts
        self.timestamp = timestamp or datetime.utcnow().isoformat() + "Z"
        
    def get_text(self) -> str:
        """Extract text content from message parts"""
        for part in self.parts:
            if part.get("kind") == "text":
                return part.get("text", "")
        return ""
        
    def get_files(self) -> List[Dict[str, Any]]:
        """Extract file parts from message"""
        files = []
        for part in self.parts:
            if part.get("kind") == "file":
                files.append(part)
        return files
        
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "A2AMessage":
        """Create message from dictionary"""
        return cls(
            role=data.get("role", "user"),
            parts=data.get("parts", []),
            timestamp=data.get("timestamp")
        )
        
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return {
            "role": self.role,
            "parts": self.parts,
            "timestamp": self.timestamp
        }


class A2AResponse:
    """Builder for A2A responses with Python-friendly API"""
    
    def __init__(self):
        self.artifacts = []
        self.error = None
        
    @staticmethod
    def text(content: str, name: str = "Response") -> "A2AResponse":
        """Create a simple text response"""
        response = A2AResponse()
        response.add_text_artifact(name, content)
        return response
        
    @staticmethod
    def error(message: str) -> "A2AResponse":
        """Create an error response"""
        response = A2AResponse()
        response.error = message
        return response
        
    def add_text_artifact(self, name: str, content: str, 
                          artifact_id: Optional[str] = None) -> "A2AResponse":
        """Add a text artifact to the response"""
        import uuid
        artifact = {
            "artifactId": artifact_id or str(uuid.uuid4()),
            "name": name,
            "parts": [{"kind": "text", "text": content}]
        }
        self.artifacts.append(artifact)
        return self
        
    def add_file_artifact(self, name: str, file_name: str, 
                          uri: str, mime_type: Optional[str] = None,
                          artifact_id: Optional[str] = None) -> "A2AResponse":
        """Add a file artifact to the response"""
        import uuid
        file_part = {"kind": "file", "name": file_name, "uri": uri}
        if mime_type:
            file_part["mimeType"] = mime_type
            
        artifact = {
            "artifactId": artifact_id or str(uuid.uuid4()),
            "name": name,
            "parts": [file_part]
        }
        self.artifacts.append(artifact)
        return self
        
    def to_dict(self) -> Dict[str, Any]:
        """Convert to A2A task result format"""
        result = {"artifacts": self.artifacts}
        if self.error:
            result["error"] = self.error
        return result


class A2AStreamingResponse:
    """Response type for streaming activities"""
    
    def __init__(self, chunks: List[str], 
                 artifact_id: str = "streaming-response",
                 artifact_name: str = "Streaming Response"):
        self.chunks = chunks
        self.artifact_id = artifact_id
        self.artifact_name = artifact_name
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to the expected activity result format"""
        return {
            "is_streaming": True,
            "chunks": self.chunks,
            "artifacts": [{
                "artifactId": self.artifact_id,
                "name": self.artifact_name,
                "parts": [{
                    "kind": "text", 
                    "text": self.chunks[-1] if self.chunks else ""
                }]
            }]
        }
    
    @classmethod
    def from_chunks(cls, chunks: List[str], 
                    artifact_id: Optional[str] = None,
                    artifact_name: Optional[str] = None) -> 'A2AStreamingResponse':
        """Create a streaming response from chunks"""
        return cls(
            chunks=chunks,
            artifact_id=artifact_id or "streaming-response",
            artifact_name=artifact_name or "Streaming Response"
        )


class A2AArtifact:
    """Represents an A2A artifact"""
    
    def __init__(self, artifact_id: str, name: str, parts: List[Dict[str, Any]]):
        self.artifact_id = artifact_id
        self.name = name
        self.parts = parts
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "artifactId": self.artifact_id,
            "name": self.name,
            "parts": self.parts
        }
    
    @classmethod
    def text(cls, artifact_id: str, name: str, text: str) -> 'A2AArtifact':
        """Create a text artifact"""
        return cls(
            artifact_id=artifact_id,
            name=name,
            parts=[{"kind": "text", "text": text}]
        )


class A2AProgressUpdate:
    """Represents a progress update for streaming - matches A2A protocol"""
    
    def __init__(self, task_id: str, status: str = "working", 
                 timestamp: Optional[str] = None,
                 append: bool = False, last_chunk: bool = False,
                 artifact: Optional[A2AArtifact] = None):
        self.task_id = task_id
        self.status = status
        self.timestamp = timestamp or datetime.utcnow().isoformat() + "Z"
        self.append = append
        self.last_chunk = last_chunk
        self.artifact = artifact
    
    def to_dict(self) -> Dict[str, Any]:
        result = {
            "taskId": self.task_id,
            "status": self.status,
            "timestamp": self.timestamp,
            "append": self.append,
            "lastChunk": self.last_chunk
        }
        if self.artifact:
            result["artifact"] = self.artifact.to_dict()
        return result


class A2ATask:
    """Represents an A2A task (for client operations)"""
    
    def __init__(self, task_id: str, agent_id: str, status: Dict[str, Any],
                 result: Optional[Dict[str, Any]] = None):
        self.id = task_id
        self.task_id = task_id  # Alias for compatibility
        self.agent_id = agent_id
        self.status = status
        self.result = result
        
    @property
    def is_completed(self) -> bool:
        """Check if task is completed"""
        return self.status.get("state") == "completed"
        
    @property
    def is_failed(self) -> bool:
        """Check if task failed"""
        return self.status.get("state") == "failed"
        
    @property
    def is_running(self) -> bool:
        """Check if task is still running"""
        return self.status.get("state") in ["submitted", "working"]
        
    def get_artifacts(self) -> List[Dict[str, Any]]:
        """Get artifacts from completed task"""
        if self.result:
            return self.result.get("artifacts", [])
        return []
        
    def get_error(self) -> Optional[str]:
        """Get error message if task failed"""
        if self.result:
            return self.result.get("error")
        return None