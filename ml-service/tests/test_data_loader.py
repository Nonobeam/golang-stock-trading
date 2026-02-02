"""
Data loading and feature extraction tests
"""
import pytest
from data.loader import (
    load_features_for_training,
    get_feature_columns,
    check_data_availability
)

def test_load_features_returns_dataframe(sample_features):
    """Test loading features returns valid DataFrame"""
    ticker = sample_features
    df = load_features_for_training(ticker)
    
    assert df is not None
    assert len(df) > 0
    assert 'target_return' in df.columns

def test_load_features_has_target(sample_features):
    """Test target_return is calculated correctly"""
    ticker = sample_features
    df = load_features_for_training(ticker)
    
    # Target should be present
    assert 'target_return' in df.columns
    
    # No NaN in target (except possibly last row)
    assert df['target_return'].notna().sum() > len(df) - 2

def test_feature_columns_complete():
    """Test all required features are defined"""
    features = get_feature_columns()
    
    required = [
        'return_1d', 'sma_20', 'rsi_14', 
        'macd', 'volatility_20d', 'volume_ratio_20d'
    ]
    
    for req in required:
        assert req in features, f"Missing required feature: {req}"

def test_check_data_availability(sample_features):
    """Test data availability checker"""
    ticker = sample_features
    info = check_data_availability(ticker)
    
    assert info['total_days'] == 600
    assert info['complete_days'] >= 250  # At least half should be complete
    assert info['earliest_date'] is not None
    assert info['latest_date'] is not None

def test_load_features_filters_incomplete(sample_features):
    """Test only complete features are loaded"""
    ticker = sample_features
    df = load_features_for_training(ticker)
    
    # Should only include rows where features_complete = TRUE
    # First 200 days marked incomplete in fixture
    assert len(df) <= 400
