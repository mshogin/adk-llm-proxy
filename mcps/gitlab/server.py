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
from .gitlab_client import GitLabClient, GitLabConfig

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class GitLabMCPServer(MCPServerBase):
    """GitLab MCP Server for code analysis and repository correlation."""

    def __init__(self):
        super().__init__("gitlab-mcp-server", "1.0.0")
        self.client: Optional[GitLabClient] = None

    async def setup(self):
        """Initialize GitLab client."""
        try:
            self.client = GitLabClient.from_env()
            if not self.client.test_connection():
                logger.error("Failed to connect to GitLab")
                raise ConnectionError("Cannot connect to GitLab")

            user_info = self.client.get_user_info()
            logger.info(f"Connected to GitLab as: {user_info.get('name', 'Unknown')}")
        except Exception as e:
            logger.error(f"GitLab setup failed: {e}")
            raise

    # User Activity Tools

    @mcp_tool(
        name="get_my_recent_commits",
        description="Get MY recent commits and push activity across all GitLab projects for the authenticated user. Returns commit titles, branches, projects, and timestamps. Use this when user asks for 'my commits', 'my recent commits', 'my latest commits', 'my pushes', 'my activity', 'my recent activity', or 'commits by me'. This tool requires NO project_id - it automatically fetches commits for the authenticated user across all their projects.",
        input_schema={
            "type": "object",
            "properties": {
                "limit": {"type": "integer", "description": "Maximum number of commits to retrieve (default: 10)"}
            }
        }
    )
    async def get_my_recent_commits(self, limit: int = 10) -> str:
        """Get authenticated user's recent commits across all projects."""
        if not self.client:
            return "Error: GitLab client not initialized"

        try:
            # Get user events filtered by 'pushed' action
            events = self.client.get_user_events(limit=limit * 3, action='pushed')

            if not events:
                return "No recent push events found for your account"

            # Extract commits from push events
            commits_info = []
            for event in events:
                if event.get('action_name') == 'pushed to' and 'push_data' in event:
                    push_data = event['push_data']
                    project_name = event.get('project', {}).get('path_with_namespace', 'Unknown')

                    commit_info = {
                        'project': project_name,
                        'branch': push_data.get('ref', 'unknown'),
                        'commit_title': push_data.get('commit_title', 'No title'),
                        'commit_count': push_data.get('commit_count', 1),
                        'created_at': event.get('created_at', ''),
                        'action': event.get('action_name', '')
                    }
                    commits_info.append(commit_info)

                    if len(commits_info) >= limit:
                        break

            if not commits_info:
                return "No commit information found in recent events"

            result = f"Your Recent Commits ({len(commits_info)} found):\n"
            result += "=" * 70 + "\n\n"

            for i, commit in enumerate(commits_info, 1):
                created_date = datetime.fromisoformat(commit['created_at'].replace('Z', '+00:00'))
                result += f"{i}. {commit['project']} ({commit['branch']})\n"
                result += f"   {commit['commit_title']}\n"
                result += f"   {created_date.strftime('%Y-%m-%d %H:%M:%S')}\n"
                if commit['commit_count'] > 1:
                    result += f"   ({commit['commit_count']} commits in this push)\n"
                result += "\n"

            return result

        except Exception as e:
            logger.error(f"Error getting user commits: {e}")
            return f"Error retrieving your recent commits: {str(e)}"

    # Repository Analysis Tools

    @mcp_tool(
        name="find_project",
        description="Search for GitLab projects by name or description",
        input_schema={
            "type": "object",
            "properties": {
                "search_term": {"type": "string", "description": "Project name or search term"},
                "limit": {"type": "integer", "description": "Maximum number of results (default: 10)"}
            },
            "required": ["search_term"]
        }
    )
    async def find_project(self, search_term: str, limit: int = 10) -> str:
        """Find GitLab projects by search term."""
        if not self.client:
            return "Error: GitLab client not initialized"

        try:
            projects = self.client.search_projects(search_term, limit)

            if not projects:
                return f"No projects found matching '{search_term}'"

            result = f"Found {len(projects)} project(s) matching '{search_term}':\n\n"
            for project in projects:
                result += f"• {project['name']} (ID: {project['id']})\n"
                result += f"  Path: {project['path_with_namespace']}\n"
                result += f"  Description: {project.get('description', 'No description')[:100]}...\n"
                result += f"  Last Activity: {project.get('last_activity_at', 'Unknown')}\n"
                result += f"  Stars: {project.get('star_count', 0)} | Forks: {project.get('forks_count', 0)}\n\n"

            return result

        except Exception as e:
            logger.error(f"Error finding projects: {e}")
            return f"Error searching for projects: {str(e)}"

    @mcp_tool(
        name="get_recent_commits",
        description="Get recent commits for a project",
        input_schema={
            "type": "object",
            "properties": {
                "project_id": {"type": "string", "description": "Project ID or path"},
                "branch": {"type": "string", "description": "Branch name (optional)"},
                "days": {"type": "integer", "description": "Number of days to look back (default: 7)"},
                "limit": {"type": "integer", "description": "Maximum number of commits (default: 20)"}
            },
            "required": ["project_id"]
        }
    )
    async def get_recent_commits(self, project_id: str, branch: str = None,
                               days: int = 7, limit: int = 20) -> str:
        """Get recent commits for a project."""
        if not self.client:
            return "Error: GitLab client not initialized"

        try:
            since_date = datetime.now() - timedelta(days=days)
            # Make since_date timezone-aware
            since_date = since_date.replace(tzinfo=datetime.now().astimezone().tzinfo)
            commits = self.client.get_project_commits(
                project_id, branch=branch, since=since_date, per_page=limit
            )

            if not commits:
                return f"No recent commits found for project {project_id}"

            result = f"Recent Commits for Project {project_id}"
            if branch:
                result += f" (Branch: {branch})"
            result += f" (Last {days} days):\n"
            result += "=" * 60 + "\n\n"

            for commit in commits:
                commit_date = datetime.fromisoformat(commit['created_at'].replace('Z', '+00:00'))
                result += f"• {commit['short_id']}: {commit['title']}\n"
                result += f"  Author: {commit.get('author_name', 'Unknown')}\n"
                result += f"  Date: {commit_date.strftime('%Y-%m-%d %H:%M')}\n"

                if commit.get('message') and len(commit['message']) > len(commit['title']):
                    body = commit['message'][len(commit['title']):].strip()
                    if body:
                        result += f"  Message: {body[:150]}{'...' if len(body) > 150 else ''}\n"

                result += "\n"

            return result

        except Exception as e:
            logger.error(f"Error getting recent commits: {e}")
            return f"Error retrieving commits: {str(e)}"

    @mcp_tool(
        name="analyze_commit_messages",
        description="Analyze commit messages for patterns and insights",
        input_schema={
            "type": "object",
            "properties": {
                "project_id": {"type": "string", "description": "Project ID or path"},
                "days": {"type": "integer", "description": "Number of days to analyze (default: 30)"},
                "limit": {"type": "integer", "description": "Maximum number of commits to analyze (default: 100)"}
            },
            "required": ["project_id"]
        }
    )
    async def analyze_commit_messages(self, project_id: str, days: int = 30, limit: int = 100) -> str:
        """Analyze commit messages for patterns."""
        if not self.client:
            return "Error: GitLab client not initialized"

        try:
            since_date = datetime.now() - timedelta(days=days)
            # Make since_date timezone-aware
            since_date = since_date.replace(tzinfo=datetime.now().astimezone().tzinfo)
            commits = self.client.get_project_commits(
                project_id, since=since_date, per_page=limit
            )

            if not commits:
                return f"No commits found for analysis in project {project_id}"

            analysis = self.client.analyze_commit_messages(commits)

            result = f"Commit Message Analysis for Project {project_id} (Last {days} days)\n"
            result += "=" * 70 + "\n\n"

            result += f"Total Commits Analyzed: {analysis['total_commits']}\n"
            result += f"Average Message Length: {analysis['avg_message_length']:.1f} characters\n"
            result += f"Commits with Task References: {analysis['commits_with_tasks']}\n\n"

            # Task references
            if analysis['task_references']:
                result += f"Task References Found ({len(analysis['task_references'])}):\n"
                for task_id in analysis['task_references'][:10]:
                    result += f"  • {task_id}\n"
                if len(analysis['task_references']) > 10:
                    result += f"  ... and {len(analysis['task_references']) - 10} more\n"
                result += "\n"

            # Commit types
            if analysis['commit_types']:
                result += "Commit Types:\n"
                for commit_type, count in list(analysis['commit_types'].items())[:10]:
                    percentage = (count / analysis['total_commits']) * 100
                    result += f"  • {commit_type}: {count} ({percentage:.1f}%)\n"
                result += "\n"

            # Top authors
            if analysis['authors']:
                result += "Most Active Authors:\n"
                for author, count in list(analysis['authors'].items())[:5]:
                    percentage = (count / analysis['total_commits']) * 100
                    result += f"  • {author}: {count} commits ({percentage:.1f}%)\n"
                result += "\n"

            # Common keywords
            if analysis['common_keywords']:
                result += "Most Common Keywords:\n"
                for keyword, count in list(analysis['common_keywords'].items())[:10]:
                    result += f"  • {keyword}: {count} times\n"

            return result

        except Exception as e:
            logger.error(f"Error analyzing commit messages: {e}")
            return f"Error analyzing commit messages: {str(e)}"

    @mcp_tool(
        name="get_merge_requests",
        description="Get merge requests for a project",
        input_schema={
            "type": "object",
            "properties": {
                "project_id": {"type": "string", "description": "Project ID or path"},
                "state": {"type": "string", "description": "MR state (opened, closed, merged, all) (default: opened)"},
                "limit": {"type": "integer", "description": "Maximum number of MRs (default: 20)"}
            },
            "required": ["project_id"]
        }
    )
    async def get_merge_requests(self, project_id: str, state: str = "opened", limit: int = 20) -> str:
        """Get merge requests for a project."""
        if not self.client:
            return "Error: GitLab client not initialized"

        try:
            merge_requests = self.client.get_merge_requests(project_id, state=state, per_page=limit)

            if not merge_requests:
                return f"No {state} merge requests found for project {project_id}"

            result = f"{state.title()} Merge Requests for Project {project_id} ({len(merge_requests)} found):\n"
            result += "=" * 70 + "\n\n"

            for mr in merge_requests:
                created_date = datetime.fromisoformat(mr['created_at'].replace('Z', '+00:00'))
                result += f"• !{mr['iid']}: {mr['title']}\n"
                result += f"  Author: {mr.get('author', {}).get('name', 'Unknown')}\n"
                result += f"  Source: {mr.get('source_branch', 'Unknown')} → {mr.get('target_branch', 'Unknown')}\n"
                result += f"  State: {mr.get('state', 'Unknown').title()}\n"
                result += f"  Created: {created_date.strftime('%Y-%m-%d %H:%M')}\n"

                if mr.get('description'):
                    description = mr['description'][:150].replace('\n', ' ')
                    result += f"  Description: {description}{'...' if len(mr['description']) > 150 else ''}\n"

                # Additional info for merged/closed MRs
                if mr.get('merged_at'):
                    merged_date = datetime.fromisoformat(mr['merged_at'].replace('Z', '+00:00'))
                    result += f"  Merged: {merged_date.strftime('%Y-%m-%d %H:%M')}\n"

                result += "\n"

            return result

        except Exception as e:
            logger.error(f"Error getting merge requests: {e}")
            return f"Error retrieving merge requests: {str(e)}"

    @mcp_tool(
        name="analyze_branch_activity",
        description="Analyze branch activity and patterns",
        input_schema={
            "type": "object",
            "properties": {
                "project_id": {"type": "string", "description": "Project ID or path"}
            },
            "required": ["project_id"]
        }
    )
    async def analyze_branch_activity(self, project_id: str) -> str:
        """Analyze branch activity patterns."""
        if not self.client:
            return "Error: GitLab client not initialized"

        try:
            branches = self.client.get_branches(project_id)

            if not branches:
                return f"No branches found for project {project_id}"

            result = f"Branch Activity Analysis for Project {project_id}\n"
            result += "=" * 60 + "\n\n"

            # Sort branches by last commit date
            active_branches = []
            stale_branches = []
            cutoff_date = datetime.now() - timedelta(days=30)
            # Make cutoff_date timezone-aware
            cutoff_date = cutoff_date.replace(tzinfo=datetime.now().astimezone().tzinfo)

            for branch in branches:
                last_commit = branch.get('commit', {})
                if last_commit.get('created_at'):
                    commit_date = datetime.fromisoformat(last_commit['created_at'].replace('Z', '+00:00'))
                    branch['last_activity'] = commit_date

                    if commit_date > cutoff_date:
                        active_branches.append(branch)
                    else:
                        stale_branches.append(branch)

            result += f"Total Branches: {len(branches)}\n"
            result += f"Active Branches (last 30 days): {len(active_branches)}\n"
            result += f"Stale Branches (older than 30 days): {len(stale_branches)}\n\n"

            # Show active branches
            if active_branches:
                result += "Recently Active Branches:\n"
                # Create timezone-aware datetime.min for default
                min_tz_aware = datetime.min.replace(tzinfo=datetime.now().astimezone().tzinfo)
                active_branches.sort(key=lambda x: x.get('last_activity', min_tz_aware), reverse=True)
                for branch in active_branches[:10]:
                    last_commit = branch.get('commit', {})
                    activity_date = branch.get('last_activity')
                    if activity_date:
                        result += f"• {branch['name']}\n"
                        result += f"  Last commit: {activity_date.strftime('%Y-%m-%d %H:%M')}\n"
                        result += f"  Author: {last_commit.get('author_name', 'Unknown')}\n"
                        result += f"  Message: {last_commit.get('title', 'No message')[:80]}...\n\n"

            # Show stale branches (potential cleanup candidates)
            if stale_branches:
                result += f"\nStale Branches (candidates for cleanup):\n"
                stale_branches.sort(key=lambda x: x.get('last_activity', min_tz_aware))
                for branch in stale_branches[:5]:
                    activity_date = branch.get('last_activity')
                    if activity_date:
                        # Make current datetime timezone-aware for comparison
                        now_tz_aware = datetime.now().replace(tzinfo=datetime.now().astimezone().tzinfo)
                        days_old = (now_tz_aware - activity_date).days
                        result += f"• {branch['name']} (last activity: {days_old} days ago)\n"

            return result

        except Exception as e:
            logger.error(f"Error analyzing branch activity: {e}")
            return f"Error analyzing branch activity: {str(e)}"

    # Code-to-Task Correlation Tools

    @mcp_tool(
        name="link_commits_to_tasks",
        description="Extract YouTrack task IDs from commit messages and link them",
        input_schema={
            "type": "object",
            "properties": {
                "project_id": {"type": "string", "description": "Project ID or path"},
                "days": {"type": "integer", "description": "Number of days to analyze (default: 30)"},
                "limit": {"type": "integer", "description": "Maximum number of commits to analyze (default: 100)"}
            },
            "required": ["project_id"]
        }
    )
    async def link_commits_to_tasks(self, project_id: str, days: int = 30, limit: int = 100) -> str:
        """Link commits to YouTrack tasks by extracting task IDs."""
        if not self.client:
            return "Error: GitLab client not initialized"

        try:
            since_date = datetime.now() - timedelta(days=days)
            # Make since_date timezone-aware
            since_date = since_date.replace(tzinfo=datetime.now().astimezone().tzinfo)
            commits = self.client.get_project_commits(
                project_id, since=since_date, per_page=limit
            )

            if not commits:
                return f"No commits found for analysis in project {project_id}"

            task_links = {}
            commits_without_tasks = 0

            result = f"Commit-to-Task Correlation Analysis for Project {project_id}\n"
            result += "=" * 70 + "\n\n"

            for commit in commits:
                message = commit.get('message', '')
                task_ids = self.client.extract_task_ids_from_text(message)

                if task_ids:
                    for task_id in task_ids:
                        if task_id not in task_links:
                            task_links[task_id] = []

                        task_links[task_id].append({
                            'commit_id': commit['short_id'],
                            'title': commit['title'],
                            'author': commit.get('author_name', 'Unknown'),
                            'date': commit['created_at']
                        })
                else:
                    commits_without_tasks += 1

            result += f"Total Commits Analyzed: {len(commits)}\n"
            result += f"Commits with Task References: {len(commits) - commits_without_tasks}\n"
            result += f"Commits without Task References: {commits_without_tasks}\n"
            result += f"Unique Tasks Referenced: {len(task_links)}\n\n"

            if task_links:
                result += "Task-to-Commit Links:\n"
                for task_id, linked_commits in sorted(task_links.items()):
                    result += f"\n{task_id} ({len(linked_commits)} commits):\n"
                    for commit_info in linked_commits[:5]:  # Show up to 5 commits per task
                        commit_date = datetime.fromisoformat(commit_info['date'].replace('Z', '+00:00'))
                        result += f"  • {commit_info['commit_id']}: {commit_info['title'][:60]}...\n"
                        result += f"    {commit_info['author']} - {commit_date.strftime('%Y-%m-%d')}\n"

                    if len(linked_commits) > 5:
                        result += f"    ... and {len(linked_commits) - 5} more commits\n"

            else:
                result += "No task references found in commit messages.\n"
                result += "Consider using task IDs in commit messages for better traceability.\n"

            # Recommendations
            if commits_without_tasks > len(commits) * 0.5:
                result += "\n⚠️  Recommendations:\n"
                result += "• More than 50% of commits lack task references\n"
                result += "• Consider establishing commit message conventions\n"
                result += "• Include task IDs (e.g., PROJECT-123) in commit messages\n"

            return result

        except Exception as e:
            logger.error(f"Error linking commits to tasks: {e}")
            return f"Error linking commits to tasks: {str(e)}"

    @mcp_tool(
        name="analyze_epic_code_changes",
        description="Analyze code changes related to a specific epic",
        input_schema={
            "type": "object",
            "properties": {
                "project_id": {"type": "string", "description": "Project ID or path"},
                "epic_id": {"type": "string", "description": "Epic ID (e.g., PROJ-100)"},
                "days": {"type": "integer", "description": "Number of days to search (default: 90)"}
            },
            "required": ["project_id", "epic_id"]
        }
    )
    async def analyze_epic_code_changes(self, project_id: str, epic_id: str, days: int = 90) -> str:
        """Analyze code changes related to a specific epic."""
        if not self.client:
            return "Error: GitLab client not initialized"

        try:
            since_date = datetime.now() - timedelta(days=days)
            # Make since_date timezone-aware
            since_date = since_date.replace(tzinfo=datetime.now().astimezone().tzinfo)
            commits = self.client.get_project_commits(
                project_id, since=since_date, per_page=200
            )

            if not commits:
                return f"No commits found for analysis in project {project_id}"

            epic_commits = []
            related_tasks = set()

            # Find commits related to the epic
            for commit in commits:
                message = commit.get('message', '')
                task_ids = self.client.extract_task_ids_from_text(message)

                # Check if this commit references the epic directly or related tasks
                epic_referenced = epic_id.upper() in message.upper()

                if epic_referenced or any(epic_id.split('-')[0] in task_id for task_id in task_ids):
                    epic_commits.append(commit)
                    related_tasks.update(task_ids)

            if not epic_commits:
                return f"No commits found related to epic {epic_id} in project {project_id}"

            result = f"Code Changes Analysis for Epic {epic_id}\n"
            result += f"Project: {project_id} | Period: Last {days} days\n"
            result += "=" * 70 + "\n\n"

            result += f"Related Commits Found: {len(epic_commits)}\n"
            result += f"Related Tasks: {len(related_tasks)}\n\n"

            # Get code metrics for epic commits
            metrics = self.client.calculate_code_metrics(project_id, epic_commits)

            result += "Code Metrics Summary:\n"
            result += f"• Lines Added: {metrics['total_additions']:,}\n"
            result += f"• Lines Deleted: {metrics['total_deletions']:,}\n"
            result += f"• Files Changed: {metrics['files_changed']:,}\n"
            result += f"• Net Change: {metrics['total_additions'] - metrics['total_deletions']:+,} lines\n\n"

            # File types analysis
            if metrics['file_types']:
                result += "File Types Modified:\n"
                for file_type, count in list(metrics['file_types'].items())[:10]:
                    result += f"• {file_type}: {count} files\n"
                result += "\n"

            # Most active files
            if metrics['most_active_files']:
                result += "Most Modified Files:\n"
                for file_path, changes in list(metrics['most_active_files'].items())[:10]:
                    result += f"• {file_path}: {changes} changes\n"
                result += "\n"

            # Recent commits
            result += "Recent Related Commits:\n"
            for commit in epic_commits[:10]:
                commit_date = datetime.fromisoformat(commit['created_at'].replace('Z', '+00:00'))
                result += f"• {commit['short_id']}: {commit['title']}\n"
                result += f"  {commit.get('author_name', 'Unknown')} - {commit_date.strftime('%Y-%m-%d')}\n"

            # Related tasks
            if related_tasks:
                result += f"\nRelated Tasks Referenced:\n"
                for task_id in sorted(related_tasks):
                    result += f"• {task_id}\n"

            return result

        except Exception as e:
            logger.error(f"Error analyzing epic code changes: {e}")
            return f"Error analyzing epic code changes: {str(e)}"

    @mcp_tool(
        name="get_code_metrics",
        description="Get detailed code metrics for a project",
        input_schema={
            "type": "object",
            "properties": {
                "project_id": {"type": "string", "description": "Project ID or path"},
                "days": {"type": "integer", "description": "Number of days to analyze (default: 30)"},
                "limit": {"type": "integer", "description": "Maximum number of commits to analyze (default: 100)"}
            },
            "required": ["project_id"]
        }
    )
    async def get_code_metrics(self, project_id: str, days: int = 30, limit: int = 100) -> str:
        """Get comprehensive code metrics for a project."""
        if not self.client:
            return "Error: GitLab client not initialized"

        try:
            since_date = datetime.now() - timedelta(days=days)
            # Make since_date timezone-aware
            since_date = since_date.replace(tzinfo=datetime.now().astimezone().tzinfo)
            commits = self.client.get_project_commits(
                project_id, since=since_date, per_page=limit
            )

            if not commits:
                return f"No commits found for metrics calculation in project {project_id}"

            metrics = self.client.calculate_code_metrics(project_id, commits)

            result = f"Code Metrics Report for Project {project_id}\n"
            result += f"Period: Last {days} days | Commits Analyzed: {metrics['total_commits']}\n"
            result += "=" * 70 + "\n\n"

            # Overall metrics
            result += "Overall Changes:\n"
            result += f"• Total Additions: {metrics['total_additions']:,} lines\n"
            result += f"• Total Deletions: {metrics['total_deletions']:,} lines\n"
            result += f"• Net Change: {metrics['total_additions'] - metrics['total_deletions']:+,} lines\n"
            result += f"• Files Modified: {metrics['files_changed']:,}\n"
            result += f"• Average per Commit: {(metrics['total_additions'] + metrics['total_deletions']) / max(metrics['total_commits'], 1):.1f} lines\n\n"

            # File types
            if metrics['file_types']:
                result += "File Types Modified:\n"
                total_file_changes = sum(metrics['file_types'].values())
                for file_type, count in list(metrics['file_types'].items())[:15]:
                    percentage = (count / total_file_changes) * 100
                    result += f"• {file_type}: {count} files ({percentage:.1f}%)\n"
                result += "\n"

            # Largest commits
            if metrics['largest_commits']:
                result += "Largest Commits:\n"
                for commit_info in metrics['largest_commits'][:5]:
                    result += f"• {commit_info['id']}: +{commit_info['additions']} -{commit_info['deletions']} lines\n"
                    result += f"  {commit_info['message'][:60]}... ({commit_info['author']})\n"
                result += "\n"

            # Most active files
            if metrics['most_active_files']:
                result += "Most Active Files (Change Frequency):\n"
                for file_path, changes in list(metrics['most_active_files'].items())[:15]:
                    result += f"• {file_path}: {changes} commits\n"

            return result

        except Exception as e:
            logger.error(f"Error getting code metrics: {e}")
            return f"Error calculating code metrics: {str(e)}"

    @mcp_tool(
        name="track_developer_activity",
        description="Track and analyze developer activity patterns",
        input_schema={
            "type": "object",
            "properties": {
                "project_id": {"type": "string", "description": "Project ID or path"},
                "days": {"type": "integer", "description": "Number of days to analyze (default: 30)"}
            },
            "required": ["project_id"]
        }
    )
    async def track_developer_activity(self, project_id: str, days: int = 30) -> str:
        """Track developer activity patterns."""
        if not self.client:
            return "Error: GitLab client not initialized"

        try:
            since_date = datetime.now() - timedelta(days=days)
            # Make since_date timezone-aware
            since_date = since_date.replace(tzinfo=datetime.now().astimezone().tzinfo)
            commits = self.client.get_project_commits(
                project_id, since=since_date, per_page=200
            )

            if not commits:
                return f"No commits found for developer activity analysis in project {project_id}"

            activity = self.client.analyze_developer_activity(commits, days)

            result = f"Developer Activity Report for Project {project_id}\n"
            result += f"Period: Last {activity['period_days']} days\n"
            result += "=" * 70 + "\n\n"

            result += f"Total Commits: {activity['total_commits']}\n"
            result += f"Active Developers: {len(activity['active_developers'])}\n\n"

            # Developer rankings
            if activity['active_developers']:
                result += "Developer Activity Rankings:\n"
                sorted_devs = sorted(activity['active_developers'].items(),
                                   key=lambda x: x[1]['commits'], reverse=True)

                for i, (developer, stats) in enumerate(sorted_devs[:10], 1):
                    percentage = (stats['commits'] / activity['total_commits']) * 100
                    result += f"{i:2d}. {developer}: {stats['commits']} commits ({percentage:.1f}%)\n"

                    first_commit = datetime.fromisoformat(stats['first_commit'])
                    last_commit = datetime.fromisoformat(stats['last_commit'])
                    result += f"    Active from {first_commit.strftime('%Y-%m-%d')} to {last_commit.strftime('%Y-%m-%d')}\n"

                result += "\n"

            # Peak activity periods
            if activity['most_productive_day']:
                day, commits_count = activity['most_productive_day']
                result += f"Most Productive Day: {day} ({commits_count} commits)\n"

            if activity['most_productive_hour']:
                hour, commits_count = activity['most_productive_hour']
                result += f"Most Productive Hour: {hour}:00 ({commits_count} commits)\n"

            # Daily activity pattern (show last 7 days)
            if activity['daily_activity']:
                result += "\nRecent Daily Activity:\n"
                sorted_days = sorted(activity['daily_activity'].items(), reverse=True)
                for day, count in sorted_days[:7]:
                    result += f"• {day}: {count} commits\n"

            return result

        except Exception as e:
            logger.error(f"Error tracking developer activity: {e}")
            return f"Error tracking developer activity: {str(e)}"

    # Advanced Code Analysis Tools

    @mcp_tool(
        name="analyze_code_complexity",
        description="Analyze code complexity trends and identify hotspots",
        input_schema={
            "type": "object",
            "properties": {
                "project_id": {"type": "string", "description": "Project ID or path"},
                "days": {"type": "integer", "description": "Number of days to analyze (default: 60)"}
            },
            "required": ["project_id"]
        }
    )
    async def analyze_code_complexity(self, project_id: str, days: int = 60) -> str:
        """Analyze code complexity trends."""
        if not self.client:
            return "Error: GitLab client not initialized"

        try:
            since_date = datetime.now() - timedelta(days=days)
            # Make since_date timezone-aware
            since_date = since_date.replace(tzinfo=datetime.now().astimezone().tzinfo)
            commits = self.client.get_project_commits(
                project_id, since=since_date, per_page=200
            )

            if not commits:
                return f"No commits found for complexity analysis in project {project_id}"

            complexity = self.client.assess_code_complexity(project_id, commits)

            result = f"Code Complexity Analysis for Project {project_id}\n"
            result += f"Period: Last {days} days | Files Analyzed: {complexity['total_files_analyzed']}\n"
            result += "=" * 70 + "\n\n"

            # Complexity indicators
            indicators = complexity['complexity_indicators']
            result += "Complexity Indicators:\n"
            result += f"• Large Commits (>500 lines): {indicators['large_commits']}\n"
            result += f"• Multi-file Commits (>10 files): {indicators['many_files_changed']}\n"
            result += f"• Refactoring Commits: {indicators['refactoring_commits']}\n"
            result += f"• Bug Fix Commits: {indicators['bug_fix_commits']}\n\n"

            # Frequently changed files (complexity hotspots)
            if indicators['frequent_changes']:
                result += "Complexity Hotspots (Frequently Changed Files):\n"
                for file_path, change_count in list(indicators['frequent_changes'].items())[:15]:
                    result += f"• {file_path}: {change_count} changes\n"
                result += "\n"

            # Risk assessment
            if complexity['risk_indicators']:
                result += "🚨 Risk Indicators:\n"
                for risk in complexity['risk_indicators']:
                    result += f"• {risk}\n"
                result += "\n"

            # Recommendations
            if complexity['recommendations']:
                result += "💡 Recommendations:\n"
                for recommendation in complexity['recommendations']:
                    result += f"• {recommendation}\n"
                result += "\n"

            # Additional insights
            total_commits = len(commits)
            if total_commits > 0:
                refactor_ratio = indicators['refactoring_commits'] / total_commits
                bug_ratio = indicators['bug_fix_commits'] / total_commits

                result += "Health Metrics:\n"
                result += f"• Refactoring Ratio: {refactor_ratio:.1%}\n"
                result += f"• Bug Fix Ratio: {bug_ratio:.1%}\n"

                if refactor_ratio < 0.1:
                    result += "⚠️  Low refactoring activity - consider proactive code maintenance\n"
                if bug_ratio > 0.3:
                    result += "⚠️  High bug fix ratio - may indicate quality issues\n"

            return result

        except Exception as e:
            logger.error(f"Error analyzing code complexity: {e}")
            return f"Error analyzing code complexity: {str(e)}"

    # Resources and Prompts

    @mcp_resource(
        uri="gitlab://project/{project_id}",
        name="Project Data",
        description="Raw GitLab project data",
        mime_type="application/json"
    )
    async def get_project_resource(self, uri: str) -> str:
        """Get project data as JSON resource."""
        if not self.client:
            return json.dumps({"error": "GitLab client not initialized"})

        project_id = uri.split('/')[-1]
        try:
            project_data = self.client.get_project(project_id)
            recent_commits = self.client.get_project_commits(project_id, per_page=10)

            resource = {
                "project": project_data,
                "recent_commits": recent_commits,
                "retrieved_at": datetime.now().isoformat()
            }

            return json.dumps(resource, indent=2)

        except Exception as e:
            return json.dumps({"error": str(e)})

    @mcp_prompt(
        name="code_review_prompt",
        description="Generate prompts for code review analysis",
        arguments=[
            {"name": "project_id", "description": "Project ID to analyze", "required": True},
            {"name": "mr_id", "description": "Merge request ID", "required": False}
        ]
    )
    async def code_review_prompt(self, project_id: str, mr_id: str = None) -> str:
        """Generate code review analysis prompt."""
        prompt = f"""Analyze the following GitLab project for code review insights:

Project ID: {project_id}"""

        if mr_id:
            prompt += f"\nMerge Request: !{mr_id}"

        prompt += """

Please provide:
1. Code quality assessment
2. Potential issues or improvements
3. Architecture and design patterns analysis
4. Testing coverage recommendations
5. Performance considerations
6. Security review points

Use the available MCP tools to gather project data, commit history, and merge request information."""

        return prompt

async def main():
    """Main entry point for GitLab MCP server."""
    from mcp.server import Server
    from mcp.server.stdio import stdio_server
    from mcp.types import Tool, TextContent

    server_impl = GitLabMCPServer()

    # Initialize GitLab client
    try:
        await server_impl.setup()
    except Exception as e:
        logger.error(f"Failed to setup GitLab server: {e}")
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