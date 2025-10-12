"""
Tests for MCP Security Framework

This module tests all security features including sandboxing, permissions,
audit logging, rate limiting, and security scanning.
"""

import unittest
import tempfile
import time
import os
from pathlib import Path
from unittest.mock import Mock, patch, MagicMock
from datetime import datetime, timedelta

import sys
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from src.infrastructure.mcp.security import (
    MCPSecurityManager, SecurityPolicy, ToolPermission, AuditEntry,
    MCPSandbox, PermissionManager, AuditLogger, RateLimiter,
    SecurityScanner, SecurityViolation, ResourceLimitExceeded
)


class TestMCPSecurity(unittest.TestCase):
    """Test MCP security framework"""

    def setUp(self):
        """Set up test environment"""
        self.policy = SecurityPolicy(
            allowed_tools={'safe_tool'},
            blocked_tools={'dangerous_tool'},
            max_execution_time=5.0,
            max_memory_mb=256,
            sandbox_enabled=True,
            audit_logging=True,
            rate_limit_per_minute=10,
            quota_per_hour=100
        )

        self.security_manager = MCPSecurityManager(self.policy)

    def test_security_policy_creation(self):
        """Test security policy creation"""
        self.assertTrue(self.policy.sandbox_enabled)
        self.assertEqual(self.policy.max_execution_time, 5.0)
        self.assertEqual(self.policy.rate_limit_per_minute, 10)

    def test_permission_manager(self):
        """Test permission management system"""
        permission = ToolPermission(
            tool_name='test_tool',
            allowed_users={'user1', 'user2'},
            allowed_roles={'admin'},
            requires_approval=True
        )

        self.security_manager.permission_manager.add_permission(permission)

        # Test user permission
        self.assertTrue(self.security_manager.permission_manager.check_permission(
            'user1', set(), 'test_tool'
        ))

        # Test role permission
        self.assertTrue(self.security_manager.permission_manager.check_permission(
            'user3', {'admin'}, 'test_tool'
        ))

        # Test denied access
        self.assertFalse(self.security_manager.permission_manager.check_permission(
            'user3', {'guest'}, 'test_tool'
        ))

        # Test approval requirement
        self.assertTrue(self.security_manager.permission_manager.requires_approval('test_tool'))

    def test_rate_limiter(self):
        """Test rate limiting functionality"""
        rate_limiter = RateLimiter()

        # Test initial request should pass
        self.assertTrue(rate_limiter.check_rate_limit('user1', 'tool1', self.policy))

        # Record requests up to limit
        for _ in range(self.policy.rate_limit_per_minute):
            rate_limiter.record_request('user1', 'tool1')

        # Next request should be blocked
        self.assertFalse(rate_limiter.check_rate_limit('user1', 'tool1', self.policy))

    def test_audit_logger(self):
        """Test audit logging system"""
        with tempfile.NamedTemporaryFile(delete=False) as temp_file:
            audit_logger = AuditLogger(temp_file.name)

            entry = AuditEntry(
                timestamp=datetime.now(),
                user_id='test_user',
                tool_name='test_tool',
                server_name='test_server',
                action='execute',
                parameters={'param': 'value'},
                result_status='success',
                execution_time=1.5,
                resource_usage={'memory_mb': 100, 'cpu_percent': 25},
                security_violations=[]
            )

            audit_logger.log_execution(entry)

            # Check entry was stored
            self.assertEqual(len(audit_logger.entries), 1)
            self.assertEqual(audit_logger.entries[0].user_id, 'test_user')

            # Test recent entries retrieval
            recent = audit_logger.get_recent_entries(1)
            self.assertEqual(len(recent), 1)

            # Cleanup
            os.unlink(temp_file.name)

    def test_security_scanner(self):
        """Test security scanner"""
        scanner = SecurityScanner()

        # Create temporary file with vulnerable code
        with tempfile.TemporaryDirectory() as temp_dir:
            vuln_file = Path(temp_dir) / "vulnerable.py"
            vuln_file.write_text("""
import os

def dangerous_function():
    os.system('rm -rf /')  # Dangerous!
    eval('print("hello")')  # Also dangerous!
""")

            vulnerabilities = scanner.scan_server_code(temp_dir)

            # Should detect vulnerabilities
            self.assertGreater(len(vulnerabilities), 0)
            self.assertIn(str(vuln_file), vulnerabilities)

            # Generate security report
            report = scanner.generate_security_report([temp_dir])
            self.assertIn('vulnerabilities', report)
            self.assertIn('risk_level', report)

    def test_sandbox_execution(self):
        """Test sandboxed execution"""
        sandbox = MCPSandbox(self.policy)

        with sandbox.execute_sandboxed('test_tool', {'param': 'value'}) as context:
            # Test sandbox context
            self.assertTrue(context.sandbox_id.startswith('test_tool_'))
            self.assertTrue(os.path.exists(context.temp_dir))

            # Test network access control
            self.assertTrue(context.check_network_access('allowed.com'))

            # Test safe file path
            safe_path = context.get_safe_file_path('../../../etc/passwd')
            self.assertTrue(safe_path.startswith(context.temp_dir))
            self.assertEqual(os.path.basename(safe_path), 'passwd')

    @patch('src.infrastructure.mcp.security.ResourceMonitor')
    def test_secure_tool_execution(self, mock_monitor):
        """Test secure tool execution with all security controls"""
        # Setup permission
        permission = ToolPermission(
            tool_name='test_tool',
            allowed_users={'test_user'}
        )
        self.security_manager.permission_manager.add_permission(permission)

        # Execute tool securely
        result = self.security_manager.execute_secure_tool(
            user_id='test_user',
            user_roles={'user'},
            server_name='test_server',
            tool_name='test_tool',
            parameters={'param': 'value'}
        )

        # Verify execution
        self.assertEqual(result['status'], 'success')

    def test_security_violations(self):
        """Test security violation handling"""
        # Test permission violation
        with self.assertRaises(SecurityViolation):
            self.security_manager.execute_secure_tool(
                user_id='unauthorized_user',
                user_roles={'guest'},
                server_name='test_server',
                tool_name='blocked_tool',
                parameters={}
            )

    def test_security_status(self):
        """Test security status reporting"""
        status = self.security_manager.get_security_status()

        self.assertIn('policy_active', status)
        self.assertIn('sandbox_enabled', status)
        self.assertIn('audit_logging', status)
        self.assertIn('recent_violations_24h', status)
        self.assertTrue(status['policy_active'])
        self.assertTrue(status['sandbox_enabled'])

    def test_role_hierarchy(self):
        """Test role hierarchy in permission management"""
        permission = ToolPermission(
            tool_name='admin_tool',
            allowed_roles={'admin'}
        )
        self.security_manager.permission_manager.add_permission(permission)

        # Admin should have access
        self.assertTrue(self.security_manager.permission_manager.check_permission(
            'user1', {'admin'}, 'admin_tool'
        ))

        # User should not have access
        self.assertFalse(self.security_manager.permission_manager.check_permission(
            'user1', {'user'}, 'admin_tool'
        ))

    def test_resource_limit_exceeded(self):
        """Test resource limit exceeded handling"""
        # This would require actual resource monitoring, so we'll mock it
        with patch('src.infrastructure.mcp.security.ResourceMonitor') as mock_monitor:
            mock_monitor.return_value.limit_resources.side_effect = ResourceLimitExceeded("Memory limit exceeded")

            sandbox = MCPSandbox(self.policy)

            with self.assertRaises(SecurityViolation):
                with sandbox.execute_sandboxed('heavy_tool', {}):
                    pass

    def test_audit_entry_serialization(self):
        """Test audit entry can be properly serialized"""
        entry = AuditEntry(
            timestamp=datetime.now(),
            user_id='test_user',
            tool_name='test_tool',
            server_name='test_server',
            action='execute',
            parameters={'complex': {'nested': 'data'}},
            result_status='success',
            execution_time=1.5,
            resource_usage={'memory_mb': 100},
            security_violations=['violation1']
        )

        # Should be able to create entry without issues
        self.assertEqual(entry.user_id, 'test_user')
        self.assertIsInstance(entry.timestamp, datetime)

    def test_comprehensive_security_workflow(self):
        """Test complete security workflow"""
        # Setup permissions
        permission = ToolPermission(
            tool_name='workflow_tool',
            allowed_users={'workflow_user'},
            requires_approval=False
        )
        self.security_manager.permission_manager.add_permission(permission)

        # Execute multiple requests to test rate limiting
        success_count = 0
        for i in range(15):  # Exceed rate limit
            try:
                result = self.security_manager.execute_secure_tool(
                    user_id='workflow_user',
                    user_roles={'user'},
                    server_name='workflow_server',
                    tool_name='workflow_tool',
                    parameters={'request': i}
                )
                success_count += 1
            except SecurityViolation:
                break

        # Should hit rate limit before 15 requests
        self.assertLess(success_count, 15)

        # Check audit log has entries
        self.assertGreater(len(self.security_manager.audit_logger.entries), 0)

        # Get security status
        status = self.security_manager.get_security_status()
        self.assertIn('recent_violations_24h', status)


if __name__ == '__main__':
    unittest.main()