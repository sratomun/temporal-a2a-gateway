"""
Agent Runner - Hides all Temporal complexity from developers
Agent 1 Sprint 4 - Step 6: Hide Workflow Registration
"""
import os
import logging
from typing import Optional, List
from temporalio.client import Client
from temporalio.worker import Worker

from .workflows import (
    AgentTaskWorkflow, AgentStreamingWorkflow, 
    register_agent,
    _agent_task_activity, _agent_streaming_activity
)
from .workflow_generator import WorkflowGenerator

logger = logging.getLogger(__name__)


class AgentRunner:
    """Runs SDK agents with all Temporal complexity hidden"""
    
    def __init__(self, agent_instance):
        """Initialize runner for an agent"""
        self.agent = agent_instance
        self.client = None
        self.worker = None
        
    async def setup_temporal_client(self, 
                                    temporal_host: Optional[str] = None,
                                    temporal_namespace: Optional[str] = None):
        """Setup Temporal client (hidden from developer)"""
        # Use environment variables or defaults
        host = temporal_host or os.getenv('TEMPORAL_HOST', 'localhost')
        port = os.getenv('TEMPORAL_PORT', '7233')
        namespace = temporal_namespace or os.getenv('TEMPORAL_NAMESPACE', 'default')
        
        connection_string = f"{host}:{port}"
        logger.info(f"🔌 Connecting to Temporal at {connection_string}")
        
        self.client = await Client.connect(
            connection_string,
            namespace=namespace
        )
    
    def _generate_workflows(self) -> List:
        """Generate workflows from agent configuration"""
        # Register the agent for activities to use
        register_agent(self.agent)
        
        workflows = []
        
        # Add task workflow
        workflows.append(AgentTaskWorkflow)
        
        # Add streaming workflow if agent supports it
        if self.agent.capabilities.get("streaming", False):
            workflows.append(AgentStreamingWorkflow)
        
        return workflows
    
    def _generate_activities(self) -> List:
        """Discover and return agent-defined activities"""
        activities = []
        
        # Discover @agent_activity decorated methods
        for attr_name in dir(self.agent):
            attr = getattr(self.agent, attr_name)
            if hasattr(attr, '_is_agent_activity'):
                activities.append(attr)
                logger.info(f"📌 Discovered agent activity: {attr_name}")
        
        # If no activities found, return the generic ones as fallback
        if not activities:
            logger.warning("⚠️ No @agent_activity methods found - using generic activities")
            activities = [_agent_task_activity, _agent_streaming_activity]
        
        return activities
    
    async def run(self):
        """Run the agent - hides all Temporal complexity"""
        logger.info(f"🚀 Starting {self.agent.name} (ID: {self.agent.agent_id})")
        
        # Setup Temporal connection if not already done
        if not self.client:
            await self.setup_temporal_client()
        
        # Generate workflows and activities
        workflows = self._generate_workflows()
        activities = self._generate_activities()
        
        # Determine task queue
        task_queue = f"{self.agent.agent_id}-tasks"
        
        # Create worker with generated components
        self.worker = Worker(
            self.client,
            task_queue=task_queue,
            workflows=workflows,
            activities=activities
        )
        
        logger.info(f"✅ {self.agent.name} ready on queue: {task_queue}")
        logger.info(f"📡 Capabilities: {self.agent.capabilities}")
        logger.info(f"🔧 Workflows: {len(workflows)}, Activities: {len(activities)}")
        
        # Run the worker
        await self.worker.run()
    
    async def shutdown(self):
        """Graceful shutdown"""
        if self.worker:
            await self.worker.shutdown()
        if self.client:
            await self.client.close()