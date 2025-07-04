#!/usr/bin/env python3

import asyncio
import os
import logging
import time
from datetime import datetime, timedelta, timezone
from typing import Dict, Any, List, Optional

from temporalio import workflow
from temporalio.activity import defn as activity
from temporalio.client import Client
from temporalio.worker import Worker

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


# A2A v0.2.5 SDK-compatible type definitions
class ArtifactPart:
    """A2A v0.2.5 compliant artifact part"""
    def __init__(self, kind: str, text: Optional[str] = None, file: Optional[Dict] = None, data: Optional[Dict] = None):
        self.kind = kind  # "text", "file", or "data"
        self.text = text
        self.file = file
        self.data = data
    
    def to_dict(self) -> Dict[str, Any]:
        result = {"kind": self.kind}
        if self.text is not None:
            result["text"] = self.text
        if self.file is not None:
            result["file"] = self.file
        if self.data is not None:
            result["data"] = self.data
        return result

class Artifact:
    """A2A v0.2.5 compliant artifact structure"""
    def __init__(self, artifact_id: str, name: str, description: Optional[str] = None, parts: Optional[List[ArtifactPart]] = None):
        self.artifact_id = artifact_id
        self.name = name
        self.description = description
        self.parts = parts or []
    
    def add_text_part(self, text: str) -> None:
        """Add a text part to the artifact"""
        self.parts.append(ArtifactPart(kind="text", text=text))
    
    def add_file_part(self, name: str, uri: str, mime_type: Optional[str] = None) -> None:
        """Add a file part to the artifact"""
        file_data = {"name": name, "uri": uri}
        if mime_type:
            file_data["mimeType"] = mime_type
        self.parts.append(ArtifactPart(kind="file", file=file_data))
    
    def add_data_part(self, data: Dict[str, Any]) -> None:
        """Add a data part to the artifact"""
        self.parts.append(ArtifactPart(kind="data", data=data))
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "artifactId": self.artifact_id,
            "name": self.name,
            "description": self.description,
            "parts": [part.to_dict() for part in self.parts]
        }

class TaskResult:
    """A2A v0.2.5 compliant task result structure"""
    def __init__(self, artifacts: Optional[List[Artifact]] = None, error: Optional[str] = None):
        self.artifacts = artifacts or []
        self.error = error
    
    def add_artifact(self, artifact: Artifact) -> None:
        """Add an artifact to the result"""
        self.artifacts.append(artifact)
    
    def to_dict(self) -> Dict[str, Any]:
        result = {
            "artifacts": [artifact.to_dict() for artifact in self.artifacts]
        }
        if self.error:
            result["error"] = self.error
        return result


class WorkflowProgressSignal:
    """Progress signal structure for streaming updates"""
    def __init__(self, task_id: str, status: str, progress: float = 0.0, 
                 result: Any = None, error: str = "", timestamp: str = None):
        self.task_id = task_id
        self.status = status
        self.progress = progress
        self.result = result
        self.error = error
        self.timestamp = timestamp  # Timestamp should be provided from workflow context
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "taskId": self.task_id,
            "status": self.status,
            "progress": self.progress,
            "result": self.result,
            "error": self.error,
            "timestamp": self.timestamp
        }


@activity
async def echo_activity(message_content: str) -> str:
    return f"Echo: {message_content}"


@workflow.defn
class EchoTaskWorkflow:
    def __init__(self):
        # Internal progress signals storage for query-based streaming
        self.progress_signals: List[Dict[str, Any]] = []
    
    @workflow.query
    def get_progress_signals(self) -> List[Dict[str, Any]]:
        """Query handler for gateway to retrieve progress signals"""
        return self.progress_signals
    
    def add_progress_signal(self, status: str, progress: float = 0.0, 
                          result: Any = None, error: str = ""):
        """Add a progress signal to the internal array"""
        task_id = workflow.info().workflow_id
        # Use workflow time for deterministic timestamps
        timestamp = workflow.now().isoformat().replace('+00:00', 'Z')
        signal = WorkflowProgressSignal(
            task_id=task_id,
            status=status,
            progress=progress,
            result=result,
            error=error,
            timestamp=timestamp
        )
        self.progress_signals.append(signal.to_dict())
        logger.info(f"📡 Added progress signal for task {task_id}: {status}")
    
    @workflow.run
    async def run(self, task_input: Dict[str, Any]) -> Dict[str, Any]:
        task_id = workflow.info().workflow_id
        logger.info(f"🚀 Starting echo workflow for task {task_id}")
        
        try:
            # Signal: Task started
            self.add_progress_signal("working", 0.1)
            
            # Get message text from input
            # The task_input IS the message object, not a wrapper
            message_data = task_input
            user_message = ""  # Default to empty string
            
            # Try to extract text from message parts
            if isinstance(message_data, dict) and "parts" in message_data:
                for part in message_data["parts"]:
                    if isinstance(part, dict) and "text" in part:
                        user_message = part["text"]
                        break
            
            # Fallback: if still empty, use a default
            if not user_message:
                user_message = "Hello"
            
            # Signal: Processing message
            self.add_progress_signal("working", 0.5)
            logger.info(f"🔄 Processing message: {user_message}")
            
            # Process with echo activity
            echo_response = await workflow.execute_activity(
                echo_activity,
                user_message,
                start_to_close_timeout=timedelta(seconds=30)
            )
            
            # Signal: Task completed with result
            # Create A2A v0.2.5 compliant artifact using SDK types
            echo_artifact = Artifact(
                artifact_id=f"echo-{task_id}",
                name="Echo Response", 
                description="Echoed user message"
            )
            echo_artifact.add_text_part(echo_response)
            
            # Create task result with artifact
            task_result = TaskResult()
            task_result.add_artifact(echo_artifact)
            result = task_result.to_dict()
            
            self.add_progress_signal("completed", 1.0, result)
            logger.info(f"✅ Echo workflow completed for task {task_id}")
            
            return result
            
        except Exception as e:
            error_msg = str(e)
            logger.error(f"❌ Echo workflow failed for task {task_id}: {error_msg}")
            
            # Signal: Task failed
            self.add_progress_signal("failed", 0.0, error=error_msg)
            
            # Create error result using A2A SDK types
            error_result = TaskResult(error=error_msg)
            return error_result.to_dict()




async def main():
    temporal_host = os.getenv('TEMPORAL_HOST', 'localhost')
    temporal_port = os.getenv('TEMPORAL_PORT', '7233')
    temporal_namespace = os.getenv('TEMPORAL_NAMESPACE', 'default')
    
    client = await Client.connect(
        f"{temporal_host}:{temporal_port}",
        namespace=temporal_namespace
    )
    
    worker = Worker(
        client,
        task_queue="echo-agent-tasks",
        workflows=[EchoTaskWorkflow],
        activities=[echo_activity]
    )
    
    logger.info("Echo worker started")
    await worker.run()


if __name__ == "__main__":
    asyncio.run(main())