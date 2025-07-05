"""
Internal Temporal workflow implementation
This is hidden from SDK users - they never see workflow complexity
"""
from typing import Dict, Any, Optional
from datetime import timedelta
from temporalio import workflow
import logging

logger = logging.getLogger(__name__)


# Global registry to store agent instances
_agent_instances = {}


# Predefined workflow classes for each agent type
@workflow.defn(name="EchoTaskWorkflow")
class EchoTaskWorkflow:
    """Workflow for echo-agent"""
    
    def __init__(self):
        self.workflow_id = None
        self.agent_id = "echo-agent"
        self.agent_name = None
        self.agent_capabilities = []
        
    @workflow.run
    async def run(self, task_input: Dict[str, Any]) -> Dict[str, Any]:
        """Execute the agent's message handler"""
        return await _execute_agent_workflow(self, task_input, self.agent_id)


@workflow.defn(name="StreamingEchoTaskWorkflow")
class StreamingEchoTaskWorkflow:
    """Workflow for streaming-echo-agent"""
    
    def __init__(self):
        self.workflow_id = None
        self.agent_id = "streaming-echo-agent"
        self.agent_name = None
        self.agent_capabilities = []
        
    @workflow.run
    async def run(self, task_input: Dict[str, Any]) -> Dict[str, Any]:
        """Execute the agent's message handler"""
        return await _execute_agent_workflow(self, task_input, self.agent_id)


async def _execute_agent_workflow(workflow_instance, task_input: Dict[str, Any], agent_id: str) -> Dict[str, Any]:
    """Common workflow execution logic"""
    workflow_instance.workflow_id = workflow.info().workflow_id
    
    # Get agent info from registry
    if agent_id in _agent_instances:
        agent = _agent_instances[agent_id]
        workflow_instance.agent_name = agent.name
        workflow_instance.agent_capabilities = agent.capabilities
    else:
        workflow_instance.agent_name = agent_id
        workflow_instance.agent_capabilities = []
        
    logger.info(f"🚀 Starting workflow {workflow_instance.workflow_id} for {workflow_instance.agent_name}")
    
    try:
        # Extract message from task input
        message_data = task_input
        if isinstance(task_input, dict):
            # Handle both direct message and wrapped formats
            if "message" in task_input:
                message_data = task_input["message"]
                
        # Check for gateway workflow ID (for streaming)
        gateway_workflow_id = None
        if isinstance(task_input, dict):
            gateway_workflow_id = task_input.get("gateway_workflow_id")
            
        # If streaming is supported and gateway workflow provided
        if gateway_workflow_id and "streaming" in workflow_instance.agent_capabilities:
            # Signal the gateway that we're starting
            await _signal_gateway(
                workflow_instance, gateway_workflow_id, "working", 0.1
            )
            
        # Process the message through the activity
        # This calls the agent's registered handler
        result = await workflow.execute_activity(
            f"process_message_{agent_id}",
            message_data,
            start_to_close_timeout=timedelta(minutes=5)
        )
        
        # If streaming, signal completion
        if gateway_workflow_id and "streaming" in workflow_instance.agent_capabilities:
            await _signal_gateway(
                workflow_instance, gateway_workflow_id, "completed", 1.0,
                artifact=result.get("artifacts", [{}])[0] if result.get("artifacts") else None
            )
            
        logger.info(f"✅ Workflow {workflow_instance.workflow_id} completed successfully")
        return result
        
    except Exception as e:
        logger.error(f"❌ Workflow {workflow_instance.workflow_id} failed: {e}")
        error_result = {"error": str(e)}
        
        # Signal failure if streaming
        if gateway_workflow_id and "streaming" in workflow_instance.agent_capabilities:
            await _signal_gateway(
                workflow_instance, gateway_workflow_id, "failed", 0.0
            )
            
        return error_result


async def _signal_gateway(workflow_instance, gateway_workflow_id: str, status: str,
                        progress: float = 0.0, 
                        artifact: Optional[Dict[str, Any]] = None):
    """Send progress signal to gateway workflow"""
    try:
        handle = workflow.get_external_workflow_handle(gateway_workflow_id)
        update = {
            "taskId": workflow_instance.workflow_id,
            "status": status,
            "progress": progress,
            "timestamp": workflow.now().isoformat().replace('+00:00', 'Z')
        }
        if artifact:
            update["artifact"] = artifact
            
        await handle.signal("progress_update", update)
        logger.info(f"📤 Sent signal to gateway: {status}")
    except Exception as e:
        logger.error(f"❌ Failed to signal gateway: {e}")


# Workflow registry mapping agent_id to workflow class
_workflow_registry = {
    "echo-agent": EchoTaskWorkflow,
    "streaming-echo-agent": StreamingEchoTaskWorkflow
}


class A2AAgentWorkflow:
    """Factory for creating agent workflows"""
    
    @staticmethod
    def create_for_agent(agent):
        """Register agent and return workflow class"""
        # Store agent instance globally
        _agent_instances[agent.agent_id] = agent
        
        # Return the specific workflow class for this agent
        if agent.agent_id in _workflow_registry:
            return _workflow_registry[agent.agent_id]
        else:
            # For unknown agents, we could create a generic workflow
            # but for now, raise an error
            raise ValueError(f"No workflow registered for agent: {agent.agent_id}")


class StreamingMixin:
    """Mixin for agents that support streaming"""
    
    async def stream_text(self, text: str, chunk_size: int = 10):
        """Stream text in chunks - to be implemented"""
        # This would integrate with the gateway workflow signals
        pass