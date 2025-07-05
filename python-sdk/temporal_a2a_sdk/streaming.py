"""
Streaming support for SDK activities
"""
from typing import Dict, Any, Optional
from temporalio import activity
import logging

from .messages import A2AArtifact, A2AProgressUpdate

logger = logging.getLogger(__name__)


class StreamingContext:
    """Context for streaming from activities - hides all complexity"""
    
    def __init__(self, message_data: Dict[str, Any]):
        self.gateway_workflow_id = message_data.get("_gateway_workflow_id")
        self.task_id = activity.info().workflow_id
        self.chunk_count = 0
        self._handle = None
        # Consistent artifact ID throughout streaming session
        self.artifact_id = f"artifact-{self.task_id.split('-')[0][:8]}"
        
        if self.gateway_workflow_id:
            try:
                self._handle = activity.get_external_workflow_handle(self.gateway_workflow_id)
            except Exception as e:
                logger.warning(f"Could not get gateway handle: {e}")
    
    async def send_chunk(self, chunk: str, artifact_name: str = "Progressive Response"):
        """Send a single chunk - hides all signal complexity"""
        self.chunk_count += 1
        
        if not self._handle:
            # No gateway to signal, just log
            logger.debug(f"Chunk {self.chunk_count}: {chunk}")
            return
        
        # Create proper A2A types
        artifact = A2AArtifact.text(
            artifact_id=self.artifact_id,
            name=artifact_name,
            text=chunk
        )
        
        update = A2AProgressUpdate(
            task_id=self.task_id,
            status="working",
            timestamp=activity.now().isoformat().replace('+00:00', 'Z'),
            append=self.chunk_count > 1,
            last_chunk=False,
            artifact=artifact
        )
        
        try:
            await self._handle.signal("progress_update", update.to_dict())
            logger.info(f"📤 Streamed chunk {self.chunk_count}")
        except Exception as e:
            logger.error(f"Failed to stream chunk: {e}")
    
    async def finish(self):
        """Send final signal if needed"""
        if self._handle and self.chunk_count > 0:
            # Signal completion is handled by workflow
            pass


def streaming_context(message_data: Dict[str, Any]) -> StreamingContext:
    """Create a streaming context for the activity"""
    return StreamingContext(message_data)