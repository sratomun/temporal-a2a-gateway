"""
Streaming support for agent activities
"""
from typing import Dict, Any
import logging

logger = logging.getLogger(__name__)


class StreamingContext:
    """Simple streaming interface for agents"""
    
    def __init__(self, message_data: Dict[str, Any]):
        # Import A2A internals (hidden from developer)
        from temporal.a2a import A2AArtifact, A2AProgressUpdate
        from temporalio import activity
        
        self._message_data = message_data
        self._chunk_count = 0
        self._setup_streaming()
        
    def _setup_streaming(self):
        """Setup streaming internals - hidden from developer"""
        from temporalio import activity
        
        self.gateway_workflow_id = self._message_data.get("_gateway_workflow_id")
        self.task_id = activity.info().workflow_id
        self.artifact_id = f"artifact-{self.task_id.split('-')[0][:8]}"
        self._handle = None
        
        if self.gateway_workflow_id:
            try:
                self._handle = activity.get_external_workflow_handle(self.gateway_workflow_id)
            except Exception as e:
                logger.warning(f"Could not get gateway handle: {e}")
    
    async def send_chunk(self, chunk: str):
        """Send a chunk - all protocol complexity hidden"""
        from temporal.a2a import A2AArtifact, A2AProgressUpdate
        from temporalio import activity
        
        self._chunk_count += 1
        
        if not self._handle:
            logger.debug(f"Chunk {self._chunk_count}: {chunk}")
            return
        
        # Create A2A protocol objects internally
        artifact = A2AArtifact.text(
            artifact_id=self.artifact_id,
            name="Progressive Response",
            text=chunk
        )
        
        update = A2AProgressUpdate(
            task_id=self.task_id,
            status="working",
            timestamp=activity.now().isoformat().replace('+00:00', 'Z'),
            append=self._chunk_count > 1,
            last_chunk=False,
            artifact=artifact
        )
        
        try:
            await self._handle.signal("progress_update", update.to_dict())
            logger.info(f"📤 Streamed chunk {self._chunk_count}")
        except Exception as e:
            logger.error(f"Failed to stream chunk: {e}")
    
    async def finish(self):
        """Finish streaming"""
        # Could send final signal here if needed
        pass


def streaming_context(message_data: Dict[str, Any]) -> StreamingContext:
    """Create a streaming context - simple interface"""
    return StreamingContext(message_data)