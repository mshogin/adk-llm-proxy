import os
import re
import logging
import requests
from typing import Dict, List, Any, Optional, Tuple
from datetime import datetime, timedelta
from dataclasses import dataclass
from collections import defaultdict

logger = logging.getLogger(__name__)

@dataclass
class GitLabConfig:
    """GitLab configuration."""
    url: str
    token: str
    project_id: Optional[str] = None
    timeout: int = 30
    verify_ssl: bool = True

class GitLabClient:
    """GitLab REST API client with authentication and error handling."""

    def __init__(self, config: GitLabConfig):
        """Initialize GitLab client.

        Args:
            config: GitLab configuration
        """
        self.config = config
        self.base_url = config.url.rstrip('/')
        self.headers = {
            'Private-Token': config.token,
            'Content-Type': 'application/json'
        }
        self.session = requests.Session()
        self.session.headers.update(self.headers)

    def _make_request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        """Make HTTP request to GitLab API.

        Args:
            method: HTTP method
            endpoint: API endpoint
            **kwargs: Additional request arguments

        Returns:
            Response object

        Raises:
            requests.RequestException: If request fails
        """
        # Ensure endpoint starts with /api/v4
        if not endpoint.startswith('/api/v4'):
            endpoint = f'/api/v4{endpoint}'

        url = f"{self.base_url}{endpoint}"

        # Set default timeout and SSL verification
        kwargs.setdefault('timeout', self.config.timeout)
        kwargs.setdefault('verify', self.config.verify_ssl)

        try:
            response = self.session.request(method, url, **kwargs)
            response.raise_for_status()
            return response
        except requests.RequestException as e:
            logger.error(f"GitLab API request failed: {method} {url} - {e}")
            raise

    def test_connection(self) -> bool:
        """Test connection to GitLab.

        Returns:
            True if connection successful, False otherwise
        """
        try:
            response = self._make_request('GET', '/user')
            logger.info("GitLab connection test successful")
            return True
        except Exception as e:
            logger.error(f"GitLab connection test failed: {e}")
            return False

    def get_user_info(self) -> Dict[str, Any]:
        """Get current user information.

        Returns:
            User information dictionary
        """
        response = self._make_request('GET', '/user')
        return response.json()

    def get_user_events(self, limit: int = 100, action: Optional[str] = None) -> List[Dict[str, Any]]:
        """Get current user's events/activity.

        Args:
            limit: Maximum number of events to retrieve
            action: Filter by action type (e.g., 'pushed', 'created', 'merged')

        Returns:
            List of user events
        """
        params = {
            'per_page': limit,
            'sort': 'desc'
        }
        if action:
            params['action'] = action

        response = self._make_request('GET', '/events', params=params)
        return response.json()

    def search_projects(self, search: str, limit: int = 20) -> List[Dict[str, Any]]:
        """Search for projects.

        Args:
            search: Search term
            limit: Maximum number of results

        Returns:
            List of projects
        """
        params = {
            'search': search,
            'per_page': limit,
            'order_by': 'last_activity_at',
            'sort': 'desc'
        }

        response = self._make_request('GET', '/projects', params=params)
        return response.json()

    def get_project(self, project_id: str) -> Dict[str, Any]:
        """Get project information.

        Args:
            project_id: Project ID or path

        Returns:
            Project information
        """
        # URL encode project ID in case it contains special characters
        from urllib.parse import quote_plus
        encoded_id = quote_plus(str(project_id))

        response = self._make_request('GET', f'/projects/{encoded_id}')
        return response.json()

    def get_project_commits(self, project_id: str, branch: str = None,
                          since: datetime = None, until: datetime = None,
                          per_page: int = 100) -> List[Dict[str, Any]]:
        """Get commits for a project.

        Args:
            project_id: Project ID
            branch: Branch name (default: project's default branch)
            since: Start date
            until: End date
            per_page: Items per page

        Returns:
            List of commits
        """
        from urllib.parse import quote_plus
        encoded_id = quote_plus(str(project_id))

        params = {'per_page': per_page}

        if branch:
            params['ref_name'] = branch
        if since:
            params['since'] = since.isoformat()
        if until:
            params['until'] = until.isoformat()

        response = self._make_request('GET', f'/projects/{encoded_id}/repository/commits', params=params)
        return response.json()

    def get_commit_details(self, project_id: str, commit_sha: str) -> Dict[str, Any]:
        """Get detailed information about a commit.

        Args:
            project_id: Project ID
            commit_sha: Commit SHA

        Returns:
            Commit details
        """
        from urllib.parse import quote_plus
        encoded_id = quote_plus(str(project_id))

        response = self._make_request('GET', f'/projects/{encoded_id}/repository/commits/{commit_sha}')
        return response.json()

    def get_commit_diff(self, project_id: str, commit_sha: str) -> List[Dict[str, Any]]:
        """Get commit diff information.

        Args:
            project_id: Project ID
            commit_sha: Commit SHA

        Returns:
            List of diff entries
        """
        from urllib.parse import quote_plus
        encoded_id = quote_plus(str(project_id))

        response = self._make_request('GET', f'/projects/{encoded_id}/repository/commits/{commit_sha}/diff')
        return response.json()

    def get_merge_requests(self, project_id: str, state: str = 'all',
                          target_branch: str = None, per_page: int = 100) -> List[Dict[str, Any]]:
        """Get merge requests for a project.

        Args:
            project_id: Project ID
            state: MR state (opened, closed, merged, all)
            target_branch: Target branch filter
            per_page: Items per page

        Returns:
            List of merge requests
        """
        from urllib.parse import quote_plus
        encoded_id = quote_plus(str(project_id))

        params = {
            'state': state,
            'per_page': per_page,
            'order_by': 'updated_at',
            'sort': 'desc'
        }

        if target_branch:
            params['target_branch'] = target_branch

        response = self._make_request('GET', f'/projects/{encoded_id}/merge_requests', params=params)
        return response.json()

    def get_merge_request_details(self, project_id: str, mr_iid: int) -> Dict[str, Any]:
        """Get detailed merge request information.

        Args:
            project_id: Project ID
            mr_iid: Merge request IID

        Returns:
            Merge request details
        """
        from urllib.parse import quote_plus
        encoded_id = quote_plus(str(project_id))

        response = self._make_request('GET', f'/projects/{encoded_id}/merge_requests/{mr_iid}')
        return response.json()

    def get_merge_request_commits(self, project_id: str, mr_iid: int) -> List[Dict[str, Any]]:
        """Get commits in a merge request.

        Args:
            project_id: Project ID
            mr_iid: Merge request IID

        Returns:
            List of commits
        """
        from urllib.parse import quote_plus
        encoded_id = quote_plus(str(project_id))

        response = self._make_request('GET', f'/projects/{encoded_id}/merge_requests/{mr_iid}/commits')
        return response.json()

    def get_branches(self, project_id: str, per_page: int = 100) -> List[Dict[str, Any]]:
        """Get branches for a project.

        Args:
            project_id: Project ID
            per_page: Items per page

        Returns:
            List of branches
        """
        from urllib.parse import quote_plus
        encoded_id = quote_plus(str(project_id))

        params = {'per_page': per_page}

        response = self._make_request('GET', f'/projects/{encoded_id}/repository/branches', params=params)
        return response.json()

    def extract_task_ids_from_text(self, text: str) -> List[str]:
        """Extract YouTrack task IDs from text.

        Args:
            text: Text to search for task IDs

        Returns:
            List of found task IDs
        """
        # Common patterns for YouTrack task IDs
        patterns = [
            r'\b[A-Z]+-\d+\b',  # PROJECT-123
            r'\b[A-Z]{2,}\s*#\d+\b',  # PROJECT #123
            r'#([A-Z]+-\d+)\b',  # #PROJECT-123
        ]

        task_ids = []
        for pattern in patterns:
            matches = re.findall(pattern, text, re.IGNORECASE)
            task_ids.extend(matches)

        # Clean up and deduplicate
        cleaned_ids = []
        for task_id in task_ids:
            # Remove # prefix and normalize
            clean_id = task_id.strip('#').upper()
            if '-' in clean_id and clean_id not in cleaned_ids:
                cleaned_ids.append(clean_id)

        return cleaned_ids

    def analyze_commit_messages(self, commits: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Analyze commit messages for patterns and insights.

        Args:
            commits: List of commit objects

        Returns:
            Analysis results
        """
        analysis = {
            'total_commits': len(commits),
            'task_references': [],
            'commit_types': defaultdict(int),
            'authors': defaultdict(int),
            'common_keywords': defaultdict(int),
            'avg_message_length': 0,
            'commits_with_tasks': 0
        }

        if not commits:
            return analysis

        total_length = 0
        keywords = ['fix', 'feat', 'refactor', 'docs', 'test', 'chore', 'style', 'perf']

        for commit in commits:
            message = commit.get('message', '').lower()
            author = commit.get('author_name', 'Unknown')

            # Track message length
            total_length += len(message)

            # Extract task IDs
            task_ids = self.extract_task_ids_from_text(message)
            if task_ids:
                analysis['commits_with_tasks'] += 1
                analysis['task_references'].extend(task_ids)

            # Analyze commit types (conventional commits)
            for keyword in keywords:
                if keyword in message:
                    analysis['commit_types'][keyword] += 1

            # Track authors
            analysis['authors'][author] += 1

            # Count common keywords
            words = message.split()
            for word in words:
                word = word.strip('.,!?;:()[]{}')
                if len(word) > 3 and word.isalpha():
                    analysis['common_keywords'][word] += 1

        # Calculate averages
        if analysis['total_commits'] > 0:
            analysis['avg_message_length'] = total_length / analysis['total_commits']

        # Convert defaultdicts to regular dicts and sort
        analysis['commit_types'] = dict(sorted(analysis['commit_types'].items(), key=lambda x: x[1], reverse=True))
        analysis['authors'] = dict(sorted(analysis['authors'].items(), key=lambda x: x[1], reverse=True))
        analysis['common_keywords'] = dict(list(sorted(analysis['common_keywords'].items(), key=lambda x: x[1], reverse=True))[:20])

        # Deduplicate task references
        analysis['task_references'] = list(set(analysis['task_references']))

        return analysis

    def calculate_code_metrics(self, project_id: str, commits: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Calculate code metrics from commits.

        Args:
            project_id: Project ID
            commits: List of commit objects

        Returns:
            Code metrics
        """
        metrics = {
            'total_commits': len(commits),
            'total_additions': 0,
            'total_deletions': 0,
            'files_changed': set(),
            'file_types': defaultdict(int),
            'largest_commits': [],
            'most_active_files': defaultdict(int)
        }

        for commit in commits:
            try:
                # Get commit details for statistics
                commit_details = self.get_commit_details(project_id, commit['id'])

                stats = commit_details.get('stats', {})
                additions = stats.get('additions', 0)
                deletions = stats.get('deletions', 0)
                total_changes = additions + deletions

                metrics['total_additions'] += additions
                metrics['total_deletions'] += deletions

                # Track largest commits
                if total_changes > 0:
                    commit_info = {
                        'id': commit['id'][:8],
                        'message': commit.get('message', '')[:100],
                        'additions': additions,
                        'deletions': deletions,
                        'total_changes': total_changes,
                        'author': commit.get('author_name', 'Unknown')
                    }
                    metrics['largest_commits'].append(commit_info)

                # Get diff information for file analysis
                try:
                    diff = self.get_commit_diff(project_id, commit['id'])
                    for file_diff in diff:
                        file_path = file_diff.get('new_path') or file_diff.get('old_path')
                        if file_path:
                            metrics['files_changed'].add(file_path)
                            metrics['most_active_files'][file_path] += 1

                            # Track file types
                            file_ext = file_path.split('.')[-1].lower() if '.' in file_path else 'no_ext'
                            metrics['file_types'][file_ext] += 1

                except Exception as e:
                    logger.warning(f"Could not get diff for commit {commit['id']}: {e}")

            except Exception as e:
                logger.warning(f"Could not get details for commit {commit['id']}: {e}")

        # Sort and limit results
        metrics['largest_commits'] = sorted(metrics['largest_commits'],
                                          key=lambda x: x['total_changes'], reverse=True)[:10]

        metrics['files_changed'] = len(metrics['files_changed'])
        metrics['file_types'] = dict(sorted(metrics['file_types'].items(), key=lambda x: x[1], reverse=True))
        metrics['most_active_files'] = dict(list(sorted(metrics['most_active_files'].items(),
                                                       key=lambda x: x[1], reverse=True))[:20])

        return metrics

    def analyze_developer_activity(self, commits: List[Dict[str, Any]],
                                 days: int = 30) -> Dict[str, Any]:
        """Analyze developer activity patterns.

        Args:
            commits: List of commit objects
            days: Number of days to analyze

        Returns:
            Developer activity analysis
        """
        cutoff_date = datetime.now() - timedelta(days=days)
        # Make cutoff_date timezone-aware
        cutoff_date = cutoff_date.replace(tzinfo=datetime.now().astimezone().tzinfo)

        activity = {
            'period_days': days,
            'total_commits': 0,
            'active_developers': {},
            'daily_activity': defaultdict(int),
            'hourly_activity': defaultdict(int),
            'most_productive_day': None,
            'most_productive_hour': None
        }

        for commit in commits:
            commit_date = datetime.fromisoformat(commit.get('created_at', '').replace('Z', '+00:00'))

            # Only include commits within the specified period
            if commit_date < cutoff_date:
                continue

            activity['total_commits'] += 1

            author = commit.get('author_name', 'Unknown')
            if author not in activity['active_developers']:
                activity['active_developers'][author] = {
                    'commits': 0,
                    'first_commit': commit_date,
                    'last_commit': commit_date
                }

            dev_stats = activity['active_developers'][author]
            dev_stats['commits'] += 1
            dev_stats['last_commit'] = max(dev_stats['last_commit'], commit_date)
            dev_stats['first_commit'] = min(dev_stats['first_commit'], commit_date)

            # Track daily and hourly patterns
            day_key = commit_date.strftime('%Y-%m-%d')
            hour_key = commit_date.hour

            activity['daily_activity'][day_key] += 1
            activity['hourly_activity'][hour_key] += 1

        # Find most productive periods
        if activity['daily_activity']:
            activity['most_productive_day'] = max(activity['daily_activity'].items(), key=lambda x: x[1])
        if activity['hourly_activity']:
            activity['most_productive_hour'] = max(activity['hourly_activity'].items(), key=lambda x: x[1])

        # Convert datetime objects to strings for JSON serialization
        for dev_stats in activity['active_developers'].values():
            dev_stats['first_commit'] = dev_stats['first_commit'].isoformat()
            dev_stats['last_commit'] = dev_stats['last_commit'].isoformat()

        # Convert defaultdicts to regular dicts
        activity['daily_activity'] = dict(activity['daily_activity'])
        activity['hourly_activity'] = dict(activity['hourly_activity'])

        return activity

    def assess_code_complexity(self, project_id: str, commits: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Assess code complexity trends from commit patterns.

        Args:
            project_id: Project ID
            commits: List of commit objects

        Returns:
            Complexity assessment
        """
        complexity = {
            'total_files_analyzed': 0,
            'complexity_indicators': {
                'large_commits': 0,  # >500 lines changed
                'many_files_changed': 0,  # >10 files in one commit
                'frequent_changes': defaultdict(int),  # Files changed frequently
                'refactoring_commits': 0,
                'bug_fix_commits': 0
            },
            'risk_indicators': [],
            'recommendations': []
        }

        file_change_frequency = defaultdict(int)

        for commit in commits:
            message = commit.get('message', '').lower()

            # Identify commit types
            if any(word in message for word in ['refactor', 'restructure', 'reorganize']):
                complexity['complexity_indicators']['refactoring_commits'] += 1

            if any(word in message for word in ['fix', 'bug', 'issue', 'error']):
                complexity['complexity_indicators']['bug_fix_commits'] += 1

            try:
                commit_details = self.get_commit_details(project_id, commit['id'])
                stats = commit_details.get('stats', {})
                total_changes = stats.get('additions', 0) + stats.get('deletions', 0)

                if total_changes > 500:
                    complexity['complexity_indicators']['large_commits'] += 1

                # Get file changes
                diff = self.get_commit_diff(project_id, commit['id'])
                files_changed = len(diff)

                if files_changed > 10:
                    complexity['complexity_indicators']['many_files_changed'] += 1

                # Track file change frequency
                for file_diff in diff:
                    file_path = file_diff.get('new_path') or file_diff.get('old_path')
                    if file_path:
                        file_change_frequency[file_path] += 1

                complexity['total_files_analyzed'] += files_changed

            except Exception as e:
                logger.warning(f"Could not analyze complexity for commit {commit['id']}: {e}")

        # Identify frequently changed files (complexity hotspots)
        for file_path, change_count in file_change_frequency.items():
            if change_count >= 5:  # Changed in 5+ commits
                complexity['complexity_indicators']['frequent_changes'][file_path] = change_count

        # Generate risk indicators
        total_commits = len(commits)
        if total_commits > 0:
            large_commit_ratio = complexity['complexity_indicators']['large_commits'] / total_commits
            if large_commit_ratio > 0.2:
                complexity['risk_indicators'].append("High ratio of large commits (>20%)")

            bug_fix_ratio = complexity['complexity_indicators']['bug_fix_commits'] / total_commits
            if bug_fix_ratio > 0.3:
                complexity['risk_indicators'].append("High ratio of bug fix commits (>30%)")

            if len(complexity['complexity_indicators']['frequent_changes']) > 10:
                complexity['risk_indicators'].append("Many files with frequent changes")

        # Generate recommendations
        if complexity['complexity_indicators']['large_commits'] > 0:
            complexity['recommendations'].append("Consider breaking down large commits")

        if complexity['complexity_indicators']['frequent_changes']:
            complexity['recommendations'].append("Review frequently changed files for refactoring opportunities")

        if complexity['complexity_indicators']['bug_fix_commits'] > complexity['complexity_indicators']['refactoring_commits'] * 2:
            complexity['recommendations'].append("Consider more proactive refactoring to reduce bug fixes")

        # Convert defaultdict to regular dict
        complexity['complexity_indicators']['frequent_changes'] = dict(
            sorted(complexity['complexity_indicators']['frequent_changes'].items(),
                  key=lambda x: x[1], reverse=True)
        )

        return complexity

    @classmethod
    def from_env(cls) -> 'GitLabClient':
        """Create client from environment variables.

        Returns:
            GitLab client instance

        Raises:
            ValueError: If required environment variables are missing
        """
        url = os.getenv('GITLAB_URL', 'https://gitlab.com')
        token = os.getenv('GITLAB_TOKEN')
        project_id = os.getenv('GITLAB_PROJECT_ID')

        if not token:
            raise ValueError("GITLAB_TOKEN environment variable is required")

        config = GitLabConfig(
            url=url,
            token=token,
            project_id=project_id,
            timeout=int(os.getenv('GITLAB_TIMEOUT', '30')),
            verify_ssl=os.getenv('GITLAB_VERIFY_SSL', 'true').lower() == 'true'
        )

        return cls(config)