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

# Workflow signal-based progressive streaming


@workflow.defn
class EchoTaskWorkflow:
    def __init__(self):
        # Internal progress signals storage for query-based streaming
        self.progress_signals: List[Dict[str, Any]] = []
    
    @workflow.query
    def get_progress_signals(self) -> List[Dict[str, Any]]:
        """Query handler for gateway to retrieve progress signals"""
        return self.progress_signals
    
    
    async def add_progress_signal(self, status: str, progress: float = 0.0, 
                          result: Any = None, error: str = ""):
        """Add a progress signal to the internal array and push to gateway"""
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
        signal_dict = signal.to_dict()
        self.progress_signals.append(signal_dict)
        logger.info(f"📡 Added progress signal for task {task_id}: {status}")
        
        # Progress signals stored for query access
    
    @workflow.run
    async def run(self, task_input: Dict[str, Any]) -> Dict[str, Any]:
        task_id = workflow.info().workflow_id
        logger.info(f"🚀 Starting echo workflow for task {task_id}")
        
        try:
            # Signal: Task started
            await self.add_progress_signal("working", 0.1)
            
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
            await self.add_progress_signal("working", 0.5)
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
            
            await self.add_progress_signal("completed", 1.0, result)
            logger.info(f"✅ Echo workflow completed for task {task_id}")
            
            return result
            
        except Exception as e:
            error_msg = str(e)
            logger.error(f"❌ Echo workflow failed for task {task_id}: {error_msg}")
            
            # Signal: Task failed
            await self.add_progress_signal("failed", 0.0, error=error_msg)
            
            # Create error result using A2A SDK types
            error_result = TaskResult(error=error_msg)
            return error_result.to_dict()


@workflow.defn
class StreamingEchoTaskWorkflow:
    def __init__(self):
        # Internal progress signals storage for query-based streaming
        self.progress_signals: List[Dict[str, Any]] = []
        
    async def _signal_gateway(self, gateway_workflow_id: str, status: str, 
                            progress: float = 0.0, artifact: Dict[str, Any] = None):
        """Send progress signal to gateway workflow"""
        try:
            handle = workflow.get_external_workflow_handle(gateway_workflow_id)
            update = {
                "taskId": workflow.info().workflow_id,
                "status": status,
                "progress": progress,
                "timestamp": workflow.now().isoformat().replace('+00:00', 'Z'),
                "append": progress > 0.1,  # Append after first chunk
                "lastChunk": status == "completed"
            }
            if artifact:
                update["artifact"] = artifact
                
            await handle.signal("progress_update", update)
            logger.info(f"📤 Sent signal to gateway workflow: {status}")
        except Exception as e:
            logger.error(f"❌ Failed to signal gateway workflow: {e}")
    
    @workflow.query
    def get_progress_signals(self) -> List[Dict[str, Any]]:
        """Query handler for gateway to retrieve progress signals"""
        return self.progress_signals
    
    
    async def add_progress_signal(self, status: str, progress: float = 0.0, 
                          result: Any = None, error: str = ""):
        """Add a progress signal to the internal array and push to gateway"""
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
        signal_dict = signal.to_dict()
        self.progress_signals.append(signal_dict)
        logger.info(f"📡 Added progress signal for task {task_id}: {status}")
        
        # Progress signals stored for query access
    
    @workflow.run
    async def run(self, task_input: Dict[str, Any]) -> Dict[str, Any]:
        task_id = workflow.info().workflow_id
        logger.info(f"🚀 Starting streaming echo workflow for task {task_id}")
        
        # Extract gateway workflow ID and message
        gateway_workflow_id = None
        message_data = task_input
        
        if isinstance(task_input, dict):
            gateway_workflow_id = task_input.get("gateway_workflow_id")
            message_data = task_input.get("message", task_input)
            
        logger.info(f"📡 Gateway workflow ID: {gateway_workflow_id}")
        
        try:
            # Signal: Task started
            await self.add_progress_signal("working", 0.1)
            
            # Send signal to gateway workflow if available
            if gateway_workflow_id:
                await self._signal_gateway(gateway_workflow_id, "working", 0.1)
            
            # Get message text from input
            user_message = ""
            
            # Try to extract text from message parts
            if isinstance(message_data, dict) and "parts" in message_data:
                for part in message_data["parts"]:
                    if isinstance(part, dict) and "text" in part:
                        user_message = part["text"]
                        break
            
            # Fallback: if still empty, use a default
            if not user_message:
                user_message = "Hello"
            
            logger.info(f"🔄 Progressive streaming for message: {user_message}")
            
            # Progressive streaming: Build response word by word
            echo_response = f"Echo: {user_message}"
            words = echo_response.split()
            
            # Stream each word progressively
            current_text = ""
            for i, word in enumerate(words):
                # Add word to current text
                if i == 0:
                    current_text = word
                else:
                    current_text += f" {word}"
                
                # Update progress
                progress = 0.3 + (0.6 * (i + 1) / len(words))  # 30% to 90%
                
                # Create progressive artifact with current text
                progressive_artifact = Artifact(
                    artifact_id=f"streaming-echo-{task_id}-chunk-{i+1}",
                    name=f"Progressive Echo (chunk {i+1}/{len(words)})",
                    description=f"Progressive echo response - word {i+1} of {len(words)}"
                )
                progressive_artifact.add_text_part(current_text)
                
                # Create progressive result
                progressive_result = TaskResult()
                progressive_result.add_artifact(progressive_artifact)
                
                # Send as intermediate progress with artifact
                await self.add_progress_signal("working", progress, progressive_result.to_dict())
                
                # Send signal to gateway workflow
                if gateway_workflow_id:
                    artifact_dict = {
                        "artifactId": progressive_artifact.artifact_id,
                        "name": progressive_artifact.name,
                        "parts": [{"kind": "text", "text": current_text}]
                    }
                    await self._signal_gateway(gateway_workflow_id, "working", progress, artifact_dict)
                
                # Small delay for demonstrating progressive streaming
                await asyncio.sleep(0.5)  # 500ms between words
            
            # Final artifact with complete response
            final_artifact = Artifact(
                artifact_id=f"streaming-echo-{task_id}-final",
                name="Complete Echo Response",
                description="Final complete echo response"
            )
            final_artifact.add_text_part(echo_response)
            final_result = TaskResult()
            final_result.add_artifact(final_artifact)
            result = final_result.to_dict()
            
            # Signal: Task completed with final result
            await self.add_progress_signal("completed", 1.0, result)
            
            # Send final signal to gateway workflow
            if gateway_workflow_id:
                final_artifact_dict = {
                    "artifactId": final_artifact.artifact_id,
                    "name": final_artifact.name,
                    "parts": [{"kind": "text", "text": echo_response}]
                }
                await self._signal_gateway(gateway_workflow_id, "completed", 1.0, final_artifact_dict)
            
            logger.info(f"✅ Streaming echo workflow completed for task {task_id}")
            
            return result
            
        except Exception as e:
            error_msg = str(e)
            logger.error(f"❌ Streaming echo workflow failed for task {task_id}: {error_msg}")
            
            # Signal: Task failed
            await self.add_progress_signal("failed", 0.0, error=error_msg)
            
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
    
    # Create workers for both echo agents
    workers = []
    
    # Basic echo agent worker
    basic_worker = Worker(
        client,
        task_queue="echo-agent-tasks",
        workflows=[EchoTaskWorkflow],
        activities=[echo_activity]
    )
    workers.append(basic_worker)
    
    # Streaming echo agent worker
    streaming_worker = Worker(
        client,
        task_queue="streaming-echo-agent-tasks",
        workflows=[StreamingEchoTaskWorkflow],
        activities=[echo_activity]
    )
    workers.append(streaming_worker)
    
    logger.info("Echo workers started - basic and streaming agents")
    
    # Run all workers concurrently
    await asyncio.gather(*[worker.run() for worker in workers])


if __name__ == "__main__":
    asyncio.run(main())