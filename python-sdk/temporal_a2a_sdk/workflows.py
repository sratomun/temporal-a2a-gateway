"""
Pre-defined workflow classes for SDK agents
These are templates that get configured at runtime
"""
from typing import Dict, Any, List, Optional
from datetime import timedelta
from temporalio import workflow, activity
import asyncio
import inspect

from .messages import A2AMessage, A2AResponse


# Global agent registry - maps agent_id to agent instance
_agent_registry = {}

# Business logic registry - maps agent_id to handler functions
_handler_registry = {}


def register_agent(agent):
    """Register an agent for workflow execution"""
    _agent_registry[agent.agent_id] = agent
    
    
def get_agent(agent_id: str):
    """Get agent instance by ID"""
    return _agent_registry.get(agent_id)


def register_handler(agent_id: str, handler_type: str, handler_func):
    """Register a handler function that can be called from activities"""
    if agent_id not in _handler_registry:
        _handler_registry[agent_id] = {}
    _handler_registry[agent_id][handler_type] = handler_func


# Generic SDK activities that delegate to agent-defined activities
@activity.defn
async def _agent_task_activity(agent_id: str, message_data: Dict[str, Any]) -> Dict[str, Any]:
    """Generic activity that finds and calls the agent's activity"""
    # This is a placeholder - the actual agent activity will be registered by the agent
    raise NotImplementedError(f"Agent {agent_id} must define its own @agent_activity")


@activity.defn  
async def _agent_streaming_activity(agent_id: str, message_data: Dict[str, Any]) -> Dict[str, Any]:
    """Generic streaming activity that finds and calls the agent's activity"""
    # This is a placeholder - the actual agent activity will be registered by the agent
    raise NotImplementedError(f"Agent {agent_id} must define its own @agent_activity")


@workflow.defn
class AgentTaskWorkflow:
    """Template workflow for SDK agents"""
    
    def __init__(self):
        self.progress_signals: List[Dict[str, Any]] = []
    
    @workflow.query
    def get_progress_signals(self) -> List[Dict[str, Any]]:
        """Query handler for gateway to retrieve progress signals"""
        return self.progress_signals
    
    async def add_progress_signal(self, status: str, progress: float = 0.0, 
                                  result: Any = None, error: str = ""):
        """Add a progress signal to the internal array"""
        task_id = workflow.info().workflow_id
        timestamp = workflow.now().isoformat().replace('+00:00', 'Z')
        signal = {
            "taskId": task_id,
            "status": status,
            "progress": progress,
            "result": result,
            "error": error,
            "timestamp": timestamp
        }
        self.progress_signals.append(signal)
        workflow.logger.info(f"📡 Added progress signal: {status}")
    
    @workflow.run
    async def run(self, task_input: Dict[str, Any]) -> Dict[str, Any]:
        task_id = workflow.info().workflow_id
        workflow.logger.info(f"🚀 Starting SDK workflow for task {task_id}")
        
        try:
            # Signal: Task started
            await self.add_progress_signal("working", 0.1)
            
            # Extract agent ID from task queue name
            task_queue = workflow.info().task_queue
            agent_id = task_queue.replace("-tasks", "")
            
            # Signal: Processing message
            await self.add_progress_signal("working", 0.5)
            
            # Use bridge activity to execute agent handler
            result = await workflow.execute_activity(
                _agent_task_activity,
                args=[agent_id, task_input],
                start_to_close_timeout=timedelta(seconds=30)
            )
            
            # Signal: Task completed
            await self.add_progress_signal("completed", 1.0, result)
            workflow.logger.info(f"✅ SDK workflow completed for task {task_id}")
            
            return result
            
        except Exception as e:
            error_msg = str(e)
            workflow.logger.error(f"❌ SDK workflow failed: {error_msg}")
            
            # Signal: Task failed
            await self.add_progress_signal("failed", 0.0, error=error_msg)
            
            # Return error result
            return {
                "artifacts": [],
                "error": error_msg
            }


@workflow.defn
class AgentStreamingWorkflow:
    """Template streaming workflow for SDK agents"""
    
    def __init__(self):
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
            workflow.logger.info(f"📤 Sent signal to gateway workflow: {status}")
        except Exception as e:
            workflow.logger.error(f"❌ Failed to signal gateway workflow: {e}")
    
    @workflow.query
    def get_progress_signals(self) -> List[Dict[str, Any]]:
        """Query handler for gateway to retrieve progress signals"""
        return self.progress_signals
    
    async def add_progress_signal(self, status: str, progress: float = 0.0, 
                                  result: Any = None, error: str = ""):
        """Add a progress signal to the internal array"""
        task_id = workflow.info().workflow_id
        timestamp = workflow.now().isoformat().replace('+00:00', 'Z')
        signal = {
            "taskId": task_id,
            "status": status,
            "progress": progress,
            "result": result,
            "error": error,
            "timestamp": timestamp
        }
        self.progress_signals.append(signal)
    
    @workflow.run
    async def run(self, task_input: Dict[str, Any]) -> Dict[str, Any]:
        task_id = workflow.info().workflow_id
        workflow.logger.info(f"🚀 Starting SDK streaming workflow for task {task_id}")
        workflow.logger.info(f"📥 Received task input: {task_input}")
        
        # Extract gateway workflow ID
        gateway_workflow_id = None
        message_data = task_input
        
        if isinstance(task_input, dict):
            gateway_workflow_id = task_input.get("gateway_workflow_id")
            message_data = task_input.get("message", task_input)
            workflow.logger.info(f"📨 Extracted message data: {message_data}")
        
        try:
            # Signal: Task started
            await self.add_progress_signal("working", 0.1)
            
            if gateway_workflow_id:
                await self._signal_gateway(gateway_workflow_id, "working", 0.1)
            
            # Extract agent ID from task queue name
            task_queue = workflow.info().task_queue
            agent_id = task_queue.replace("-tasks", "")
            
            # Pass gateway workflow ID to activity for real-time streaming
            activity_message_data = dict(message_data)
            if gateway_workflow_id:
                activity_message_data["_gateway_workflow_id"] = gateway_workflow_id
            
            # Execute streaming activity
            streaming_result = await workflow.execute_activity(
                _agent_streaming_activity,
                args=[agent_id, activity_message_data],
                start_to_close_timeout=timedelta(seconds=300)  # 5 min timeout for streaming
            )
            
            # Process streaming result
            if streaming_result.get("is_streaming"):
                # Handle progressive chunks
                chunks = streaming_result.get("chunks", [])
                workflow.logger.info(f"📊 Received {len(chunks)} chunks from activity")
                chunk_count = 0
                
                # Use consistent artifact ID for all chunks
                short_task_id = task_id.split('-')[0][:8]  # First 8 chars of UUID
                artifact_id = f"artifact-{short_task_id}"
                
                for chunk in chunks:
                    chunk_count += 1
                    workflow.logger.info(f"📝 Processing chunk {chunk_count}: '{chunk}'")
                    
                    # Calculate progress
                    progress = 0.3 + (0.6 * min(chunk_count / 10, 0.9))
                    
                    # Create progressive artifact with consistent ID
                    artifact = {
                        "artifactId": artifact_id,
                        "name": f"Progressive Response",
                        "parts": [{"kind": "text", "text": chunk}]
                    }
                    
                    progressive_result = {"artifacts": [artifact]}
                    await self.add_progress_signal("working", progress, progressive_result)
                    
                    if gateway_workflow_id:
                        await self._signal_gateway(gateway_workflow_id, "working", progress, artifact)
                
                # Final result with last chunk
                final_text = chunks[-1] if chunks else ""
                final_artifact = {
                    "artifactId": artifact_id,
                    "name": "Complete Response",
                    "parts": [{"kind": "text", "text": final_text}]
                }
                final_result = {"artifacts": [final_artifact]}
            else:
                # Non-streaming result
                final_result = streaming_result
            
            # Signal: Task completed
            await self.add_progress_signal("completed", 1.0, final_result)
            
            if gateway_workflow_id and streaming_result.get("is_streaming"):
                await self._signal_gateway(gateway_workflow_id, "completed", 1.0, final_artifact)
            
            workflow.logger.info(f"✅ SDK streaming workflow completed for task {task_id}")
            return final_result
            
        except Exception as e:
            error_msg = str(e)
            workflow.logger.error(f"❌ SDK streaming workflow failed: {error_msg}")
            
            await self.add_progress_signal("failed", 0.0, error=error_msg)
            
            return {
                "artifacts": [],
                "error": error_msg
            }