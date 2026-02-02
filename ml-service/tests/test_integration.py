"""
End-to-end integration tests
"""
import pytest
from datetime import datetime, timedelta
import json
from pathlib import Path
import tempfile
import shutil
from train import (
    validate_data_sufficiency,
    save_models,
    register_models_in_db
)
from data.loader import load_features_for_training, get_feature_columns
from models.trainer import train_all_quantiles
import config

def test_validate_data_sufficiency_passes(sample_features):
    """Test validation passes with sufficient data"""
    ticker = sample_features
    
    # Should not raise exception
    result = validate_data_sufficiency(ticker)
    assert result == True

def test_validate_data_sufficiency_fails_insufficient_data(db_connection, sample_ticker):
    """Test validation fails with insufficient data"""
    # Insert only 100 days (insufficient)
    cursor = db_connection.cursor()
    
    cursor.execute("DELETE FROM features WHERE ticker = %s", (sample_ticker,))
    
    start_date = datetime(2024, 1, 1)
    
    for i in range(100):
        current_date = start_date + timedelta(days=i)
        
        cursor.execute("""
            INSERT INTO features (
                ticker, date, 
                return_1d, return_5d, return_20d, return_60d,
                sma_5, sma_10, sma_20, sma_50, sma_200,
                ema_12, ema_26,
                rsi_14, macd, macd_signal, macd_hist,
                bb_upper, bb_middle, bb_lower,
                volatility_20d,
                volume_ratio_20d,
                atr_14,
                features_complete
            ) VALUES (
                %(ticker)s, %(date)s,
                0, 0, 0, 0,
                100, 100, 100, 100, 100,
                100, 100,
                50, 0, 0, 0,
                102, 100, 98,
                0.02,
                1.0,
                0.02,
                %(features_complete)s
            )
        """, {
            'ticker': sample_ticker,
            'date': current_date,
            'features_complete': True
        })
    
    db_connection.commit()
    cursor.close()
    
    with pytest.raises(ValueError):
        validate_data_sufficiency(sample_ticker)

def test_end_to_end_training(sample_features):
    """Test complete training workflow"""
    ticker = sample_features
    
    # Load data
    df = load_features_for_training(ticker)
    assert len(df) > 0
    
    # Train models
    feature_cols = get_feature_columns()
    models, metrics = train_all_quantiles(
        df, feature_cols, config.HYPERPARAMETERS,
        config.TRAINING_WINDOW,
        config.VALIDATION_WINDOW,
        config.TEST_WINDOW
    )
    
    # Verify models trained
    assert len(models) == 3
    assert all(model is not None for model in models.values())
    
    # Verify metrics
    assert 'quantiles' in metrics
    assert len(metrics['quantiles']) == 3

def test_save_models_creates_files(sample_features):
    """Test model saving creates all files"""
    ticker = sample_features
    
    # Train models
    df = load_features_for_training(ticker)
    feature_cols = get_feature_columns()
    models, metrics = train_all_quantiles(
        df, feature_cols, config.HYPERPARAMETERS,
        252, 80, 20
    ) # Defaults to horizons=[1]
    
    # Save to temporary directory
    with tempfile.TemporaryDirectory() as tmpdir:
        version_dir = save_models(models, ticker, metrics, tmpdir)
        
        # Check model files exist (now with horizon suffix)
        assert (version_dir / f"model_{ticker}_p10_1d.json").exists()
        assert (version_dir / f"model_{ticker}_p50_1d.json").exists()
        assert (version_dir / f"model_{ticker}_p90_1d.json").exists()
        
        # Check metadata exists
        metadata_file = version_dir / f"training_info_{ticker}.json"
        assert metadata_file.exists()
        
        # Verify metadata content
        with open(metadata_file) as f:
            metadata = json.load(f)
        
        assert metadata['ticker'] == ticker
        assert 'hyperparameters' in metadata
        assert 'metrics' in metadata
        assert len(metadata['model_paths']) == 3

def test_models_registered_in_database(sample_features, db_connection):
    """Test models are registered in database"""
    ticker = sample_features
    
    # Train and save
    df = load_features_for_training(ticker)
    feature_cols = get_feature_columns()
    models, metrics = train_all_quantiles(
        df, feature_cols, config.HYPERPARAMETERS,
        252, 80, 20
    )
    
    with tempfile.TemporaryDirectory() as tmpdir:
        save_models(models, ticker, metrics, tmpdir)
        
        # Check database
        cursor = db_connection.cursor()
        cursor.execute("""
            SELECT COUNT(*) as count
            FROM model_metadata
            WHERE ticker = %s AND in_production = TRUE
        """, (ticker,))
        
        result = cursor.fetchone()
        assert result['count'] == 3  # Three quantile models
        
        cursor.close()

def test_model_deactivation_on_retrain(sample_features, db_connection):
    """Test old models deactivated when retraining"""
    ticker = sample_features
    
    df = load_features_for_training(ticker)
    feature_cols = get_feature_columns()
    models, metrics = train_all_quantiles(
        df, feature_cols, config.HYPERPARAMETERS,
        252, 80, 20
    )
    
    with tempfile.TemporaryDirectory() as tmpdir:
        # First training
        save_models(models, ticker, metrics, tmpdir)
        
        cursor = db_connection.cursor()
        cursor.execute("""
            SELECT COUNT(*) as count
            FROM model_metadata
            WHERE ticker = %s AND in_production = TRUE
        """, (ticker,))
        assert cursor.fetchone()['count'] == 3
        
        # Second training (retrain)
        save_models(models, ticker, metrics, tmpdir)
        
        # Should still have only 3 active models
        cursor.execute("""
            SELECT COUNT(*) as count
            FROM model_metadata
            WHERE ticker = %s AND in_production = TRUE
        """, (ticker,))
        assert cursor.fetchone()['count'] == 3
        
        # But total should be 6 (3 old inactive + 3 new active)
        cursor.execute("""
            SELECT COUNT(*) as count
            FROM model_metadata
            WHERE ticker = %s
        """, (ticker,))
        assert cursor.fetchone()['count'] == 6
        
        cursor.close()

def test_prediction_on_new_data(sample_features):
    """Test making predictions with trained model"""
    import xgboost as xgb
    
    ticker = sample_features
    
    # Train
    df = load_features_for_training(ticker)
    feature_cols = get_feature_columns()
    models, metrics = train_all_quantiles(
        df, feature_cols, config.HYPERPARAMETERS,
        252, 80, 20
    )
    
    # Get latest data
    latest_features = df[feature_cols].iloc[-1:].values
    
    # Make predictions with all models
    dmatrix = xgb.DMatrix(latest_features)
    
    pred_p10 = models['p10_1d'].predict(dmatrix)[0]
    pred_p50 = models['p50_1d'].predict(dmatrix)[0]
    pred_p90 = models['p90_1d'].predict(dmatrix)[0]
    
    # Verify ordering
    assert pred_p10 <= pred_p50 <= pred_p90
    
    # Verify reasonable range (within -50% to +50%)
    assert -0.5 < pred_p10 < 0.5
    assert -0.5 < pred_p50 < 0.5
    assert -0.5 < pred_p90 < 0.5
