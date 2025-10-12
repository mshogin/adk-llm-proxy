import os
import logging
import requests
from typing import Dict, List, Any, Optional
from datetime import datetime, timedelta
from dataclasses import dataclass
from pydantic import BaseModel, HttpUrl

logger = logging.getLogger(__name__)

@dataclass
class YouTrackConfig:
    """YouTrack configuration."""
    url: str
    token: str
    timeout: int = 30
    verify_ssl: bool = True

class YouTrackIssue(BaseModel):
    """YouTrack issue model."""
    id: str
    idReadable: str
    summary: str
    description: Optional[str] = None
    project: Optional[Dict[str, Any]] = None
    customFields: List[Dict[str, Any]] = []
    comments: List[Dict[str, Any]] = []
    links: List[Dict[str, Any]] = []
    tags: List[Dict[str, Any]] = []
    created: Optional[int] = None
    updated: Optional[int] = None
    resolved: Optional[int] = None
    reporter: Optional[Dict[str, Any]] = None
    updater: Optional[Dict[str, Any]] = None

class YouTrackClient:
    """YouTrack REST API client with authentication and error handling."""

    def __init__(self, config: YouTrackConfig):
        """Initialize YouTrack client.

        Args:
            config: YouTrack configuration
        """
        self.config = config
        self.base_url = config.url.rstrip('/')
        self.headers = {
            'Authorization': f'Bearer {config.token}',
            'Accept': 'application/json',
            'Content-Type': 'application/json'
        }
        self.session = requests.Session()
        self.session.headers.update(self.headers)

    def _make_request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        """Make HTTP request to YouTrack API.

        Args:
            method: HTTP method
            endpoint: API endpoint
            **kwargs: Additional request arguments

        Returns:
            Response object

        Raises:
            requests.RequestException: If request fails
        """
        url = f"{self.base_url}/api{endpoint}"

        # Set default timeout and SSL verification
        kwargs.setdefault('timeout', self.config.timeout)
        kwargs.setdefault('verify', self.config.verify_ssl)

        try:
            response = self.session.request(method, url, **kwargs)
            response.raise_for_status()
            return response
        except requests.RequestException as e:
            logger.error(f"YouTrack API request failed: {method} {url} - {e}")
            raise

    def test_connection(self) -> bool:
        """Test connection to YouTrack.

        Returns:
            True if connection successful, False otherwise
        """
        try:
            response = self._make_request('GET', '/users/me')
            logger.info("YouTrack connection test successful")
            return True
        except Exception as e:
            logger.error(f"YouTrack connection test failed: {e}")
            return False

    def get_user_info(self) -> Dict[str, Any]:
        """Get current user information.

        Returns:
            User information dictionary
        """
        response = self._make_request('GET', '/users/me')
        return response.json()

    def search_issues(self, query: str, fields: Optional[str] = None, top: int = 100) -> List[Dict[str, Any]]:
        """Search for issues using YouTrack query language.

        Args:
            query: YouTrack search query
            fields: Comma-separated list of fields to return
            top: Maximum number of results

        Returns:
            List of issues
        """
        if not fields:
            fields = "id,idReadable,summary,description,project(name,shortName),customFields(name,value),created,updated,resolved,reporter(name),updater(name)"

        params = {
            'query': query,
            'fields': fields,
            '$top': top
        }

        response = self._make_request('GET', '/issues', params=params)
        return response.json()

    def get_issue(self, issue_id: str, fields: Optional[str] = None) -> Dict[str, Any]:
        """Get specific issue by ID.

        Args:
            issue_id: Issue ID (can be readable ID like PROJECT-123)
            fields: Comma-separated list of fields to return

        Returns:
            Issue data
        """
        if not fields:
            fields = "id,idReadable,summary,description,project(name,shortName),customFields(name,value),created,updated,resolved,reporter(name),updater(name),tags(name),links(direction,linkType(name),issues(id,idReadable,summary))"

        params = {'fields': fields}
        response = self._make_request('GET', f'/issues/{issue_id}', params=params)
        return response.json()

    def get_issue_comments(self, issue_id: str, top: int = 100) -> List[Dict[str, Any]]:
        """Get comments for an issue.

        Args:
            issue_id: Issue ID
            top: Maximum number of comments

        Returns:
            List of comments
        """
        params = {
            'fields': 'id,text,created,updated,author(name),deleted',
            '$top': top,
            '$orderBy': 'created desc'
        }

        response = self._make_request('GET', f'/issues/{issue_id}/comments', params=params)
        return response.json()

    def get_issue_history(self, issue_id: str, top: int = 100) -> List[Dict[str, Any]]:
        """Get change history for an issue.

        Args:
            issue_id: Issue ID
            top: Maximum number of history entries

        Returns:
            List of history entries
        """
        params = {
            'fields': 'id,timestamp,author(name),added(name),removed(name),field(name)',
            '$top': top,
            '$orderBy': 'timestamp desc'
        }

        response = self._make_request('GET', f'/issues/{issue_id}/activities', params=params)
        return response.json()

    def find_epics(self, search_term: str, project: Optional[str] = None) -> List[Dict[str, Any]]:
        """Find epics by name, ID, or tag.

        Args:
            search_term: Search term
            project: Optional project filter

        Returns:
            List of matching epics
        """
        # Build search query for epics
        query_parts = []

        # Search in summary and description
        query_parts.append(f'(summary: {search_term} or description: {search_term})')

        # Search by readable ID
        if search_term.upper().replace('-', '').replace('_', '').isalnum():
            query_parts.append(f'issue id: {search_term}')

        # Search by tags
        query_parts.append(f'tag: {search_term}')

        # Filter by epic type (assuming Epic is a custom field or issue type)
        query_parts.append('Type: Epic')

        if project:
            query_parts.append(f'project: {project}')

        # Combine with OR logic for search terms, AND for filters
        search_query = f"({' or '.join(query_parts[:3])}) and Type: Epic"
        if project:
            search_query += f" and project: {project}"

        return self.search_issues(search_query)

    def get_epic_tasks(self, epic_id: str) -> List[Dict[str, Any]]:
        """Get all tasks linked to an epic.

        Args:
            epic_id: Epic ID

        Returns:
            List of tasks in the epic
        """
        # Search for issues linked to the epic
        query = f'links: {epic_id}'
        return self.search_issues(query)

    def get_projects(self) -> List[Dict[str, Any]]:
        """Get list of accessible projects.

        Returns:
            List of projects
        """
        params = {
            'fields': 'id,name,shortName,description'
        }

        response = self._make_request('GET', '/admin/projects', params=params)
        return response.json()

    def get_custom_fields(self, project_id: Optional[str] = None) -> List[Dict[str, Any]]:
        """Get custom fields.

        Args:
            project_id: Optional project ID filter

        Returns:
            List of custom fields
        """
        endpoint = '/admin/customFieldSettings/customFields'
        params = {
            'fields': 'id,name,fieldType(id),isPrivate'
        }

        response = self._make_request('GET', endpoint, params=params)
        return response.json()

    def get_issue_custom_field_value(self, issue: Dict[str, Any], field_name: str) -> Any:
        """Extract custom field value from issue.

        Args:
            issue: Issue data
            field_name: Custom field name

        Returns:
            Field value or None if not found
        """
        custom_fields = issue.get('customFields', [])
        for field in custom_fields:
            if field.get('name') == field_name:
                value = field.get('value')
                if isinstance(value, list) and len(value) > 0:
                    # Handle multi-value fields
                    return [item.get('name', item) for item in value]
                elif isinstance(value, dict):
                    return value.get('name', value)
                return value
        return None

    def calculate_epic_progress(self, epic_id: str) -> Dict[str, Any]:
        """Calculate progress statistics for an epic.

        Args:
            epic_id: Epic ID

        Returns:
            Progress statistics
        """
        tasks = self.get_epic_tasks(epic_id)

        stats = {
            'total_tasks': len(tasks),
            'completed_tasks': 0,
            'in_progress_tasks': 0,
            'open_tasks': 0,
            'blocked_tasks': 0,
            'story_points_total': 0,
            'story_points_completed': 0,
            'completion_percentage': 0
        }

        for task in tasks:
            # Get task state
            state = self.get_issue_custom_field_value(task, 'State')
            if not state:
                # Fallback to resolved field
                state = 'Done' if task.get('resolved') else 'Open'

            # Count by state
            if state in ['Done', 'Fixed', 'Completed', 'Resolved']:
                stats['completed_tasks'] += 1
            elif state in ['In Progress', 'Implementing', 'Testing']:
                stats['in_progress_tasks'] += 1
            elif state in ['Blocked', 'On Hold']:
                stats['blocked_tasks'] += 1
            else:
                stats['open_tasks'] += 1

            # Sum story points if available
            story_points = self.get_issue_custom_field_value(task, 'Story Points')
            if story_points and isinstance(story_points, (int, float)):
                stats['story_points_total'] += story_points
                if state in ['Done', 'Fixed', 'Completed', 'Resolved']:
                    stats['story_points_completed'] += story_points

        # Calculate completion percentage
        if stats['total_tasks'] > 0:
            stats['completion_percentage'] = (stats['completed_tasks'] / stats['total_tasks']) * 100

        return stats

    def get_recent_activity(self, issue_id: str, days: int = 7) -> List[Dict[str, Any]]:
        """Get recent activity for an issue.

        Args:
            issue_id: Issue ID
            days: Number of days to look back

        Returns:
            List of recent activities
        """
        cutoff_timestamp = int((datetime.now() - timedelta(days=days)).timestamp() * 1000)

        history = self.get_issue_history(issue_id)
        recent_activity = []

        for activity in history:
            if activity.get('timestamp', 0) >= cutoff_timestamp:
                recent_activity.append(activity)

        return recent_activity

    @classmethod
    def from_env(cls) -> 'YouTrackClient':
        """Create client from environment variables.

        Returns:
            YouTrack client instance

        Raises:
            ValueError: If required environment variables are missing
        """
        url = os.getenv('YOUTRACK_URL') or os.getenv('YOUTRACK_BASE_URL')
        token = os.getenv('YOUTRACK_TOKEN')

        if not url:
            raise ValueError("YOUTRACK_URL or YOUTRACK_BASE_URL environment variable is required")
        if not token:
            raise ValueError("YOUTRACK_TOKEN environment variable is required")

        config = YouTrackConfig(
            url=url,
            token=token,
            timeout=int(os.getenv('YOUTRACK_TIMEOUT', '30')),
            verify_ssl=os.getenv('YOUTRACK_VERIFY_SSL', 'true').lower() == 'true'
        )

        return cls(config)