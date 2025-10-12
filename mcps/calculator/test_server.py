"""
Tests for Calculator MCP Server

Run with: pytest mcps/calculator/test_server.py
"""

import pytest
from mcps.calculator.server import CalculatorMCPServer


@pytest.fixture
def calculator_server():
    """Create a calculator server instance for testing."""
    return CalculatorMCPServer()


class TestAddition:
    """Tests for addition operation."""

    @pytest.mark.asyncio
    async def test_add_positive_numbers(self, calculator_server):
        """Test adding two positive numbers."""
        result = await calculator_server.add(5, 3)
        assert len(result) == 1
        assert "5 + 3 = 8" in result[0].text

    @pytest.mark.asyncio
    async def test_add_negative_numbers(self, calculator_server):
        """Test adding negative numbers."""
        result = await calculator_server.add(-5, -3)
        assert "-5 + -3 = -8" in result[0].text

    @pytest.mark.asyncio
    async def test_add_with_decimals(self, calculator_server):
        """Test adding decimal numbers."""
        result = await calculator_server.add(2.5, 3.7)
        assert "2.5 + 3.7 = 6.2" in result[0].text


class TestSubtraction:
    """Tests for subtraction operation."""

    @pytest.mark.asyncio
    async def test_subtract_positive_numbers(self, calculator_server):
        """Test subtracting positive numbers."""
        result = await calculator_server.subtract(10, 3)
        assert "10 - 3 = 7" in result[0].text

    @pytest.mark.asyncio
    async def test_subtract_negative_result(self, calculator_server):
        """Test subtraction resulting in negative number."""
        result = await calculator_server.subtract(3, 10)
        assert "3 - 10 = -7" in result[0].text


class TestMultiplication:
    """Tests for multiplication operation."""

    @pytest.mark.asyncio
    async def test_multiply_positive_numbers(self, calculator_server):
        """Test multiplying positive numbers."""
        result = await calculator_server.multiply(5, 3)
        assert "5 × 3 = 15" in result[0].text

    @pytest.mark.asyncio
    async def test_multiply_by_zero(self, calculator_server):
        """Test multiplying by zero."""
        result = await calculator_server.multiply(5, 0)
        assert "5 × 0 = 0" in result[0].text

    @pytest.mark.asyncio
    async def test_multiply_negative_numbers(self, calculator_server):
        """Test multiplying negative numbers."""
        result = await calculator_server.multiply(-5, -3)
        assert "-5 × -3 = 15" in result[0].text


class TestDivision:
    """Tests for division operation."""

    @pytest.mark.asyncio
    async def test_divide_positive_numbers(self, calculator_server):
        """Test dividing positive numbers."""
        result = await calculator_server.divide(10, 2)
        assert "10 ÷ 2 = 5" in result[0].text

    @pytest.mark.asyncio
    async def test_divide_by_zero(self, calculator_server):
        """Test division by zero error handling."""
        result = await calculator_server.divide(10, 0)
        assert "Error" in result[0].text
        assert "zero" in result[0].text.lower()

    @pytest.mark.asyncio
    async def test_divide_with_decimals(self, calculator_server):
        """Test division resulting in decimal."""
        result = await calculator_server.divide(10, 3)
        assert "10 ÷ 3 = 3.333" in result[0].text or "10 ÷ 3" in result[0].text


class TestPower:
    """Tests for power operation."""

    @pytest.mark.asyncio
    async def test_power_positive_exponent(self, calculator_server):
        """Test power with positive exponent."""
        result = await calculator_server.power(2, 3)
        assert "2^3 = 8" in result[0].text

    @pytest.mark.asyncio
    async def test_power_zero_exponent(self, calculator_server):
        """Test power with zero exponent."""
        result = await calculator_server.power(5, 0)
        assert "5^0 = 1" in result[0].text

    @pytest.mark.asyncio
    async def test_power_negative_exponent(self, calculator_server):
        """Test power with negative exponent."""
        result = await calculator_server.power(2, -2)
        assert "2^-2 = 0.25" in result[0].text


class TestStatistics:
    """Tests for server statistics."""

    @pytest.mark.asyncio
    async def test_initial_stats(self, calculator_server):
        """Test initial statistics."""
        result = await calculator_server.get_stats()
        assert "Operations performed: 0" in result[0].text

    @pytest.mark.asyncio
    async def test_stats_after_operations(self, calculator_server):
        """Test statistics after performing operations."""
        # Perform some operations
        await calculator_server.add(5, 3)
        await calculator_server.subtract(10, 2)
        await calculator_server.multiply(4, 3)

        # Check stats
        result = await calculator_server.get_stats()
        assert "Operations performed: 3" in result[0].text


class TestServerIntegration:
    """Integration tests for the calculator server."""

    @pytest.mark.asyncio
    async def test_multiple_operations_sequence(self, calculator_server):
        """Test performing multiple operations in sequence."""
        # Perform a series of operations
        await calculator_server.add(5, 3)
        await calculator_server.multiply(2, 4)
        await calculator_server.divide(10, 2)
        await calculator_server.power(2, 3)

        # Verify stats
        result = await calculator_server.get_stats()
        assert "Operations performed: 4" in result[0].text

    @pytest.mark.asyncio
    async def test_server_initialization(self, calculator_server):
        """Test that server initializes correctly."""
        assert calculator_server.server is not None
        assert calculator_server.operations_count == 0
