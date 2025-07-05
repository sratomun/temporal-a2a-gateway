"""
Workflow Bridge - Connects SDK agents to Temporal workflows
Agent 1 Sprint 4 - Step 4: Bridge SDK to Existing Workflows
"""
import asyncio
import inspect
from typing import Dict, Any, List
from temporalio import workflow
from datetime import timedelta

from .messages import A2AMessage, A2AResponse


class SDKWorkflowBridge:
    """Bridge between SDK agents and Temporal workflows"""
    
    @staticmethod
    def create_task_workflow(agent_instance):
        """Create a Temporal workflow class for an SDK agent"""
        
        @workflow.defn(name=f"{agent_instance.agent_id}_TaskWorkflow")
        class SDKBridgedTaskWorkflow:
            def __init__(self):
                self.progress_signals: List[Dict[str, Any]] = []
                self.agent = agent_instance
            
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
                workflow.logger.info(f"🚀 Starting SDK-bridged workflow for task {task_id}")
                
                try:
                    # Signal: Task started
                    await self.add_progress_signal("working", 0.1)
                    
                    # Convert task input to A2A message
                    message = A2AMessage.from_dict(task_input)
                    
                    # Signal: Processing message
                    await self.add_progress_signal("working", 0.5)
                    
                    # Use SDK agent's handler
                    handler = self.agent.get_handler('message')
                    if not handler:
                        raise ValueError("No message handler found in agent")
                    
                    # Call the handler
                    if asyncio.iscoroutinefunction(handler):
                        response = await handler(message)
                    else:
                        response = handler(message)
                    
                    # Convert response to dict
                    result = response.to_dict()
                    
                    # Signal: Task completed
                    await self.add_progress_signal("completed", 1.0, result)
                    workflow.logger.info(f"✅ SDK-bridged workflow completed for task {task_id}")
                    
                    return result
                    
                except Exception as e:
                    error_msg = str(e)
                    workflow.logger.error(f"❌ SDK-bridged workflow failed: {error_msg}")
                    
                    # Signal: Task failed
                    await self.add_progress_signal("failed", 0.0, error=error_msg)
                    
                    # Return error result
                    return {
                        "artifacts": [],
                        "error": error_msg
                    }
        
        return SDKBridgedTaskWorkflow
    
    @staticmethod
    def create_streaming_workflow(agent_instance):
        """Create a streaming Temporal workflow for an SDK agent"""
        
        @workflow.defn(name=f"{agent_instance.agent_id}_StreamingTaskWorkflow")
        class SDKBridgedStreamingWorkflow:
            def __init__(self):
                self.progress_signals: List[Dict[str, Any]] = []
                self.agent = agent_instance
                
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
                
                # Extract gateway workflow ID
                gateway_workflow_id = None
                message_data = task_input
                
                if isinstance(task_input, dict):
                    gateway_workflow_id = task_input.get("gateway_workflow_id")
                    message_data = task_input.get("message", task_input)
                
                try:
                    # Signal: Task started
                    await self.add_progress_signal("working", 0.1)
                    
                    if gateway_workflow_id:
                        await self._signal_gateway(gateway_workflow_id, "working", 0.1)
                    
                    # Convert to A2A message
                    message = A2AMessage.from_dict(message_data)
                    
                    # Get streaming handler
                    handler = self.agent.get_handler('streaming')
                    if not handler:
                        # Fallback to message handler
                        handler = self.agent.get_handler('message')
                    
                    if not handler:
                        raise ValueError("No handler found in agent")
                    
                    # Check if handler returns an async generator (streaming)
                    if inspect.isasyncgenfunction(handler):
                        # Streaming handler - collect chunks from async generator
                        chunks = []
                        chunk_count = 0
                        
                        async for chunk in handler(message):
                            chunk_count += 1
                            chunks.append(chunk)
                            
                            # Calculate progress
                            progress = 0.3 + (0.6 * min(chunk_count / 10, 0.9))  # Estimate progress
                            
                            # Create progressive artifact
                            artifact = {
                                "artifactId": f"sdk-streaming-{task_id}-chunk-{chunk_count}",
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
                            "artifactId": f"sdk-streaming-{task_id}-final",
                            "name": "Complete Response",
                            "parts": [{"kind": "text", "text": final_text}]
                        }
                        final_result = {"artifacts": [final_artifact]}
                    else:
                        # Non-streaming handler
                        if asyncio.iscoroutinefunction(handler):
                            result = await handler(message)
                        else:
                            result = handler(message)
                        
                        # Convert result
                        final_result = result.to_dict() if hasattr(result, 'to_dict') else result
                    
                    # Signal: Task completed
                    await self.add_progress_signal("completed", 1.0, final_result)
                    
                    if gateway_workflow_id and isinstance(result, list):
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
        
        return SDKBridgedStreamingWorkflow