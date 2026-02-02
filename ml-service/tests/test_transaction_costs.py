"""
Unit Tests for Transaction Cost Module

Tests Vietnamese market fee calculations, profit thresholds,
and fee-adjusted returns.

Author: ML Trading System
Created: 2026-02-02
"""

import pytest
from decimal import Decimal
from ml_service.validation.transaction_costs import (
    calculate_fees,
    calculate_round_trip_cost,
    get_minimum_profit_threshold,
    calculate_fee_adjusted_return,
    is_profitable_after_fees,
    calculate_profit_factor,
    BROKERAGE_FEE_RATE,
    TAX_RATE,
    ROUND_TRIP_COST
)


class TestCalculateFees:
    """Test basic fee calculations"""
    
    def test_buy_fees(self):
        """Test buy transaction (no tax)"""
        result = calculate_fees(10_000_000, is_sell=False)
        
        assert result['brokerage_fee'] == 15_000  # 0.15%
        assert result['tax'] == 0
        assert result['total_fee'] == 15_000
    
    def test_sell_fees(self):
        """Test sell transaction (includes tax)"""
        result = calculate_fees(10_000_000, is_sell=True)
        
        assert result['brokerage_fee'] == 15_000  # 0.15%
        assert result['tax'] == 10_000  # 0.1%
        assert result['total_fee'] == 25_000  # 0.25%
    
    def test_small_transaction(self):
        """Test minimum fee calculations"""
        result = calculate_fees(1_000_000, is_sell=False)
        
        assert result['brokerage_fee'] == 1_500
        assert result['total_fee'] == 1_500


class TestRoundTripCost:
    """Test round-trip fee calculations"""
    
    def test_profitable_trade(self):
        """Test trade with profit after fees"""
        result = calculate_round_trip_cost(
            entry_price=100_000,
            exit_price=105_000,
            shares=100
        )
        
        # Buy value: 10M, Sell value: 10.5M
        assert result['buy_fees'] == pytest.approx(15_000, rel=1)  # 0.15% of 10M
        assert result['sell_fees'] == pytest.approx(26_250, rel=1)  # 0.25% of 10.5M
        assert result['total_fees'] == pytest.approx(41_250, rel=1)
        assert result['gross_profit'] == 500_000
        assert result['net_profit'] == pytest.approx(458_750, rel=1)
        assert result['fee_drag'] == pytest.approx(0.004125, rel=0.01)  # ~0.4%
    
    def test_losing_trade(self):
        """Test trade with loss plus fees"""
        result = calculate_round_trip_cost(
            entry_price=100_000,
            exit_price=95_000,
            shares=100
        )
        
        assert result['gross_profit'] < 0
        assert result['net_profit'] < result['gross_profit']  # Fees worsen loss
        assert result['total_fees'] > 0


class TestMinimumProfitThreshold:
    """Test minimum profit thresholds by horizon"""
    
    def test_1day_threshold(self):
        """1-day horizon requires 1.0% profit"""
        threshold = get_minimum_profit_threshold(1)
        assert threshold == 0.01
    
    def test_5day_threshold(self):
        """5-day horizon requires 1.5% profit"""
        threshold = get_minimum_profit_threshold(5)
        assert threshold == 0.015
    
    def test_10day_threshold(self):
        """10-day horizon requires 2.0% profit"""
        threshold = get_minimum_profit_threshold(10)
        assert threshold == 0.02
    
    def test_unknown_horizon(self):
        """Unknown horizon defaults to 1.5%"""
        threshold = get_minimum_profit_threshold(7)
        assert threshold == 0.015


class TestFeeAdjustedReturn:
    """Test fee-adjusted return calculations"""
    
    def test_simplified_calculation(self):
        """Test without entry price (flat 0.4% subtraction)"""
        gross_return = 0.025  # 2.5%
        net_return = calculate_fee_adjusted_return(gross_return)
        
        assert net_return == pytest.approx(0.021, rel=0.01)  # 2.5% - 0.4% = 2.1%
    
    def test_exact_calculation(self):
        """Test with entry price (exact fee calculation)"""
        net_return = calculate_fee_adjusted_return(
            gross_return=0.025,
            entry_price=100_000,
            shares=100
        )
        
        # Should be close to 2.1% but slightly different due to exact calculation
        assert 0.020 < net_return < 0.022
    
    def test_negative_return(self):
        """Test loss worsened by fees"""
        net_return = calculate_fee_adjusted_return(-0.02)  # -2%
        
        assert net_return == pytest.approx(-0.024, rel=0.01)  # -2% - 0.4% = -2.4%


class TestProfitabilityCheck:
    """Test profitability validation"""
    
    def test_profitable_1day(self):
        """Test profitable 1-day prediction"""
        is_profitable, message = is_profitable_after_fees(0.015, horizon_days=1)
        
        assert is_profitable is True
        assert "exceeds" in message.lower()
    
    def test_unprofitable_1day(self):
        """Test unprofitable 1-day prediction"""
        is_profitable, message = is_profitable_after_fees(0.008, horizon_days=1)
        
        assert is_profitable is False
        assert "below" in message.lower()
    
    def test_borderline_5day(self):
        """Test borderline 5-day prediction"""
        is_profitable, _ = is_profitable_after_fees(0.0155, horizon_days=5)
        
        assert is_profitable is True  # Just above 1.5% threshold


class TestProfitFactor:
    """Test profit factor calculations"""
    
    def test_profitable_strategy(self):
        """Test strategy with strong profit factor"""
        winning_trades = [0.025, 0.018, 0.031, 0.022]  # 4 winners
        losing_trades = [-0.012, -0.008, -0.015]  # 3 losers
        
        result = calculate_profit_factor(winning_trades, losing_trades)
        
        assert result['num_winners'] == 4
        assert result['num_losers'] == 3
        assert result['gross_profit_factor'] > 1.5
        assert result['net_profit_factor'] > 1.0  # Still profitable after fees
        assert result['net_profit_factor'] < result['gross_profit_factor']  # Fees reduce PF
    
    def test_losing_strategy(self):
        """Test strategy with poor profit factor"""
        winning_trades = [0.008, 0.005]
        losing_trades = [-0.025, -0.018, -0.022]
        
        result = calculate_profit_factor(winning_trades, losing_trades)
        
        assert result['net_profit_factor'] < 1.0  # Losing strategy
    
    def test_no_losers(self):
        """Test perfect strategy (no losses)"""
        winning_trades = [0.025, 0.018]
        losing_trades = []
        
        result = calculate_profit_factor(winning_trades, losing_trades)
        
        assert result['net_profit_factor'] == float('inf')  # No losses = infinite PF


class TestFeeConstants:
    """Verify Vietnamese market fee constants"""
    
    def test_brokerage_rate(self):
        """Brokerage should be 0.15%"""
        assert BROKERAGE_FEE_RATE == Decimal('0.0015')
    
    def test_tax_rate(self):
        """Tax should be 0.1%"""
        assert TAX_RATE == Decimal('0.001')
    
    def test_round_trip_cost(self):
        """Total round-trip should be 0.4%"""
        assert ROUND_TRIP_COST == Decimal('0.004')


if __name__ == '__main__':
    pytest.main([__file__, '-v'])
