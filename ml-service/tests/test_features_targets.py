
import pytest
import pandas as pd
import numpy as np
import sys
import os

# Add parent directory to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from features.calculator import calculate_targets

def test_calculate_targets():
    # Create sample data
    # We need at least 11 rows to get a valid 10d target for the first row
    prices = [100.0 * (1.01 ** i) for i in range(20)] # 1% daily increase
    df = pd.DataFrame({
        'close': prices,
        'date': pd.date_range(start='2023-01-01', periods=20)
    })
    
    # Calculate targets
    df = calculate_targets(df)
    
    # Check 1-day target
    # target_return_1d at index 0 should be (price[1] - price[0])/price[0]
    expected_1d = (prices[1] - prices[0]) / prices[0]
    assert np.isclose(df.iloc[0]['target_return_1d'], expected_1d)
    
    # Check 5-day target
    # target_return_5d at index 0 should be (price[5] - price[0])/price[0]
    expected_5d = (prices[5] - prices[0]) / prices[0]
    assert np.isclose(df.iloc[0]['target_return_5d'], expected_5d)
    
    # Check 10-day target
    # target_return_10d at index 0 should be (price[10] - price[0])/price[0]
    expected_10d = (prices[10] - prices[0]) / prices[0]
    assert np.isclose(df.iloc[0]['target_return_10d'], expected_10d)
    
    # Check shift behavior (last rows should be NaN)
    assert pd.isna(df.iloc[-1]['target_return_1d'])
    assert pd.isna(df.iloc[-5]['target_return_5d'])
    assert pd.isna(df.iloc[-10]['target_return_10d'])

def test_calculate_targets_ordering():
    """Verify that in an uptrend, returns increase with horizon"""
    prices = [100.0 + i for i in range(20)] # Linear increase
    df = pd.DataFrame({'close': prices})
    
    df = calculate_targets(df)
    
    # For index 0: 1d return < 5d return < 10d return
    r1 = df.iloc[0]['target_return_1d']
    r5 = df.iloc[0]['target_return_5d']
    r10 = df.iloc[0]['target_return_10d']
    
    assert r1 > 0
    assert r5 > r1
    assert r10 > r5

if __name__ == "__main__":
    test_calculate_targets()
    test_calculate_targets_ordering()
    print("All tests passed!")
