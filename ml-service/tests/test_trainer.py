"""
Model training logic tests
"""
import pytest
import numpy as np
import pandas as pd
from models.trainer import (
    split_train_val_test,
    train_quantile_model,
    calculate_quantile_calibration,
    train_all_quantiles
)
from data.loader import get_feature_columns
import config

@pytest.fixture
def mock_training_data():
    """Create mock training data"""
    np.random.seed(42)
    n_samples = 400
    
    feature_cols = get_feature_columns()
    n_features = len(feature_cols)
    
    # Create synthetic data
    X = np.random.randn(n_samples, n_features)
    y = 0.01 * X[:, 0] + 0.005 * X[:, 1] + np.random.normal(0, 0.02, n_samples)
    
    dates = pd.date_range('2022-01-01', periods=n_samples, freq='D')
    
    df = pd.DataFrame(X, columns=feature_cols)
    df['date'] = dates
    df['target_return'] = y
    
    return df

def test_split_train_val_test_sizes(mock_training_data):
    """Test data split produces correct sizes"""
    df = mock_training_data
    
    train_df, val_df, test_df = split_train_val_test(df, 252, 80, 20)
    
    assert len(train_df) == 252
    assert len(val_df) == 80
    assert len(test_df) == 20

def test_split_train_val_test_chronological(mock_training_data):
    """Test split maintains chronological order"""
    df = mock_training_data
    
    train_df, val_df, test_df = split_train_val_test(df, 252, 80, 20)
    
    # Training should be earliest
    assert train_df['date'].max() < val_df['date'].min()
    
    # Validation should be middle
    assert val_df['date'].max() < test_df['date'].min()

def test_split_insufficient_data():
    """Test split raises error with insufficient data"""
    df = pd.DataFrame({
        'date': pd.date_range('2022-01-01', periods=100, freq='D'),
        'target_return': np.random.randn(100)
    })
    
    with pytest.raises(ValueError):
        split_train_val_test(df, 252, 80, 20)

def test_train_quantile_model_returns_valid(mock_training_data):
    """Test training returns valid model"""
    df = mock_training_data
    train_df, val_df, test_df = split_train_val_test(df, 252, 80, 20)
    
    feature_cols = get_feature_columns()
    X_train = train_df[feature_cols].values
    y_train = train_df['target_return'].values
    X_val = val_df[feature_cols].values
    y_val = val_df['target_return'].values
    
    model, mae = train_quantile_model(
        X_train, y_train, X_val, y_val, 
        0.50, config.HYPERPARAMETERS
    )
    
    assert model is not None
    assert isinstance(mae, float)
    assert mae >= 0

def test_quantile_calibration(mock_training_data):
    """Test quantile calibration calculation"""
    df = mock_training_data
    train_df, val_df, test_df = split_train_val_test(df, 252, 80, 20)
    
    feature_cols = get_feature_columns()
    X_train = train_df[feature_cols].values
    y_train = train_df['target_return'].values
    X_test = test_df[feature_cols].values
    y_test = test_df['target_return'].values
    
    # Train p10 model
    model, _ = train_quantile_model(
        X_train, y_train, X_test, y_test,
        0.10, config.HYPERPARAMETERS
    )
    
    # Check calibration
    coverage, expected = calculate_quantile_calibration(model, X_test, y_test, 0.10)
    
    assert 0 <= coverage <= 1
    assert expected == 0.10

def test_train_all_quantiles_returns_three_models(mock_training_data):
    """Test training all quantiles returns 3 models for 1 horizon"""
    df = mock_training_data
    feature_cols = get_feature_columns()
    
    models, metrics = train_all_quantiles(
        df, feature_cols, config.HYPERPARAMETERS,
        252, 80, 20, horizons=[1]
    )
    
    assert len(models) == 3
    assert 'p10_1d' in models
    assert 'p50_1d' in models
    assert 'p90_1d' in models

def test_quantile_ordering(mock_training_data):
    """Test p10 < p50 < p90 ordering"""
    df = mock_training_data
    feature_cols = get_feature_columns()
    
    models, metrics = train_all_quantiles(
        df, feature_cols, config.HYPERPARAMETERS,
        252, 80, 20, horizons=[1]
    )
    
    # Check ordering flag
    assert metrics['quantile_ordering_valid']['1d'] == True

def test_model_metrics_present(mock_training_data):
    """Test all required metrics are calculated"""
    df = mock_training_data
    feature_cols = get_feature_columns()
    
    models, metrics = train_all_quantiles(
        df, feature_cols, config.HYPERPARAMETERS,
        252, 80, 20, horizons=[1]
    )
    
    required_metrics = ['val_mae', 'test_mae', 'coverage', 'expected_coverage', 'horizon']
    
    for quantile_name in ['p10_1d', 'p50_1d', 'p90_1d']:
        for metric in required_metrics:
            assert metric in metrics['quantiles'][quantile_name]

def test_train_multi_horizon(mock_training_data):
    """Test training with multiple horizons"""
    df = mock_training_data
    feature_cols = get_feature_columns()
    
    # Mock target_return_5d (copy of 1d for simplicity)
    df['target_return_5d'] = df['target_return']
    
    models, metrics = train_all_quantiles(
        df, feature_cols, config.HYPERPARAMETERS,
        252, 80, 20, horizons=[1, 5]
    )
    
    assert len(models) == 6 # 3 quantiles * 2 horizons
    assert 'p10_1d' in models
    assert 'p10_5d' in models
    
    assert '1d' in metrics['quantile_ordering_valid']
    assert '5d' in metrics['quantile_ordering_valid']

