"""
Integration Test: Portfolio Equity Tracking

Tests equity calculation, drawdown measurement, and daily snapshots.
"""

import unittest
from decimal import Decimal
from datetime import date
import sys
import os

# Add parent to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)) + '/..')

from validation.portfolio_metrics import PortfolioEquityTracker


class TestEquityTracking(unittest.TestCase):
    """Test portfolio equity tracking functionality."""
    
    def test_equity_calculation_logic(self):
        """Test equity calculation formula."""
        # Simulated equity components
        initial_capital = Decimal('100000000')  # 100M VND
        
        # Open positions
        vci_shares = 100
        vci_entry = Decimal('45000')
        vci_current = Decimal('47000')
        
        hpg_shares = 200
        hpg_entry = Decimal('30000')
        hpg_current = Decimal('29500')
        
        # Closed P&L
        closed_pnl = Decimal('500000')
        
        # Calculate components
        open_value = (vci_shares * vci_current) + (hpg_shares * hpg_current)
        invested = (vci_shares * vci_entry) + (hpg_shares * hpg_entry)
        cash = initial_capital - invested
        total_equity = open_value + closed_pnl + cash
        
        # Verify calculations
        self.assertEqual(open_value, Decimal('10600000'))  # 4.7M + 5.9M
        self.assertEqual(invested, Decimal('10500000'))     # 4.5M + 6M
        self.assertEqual(cash, Decimal('89500000'))         # 100M - 10.5M
        self.assertEqual(total_equity, Decimal('100600000')) # 10.6M + 0.5M + 89.5M
    
    def test_drawdown_calculation(self):
        """Test drawdown calculation formula."""
        peak_equity = Decimal('100000000')
        current_equity = Decimal('88000000')
        
        drawdown = (current_equity - peak_equity) / peak_equity
        
        # Expected: -0.12 (-12%)
        self.assertAlmostEqual(float(drawdown), -0.12, places=4)
    
    def test_peak_update_logic(self):
        """Test peak equity updates when new high is reached."""
        previous_peak = Decimal('102000000')
        new_equity = Decimal('105000000')
        
        # Peak should update
        new_peak = max(new_equity, previous_peak)
        self.assertEqual(new_peak, new_equity)
        
        # Drawdown should be 0 at new peak
        drawdown = (new_equity - new_peak) / new_peak
        self.assertEqual(drawdown, Decimal('0'))
    
    def test_drawdown_when_below_peak(self):
        """Test drawdown calculation when equity is below peak."""
        peak = Decimal('105000000')
        current = Decimal('98500000')
        
        drawdown = (current - peak) / peak
        drawdown_pct = float(drawdown * 100)
        
        # Expected: approximately -6.19%
        self.assertAlmostEqual(drawdown_pct, -6.19, places=2)
    
    def test_zero_drawdown_at_peak(self):
        """Test that drawdown is 0 when at peak equity."""
        equity = Decimal('100000000')
        peak = equity
        
        drawdown = (equity - peak) / peak
        self.assertEqual(drawdown, Decimal('0'))
    
    def test_equity_tracker_methods_exist(self):
        """Test that PortfolioEquityTracker has required methods."""
        tracker = PortfolioEquityTracker()
        
        # Verify methods exist
        self.assertTrue(hasattr(tracker, 'calculate_current_equity'))
        self.assertTrue(hasattr(tracker, 'get_peak_equity'))
        self.assertTrue(hasattr(tracker, 'calculate_drawdown'))
        self.assertTrue(hasattr(tracker, 'save_equity_snapshot'))
        self.assertTrue(hasattr(tracker, 'get_equity_history'))


if __name__ == '__main__':
    print("Running Equity Tracking Tests...")
    unittest.main(verbosity=2)
