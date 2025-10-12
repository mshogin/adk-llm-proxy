#!/usr/bin/env python3

import asyncio
import json
import logging
import os
import sys
import time
import unittest.mock
from typing import Dict, Any, List
from datetime import datetime

# Add parent directories to path for imports
sys.path.append(os.path.dirname(os.path.dirname(os.path.dirname(__file__))))

from youtrack_client import YouTrackClient, YouTrackConfig
from server import YouTrackMCPServer

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class MockYouTrackClient:
    """Mock YouTrack client for testing."""

    def __init__(self, config: YouTrackConfig):
        self.config = config
        self.mock_data = self._generate_mock_data()

    def _generate_mock_data(self) -> Dict[str, Any]:
        """Generate mock data for testing."""
        return {
            'user_info': {
                'id': 'test-user',
                'name': 'Test User',
                'email': 'test@example.com'
            },
            'epics': [
                {
                    'id': 'epic-1',
                    'idReadable': 'PROJ-100',
                    'summary': 'Test Epic 1 - User Authentication',
                    'description': 'Implement user authentication system',
                    'project': {'name': 'Test Project', 'shortName': 'PROJ'},
                    'customFields': [
                        {'name': 'Type', 'value': {'name': 'Epic'}},
                        {'name': 'State', 'value': {'name': 'In Progress'}},
                        {'name': 'Story Points', 'value': 50}
                    ],
                    'created': int(datetime(2024, 1, 1).timestamp() * 1000),
                    'updated': int(datetime.now().timestamp() * 1000),
                    'tags': [{'name': 'authentication'}, {'name': 'security'}]
                },
                {
                    'id': 'epic-2',
                    'idReadable': 'PROJ-200',
                    'summary': 'Test Epic 2 - API Integration',
                    'description': 'Integrate with external APIs',
                    'project': {'name': 'Test Project', 'shortName': 'PROJ'},
                    'customFields': [
                        {'name': 'Type', 'value': {'name': 'Epic'}},
                        {'name': 'State', 'value': {'name': 'Open'}},
                        {'name': 'Story Points', 'value': 30}
                    ],
                    'created': int(datetime(2024, 1, 15).timestamp() * 1000),
                    'updated': int(datetime.now().timestamp() * 1000),
                    'tags': [{'name': 'api'}, {'name': 'integration'}]
                }
            ],
            'tasks': [
                {
                    'id': 'task-1',
                    'idReadable': 'PROJ-101',
                    'summary': 'Implement login form',
                    'description': 'Create login form with validation',
                    'project': {'name': 'Test Project', 'shortName': 'PROJ'},
                    'customFields': [
                        {'name': 'State', 'value': {'name': 'Done'}},
                        {'name': 'Assignee', 'value': {'name': 'Alice Smith'}},
                        {'name': 'Story Points', 'value': 5}
                    ],
                    'created': int(datetime(2024, 1, 2).timestamp() * 1000),
                    'updated': int(datetime.now().timestamp() * 1000),
                    'resolved': int(datetime(2024, 1, 10).timestamp() * 1000),
                    'links': [{'issues': [{'id': 'epic-1', 'idReadable': 'PROJ-100'}]}]
                },
                {
                    'id': 'task-2',
                    'idReadable': 'PROJ-102',
                    'summary': 'Add password validation',
                    'description': 'Implement strong password validation',
                    'project': {'name': 'Test Project', 'shortName': 'PROJ'},
                    'customFields': [
                        {'name': 'State', 'value': {'name': 'In Progress'}},
                        {'name': 'Assignee', 'value': {'name': 'Bob Johnson'}},
                        {'name': 'Story Points', 'value': 3}
                    ],
                    'created': int(datetime(2024, 1, 3).timestamp() * 1000),
                    'updated': int(datetime.now().timestamp() * 1000),
                    'links': [{'issues': [{'id': 'epic-1', 'idReadable': 'PROJ-100'}]}]
                },
                {
                    'id': 'task-3',
                    'idReadable': 'PROJ-103',
                    'summary': 'Implement session management',
                    'description': 'Add secure session management',
                    'project': {'name': 'Test Project', 'shortName': 'PROJ'},
                    'customFields': [
                        {'name': 'State', 'value': {'name': 'Blocked'}},
                        {'name': 'Assignee', 'value': {'name': 'Charlie Brown'}},
                        {'name': 'Story Points', 'value': 8}
                    ],
                    'created': int(datetime(2024, 1, 4).timestamp() * 1000),
                    'updated': int(datetime.now().timestamp() * 1000),
                    'links': [{'issues': [{'id': 'epic-1', 'idReadable': 'PROJ-100'}]}]
                }
            ],
            'comments': [
                {
                    'id': 'comment-1',
                    'text': 'Login form implementation is complete. Added proper validation.',
                    'created': int(datetime(2024, 1, 10).timestamp() * 1000),
                    'author': {'name': 'Alice Smith'},
                    'deleted': False
                },
                {
                    'id': 'comment-2',
                    'text': 'Need to discuss password requirements with security team.',
                    'created': int(datetime(2024, 1, 8).timestamp() * 1000),
                    'author': {'name': 'Bob Johnson'},
                    'deleted': False
                }
            ],
            'history': [
                {
                    'id': 'history-1',
                    'timestamp': int(datetime(2024, 1, 10).timestamp() * 1000),
                    'author': {'name': 'Alice Smith'},
                    'field': {'name': 'State'},
                    'removed': [{'name': 'In Progress'}],
                    'added': [{'name': 'Done'}]
                },
                {
                    'id': 'history-2',
                    'timestamp': int(datetime(2024, 1, 9).timestamp() * 1000),
                    'author': {'name': 'Alice Smith'},
                    'field': {'name': 'Assignee'},
                    'removed': [{'name': 'Unassigned'}],
                    'added': [{'name': 'Alice Smith'}]
                }
            ]
        }

    def test_connection(self) -> bool:
        """Mock connection test."""
        return True

    def get_user_info(self) -> Dict[str, Any]:
        """Mock user info."""
        return self.mock_data['user_info']

    def find_epics(self, search_term: str, project: str = None) -> List[Dict[str, Any]]:
        """Mock epic search."""
        epics = []
        for epic in self.mock_data['epics']:
            if (search_term.lower() in epic['summary'].lower() or
                search_term.lower() in epic['idReadable'].lower() or
                any(search_term.lower() in tag['name'].lower() for tag in epic.get('tags', []))):
                epics.append(epic)
        return epics

    def get_issue(self, issue_id: str, fields: str = None) -> Dict[str, Any]:
        """Mock get issue."""
        # Look for epic first
        for epic in self.mock_data['epics']:
            if epic['idReadable'] == issue_id or epic['id'] == issue_id:
                return epic

        # Look for task
        for task in self.mock_data['tasks']:
            if task['idReadable'] == issue_id or task['id'] == issue_id:
                return task

        raise ValueError(f"Issue not found: {issue_id}")

    def get_epic_tasks(self, epic_id: str) -> List[Dict[str, Any]]:
        """Mock get epic tasks."""
        tasks = []
        for task in self.mock_data['tasks']:
            for link in task.get('links', []):
                for linked_issue in link.get('issues', []):
                    if linked_issue.get('idReadable') == epic_id or linked_issue.get('id') == epic_id:
                        tasks.append(task)
        return tasks

    def get_issue_comments(self, issue_id: str, top: int = 100) -> List[Dict[str, Any]]:
        """Mock get issue comments."""
        return self.mock_data['comments'][:top]

    def get_issue_history(self, issue_id: str, top: int = 100) -> List[Dict[str, Any]]:
        """Mock get issue history."""
        return self.mock_data['history'][:top]

    def get_issue_custom_field_value(self, issue: Dict[str, Any], field_name: str) -> Any:
        """Mock get custom field value."""
        custom_fields = issue.get('customFields', [])
        for field in custom_fields:
            if field.get('name') == field_name:
                value = field.get('value')
                if isinstance(value, dict):
                    return value.get('name', value)
                return value
        return None

    def calculate_epic_progress(self, epic_id: str) -> Dict[str, Any]:
        """Mock calculate epic progress."""
        tasks = self.get_epic_tasks(epic_id)

        stats = {
            'total_tasks': len(tasks),
            'completed_tasks': 0,
            'in_progress_tasks': 0,
            'open_tasks': 0,
            'blocked_tasks': 0,
            'story_points_total': 0,
            'story_points_completed': 0
        }

        for task in tasks:
            state = self.get_issue_custom_field_value(task, 'State')
            story_points = self.get_issue_custom_field_value(task, 'Story Points')

            if story_points:
                stats['story_points_total'] += story_points

            if state == 'Done':
                stats['completed_tasks'] += 1
                if story_points:
                    stats['story_points_completed'] += story_points
            elif state == 'In Progress':
                stats['in_progress_tasks'] += 1
            elif state == 'Blocked':
                stats['blocked_tasks'] += 1
            else:
                stats['open_tasks'] += 1

        if stats['total_tasks'] > 0:
            stats['completion_percentage'] = (stats['completed_tasks'] / stats['total_tasks']) * 100
        else:
            stats['completion_percentage'] = 0

        return stats

    def get_recent_activity(self, issue_id: str, days: int = 7) -> List[Dict[str, Any]]:
        """Mock get recent activity."""
        return self.mock_data['history']

class YouTrackServerTester:
    """Comprehensive tester for YouTrack MCP Server."""

    def __init__(self):
        self.results: List[Dict[str, Any]] = []

    async def run_all_tests(self) -> Dict[str, Any]:
        """Run all tests and return results."""
        logger.info("Starting YouTrack MCP Server comprehensive testing")

        # Test environment setup
        await self._test_setup()

        # Epic management tools
        await self._test_epic_tools()

        # Task analysis tools
        await self._test_task_tools()

        # Analytics tools
        await self._test_analytics_tools()

        # Resource and prompt tests
        await self._test_resources_and_prompts()

        # Performance benchmarking
        await self._test_performance()

        # Generate summary
        return self._generate_test_summary()

    async def _test_setup(self):
        """Test server setup and connection."""
        logger.info("Testing server setup...")

        try:
            # Mock environment variables
            os.environ['YOUTRACK_URL'] = 'https://test.youtrack.cloud'
            os.environ['YOUTRACK_TOKEN'] = 'test-token-123'

            # Create server with mocked client
            server = YouTrackMCPServer()

            # Mock the client creation
            with unittest.mock.patch('youtrack_client.YouTrackClient.from_env') as mock_from_env:
                mock_client = MockYouTrackClient(YouTrackConfig(
                    url='https://test.youtrack.cloud',
                    token='test-token-123'
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

    async def _test_epic_tools(self):
        """Test epic management tools."""
        logger.info("Testing epic management tools...")

        # Mock server with client
        server = YouTrackMCPServer()
        server.client = MockYouTrackClient(YouTrackConfig(
            url='https://test.youtrack.cloud',
            token='test-token-123'
        ))

        # Test find_epic
        try:
            result = await server.find_epic("authentication")
            self.results.append({
                'test': 'find_epic',
                'passed': 'PROJ-100' in result,
                'message': f'Find epic result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'find_epic',
                'passed': False,
                'message': f'Find epic failed: {e}'
            })

        # Test get_epic_details
        try:
            result = await server.get_epic_details("PROJ-100")
            self.results.append({
                'test': 'get_epic_details',
                'passed': 'User Authentication' in result,
                'message': f'Epic details result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'get_epic_details',
                'passed': False,
                'message': f'Get epic details failed: {e}'
            })

        # Test list_epic_tasks
        try:
            result = await server.list_epic_tasks("PROJ-100")
            self.results.append({
                'test': 'list_epic_tasks',
                'passed': 'PROJ-101' in result,
                'message': f'Epic tasks result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'list_epic_tasks',
                'passed': False,
                'message': f'List epic tasks failed: {e}'
            })

        # Test get_epic_status_summary
        try:
            result = await server.get_epic_status_summary("PROJ-100")
            self.results.append({
                'test': 'get_epic_status_summary',
                'passed': 'Total Tasks:' in result,
                'message': f'Epic status result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'get_epic_status_summary',
                'passed': False,
                'message': f'Get epic status failed: {e}'
            })

    async def _test_task_tools(self):
        """Test task analysis tools."""
        logger.info("Testing task analysis tools...")

        # Mock server with client
        server = YouTrackMCPServer()
        server.client = MockYouTrackClient(YouTrackConfig(
            url='https://test.youtrack.cloud',
            token='test-token-123'
        ))

        # Test get_task_details
        try:
            result = await server.get_task_details("PROJ-101")
            self.results.append({
                'test': 'get_task_details',
                'passed': 'Implement login form' in result,
                'message': f'Task details result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'get_task_details',
                'passed': False,
                'message': f'Get task details failed: {e}'
            })

        # Test get_task_comments
        try:
            result = await server.get_task_comments("PROJ-101")
            self.results.append({
                'test': 'get_task_comments',
                'passed': 'Alice Smith' in result,
                'message': f'Task comments result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'get_task_comments',
                'passed': False,
                'message': f'Get task comments failed: {e}'
            })

        # Test get_task_history
        try:
            result = await server.get_task_history("PROJ-101")
            self.results.append({
                'test': 'get_task_history',
                'passed': 'State' in result,
                'message': f'Task history result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'get_task_history',
                'passed': False,
                'message': f'Get task history failed: {e}'
            })

        # Test analyze_task_activity
        try:
            result = await server.analyze_task_activity("PROJ-101")
            self.results.append({
                'test': 'analyze_task_activity',
                'passed': 'Activity Analysis' in result,
                'message': f'Task activity result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'analyze_task_activity',
                'passed': False,
                'message': f'Analyze task activity failed: {e}'
            })

    async def _test_analytics_tools(self):
        """Test analytics tools."""
        logger.info("Testing analytics tools...")

        # Mock server with client
        server = YouTrackMCPServer()
        server.client = MockYouTrackClient(YouTrackConfig(
            url='https://test.youtrack.cloud',
            token='test-token-123'
        ))

        # Test analyze_epic_progress
        try:
            result = await server.analyze_epic_progress("PROJ-100")
            self.results.append({
                'test': 'analyze_epic_progress',
                'passed': 'Progress Analysis' in result,
                'message': f'Epic progress analysis result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'analyze_epic_progress',
                'passed': False,
                'message': f'Analyze epic progress failed: {e}'
            })

        # Test generate_epic_report
        try:
            result = await server.generate_epic_report("PROJ-100")
            self.results.append({
                'test': 'generate_epic_report',
                'passed': 'COMPREHENSIVE REPORT' in result,
                'message': f'Epic report result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'generate_epic_report',
                'passed': False,
                'message': f'Generate epic report failed: {e}'
            })

    async def _test_resources_and_prompts(self):
        """Test resources and prompts."""
        logger.info("Testing resources and prompts...")

        # Mock server with client
        server = YouTrackMCPServer()
        server.client = MockYouTrackClient(YouTrackConfig(
            url='https://test.youtrack.cloud',
            token='test-token-123'
        ))

        # Test epic resource
        try:
            result = await server.get_epic_resource("youtrack://epic/PROJ-100")
            parsed = json.loads(result)
            self.results.append({
                'test': 'epic_resource',
                'passed': 'epic' in parsed and 'progress' in parsed,
                'message': f'Epic resource result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'epic_resource',
                'passed': False,
                'message': f'Epic resource failed: {e}'
            })

        # Test epic analysis prompt
        try:
            result = await server.epic_analysis_prompt("PROJ-100")
            self.results.append({
                'test': 'epic_analysis_prompt',
                'passed': 'PROJ-100' in result and 'analyze' in result.lower(),
                'message': f'Epic analysis prompt result: {len(result)} chars'
            })
        except Exception as e:
            self.results.append({
                'test': 'epic_analysis_prompt',
                'passed': False,
                'message': f'Epic analysis prompt failed: {e}'
            })

    async def _test_performance(self):
        """Test performance benchmarks."""
        logger.info("Running performance benchmarks...")

        # Mock server with client
        server = YouTrackMCPServer()
        server.client = MockYouTrackClient(YouTrackConfig(
            url='https://test.youtrack.cloud',
            token='test-token-123'
        ))

        # Benchmark epic search
        start_time = time.time()
        try:
            await server.find_epic("authentication")
            duration = time.time() - start_time
            self.results.append({
                'test': 'performance_epic_search',
                'passed': duration < 1.0,
                'message': f'Epic search took {duration:.3f}s'
            })
        except Exception as e:
            self.results.append({
                'test': 'performance_epic_search',
                'passed': False,
                'message': f'Performance test failed: {e}'
            })

        # Benchmark report generation
        start_time = time.time()
        try:
            await server.generate_epic_report("PROJ-100")
            duration = time.time() - start_time
            self.results.append({
                'test': 'performance_report_generation',
                'passed': duration < 2.0,
                'message': f'Report generation took {duration:.3f}s'
            })
        except Exception as e:
            self.results.append({
                'test': 'performance_report_generation',
                'passed': False,
                'message': f'Performance test failed: {e}'
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
    tester = YouTrackServerTester()
    summary = await tester.run_all_tests()

    print("\n" + "="*60)
    print("YOUTRACK MCP SERVER TEST RESULTS")
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
        print("\n🎉 All tests passed! YouTrack MCP Server is working correctly.")
    else:
        print(f"\n⚠️  {summary['failed_tests']} test(s) failed. Please review the issues above.")

    return 0 if summary['failed_tests'] == 0 else 1

if __name__ == "__main__":
    exit(asyncio.run(main()))