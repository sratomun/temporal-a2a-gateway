#!/usr/bin/env python3

import asyncio
import os
import logging
from datetime import timedelta
from typing import Dict, Any

from temporalio import workflow
from temporalio.activity import defn as activity
from temporalio.client import Client
from temporalio.worker import Worker

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


@activity
async def echo_activity(message_content: str) -> str:
    return f"Echo: {message_content}"


@workflow.defn
class EchoTaskWorkflow:
    @workflow.run
    async def run(self, task_input: Dict[str, Any]) -> Dict[str, Any]:
        try:
            # Get message text from input
            message_data = task_input.get("message", {})
            user_message = "Hello"  # Simple default
            
            # Try to extract text from message parts
            if isinstance(message_data, dict) and "parts" in message_data:
                for part in message_data["parts"]:
                    if isinstance(part, dict) and "text" in part:
                        user_message = part["text"]
                        break
            
            # Process with echo activity
            echo_response = await workflow.execute_activity(
                echo_activity,
                user_message,
                start_to_close_timeout=timedelta(seconds=30)
            )
            
            # Return simple result
            return {
                "status": "completed",
                "messages": [
                    message_data,
                    {
                        "messageId": f"echo-{workflow.info().workflow_id}",
                        "role": "agent",
                        "parts": [{"text": echo_response}]
                    }
                ]
            }
            
        except Exception as e:
            return {
                "status": "failed",
                "error": str(e),
                "messages": []
            }




async def main():
    temporal_host = os.getenv('TEMPORAL_HOST', 'localhost')
    temporal_port = os.getenv('TEMPORAL_PORT', '7233')
    temporal_namespace = os.getenv('TEMPORAL_NAMESPACE', 'default')
    
    client = await Client.connect(
        f"{temporal_host}:{temporal_port}",
        namespace=temporal_namespace
    )
    
    worker = Worker(
        client,
        task_queue="echo-agent-tasks",
        workflows=[EchoTaskWorkflow],
        activities=[echo_activity]
    )
    
    logger.info("Echo worker started")
    await worker.run()


if __name__ == "__main__":
    asyncio.run(main())