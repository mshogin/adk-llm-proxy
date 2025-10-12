#!/usr/bin/env python3

import asyncio
import json
import logging
import os
import sys
import time
import unittest.mock
from typing import Dict, Any, List
from datetime import datetime, timedelta

# Add parent directories to path for imports
sys.path.append(os.path.dirname(os.path.dirname(os.path.dirname(__file__))))

from gitlab_client import GitLabClient, GitLabConfig
from server import GitLabMCPServer

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class MockGitLabClient:
    """Mock GitLab client for testing."""

    def __init__(self, config: GitLabConfig):
        self.config = config
        self.mock_data = self._generate_mock_data()

    def _generate_mock_data(self) -> Dict[str, Any]:
        """Generate mock data for testing."""
        base_time = datetime.now()
        return {
            'user_info': {
                'id': 1,
                'name': 'Test User',
                'username': 'testuser',
                'email': 'test@example.com'
            },
            'projects': [
                {
                    'id': 123,
                    'name': 'Test Project',
                    'path_with_namespace': 'group/test-project',
                    'description': 'A test project for MCP GitLab integration',
                    'last_activity_at': base_time.isoformat(),
                    'star_count': 5,
                    'forks_count': 2,
                    'default_branch': 'main'
                },
                {
                    'id': 456,
                    'name': 'Another Project',
                    'path_with_namespace': 'group/another-project',
                    'description': 'Another test project with API integration',
                    'last_activity_at': (base_time - timedelta(days=1)).isoformat(),
                    'star_count': 10,
                    'forks_count': 5,
                    'default_branch': 'master'
                }
            ],
            'commits': [
                {
                    'id': 'abc123def456',
                    'short_id': 'abc123d',
                    'title': 'Fix authentication bug PROJ-101',
                    'message': 'Fix authentication bug PROJ-101\n\nResolves issue with login validation',
                    'author_name': 'Alice Developer',
                    'author_email': 'alice@example.com',
                    'created_at': (base_time - timedelta(hours=2)).isoformat() + 'Z',
                    'stats': {'additions': 25, 'deletions': 10, 'total': 35}
                },
                {
                    'id': 'def456ghi789',
                    'short_id': 'def456g',
                    'title': 'Add user profile feature PROJ-102',
                    'message': 'Add user profile feature PROJ-102\n\nImplements new profile management',
                    'author_name': 'Bob Developer',
                    'author_email': 'bob@example.com',
                    'created_at': (base_time - timedelta(hours=5)).isoformat() + 'Z',
                    'stats': {'additions': 150, 'deletions': 5, 'total': 155}
                },
                {
                    'id': 'ghi789jkl012',
                    'short_id': 'ghi789j',
                    'title': 'Refactor database queries',
                    'message': 'Refactor database queries\n\nImprove performance and maintainability',
                    'author_name': 'Charlie Developer',
                    'author_email': 'charlie@example.com',
                    'created_at': (base_time - timedelta(days=1)).isoformat() + 'Z',
                    'stats': {'additions': 75, 'deletions': 120, 'total': 195}
                }
            ],
            'merge_requests': [
                {
                    'id': 1001,
                    'iid': 10,
                    'title': 'Feature: Enhanced user authentication PROJ-100',
                    'description': 'This MR implements enhanced user authentication features including 2FA support.',
                    'state': 'opened',
                    'created_at': (base_time - timedelta(hours=6)).isoformat() + 'Z',
                    'updated_at': (base_time - timedelta(hours=1)).isoformat() + 'Z',
                    'source_branch': 'feature/enhanced-auth',
                    'target_branch': 'main',
                    'author': {'name': 'Alice Developer', 'username': 'alice'},
                    'assignees': [{'name': 'Bob Developer', 'username': 'bob'}]
                },
                {
                    'id': 1002,
                    'iid': 11,
                    'title': 'Bugfix: API response formatting PROJ-103',
                    'description': 'Fixes API response formatting issues.',
                    'state': 'merged',
                    'created_at': (base_time - timedelta(days=2)).isoformat() + 'Z',
                    'updated_at': (base_time - timedelta(days=1)).isoformat() + 'Z',
                    'merged_at': (base_time - timedelta(days=1)).isoformat() + 'Z',
                    'source_branch': 'fix/api-response',
                    'target_branch': 'main',
                    'author': {'name': 'Charlie Developer', 'username': 'charlie'}
                }
            ],
            'branches': [
                {
                    'name': 'main',
                    'default': True,
                    'commit': {
                        'id': 'abc123def456',
                        'title': 'Fix authentication bug PROJ-101',
                        'created_at': (base_time - timedelta(hours=2)).isoformat() + 'Z',
                        'author_name': 'Alice Developer'
                    }
                },
                {
                    'name': 'feature/enhanced-auth',
                    'default': False,
                    'commit': {
                        'id': 'def456ghi789',
                        'title': 'Add user profile feature PROJ-102',
                        'created_at': (base_time - timedelta(hours=5)).isoformat() + 'Z',
                        'author_name': 'Bob Developer'
                    }
                },
                {
                    'name': 'old/deprecated-feature',
                    'default': False,
                    'commit': {
                        'id': 'old123old456',
                        'title': 'Old feature implementation',
                        'created_at': (base_time - timedelta(days=45)).isoformat() + 'Z',
                        'author_name': 'Old Developer'
                    }
                }
            ],
            'diffs': [
                {
                    'new_path': 'src/auth/authentication.py',
                    'old_path': 'src/auth/authentication.py',
                    'diff': '@@ -10,5 +10,8 @@ def authenticate(user):\n+    if not user.is_active:\n+        return False\n     return validate_password(user.password)'
                },
                {
                    'new_path': 'src/models/user.py',
                    'old_path': 'src/models/user.py',
                    'diff': '@@ -5,2 +5,5 @@ class User:\n+    def is_active(self):\n+        return self.status == "active"'
                }
            ]
        }

    def test_connection(self) -> bool:
        """Mock connection test."""
        return True

    def get_user_info(self) -> Dict[str, Any]:
        """Mock user info."""
        return self.mock_data['user_info']

    def search_projects(self, search: str, limit: int = 20) -> List[Dict[str, Any]]:
        """Mock project search."""
        projects = []
        for project in self.mock_data['projects']:
            if (search.lower() in project['name'].lower() or
                search.lower() in project['description'].lower()):
                projects.append(project)
        return projects[:limit]

    def get_project(self, project_id: str) -> Dict[str, Any]:
        """Mock get project."""
        for project in self.mock_data['projects']:
            if str(project['id']) == str(project_id) or project['path_with_namespace'] == project_id:
                return project
        raise ValueError(f"Project not found: {project_id}")

    def get_project_commits(self, project_id: str, branch: str = None,
                           since: datetime = None, until: datetime = None,
                           per_page: int = 100) -> List[Dict[str, Any]]:
        """Mock get project commits."""
        commits = self.mock_data['commits'].copy()

        # Filter by date if specified
        if since:
            commits = [
                commit for commit in commits
                if datetime.fromisoformat(commit['created_at'].replace('Z', '+00:00')) >= since
            ]

        return commits[:per_page]

    def get_commit_details(self, project_id: str, commit_sha: str) -> Dict[str, Any]:
        """Mock get commit details."""
        for commit in self.mock_data['commits']:
            if commit['id'].startswith(commit_sha) or commit['short_id'] == commit_sha:
                commit_details = commit.copy()
                commit_details['stats'] = commit.get('stats', {'additions': 10, 'deletions': 5, 'total': 15})
                return commit_details
        raise ValueError(f"Commit not found: {commit_sha}")

    def get_commit_diff(self, project_id: str, commit_sha: str) -> List[Dict[str, Any]]:
        """Mock get commit diff."""
        return self.mock_data['diffs']

    def get_merge_requests(self, project_id: str, state: str = 'all',
                          target_branch: str = None, per_page: int = 100) -> List[Dict[str, Any]]:
        """Mock get merge requests."""
        mrs = self.mock_data['merge_requests'].copy()

        if state != 'all':
            mrs = [mr for mr in mrs if mr['state'] == state]

        return mrs[:per_page]

    def get_branches(self, project_id: str, per_page: int = 100) -> List[Dict[str, Any]]:
        """Mock get branches."""
        return self.mock_data['branches'][:per_page]

    def analyze_commit_messages(self, commits: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Mock analyze commit messages."""
        return {
            'total_commits': len(commits),
            'task_references': ['PROJ-101', 'PROJ-102', 'PROJ-103'],
            'commit_types': {'fix': 2, 'feat': 1, 'refactor': 1},
            'authors': {'Alice Developer': 1, 'Bob Developer': 1, 'Charlie Developer': 1},
            'common_keywords': {'bug': 1, 'feature': 1, 'user': 2, 'authentication': 2},
            'avg_message_length': 65.5,
            'commits_with_tasks': 2
        }

    def calculate_code_metrics(self, project_id: str, commits: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Mock calculate code metrics."""
        return {
            'total_commits': len(commits),
            'total_additions': 250,
            'total_deletions': 135,
            'files_changed': 15,
            'file_types': {'py': 8, 'js': 4, 'html': 2, 'css': 1},
            'largest_commits': [
                {
                    'id': 'def456g',
                    'message': 'Add user profile feature PROJ-102',
                    'additions': 150,
                    'deletions': 5,
                    'total_changes': 155,
                    'author': 'Bob Developer'
                }
            ],
            'most_active_files': {
                'src/auth/authentication.py': 3,
                'src/models/user.py': 2,
                'src/api/views.py': 2
            }
        }

    def analyze_developer_activity(self, commits: List[Dict[str, Any]], days: int = 30) -> Dict[str, Any]:
        """Mock analyze developer activity."""
        base_time = datetime.now()
        return {
            'period_days': days,
            'total_commits': len(commits),
            'active_developers': {
                'Alice Developer': {
                    'commits': 1,
                    'first_commit': (base_time - timedelta(hours=2)).isoformat(),
                    'last_commit': (base_time - timedelta(hours=2)).isoformat()
                },
                'Bob Developer': {
                    'commits': 1,
                    'first_commit': (base_time - timedelta(hours=5)).isoformat(),
                    'last_commit': (base_time - timedelta(hours=5)).isoformat()
                },
                'Charlie Developer': {
                    'commits': 1,
                    'first_commit': (base_time - timedelta(days=1)).isoformat(),
                    'last_commit': (base_time - timedelta(days=1)).isoformat()
                }
            },
            'daily_activity': {
                base_time.strftime('%Y-%m-%d'): 2,
                (base_time - timedelta(days=1)).strftime('%Y-%m-%d'): 1
            },
            'hourly_activity': {14: 2, 9: 1},
            'most_productive_day': (base_time.strftime('%Y-%m-%d'), 2),
            'most_productive_hour': (14, 2)
        }

    def assess_code_complexity(self, project_id: str, commits: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Mock assess code complexity."""
        return {
            'total_files_analyzed': 15,
            'complexity_indicators': {
                'large_commits': 1,
                'many_files_changed': 0,
                'frequent_changes': {
                    'src/auth/authentication.py': 5,
                    'src/models/user.py': 3
                },
                'refactoring_commits': 1,
                'bug_fix_commits': 2
            },
            'risk_indicators': ['High ratio of bug fix commits (>30%)'],
            'recommendations': [
                'Consider breaking down large commits',
                'Review frequently changed files for refactoring opportunities'
            ]
        }

    def extract_task_ids_from_text(self, text: str) -> List[str]:
        """Mock extract task IDs."""
        import re
        pattern = r'\b[A-Z]+-\d+\b'
        matches = re.findall(pattern, text, re.IGNORECASE)
        return [match.upper() for match in matches]

class GitLabServerTester:
    """Comprehensive tester for GitLab MCP Server."""

    def __init__(self):
        self.results: List[Dict[str, Any]] = []

    async def run_all_tests(self) -> Dict[str, Any]:
        """Run all tests and return results."""
        logger.info("Starting GitLab MCP Server comprehensive testing")

        # Test environment setup
        await self._test_setup()

        # Repository analysis tools
        await self._test_repository_tools()

        # Code-to-task correlation
        await self._test_correlation_tools()

        # Advanced analysis tools
        await self._test_advanced_analysis_tools()

        # Resource and prompt tests
        await self._test_resources_and_prompts()

        # Performance benchmarking
        await self._test_performance()

        # YouTrack-GitLab correlation
        await self._test_youtrack_gitlab_correlation()

        # Generate summary
        return self._generate_test_summary()

    async def _test_setup(self):
        """Test server setup and connection."""
        logger.info("Testing server setup...")

        try:
            # Mock environment variables
            os.environ['GITLAB_URL'] = 'https://gitlab.example.com'
            os.environ['GITLAB_TOKEN'] = 'test-token-123'
            os.environ['GITLAB_PROJECT_ID'] = '123'

            # Create server with mocked client
            server = GitLabMCPServer()

            # Mock the client creation
            with unittest.mock.patch('gitlab_client.GitLabClient.from_env') as mock_from_env:
                mock_client = MockGitLabClient(GitLabConfig(
                    url='https://gitlab.example.com',
                    token='test-token-123',
                    project_id='123'
                ))
                mock_from_env.return_value = mock_client

                await server.setup()

                self.results.append({
                    'test': 'server_setup',
                    'passed': True,
                    'message': 'Server setup successful'
                })

        except Exception as e:
            self.results.append({
                'test': 'server_setup',
                'passed': False,
                'message': f'Server setup failed: {e}'
            })

    async def _test_repository_tools(self):
        """Test repository analysis tools."""
        logger.info("Testing repository analysis tools...")

        # Mock server with client
        server = GitLabMCPServer()
        server.client = MockGitLabClient(GitLabConfig(
            url='https://gitlab.example.com',
            token='test-token-123',
            project_id='123'
        ))

        # Test find_project
        try:
            result = await server.find_project("test")
            self.results.append({
                'test': 'find_project',
                'passed': 'Test Project' in result,
                'message': f'Project search result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'find_project',
                'passed': False,
                'message': f'Find project failed: {e}'
            })

        # Test get_recent_commits
        try:
            result = await server.get_recent_commits("123")
            self.results.append({
                'test': 'get_recent_commits',
                'passed': 'PROJ-101' in result,
                'message': f'Recent commits result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'get_recent_commits',
                'passed': False,
                'message': f'Get recent commits failed: {e}'
            })

        # Test analyze_commit_messages
        try:
            result = await server.analyze_commit_messages("123")
            self.results.append({
                'test': 'analyze_commit_messages',
                'passed': 'Analysis' in result and 'PROJ-' in result,
                'message': f'Commit analysis result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'analyze_commit_messages',
                'passed': False,
                'message': f'Analyze commit messages failed: {e}'
            })

        # Test get_merge_requests
        try:
            result = await server.get_merge_requests("123")
            self.results.append({
                'test': 'get_merge_requests',
                'passed': 'Enhanced user authentication' in result,
                'message': f'Merge requests result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'get_merge_requests',
                'passed': False,
                'message': f'Get merge requests failed: {e}'
            })

        # Test analyze_branch_activity
        try:
            result = await server.analyze_branch_activity("123")
            self.results.append({
                'test': 'analyze_branch_activity',
                'passed': 'Branch Activity' in result,
                'message': f'Branch activity result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'analyze_branch_activity',
                'passed': False,
                'message': f'Analyze branch activity failed: {e}'
            })

    async def _test_correlation_tools(self):
        """Test code-to-task correlation tools."""
        logger.info("Testing correlation tools...")

        # Mock server with client
        server = GitLabMCPServer()
        server.client = MockGitLabClient(GitLabConfig(
            url='https://gitlab.example.com',
            token='test-token-123',
            project_id='123'
        ))

        # Test link_commits_to_tasks
        try:
            result = await server.link_commits_to_tasks("123")
            self.results.append({
                'test': 'link_commits_to_tasks',
                'passed': 'PROJ-101' in result and 'Correlation' in result,
                'message': f'Task linking result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'link_commits_to_tasks',
                'passed': False,
                'message': f'Link commits to tasks failed: {e}'
            })

        # Test analyze_epic_code_changes
        try:
            result = await server.analyze_epic_code_changes("123", "PROJ-100")
            self.results.append({
                'test': 'analyze_epic_code_changes',
                'passed': 'Code Changes Analysis' in result or 'No commits found' in result,
                'message': f'Epic code analysis result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'analyze_epic_code_changes',
                'passed': False,
                'message': f'Analyze epic code changes failed: {e}'
            })

        # Test get_code_metrics
        try:
            result = await server.get_code_metrics("123")
            self.results.append({
                'test': 'get_code_metrics',
                'passed': 'Code Metrics Report' in result,
                'message': f'Code metrics result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'get_code_metrics',
                'passed': False,
                'message': f'Get code metrics failed: {e}'
            })

        # Test track_developer_activity
        try:
            result = await server.track_developer_activity("123")
            self.results.append({
                'test': 'track_developer_activity',
                'passed': 'Developer Activity Report' in result,
                'message': f'Developer activity result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'track_developer_activity',
                'passed': False,
                'message': f'Track developer activity failed: {e}'
            })

    async def _test_advanced_analysis_tools(self):
        """Test advanced analysis tools."""
        logger.info("Testing advanced analysis tools...")

        # Mock server with client
        server = GitLabMCPServer()
        server.client = MockGitLabClient(GitLabConfig(
            url='https://gitlab.example.com',
            token='test-token-123',
            project_id='123'
        ))

        # Test analyze_code_complexity
        try:
            result = await server.analyze_code_complexity("123")
            self.results.append({
                'test': 'analyze_code_complexity',
                'passed': 'Code Complexity Analysis' in result,
                'message': f'Code complexity result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'analyze_code_complexity',
                'passed': False,
                'message': f'Analyze code complexity failed: {e}'
            })

    async def _test_resources_and_prompts(self):
        """Test resources and prompts."""
        logger.info("Testing resources and prompts...")

        # Mock server with client
        server = GitLabMCPServer()
        server.client = MockGitLabClient(GitLabConfig(
            url='https://gitlab.example.com',
            token='test-token-123',
            project_id='123'
        ))

        # Test project resource
        try:
            result = await server.get_project_resource("gitlab://project/123")
            parsed = json.loads(result)
            self.results.append({
                'test': 'project_resource',
                'passed': 'project' in parsed and 'recent_commits' in parsed,
                'message': f'Project resource result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'project_resource',
                'passed': False,
                'message': f'Project resource failed: {e}'
            })

        # Test code review prompt
        try:
            result = await server.code_review_prompt("123", "10")
            self.results.append({
                'test': 'code_review_prompt',
                'passed': '123' in result and '10' in result and 'analyze' in result.lower(),
                'message': f'Code review prompt result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'code_review_prompt',
                'passed': False,
                'message': f'Code review prompt failed: {e}'
            })

    async def _test_performance(self):
        """Test performance benchmarks."""
        logger.info("Running performance benchmarks...")

        # Mock server with client
        server = GitLabMCPServer()
        server.client = MockGitLabClient(GitLabConfig(
            url='https://gitlab.example.com',
            token='test-token-123',
            project_id='123'
        ))

        # Benchmark project search
        start_time = time.time()
        try:
            await server.find_project("test")
            duration = time.time() - start_time
            self.results.append({
                'test': 'performance_project_search',
                'passed': duration < 1.0,
                'message': f'Project search took {duration:.3f}s'
            })
        except Exception as e:
            self.results.append({
                'test': 'performance_project_search',
                'passed': False,
                'message': f'Performance test failed: {e}'
            })

        # Benchmark commit analysis
        start_time = time.time()
        try:
            await server.analyze_commit_messages("123")
            duration = time.time() - start_time
            self.results.append({
                'test': 'performance_commit_analysis',
                'passed': duration < 2.0,
                'message': f'Commit analysis took {duration:.3f}s'
            })
        except Exception as e:
            self.results.append({
                'test': 'performance_commit_analysis',
                'passed': False,
                'message': f'Performance test failed: {e}'
            })

    async def _test_youtrack_gitlab_correlation(self):
        """Test YouTrack-GitLab correlation functionality."""
        logger.info("Testing YouTrack-GitLab correlation...")

        # Mock server with client
        server = GitLabMCPServer()
        server.client = MockGitLabClient(GitLabConfig(
            url='https://gitlab.example.com',
            token='test-token-123',
            project_id='123'
        ))

        # Test task ID extraction
        try:
            task_ids = server.client.extract_task_ids_from_text(
                "Fix authentication bug PROJ-101 and implement PROJ-102"
            )
            self.results.append({
                'test': 'task_id_extraction',
                'passed': 'PROJ-101' in task_ids and 'PROJ-102' in task_ids,
                'message': f'Extracted task IDs: {task_ids}'
            })
        except Exception as e:
            self.results.append({
                'test': 'task_id_extraction',
                'passed': False,
                'message': f'Task ID extraction failed: {e}'
            })

        # Test commit-to-task correlation
        try:
            result = await server.link_commits_to_tasks("123")
            self.results.append({
                'test': 'youtrack_gitlab_correlation',
                'passed': 'PROJ-' in result and 'Correlation' in result,
                'message': f'Correlation test result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'youtrack_gitlab_correlation',
                'passed': False,
                'message': f'YouTrack-GitLab correlation failed: {e}'
            })

    def _generate_test_summary(self) -> Dict[str, Any]:
        """Generate test summary."""
        total_tests = len(self.results)
        passed_tests = sum(1 for result in self.results if result['passed'])
        failed_tests = total_tests - passed_tests

        summary = {
            'total_tests': total_tests,
            'passed_tests': passed_tests,
            'failed_tests': failed_tests,
            'success_rate': (passed_tests / total_tests * 100) if total_tests > 0 else 0,
            'results': self.results
        }

        return summary

async def main():
    """Run comprehensive tests."""
    tester = GitLabServerTester()
    summary = await tester.run_all_tests()

    print("\n" + "="*60)
    print("GITLAB MCP SERVER TEST RESULTS")
    print("="*60)
    print(f"Total Tests: {summary['total_tests']}")
    print(f"Passed: {summary['passed_tests']}")
    print(f"Failed: {summary['failed_tests']}")
    print(f"Success Rate: {summary['success_rate']:.1f}%")

    print("\nDetailed Results:")
    for result in summary['results']:
        status = "✅ PASS" if result['passed'] else "❌ FAIL"
        print(f"{status} {result['test']}: {result['message']}")

    if summary['failed_tests'] == 0:
        print("\n🎉 All tests passed! GitLab MCP Server is working correctly.")
    else:
        print(f"\n⚠️  {summary['failed_tests']} test(s) failed. Please review the issues above.")

    return 0 if summary['failed_tests'] == 0 else 1

if __name__ == "__main__":
    exit(asyncio.run(main()))