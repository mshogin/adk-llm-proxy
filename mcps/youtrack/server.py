#!/usr/bin/env python3

import asyncio
import json
import logging
import os
import sys
from datetime import datetime, timedelta
from typing import Any, Dict, List, Optional

# Add parent directories to path for imports
sys.path.append(os.path.dirname(os.path.dirname(os.path.dirname(__file__))))

from src.infrastructure.mcp.server_base import MCPServerBase, mcp_tool, mcp_resource, mcp_prompt
from .youtrack_client import YouTrackClient, YouTrackConfig

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class YouTrackMCPServer(MCPServerBase):
    """YouTrack MCP Server for epic and task tracking."""

    def __init__(self):
        super().__init__("youtrack-mcp-server", "1.0.0")
        self.client: Optional[YouTrackClient] = None

    async def setup(self):
        """Initialize YouTrack client."""
        try:
            self.client = YouTrackClient.from_env()
            if not self.client.test_connection():
                logger.error("Failed to connect to YouTrack")
                raise ConnectionError("Cannot connect to YouTrack")

            user_info = self.client.get_user_info()
            logger.info(f"Connected to YouTrack as: {user_info.get('name', 'Unknown')}")
        except Exception as e:
            logger.error(f"YouTrack setup failed: {e}")
            raise

    # Epic Management Tools

    @mcp_tool(
        name="find_epic",
        description="Search for epics by name, ID, or tag",
        input_schema={
            "type": "object",
            "properties": {
                "search_term": {"type": "string", "description": "Epic name, ID, or tag to search for"},
                "project": {"type": "string", "description": "Optional project filter"}
            },
            "required": ["search_term"]
        }
    )
    async def find_epic(self, search_term: str, project: Optional[str] = None) -> str:
        """Find epics by search term."""
        if not self.client:
            return "Error: YouTrack client not initialized"

        try:
            epics = self.client.find_epics(search_term, project)

            if not epics:
                return f"No epics found matching '{search_term}'"

            result = f"Found {len(epics)} epic(s) matching '{search_term}':\n\n"
            for epic in epics:
                result += f"• {epic['idReadable']}: {epic['summary']}\n"
                if epic.get('project'):
                    result += f"  Project: {epic['project'].get('name', 'Unknown')}\n"
                result += "\n"

            return result

        except Exception as e:
            logger.error(f"Error finding epics: {e}")
            return f"Error searching for epics: {str(e)}"

    @mcp_tool(
        name="find_assigned_tickets",
        description="Find tickets assigned to the current user",
        input_schema={
            "type": "object",
            "properties": {
                "state": {"type": "string", "description": "Optional state filter (e.g., 'Open', 'Resolved')", "default": "Open"},
                "project": {"type": "string", "description": "Optional project filter"}
            },
            "required": []
        }
    )
    async def find_assigned_tickets(self, state: str = "Open", project: Optional[str] = None) -> str:
        """Find tickets assigned to the current user."""
        if not self.client:
            return "Error: YouTrack client not initialized"

        try:
            # Use proper YouTrack query syntax for assigned tickets
            query = "for: me"

            if state and state != "All":
                if state == "Open":
                    query += " #Unresolved"
                else:
                    query += f" State: {state}"

            if project:
                query += f" project: {project}"

            # Get assigned tickets using the search method
            tickets = self.client.search_issues(query)

            if not tickets:
                return f"No tickets assigned to you"

            result = f"Found {len(tickets)} ticket(s) assigned to you:\n\n"
            for ticket in tickets:
                result += f"• {ticket['idReadable']}: {ticket['summary']}\n"
                if ticket.get('project'):
                    result += f"  Project: {ticket['project'].get('name', 'Unknown')}\n"

                # Add assignee info to confirm
                if ticket.get('customFields'):
                    for field in ticket['customFields']:
                        if field.get('name') == 'Assignee' and field.get('value'):
                            assignee = field['value']
                            if isinstance(assignee, dict):
                                assignee_name = assignee.get('name', 'Unknown')
                            else:
                                assignee_name = str(assignee)
                            result += f"  Assignee: {assignee_name}\n"

                # Add state
                state_info = ticket.get('customFields', [])
                for field in state_info:
                    if field.get('name') == 'State' and field.get('value'):
                        state_val = field['value']
                        if isinstance(state_val, dict):
                            state_name = state_val.get('name', 'Unknown')
                        else:
                            state_name = str(state_val)
                        result += f"  State: {state_name}\n"

                result += "\n"

            return result

        except Exception as e:
            logger.error(f"Error finding assigned tickets: {e}")
            return f"Error searching for assigned tickets: {str(e)}"

    @mcp_tool(
        name="get_epic_details",
        description="Get detailed information about a specific epic",
        input_schema={
            "type": "object",
            "properties": {
                "epic_id": {"type": "string", "description": "Epic ID (e.g., PROJECT-123)"}
            },
            "required": ["epic_id"]
        }
    )
    async def get_epic_details(self, epic_id: str) -> str:
        """Get epic details."""
        if not self.client:
            return "Error: YouTrack client not initialized"

        try:
            epic = self.client.get_issue(epic_id)

            result = f"Epic Details: {epic['idReadable']}\n"
            result += "=" * 50 + "\n\n"
            result += f"Summary: {epic['summary']}\n"

            if epic.get('description'):
                result += f"Description: {epic['description'][:200]}{'...' if len(epic['description']) > 200 else ''}\n"

            if epic.get('project'):
                result += f"Project: {epic['project'].get('name', 'Unknown')}\n"

            # Custom fields
            custom_fields = epic.get('customFields', [])
            if custom_fields:
                result += "\nCustom Fields:\n"
                for field in custom_fields:
                    value = field.get('value')
                    if isinstance(value, dict):
                        value = value.get('name', str(value))
                    elif isinstance(value, list):
                        value = ', '.join([item.get('name', str(item)) for item in value])
                    result += f"  {field['name']}: {value}\n"

            # Dates
            if epic.get('created'):
                created_date = datetime.fromtimestamp(epic['created'] / 1000)
                result += f"Created: {created_date.strftime('%Y-%m-%d %H:%M')}\n"

            if epic.get('updated'):
                updated_date = datetime.fromtimestamp(epic['updated'] / 1000)
                result += f"Updated: {updated_date.strftime('%Y-%m-%d %H:%M')}\n"

            return result

        except Exception as e:
            logger.error(f"Error getting epic details: {e}")
            return f"Error retrieving epic details: {str(e)}"

    @mcp_tool(
        name="list_epic_tasks",
        description="List all tasks in an epic",
        input_schema={
            "type": "object",
            "properties": {
                "epic_id": {"type": "string", "description": "Epic ID (e.g., PROJECT-123)"}
            },
            "required": ["epic_id"]
        }
    )
    async def list_epic_tasks(self, epic_id: str) -> str:
        """List tasks in an epic."""
        if not self.client:
            return "Error: YouTrack client not initialized"

        try:
            tasks = self.client.get_epic_tasks(epic_id)

            if not tasks:
                return f"No tasks found for epic {epic_id}"

            result = f"Tasks in Epic {epic_id} ({len(tasks)} tasks):\n"
            result += "=" * 50 + "\n\n"

            for task in tasks:
                state = self.client.get_issue_custom_field_value(task, 'State') or 'Unknown'
                assignee = self.client.get_issue_custom_field_value(task, 'Assignee') or 'Unassigned'

                result += f"• {task['idReadable']}: {task['summary']}\n"
                result += f"  Status: {state} | Assignee: {assignee}\n"

                if task.get('updated'):
                    updated_date = datetime.fromtimestamp(task['updated'] / 1000)
                    result += f"  Updated: {updated_date.strftime('%Y-%m-%d')}\n"

                result += "\n"

            return result

        except Exception as e:
            logger.error(f"Error listing epic tasks: {e}")
            return f"Error listing tasks for epic: {str(e)}"

    @mcp_tool(
        name="get_epic_status_summary",
        description="Get current status summary for an epic",
        input_schema={
            "type": "object",
            "properties": {
                "epic_id": {"type": "string", "description": "Epic ID (e.g., PROJECT-123)"}
            },
            "required": ["epic_id"]
        }
    )
    async def get_epic_status_summary(self, epic_id: str) -> str:
        """Get epic status summary."""
        if not self.client:
            return "Error: YouTrack client not initialized"

        try:
            progress = self.client.calculate_epic_progress(epic_id)

            result = f"Epic Status Summary: {epic_id}\n"
            result += "=" * 50 + "\n\n"
            result += f"Total Tasks: {progress['total_tasks']}\n"
            result += f"Completed: {progress['completed_tasks']}\n"
            result += f"In Progress: {progress['in_progress_tasks']}\n"
            result += f"Open: {progress['open_tasks']}\n"
            result += f"Blocked: {progress['blocked_tasks']}\n"
            result += f"Completion: {progress['completion_percentage']:.1f}%\n"

            if progress['story_points_total'] > 0:
                result += f"\nStory Points: {progress['story_points_completed']}/{progress['story_points_total']}\n"

            return result

        except Exception as e:
            logger.error(f"Error getting epic status: {e}")
            return f"Error retrieving epic status: {str(e)}"

    # Task Analysis Tools

    @mcp_tool(
        name="get_task_details",
        description="Get detailed information about a specific task",
        input_schema={
            "type": "object",
            "properties": {
                "task_id": {"type": "string", "description": "Task ID (e.g., PROJECT-456)"}
            },
            "required": ["task_id"]
        }
    )
    async def get_task_details(self, task_id: str) -> str:
        """Get task details."""
        if not self.client:
            return "Error: YouTrack client not initialized"

        try:
            task = self.client.get_issue(task_id)

            result = f"Task Details: {task['idReadable']}\n"
            result += "=" * 50 + "\n\n"
            result += f"Summary: {task['summary']}\n"

            if task.get('description'):
                result += f"Description: {task['description'][:300]}{'...' if len(task['description']) > 300 else ''}\n"

            # Status and assignee
            state = self.client.get_issue_custom_field_value(task, 'State') or 'Unknown'
            assignee = self.client.get_issue_custom_field_value(task, 'Assignee') or 'Unassigned'
            priority = self.client.get_issue_custom_field_value(task, 'Priority') or 'Unknown'

            result += f"\nStatus: {state}\n"
            result += f"Assignee: {assignee}\n"
            result += f"Priority: {priority}\n"

            # Story points
            story_points = self.client.get_issue_custom_field_value(task, 'Story Points')
            if story_points:
                result += f"Story Points: {story_points}\n"

            # Dates
            if task.get('created'):
                created_date = datetime.fromtimestamp(task['created'] / 1000)
                result += f"Created: {created_date.strftime('%Y-%m-%d %H:%M')}\n"

            if task.get('updated'):
                updated_date = datetime.fromtimestamp(task['updated'] / 1000)
                result += f"Updated: {updated_date.strftime('%Y-%m-%d %H:%M')}\n"

            return result

        except Exception as e:
            logger.error(f"Error getting task details: {e}")
            return f"Error retrieving task details: {str(e)}"

    @mcp_tool(
        name="get_task_comments",
        description="Get recent comments for a task",
        input_schema={
            "type": "object",
            "properties": {
                "task_id": {"type": "string", "description": "Task ID (e.g., PROJECT-456)"},
                "limit": {"type": "integer", "description": "Maximum number of comments (default: 10)"}
            },
            "required": ["task_id"]
        }
    )
    async def get_task_comments(self, task_id: str, limit: int = 10) -> str:
        """Get task comments."""
        if not self.client:
            return "Error: YouTrack client not initialized"

        try:
            comments = self.client.get_issue_comments(task_id, limit)

            if not comments:
                return f"No comments found for task {task_id}"

            result = f"Comments for Task {task_id} (showing {len(comments)} most recent):\n"
            result += "=" * 50 + "\n\n"

            for comment in comments:
                if comment.get('deleted'):
                    continue

                author = comment.get('author', {}).get('name', 'Unknown')
                created_date = datetime.fromtimestamp(comment['created'] / 1000)
                text = comment.get('text', '')[:200]

                result += f"• {author} ({created_date.strftime('%Y-%m-%d %H:%M')}):\n"
                result += f"  {text}{'...' if len(comment.get('text', '')) > 200 else ''}\n\n"

            return result

        except Exception as e:
            logger.error(f"Error getting task comments: {e}")
            return f"Error retrieving task comments: {str(e)}"

    @mcp_tool(
        name="get_task_history",
        description="Get change history for a task",
        input_schema={
            "type": "object",
            "properties": {
                "task_id": {"type": "string", "description": "Task ID (e.g., PROJECT-456)"},
                "limit": {"type": "integer", "description": "Maximum number of history entries (default: 10)"}
            },
            "required": ["task_id"]
        }
    )
    async def get_task_history(self, task_id: str, limit: int = 10) -> str:
        """Get task history."""
        if not self.client:
            return "Error: YouTrack client not initialized"

        try:
            history = self.client.get_issue_history(task_id, limit)

            if not history:
                return f"No history found for task {task_id}"

            result = f"Change History for Task {task_id} (showing {len(history)} most recent):\n"
            result += "=" * 50 + "\n\n"

            for entry in history:
                author = entry.get('author', {}).get('name', 'Unknown')
                timestamp = datetime.fromtimestamp(entry['timestamp'] / 1000)
                field = entry.get('field', {}).get('name', 'Unknown')

                result += f"• {author} ({timestamp.strftime('%Y-%m-%d %H:%M')}):\n"
                result += f"  Changed {field}\n"

                removed = entry.get('removed')
                added = entry.get('added')

                if removed:
                    removed_str = removed[0].get('name', str(removed[0])) if isinstance(removed, list) else str(removed)
                    result += f"  From: {removed_str}\n"

                if added:
                    added_str = added[0].get('name', str(added[0])) if isinstance(added, list) else str(added)
                    result += f"  To: {added_str}\n"

                result += "\n"

            return result

        except Exception as e:
            logger.error(f"Error getting task history: {e}")
            return f"Error retrieving task history: {str(e)}"

    @mcp_tool(
        name="analyze_task_activity",
        description="Analyze task activity over the last week",
        input_schema={
            "type": "object",
            "properties": {
                "task_id": {"type": "string", "description": "Task ID (e.g., PROJECT-456)"},
                "days": {"type": "integer", "description": "Number of days to analyze (default: 7)"}
            },
            "required": ["task_id"]
        }
    )
    async def analyze_task_activity(self, task_id: str, days: int = 7) -> str:
        """Analyze recent task activity."""
        if not self.client:
            return "Error: YouTrack client not initialized"

        try:
            recent_activity = self.client.get_recent_activity(task_id, days)

            if not recent_activity:
                return f"No activity found for task {task_id} in the last {days} days"

            result = f"Task Activity Analysis: {task_id} (Last {days} days)\n"
            result += "=" * 50 + "\n\n"
            result += f"Total activities: {len(recent_activity)}\n\n"

            # Group activities by type
            activity_types = {}
            for activity in recent_activity:
                field = activity.get('field', {}).get('name', 'Unknown')
                activity_types[field] = activity_types.get(field, 0) + 1

            result += "Activity breakdown:\n"
            for field, count in sorted(activity_types.items()):
                result += f"• {field}: {count} changes\n"

            result += "\nRecent changes:\n"
            for activity in recent_activity[:5]:  # Show top 5
                author = activity.get('author', {}).get('name', 'Unknown')
                timestamp = datetime.fromtimestamp(activity['timestamp'] / 1000)
                field = activity.get('field', {}).get('name', 'Unknown')

                result += f"• {timestamp.strftime('%m/%d %H:%M')} - {author} changed {field}\n"

            return result

        except Exception as e:
            logger.error(f"Error analyzing task activity: {e}")
            return f"Error analyzing task activity: {str(e)}"

    @mcp_tool(
        name="analyze_epic_progress",
        description="Analyze epic progress over time",
        input_schema={
            "type": "object",
            "properties": {
                "epic_id": {"type": "string", "description": "Epic ID (e.g., PROJECT-123)"}
            },
            "required": ["epic_id"]
        }
    )
    async def analyze_epic_progress(self, epic_id: str) -> str:
        """Analyze epic progress trends."""
        if not self.client:
            return "Error: YouTrack client not initialized"

        try:
            # Get current progress
            progress = self.client.calculate_epic_progress(epic_id)

            # Get epic details
            epic = self.client.get_issue(epic_id)

            result = f"Epic Progress Analysis: {epic_id}\n"
            result += "=" * 50 + "\n\n"

            # Current status
            result += "Current Status:\n"
            result += f"• Completion: {progress['completion_percentage']:.1f}%\n"
            result += f"• Tasks: {progress['completed_tasks']}/{progress['total_tasks']}\n"

            if progress['story_points_total'] > 0:
                sp_percentage = (progress['story_points_completed'] / progress['story_points_total']) * 100
                result += f"• Story Points: {progress['story_points_completed']}/{progress['story_points_total']} ({sp_percentage:.1f}%)\n"

            # Risk assessment
            result += "\nRisk Assessment:\n"
            if progress['blocked_tasks'] > 0:
                result += f"⚠️  {progress['blocked_tasks']} blocked tasks need attention\n"

            if progress['in_progress_tasks'] > progress['completed_tasks']:
                result += "⚠️  More tasks in progress than completed - potential bottleneck\n"

            if progress['completion_percentage'] < 50 and progress['total_tasks'] > 10:
                result += "⚠️  Large epic with low completion rate\n"

            if progress['blocked_tasks'] == 0 and progress['completion_percentage'] > 80:
                result += "✅ Epic is on track with no blockers\n"

            return result

        except Exception as e:
            logger.error(f"Error analyzing epic progress: {e}")
            return f"Error analyzing epic progress: {str(e)}"

    @mcp_tool(
        name="generate_epic_report",
        description="Generate a comprehensive epic report",
        input_schema={
            "type": "object",
            "properties": {
                "epic_id": {"type": "string", "description": "Epic ID (e.g., PROJECT-123)"}
            },
            "required": ["epic_id"]
        }
    )
    async def generate_epic_report(self, epic_id: str) -> str:
        """Generate comprehensive epic report."""
        if not self.client:
            return "Error: YouTrack client not initialized"

        try:
            # Get epic details
            epic = self.client.get_issue(epic_id)
            progress = self.client.calculate_epic_progress(epic_id)
            tasks = self.client.get_epic_tasks(epic_id)

            report = f"EPIC COMPREHENSIVE REPORT\n"
            report += "=" * 60 + "\n\n"

            # Epic overview
            report += f"Epic: {epic['idReadable']} - {epic['summary']}\n"
            report += f"Project: {epic.get('project', {}).get('name', 'Unknown')}\n"

            if epic.get('created'):
                created_date = datetime.fromtimestamp(epic['created'] / 1000)
                report += f"Created: {created_date.strftime('%Y-%m-%d')}\n"

            report += "\n" + "PROGRESS OVERVIEW" + "\n"
            report += "-" * 20 + "\n"
            report += f"Completion: {progress['completion_percentage']:.1f}%\n"
            report += f"Total Tasks: {progress['total_tasks']}\n"
            report += f"✅ Completed: {progress['completed_tasks']}\n"
            report += f"🔄 In Progress: {progress['in_progress_tasks']}\n"
            report += f"📋 Open: {progress['open_tasks']}\n"
            report += f"🚫 Blocked: {progress['blocked_tasks']}\n"

            if progress['story_points_total'] > 0:
                sp_percentage = (progress['story_points_completed'] / progress['story_points_total']) * 100
                report += f"Story Points: {progress['story_points_completed']}/{progress['story_points_total']} ({sp_percentage:.1f}%)\n"

            # Task breakdown by status
            report += "\n" + "TASK BREAKDOWN" + "\n"
            report += "-" * 20 + "\n"

            status_groups = {}
            for task in tasks:
                state = self.client.get_issue_custom_field_value(task, 'State') or 'Unknown'
                if state not in status_groups:
                    status_groups[state] = []
                status_groups[state].append(task)

            for status, task_list in status_groups.items():
                report += f"\n{status} ({len(task_list)} tasks):\n"
                for task in task_list[:5]:  # Show max 5 per status
                    assignee = self.client.get_issue_custom_field_value(task, 'Assignee') or 'Unassigned'
                    report += f"  • {task['idReadable']}: {task['summary'][:50]}... ({assignee})\n"
                if len(task_list) > 5:
                    report += f"  ... and {len(task_list) - 5} more\n"

            # Recommendations
            report += "\n" + "RECOMMENDATIONS" + "\n"
            report += "-" * 20 + "\n"

            if progress['blocked_tasks'] > 0:
                report += f"🚨 Address {progress['blocked_tasks']} blocked tasks immediately\n"

            if progress['completion_percentage'] < 30:
                report += "📈 Consider breaking down large tasks for better progress tracking\n"

            if progress['in_progress_tasks'] > 5:
                report += "⚠️  High number of in-progress tasks - consider focus on completion\n"

            if progress['completion_percentage'] > 90:
                report += "🎉 Epic is nearly complete - prepare for closure\n"

            return report

        except Exception as e:
            logger.error(f"Error generating epic report: {e}")
            return f"Error generating epic report: {str(e)}"

    # Resources

    @mcp_resource(
        uri="youtrack://epic/{epic_id}",
        name="Epic Data",
        description="Raw YouTrack epic data",
        mime_type="application/json"
    )
    async def get_epic_resource(self, uri: str) -> str:
        """Get epic data as JSON resource."""
        if not self.client:
            return json.dumps({"error": "YouTrack client not initialized"})

        epic_id = uri.split('/')[-1]
        try:
            epic_data = self.client.get_issue(epic_id)
            progress_data = self.client.calculate_epic_progress(epic_id)

            resource = {
                "epic": epic_data,
                "progress": progress_data,
                "retrieved_at": datetime.now().isoformat()
            }

            return json.dumps(resource, indent=2)

        except Exception as e:
            return json.dumps({"error": str(e)})

    # Prompts

    @mcp_prompt(
        name="epic_analysis_prompt",
        description="Generate prompts for epic analysis",
        arguments=[
            {"name": "epic_id", "description": "Epic ID to analyze", "required": True}
        ]
    )
    async def epic_analysis_prompt(self, epic_id: str) -> str:
        """Generate epic analysis prompt."""
        return f"""Analyze the following epic and provide insights:

Epic ID: {epic_id}

Please provide:
1. Current progress assessment
2. Risk factors and blockers
3. Timeline predictions
4. Resource allocation recommendations
5. Next steps and priorities

Use the available MCP tools to gather epic data and task information."""

async def main():
    """Main entry point for YouTrack MCP server."""
    from mcp.server import Server
    from mcp.server.stdio import stdio_server
    from mcp.types import Tool, TextContent

    server_impl = YouTrackMCPServer()

    # Initialize YouTrack client
    try:
        await server_impl.setup()
    except Exception as e:
        logger.error(f"Failed to setup YouTrack server: {e}")
        # Continue anyway - tools will return errors

    mcp_server = Server(server_impl.name)

    # Build list of tools from server_impl
    tools_list = []
    for tool_name, handler in server_impl._tools.items():
        tool = Tool(
            name=getattr(handler, '_mcp_tool_name', tool_name),
            description=getattr(handler, '_mcp_tool_description', handler.__doc__ or ""),
            inputSchema=getattr(handler, '_mcp_tool_input_schema', {"type": "object", "properties": {}})
        )
        tools_list.append(tool)

    # Register list_tools handler
    @mcp_server.list_tools()
    async def list_tools():
        return tools_list

    # Register call_tool handler
    @mcp_server.call_tool()
    async def call_tool(name: str, arguments: dict):
        result = await server_impl.call_tool(name, arguments)
        content = result.get("content", [])
        return content

    # Register resource handlers
    @mcp_server.list_resources()
    async def list_resources():
        resources_list = await server_impl.list_resources()
        return resources_list.get("resources", [])

    @mcp_server.read_resource()
    async def read_resource(uri: str):
        result = await server_impl.read_resource(uri)
        return result.get("contents", [])

    # Register prompt handlers
    @mcp_server.list_prompts()
    async def list_prompts():
        prompts_list = await server_impl.list_prompts()
        return prompts_list.get("prompts", [])

    @mcp_server.get_prompt()
    async def get_prompt(name: str, arguments: dict):
        result = await server_impl.get_prompt(name, arguments)
        return result.get("messages", [])

    # Run the server
    async with stdio_server() as (read_stream, write_stream):
        await mcp_server.run(read_stream, write_stream, mcp_server.create_initialization_options())

if __name__ == "__main__":
    asyncio.run(main())