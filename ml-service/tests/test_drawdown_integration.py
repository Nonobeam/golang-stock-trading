"""
Integration Test: Drawdown-Based Position Sizing

Tests portfolio drawdown calculation and position sizing adjustments.
"""

import unittest
from decimal import Decimal
from datetime import date
import sys
import os

# Add parent to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)) + '/..')

from position_sizing.kelly import PositionSizer
from position_sizing.drawdown_manager import DrawdownManager


class TestDrawdownIntegration(unittest.TestCase):
    """Test drawdown-based position sizing integration."""
    
    def setUp(self):
        """Set up test fixtures."""
        self.sizer = PositionSizer(base_fraction=0.10, max_allocation=0.20)
        self.prediction = {'p10': -0.01, 'p50': 0.03, 'p90': 0.08}
        self.horizon = 5
    
    def test_normal_sizing_no_drawdown(self):
        """Test normal position sizing when no drawdown."""
        # No drawdown: multiplier = 1.0
        size = self.sizer.calculate_size(
            self.prediction,
            horizon=self.horizon,
            drawdown_multiplier=1.0
        )
        
        # Prediction range = 0.08 - (-0.01) = 0.09 → confidence_multiplier = 1.0
        # horizon = 5 → horizon_multiplier = 1.0
        # Expected: 0.10 * 1.0 * 1.0 * 1.0 = 0.10 (10%)
        self.assertAlmostEqual(size, 0.10, places=2)
    
    def test_half_sizing_at_12_percent_drawdown(self):
        """Test position size reduced by 50% at -12% drawdown."""
        # Drawdown -12%: multiplier = 0.5
        size = self.sizer.calculate_size(
            self.prediction,
            horizon=self.horizon,
            drawdown_multiplier=0.5
        )
        
        # Expected: 0.10 * 1.0 * 1.0 * 0.5 = 0.05 (5%)
        self.assertAlmostEqual(size, 0.05, places=2)
    
    def test_zero_sizing_at_16_percent_drawdown(self):
        """Test trading stopped at -16% drawdown."""
        # Drawdown -16%: multiplier = 0.0
        size = self.sizer.calculate_size(
            self.prediction,
            horizon=self.horizon,
            drawdown_multiplier=0.0
        )
        
        # Expected: 0.10 * 1.0 * 1.0 * 0.0 = 0.0 (0%)
        self.assertEqual(size, 0.0)
    
    def test_drawdown_manager_multipliers(self):
        """Test DrawdownManager returns correct multipliers."""
        mgr = DrawdownManager()
        
        # Test threshold boundaries
        self.assertEqual(mgr.MULTIPLIER_NORMAL, 1.0)
        self.assertEqual(mgr.MULTIPLIER_HALF, 0.5)
        self.assertEqual(mgr.MULTIPLIER_STOP, 0.0)
        
        self.assertEqual(mgr.THRESHOLD_WARNING, -0.10)
        self.assertEqual(mgr.THRESHOLD_CRITICAL, -0.15)
    
    def test_calculate_shares_with_drawdown(self):
        """Test share calculation with drawdown adjustment."""
        account_value = 100_000_000  # 100M VND
        price = 50_000  # 50k VND
        
        # Normal sizing
        shares_normal = self.sizer.calculate_shares(
            account_value, price, self.prediction, self.horizon, drawdown_multiplier=1.0
        )
        
        # Half sizing
        shares_half = self.sizer.calculate_shares(
            account_value, price, self.prediction, self.horizon, drawdown_multiplier=0.5
        )
        
        # Shares should be halved
        # Normal: 100M * 0.10 / 50k = 200 shares
        # Half: 100M * 0.05 / 50k = 100 shares
        self.assertEqual(shares_normal, 200)
        self.assertEqual(shares_half, 100)
    
    def test_high_confidence_with_drawdown(self):
        """Test high confidence prediction with drawdown adjustment."""
        # Narrow prediction range → high confidence
        high_conf_prediction = {'p10': 0.02, 'p50': 0.04, 'p90': 0.06}
        # Range = 0.04 → confidence_multiplier = 1.5
        
        # Normal sizing
        size_normal = self.sizer.calculate_size(
            high_conf_prediction,
            horizon=5,
            drawdown_multiplier=1.0
        )
        
        # With drawdown
        size_drawdown = self.sizer.calculate_size(
            high_conf_prediction,
            horizon=5,
            drawdown_multiplier=0.5
        )
        
        # Expected normal: 0.10 * 1.5 * 1.0 * 1.0 = 0.15
        # Expected drawdown: 0.10 * 1.5 * 1.0 * 0.5 = 0.075
        self.assertAlmostEqual(size_normal, 0.15, places=2)
        self.assertAlmostEqual(size_drawdown, 0.075, places=3)


if __name__ == '__main__':
    print("Running Drawdown Integration Tests...")
    unittest.main(verbosity=2)
