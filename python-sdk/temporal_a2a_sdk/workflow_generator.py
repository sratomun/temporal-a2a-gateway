"""
Workflow Generator - Auto-generates Temporal workflows and activities from SDK agents
Agent 1 Sprint 4 - Step 5: Auto-Generate Activities from Handlers
"""
import inspect
from typing import Callable, Type, Dict, Any, List
from temporalio import activity, workflow

from .messages import A2AMessage, A2AResponse


class WorkflowGenerator:
    """Generates Temporal workflows and activities from SDK agents"""
    
    @staticmethod
    def create_activity_from_handler(agent_instance, handler_name: str, handler_func: Callable) -> Callable:
        """Auto-generate Temporal activity from SDK handler"""
        
        # Determine if handler expects A2AMessage or raw text
        sig = inspect.signature(handler_func)
        params = list(sig.parameters.values())
        
        # Skip 'self' parameter
        if params and params[0].name == 'self':
            params = params[1:]
        
        expects_message = (
            params and 
            params[0].annotation in [A2AMessage, 'A2AMessage'] or
            'message' in params[0].name.lower()
        )
        
        if expects_message:
            # Handler expects A2AMessage
            @activity.defn(name=f"{agent_instance.agent_id}_{handler_name}_activity")
            async def generated_message_activity(message_data: Dict[str, Any]) -> Dict[str, Any]:
                """Generated activity that handles A2AMessage"""
                message = A2AMessage.from_dict(message_data)
                
                # Call the handler
                if inspect.iscoroutinefunction(handler_func):
                    response = await handler_func(agent_instance, message)
                else:
                    response = handler_func(agent_instance, message)
                
                # Convert response
                if hasattr(response, 'to_dict'):
                    return response.to_dict()
                elif isinstance(response, dict):
                    return response
                else:
                    # Wrap in A2AResponse if needed
                    return A2AResponse.text(str(response)).to_dict()
            
            return generated_message_activity
        else:
            # Handler expects raw text (legacy style)
            @activity.defn(name=f"{agent_instance.agent_id}_{handler_name}_activity")
            async def generated_text_activity(message_text: str) -> Any:
                """Generated activity that handles raw text"""
                if inspect.iscoroutinefunction(handler_func):
                    return await handler_func(agent_instance, message_text)
                else:
                    return handler_func(agent_instance, message_text)
            
            return generated_text_activity
    
    @staticmethod
    def create_activities_from_agent(agent_instance) -> List[Callable]:
        """Generate all activities from an agent's handlers"""
        activities = []
        
        # Generate activities for all handler types
        for handler_type, handlers in agent_instance._handlers.items():
            for handler_name, handler_func in handlers.items():
                activity_func = WorkflowGenerator.create_activity_from_handler(
                    agent_instance, handler_name, handler_func
                )
                activities.append(activity_func)
        
        # Also check for direct methods (backward compatibility)
        if hasattr(agent_instance, 'handle_message'):
            handler_func = getattr(agent_instance, 'handle_message')
            if handler_func not in agent_instance._handlers.get('message', {}).values():
                activity_func = WorkflowGenerator.create_activity_from_handler(
                    agent_instance, 'handle_message', handler_func
                )
                activities.append(activity_func)
        
        if hasattr(agent_instance, 'handle_streaming_message'):
            handler_func = getattr(agent_instance, 'handle_streaming_message')
            if handler_func not in agent_instance._handlers.get('streaming', {}).values():
                activity_func = WorkflowGenerator.create_activity_from_handler(
                    agent_instance, 'handle_streaming_message', handler_func
                )
                activities.append(activity_func)
        
        return activities
    
    @staticmethod
    def create_workflow_from_agent(agent_instance) -> Type:
        """Auto-generate workflow class from SDK agent"""
        agent_id = agent_instance.agent_id
        
        @workflow.defn(name=f"{agent_id}_GeneratedWorkflow")
        class GeneratedWorkflow:
            def __init__(self):
                self.progress_signals: List[Dict[str, Any]] = []
            
            @workflow.query
            def get_progress_signals(self) -> List[Dict[str, Any]]:
                """Query handler for progress signals"""
                return self.progress_signals
            
            async def add_progress_signal(self, status: str, progress: float = 0.0, 
                                          result: Any = None, error: str = ""):
                """Add a progress signal"""
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
                workflow.logger.info(f"🚀 Starting generated workflow for {agent_id} task {task_id}")
                
                try:
                    await self.add_progress_signal("working", 0.1)
                    
                    # Determine which activity to use
                    activity_name = f"{agent_id}_handle_message_activity"
                    
                    await self.add_progress_signal("working", 0.5)
                    
                    # Execute the activity
                    result = await workflow.execute_activity(
                        activity_name,
                        task_input,
                        start_to_close_timeout=workflow.timedelta(seconds=30)
                    )
                    
                    await self.add_progress_signal("completed", 1.0, result)
                    workflow.logger.info(f"✅ Generated workflow completed for task {task_id}")
                    
                    return result
                    
                except Exception as e:
                    error_msg = str(e)
                    workflow.logger.error(f"❌ Generated workflow failed: {error_msg}")
                    await self.add_progress_signal("failed", 0.0, error=error_msg)
                    return {"artifacts": [], "error": error_msg}
        
        return GeneratedWorkflow