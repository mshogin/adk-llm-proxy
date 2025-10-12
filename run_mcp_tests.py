#!/usr/bin/env python3
"""
Test runner for MCP integration tests.

This script runs comprehensive tests for the MCP system, including unit tests
and integration tests with a real MCP server.
"""

import sys
import subprocess
import argparse
from pathlib import Path
import os


def run_command(cmd, cwd=None):
    """Run a command and return the result."""
    print(f"Running: {' '.join(cmd)}")
    result = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True)

    if result.stdout:
        print("STDOUT:", result.stdout)
    if result.stderr:
        print("STDERR:", result.stderr)

    return result.returncode == 0


def install_dependencies():
    """Install test dependencies."""
    print("Installing test dependencies...")

    # Install pytest if not available
    try:
        import pytest
        print("✅ pytest is available")
    except ImportError:
        print("Installing pytest...")
        if not run_command([sys.executable, "-m", "pip", "install", "pytest", "pytest-asyncio"]):
            print("❌ Failed to install pytest")
            return False

    return True


def run_unit_tests():
    """Run unit tests."""
    print("\n" + "="*60)
    print("RUNNING UNIT TESTS")
    print("="*60)

    test_files = [
        "tests/test_mcp_client.py",
        "tests/test_mcp_registry.py",
        "tests/test_mcp_discovery.py"
    ]

    all_passed = True

    for test_file in test_files:
        print(f"\n🧪 Running {test_file}...")

        if not Path(test_file).exists():
            print(f"⚠️  Test file not found: {test_file}")
            continue

        success = run_command([
            sys.executable, "-m", "pytest",
            test_file,
            "-v",
            "--tb=short"
        ])

        if success:
            print(f"✅ {test_file} passed")
        else:
            print(f"❌ {test_file} failed")
            all_passed = False

    return all_passed


def run_integration_tests():
    """Run integration tests with real MCP server."""
    print("\n" + "="*60)
    print("RUNNING INTEGRATION TESTS")
    print("="*60)

    # Check if test server exists
    test_server_path = "test_mcp_server.py"
    if not Path(test_server_path).exists():
        print(f"❌ Test MCP server not found: {test_server_path}")
        return False

    # Make sure test server is executable
    if not os.access(test_server_path, os.X_OK):
        os.chmod(test_server_path, 0o755)

    print("🧪 Running integration tests...")

    success = run_command([
        sys.executable, "-m", "pytest",
        "tests/test_mcp_integration.py",
        "-v",
        "--tb=short",
        "-m", "integration",
        "-s"  # Don't capture output for integration tests
    ])

    if success:
        print("✅ Integration tests passed")
    else:
        print("❌ Integration tests failed")
        print("\n💡 Integration test failures may be due to:")
        print("   - Missing MCP dependencies")
        print("   - Test server connection issues")
        print("   - Environment-specific problems")
        print("   - This is expected if MCP library is not installed")

    return success


def validate_test_server():
    """Validate that the test MCP server works."""
    print("\n" + "="*60)
    print("VALIDATING TEST MCP SERVER")
    print("="*60)

    test_server_path = "test_mcp_server.py"
    if not Path(test_server_path).exists():
        print(f"❌ Test server not found: {test_server_path}")
        return False

    print("🔍 Testing MCP server basic functionality...")

    # Test server can start and respond to basic JSON-RPC
    test_message = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "initialize",
        "params": {"protocolVersion": "2024-11-05"}
    }

    try:
        import json

        # Start server process
        proc = subprocess.Popen(
            [sys.executable, test_server_path],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )

        # Send test message
        stdout, stderr = proc.communicate(
            input=json.dumps(test_message) + "\n",
            timeout=5
        )

        if stderr:
            print(f"Server stderr: {stderr}")

        if stdout:
            try:
                response = json.loads(stdout.strip())
                if response.get("id") == 1 and "result" in response:
                    print("✅ Test server responds correctly")
                    return True
                else:
                    print(f"❌ Unexpected response: {response}")
            except json.JSONDecodeError as e:
                print(f"❌ Invalid JSON response: {e}")
                print(f"Raw output: {stdout}")
        else:
            print("❌ No response from server")

        return False

    except subprocess.TimeoutExpired:
        print("❌ Server test timed out")
        proc.kill()
        return False
    except Exception as e:
        print(f"❌ Server test failed: {e}")
        return False


def run_configuration_validation():
    """Validate MCP configuration components."""
    print("\n" + "="*60)
    print("VALIDATING MCP CONFIGURATION")
    print("="*60)

    try:
        # Test basic imports
        print("🔍 Testing MCP component imports...")

        from src.infrastructure.config.config import MCPServerConfig, Config
        print("✅ Configuration classes imported successfully")

        # Test config validation
        print("🔍 Testing configuration validation...")

        valid_config = MCPServerConfig(
            name="test-validation",
            transport="stdio",
            command="python",
            args=["test.py"]
        )

        assert valid_config.validate() == True
        print("✅ Valid configuration passes validation")

        # Test invalid config
        try:
            invalid_config = MCPServerConfig(
                name="test-invalid",
                transport="stdio"
                # Missing required command
            )
            invalid_config.validate()
            print("❌ Invalid configuration should have failed validation")
            return False
        except ValueError:
            print("✅ Invalid configuration properly rejected")

        # Test main config
        config = Config()
        print(f"✅ Main configuration loaded (MCP enabled: {config.ENABLE_MCP})")

        return True

    except ImportError as e:
        print(f"❌ Failed to import MCP components: {e}")
        return False
    except Exception as e:
        print(f"❌ Configuration validation failed: {e}")
        return False


def main():
    """Main test runner."""
    parser = argparse.ArgumentParser(description="Run MCP tests")
    parser.add_argument("--unit", action="store_true", help="Run only unit tests")
    parser.add_argument("--integration", action="store_true", help="Run only integration tests")
    parser.add_argument("--validate", action="store_true", help="Run only validation tests")
    parser.add_argument("--all", action="store_true", help="Run all tests (default)")

    args = parser.parse_args()

    # Default to all tests if no specific type selected
    if not any([args.unit, args.integration, args.validate]):
        args.all = True

    print("🚀 MCP Test Runner")
    print("="*60)

    # Install dependencies
    if not install_dependencies():
        print("❌ Failed to install dependencies")
        sys.exit(1)

    success = True

    # Run validation tests
    if args.all or args.validate:
        if not run_configuration_validation():
            success = False

        if not validate_test_server():
            success = False

    # Run unit tests
    if args.all or args.unit:
        if not run_unit_tests():
            success = False

    # Run integration tests
    if args.all or args.integration:
        if not run_integration_tests():
            success = False
            # Integration test failures are not critical for development
            print("⚠️  Integration test failures are not critical")

    print("\n" + "="*60)
    print("TEST SUMMARY")
    print("="*60)

    if success:
        print("✅ All tests passed successfully!")
        print("\n🎉 MCP implementation is ready!")
        print("\nNext steps:")
        print("  1. Install MCP dependencies: pip install -r requirements.txt")
        print("  2. Set up your config.yaml with MCP servers")
        print("  3. Start the ADK server: python main.py")
    else:
        print("❌ Some tests failed")
        print("\n🔧 This is expected during development")
        print("   Unit test failures indicate code issues")
        print("   Integration test failures may be environment-related")

    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()