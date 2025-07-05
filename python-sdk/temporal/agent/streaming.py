"""
Streaming support for SDK activities
"""
from typing import Dict, Any, Optional, List
from temporalio import activity
import logging

from temporal.a2a.messages import A2AArtifact, A2AProgressUpdate

logger = logging.getLogger(__name__)


class StreamingContext:
    """Context for streaming from activities - collects chunks for batch return"""
    
    def __init__(self, message_data: Dict[str, Any]):
        self.gateway_workflow_id = message_data.get("_gateway_workflow_id")
        self.task_id = activity.info().workflow_id
        self.chunk_count = 0
        self.chunks = []  # Collect all chunks for return
        # Consistent artifact ID throughout streaming session
        self.artifact_id = f"artifact-{self.task_id.split('-')[0][:8]}"
    
    async def send_chunk(self, chunk: str, artifact_name: str = "Progressive Response"):
        """Collect chunk - will be returned at end of activity"""
        self.chunk_count += 1
        self.chunks.append(chunk)
        logger.debug(f"Collected chunk {self.chunk_count}: '{chunk}'")
    
    async def finish(self):
        """Finish collecting chunks"""
        logger.info(f"✅ Collected {len(self.chunks)} chunks for batch return")
    
    def get_chunks(self) -> List[str]:
        """Get all collected chunks for activity return"""
        return self.chunks


def streaming_context(message_data: Dict[str, Any]) -> StreamingContext:
    """Create a streaming context for the activity"""
    return StreamingContext(message_data)