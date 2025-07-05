"""
A2A Client - Google A2A SDK compatible client interface
Direct Temporal workflow integration for Sprint 4
"""
import asyncio
import uuid
import os
from typing import Dict, Any, Optional, AsyncIterator, Union
from temporalio import workflow
from temporalio.client import Client, WorkflowHandle
from .messages import A2ATask, A2AMessage
from .exceptions import A2AException, TaskNotFoundException


class A2AClient:
    """Client for interacting with A2A agents - Direct Temporal integration"""
    
    def __init__(self, 
                 temporal_host: str = "localhost:7233",
                 namespace: str = "default"):
        """Initialize A2A client
        
        Args:
            temporal_host: Temporal server host:port
            namespace: Temporal namespace
        """
        self.temporal_host = temporal_host
        self.namespace = namespace
        self._client = None
        
    async def __aenter__(self):
        """Async context manager entry"""
        await self._get_client()
        return self
        
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit"""
        # Temporal client doesn't need explicit cleanup
        pass
            
    async def _get_client(self) -> Client:
        """Get or create Temporal client"""
        if self._client is None:
            self._client = await Client.connect(
                self.temporal_host,
                namespace=self.namespace
            )
        return self._client
        
    async def send_message(self, agent_id: str, message: Union[str, A2AMessage]) -> A2ATask:
        """Send a message to an agent by starting a Temporal workflow
        
        Args:
            agent_id: ID of the agent to send to
            message: Message to send (str or A2AMessage)
            
        Returns:
            A2ATask object representing the task
        """
        client = await self._get_client()
        
        # Convert string to A2A message format if needed
        if isinstance(message, str):
            message_data = {
                "role": "user",
                "parts": [{"kind": "text", "text": message}]
            }
        else:
            message_data = message.to_dict()
        
        # Generate unique task ID
        task_id = str(uuid.uuid4())
        
        # Start the agent workflow
        try:
            handle = await client.start_workflow(
                "AgentTaskWorkflow",  # Workflow type name
                args=[message_data],  # A2A message format
                id=task_id,
                task_queue=f"{agent_id}-tasks"  # Agent-specific task queue
            )
            
            return A2ATask(
                task_id=handle.id,
                agent_id=agent_id,
                status="submitted"
            )
        except Exception as e:
            raise A2AException(f"Failed to start workflow for agent {agent_id}: {e}")
            
    async def get_task(self, task_id: str) -> A2ATask:
        """Get task status and result by querying Temporal workflow
        
        Args:
            task_id: ID of the task to retrieve
            
        Returns:
            A2ATask object with current status
        """
        client = await self._get_client()
        
        try:
            # Get workflow handle
            handle = client.get_workflow_handle(task_id)
            
            # Check if workflow is running
            try:
                result = await handle.result()
                # Workflow completed
                return A2ATask(
                    task_id=task_id,
                    agent_id="unknown",  # Could be extracted from workflow info
                    status="completed",
                    result=result
                )
            except asyncio.CancelledError:
                # Workflow was cancelled
                return A2ATask(
                    task_id=task_id,
                    agent_id="unknown",
                    status="cancelled"
                )
            except Exception:
                # Workflow still running, check status
                describe = await handle.describe()
                status = describe.status.name.lower()
                
                return A2ATask(
                    task_id=task_id,
                    agent_id="unknown",
                    status=status
                )
                
        except Exception as e:
            raise TaskNotFoundException(f"Task {task_id} not found: {e}")
            
    async def cancel_task(self, task_id: str) -> bool:
        """Cancel a running task by cancelling the Temporal workflow
        
        Args:
            task_id: ID of the task to cancel
            
        Returns:
            True if cancelled successfully
        """
        client = await self._get_client()
        
        try:
            # Get workflow handle and cancel it
            handle = client.get_workflow_handle(task_id)
            await handle.cancel()
            return True
        except Exception:
            return False
            
    async def stream_message(self, agent_id: str, message: Union[str, A2AMessage]) -> AsyncIterator[Dict[str, Any]]:
        """Stream message responses using existing SSE infrastructure
        
        Args:
            agent_id: ID of the agent to send to
            message: Message to send
            
        Yields:
            Streaming events from the agent
        """
        import aiohttp
        import json
        
        # Convert string to A2A message format if needed
        if isinstance(message, str):
            message_data = {
                "role": "user",
                "parts": [{"kind": "text", "text": message}]
            }
        else:
            message_data = message.to_dict()
        
        # Use existing SSE endpoint that already works
        gateway_url = "http://localhost:8080"  # TODO: Make configurable
        
        async with aiohttp.ClientSession() as session:
            request_data = {
                "jsonrpc": "2.0",
                "method": "message/stream",
                "params": {
                    "message": message_data
                },
                "id": "1"
            }
            
            url = f"{gateway_url}/{agent_id}"
            async with session.post(url, json=request_data) as response:
                if response.status != 200:
                    text = await response.text()
                    raise A2AException(f"Failed to stream message: {response.status} - {text}")
                    
                # Process SSE stream (existing working implementation)
                async for line in response.content:
                    line = line.decode('utf-8').strip()
                    if line.startswith('data: '):
                        try:
                            event_data = json.loads(line[6:])
                            yield event_data
                        except json.JSONDecodeError:
                            continue
                        
    # Synchronous convenience methods
    def send_message_sync(self, agent_id: str, message: Union[str, A2AMessage]) -> A2ATask:
        """Synchronous version of send_message"""
        return asyncio.run(self.send_message(agent_id, message))
        
    def get_task_sync(self, task_id: str) -> A2ATask:
        """Synchronous version of get_task"""
        return asyncio.run(self.get_task(task_id))
        
    def cancel_task_sync(self, task_id: str) -> bool:
        """Synchronous version of cancel_task"""
        return asyncio.run(self.cancel_task(task_id))
        
    def close(self):
        """Close the client - Temporal client doesn't need explicit cleanup"""
        self._client = None