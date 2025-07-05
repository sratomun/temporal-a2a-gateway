"""
A2A Client - Google A2A SDK compatible client interface
"""
import asyncio
import aiohttp
import json
from typing import Dict, Any, Optional, AsyncIterator
from .messages import A2ATask, A2AMessage
from .exceptions import A2AException, TaskNotFoundException


class A2AClient:
    """Client for interacting with A2A agents - Google SDK compatible"""
    
    def __init__(self, gateway_url: str = "http://localhost:8080"):
        """Initialize A2A client
        
        Args:
            gateway_url: URL of the A2A gateway
        """
        self.gateway_url = gateway_url.rstrip('/')
        self._session = None
        
    async def __aenter__(self):
        """Async context manager entry"""
        self._session = aiohttp.ClientSession()
        return self
        
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit"""
        if self._session:
            await self._session.close()
            
    def _get_session(self) -> aiohttp.ClientSession:
        """Get or create session"""
        if self._session is None:
            self._session = aiohttp.ClientSession()
        return self._session
        
    async def send_message(self, agent_id: str, message: A2AMessage) -> A2ATask:
        """Send a message to an agent
        
        Args:
            agent_id: ID of the agent to send to
            message: Message to send
            
        Returns:
            A2ATask object representing the task
        """
        session = self._get_session()
        
        # Prepare JSON-RPC request
        request_data = {
            "jsonrpc": "2.0",
            "method": "message/send",
            "params": {
                "message": message.to_dict()
            },
            "id": "1"
        }
        
        # Send to agent endpoint
        url = f"{self.gateway_url}/{agent_id}"
        async with session.post(url, json=request_data) as response:
            if response.status != 200:
                text = await response.text()
                raise A2AException(f"Failed to send message: {response.status} - {text}")
                
            data = await response.json()
            
            # Check for JSON-RPC error
            if "error" in data:
                raise A2AException(f"RPC Error: {data['error']}")
                
            # Extract task from result
            result = data.get("result", {})
            task = result.get("task", {})
            
            return A2ATask(
                task_id=task.get("id"),
                agent_id=agent_id,
                status=task.get("status", {}),
                result=task.get("result")
            )
            
    async def get_task(self, task_id: str) -> A2ATask:
        """Get task status and result
        
        Args:
            task_id: ID of the task to retrieve
            
        Returns:
            A2ATask object with current status
        """
        session = self._get_session()
        
        # Use any agent endpoint (task IDs are global)
        request_data = {
            "jsonrpc": "2.0",
            "method": "tasks/get",
            "params": {
                "taskIds": [task_id]
            },
            "id": "1"
        }
        
        # Send to gateway root (tasks are agent-agnostic)
        url = f"{self.gateway_url}/echo-agent"  # Any agent works
        async with session.post(url, json=request_data) as response:
            if response.status != 200:
                text = await response.text()
                raise A2AException(f"Failed to get task: {response.status} - {text}")
                
            data = await response.json()
            
            # Check for JSON-RPC error
            if "error" in data:
                raise A2AException(f"RPC Error: {data['error']}")
                
            # Extract task from result
            result = data.get("result", {})
            tasks = result.get("tasks", [])
            
            if not tasks:
                raise TaskNotFoundException(f"Task {task_id} not found")
                
            task = tasks[0]
            return A2ATask(
                task_id=task.get("id"),
                agent_id=task.get("agentId", "unknown"),
                status=task.get("status", {}),
                result=task.get("result")
            )
            
    async def cancel_task(self, task_id: str) -> bool:
        """Cancel a running task
        
        Args:
            task_id: ID of the task to cancel
            
        Returns:
            True if cancelled successfully
        """
        session = self._get_session()
        
        request_data = {
            "jsonrpc": "2.0",
            "method": "tasks/cancel",
            "params": {
                "taskIds": [task_id]
            },
            "id": "1"
        }
        
        url = f"{self.gateway_url}/echo-agent"  # Any agent works
        async with session.post(url, json=request_data) as response:
            if response.status != 200:
                return False
                
            data = await response.json()
            return "error" not in data
            
    async def stream_message(self, agent_id: str, message: A2AMessage) -> AsyncIterator[Dict[str, Any]]:
        """Stream message responses using Server-Sent Events
        
        Args:
            agent_id: ID of the agent to send to
            message: Message to send
            
        Yields:
            Streaming events from the agent
        """
        session = self._get_session()
        
        request_data = {
            "jsonrpc": "2.0",
            "method": "message/stream",
            "params": {
                "message": message.to_dict()
            },
            "id": "1"
        }
        
        url = f"{self.gateway_url}/{agent_id}"
        async with session.post(url, json=request_data) as response:
            if response.status != 200:
                text = await response.text()
                raise A2AException(f"Failed to stream message: {response.status} - {text}")
                
            # Process SSE stream
            async for line in response.content:
                line = line.decode('utf-8').strip()
                if line.startswith('data: '):
                    try:
                        event_data = json.loads(line[6:])
                        yield event_data
                    except json.JSONDecodeError:
                        continue
                        
    # Synchronous convenience methods
    def send_message_sync(self, agent_id: str, message: A2AMessage) -> A2ATask:
        """Synchronous version of send_message"""
        return asyncio.run(self.send_message(agent_id, message))
        
    def get_task_sync(self, task_id: str) -> A2ATask:
        """Synchronous version of get_task"""
        return asyncio.run(self.get_task(task_id))
        
    def cancel_task_sync(self, task_id: str) -> bool:
        """Synchronous version of cancel_task"""
        return asyncio.run(self.cancel_task(task_id))
        
    def close(self):
        """Close the client session"""
        if self._session:
            asyncio.run(self._session.close())