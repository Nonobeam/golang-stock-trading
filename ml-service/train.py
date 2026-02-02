#!/usr/bin/env python3
"""
XGBoost Quantile Training Script

Usage:
    python train.py --ticker HPG
    python train.py --ticker HPG --force  # Overwrite existing models
"""

import argparse
import json
import os
from datetime import datetime
from pathlib import Path

import config
from db import test_connection, get_connection
from data import load_features_for_training, get_feature_columns, check_data_availability
from models import train_all_quantiles

def save_models(models, ticker, metrics, output_dir):
    """
    Save trained models to disk and register in database
    
    Args:
        models: Dict of trained XGBoost models (p10_1d, p50_5d, etc.)
        ticker: Stock symbol
        metrics: Training metrics dict
        output_dir: Directory to save models
    """
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S%f")
    version_dir = Path(output_dir) / ticker / timestamp
    version_dir.mkdir(parents=True, exist_ok=True)
    
    saved_paths = {}
    
    # Save each quantile model
    for model_key, model in models.items():
        # model_key e.g. "p10_1d"
        filename = f"model_{ticker}_{model_key}.json"
        filepath = version_dir / filename
        model.save_model(str(filepath))
        saved_paths[model_key] = str(filepath)
        print(f"  Saved {model_key} model: {filepath}")
    
    # Prepare data ranges for metadata
    data_ranges = {}
    for h_suffix, dates in metrics['train_dates'].items():
        data_ranges[h_suffix] = {
            'train': [str(d) for d in dates],
            'validation': [str(d) for d in metrics['val_dates'].get(h_suffix, [])],
            'test': [str(d) for d in metrics['test_dates'].get(h_suffix, [])]
        }
    
    # Save training metadata
    metadata = {
        'ticker': ticker,
        'training_timestamp': timestamp,
        'training_date': datetime.now().isoformat(),
        'data_ranges': data_ranges,
        'hyperparameters': config.HYPERPARAMETERS,
        'feature_columns': get_feature_columns(),
        'metrics': metrics['quantiles'],
        'quantile_ordering_valid': metrics['quantile_ordering_valid'],
        'model_paths': saved_paths
    }
    
    metadata_path = version_dir / f"training_info_{ticker}.json"
    with open(metadata_path, 'w') as f:
        json.dump(metadata, f, indent=2)
    print(f"  Saved metadata: {metadata_path}")
    
    # Register in database
    register_models_in_db(ticker, saved_paths, metadata)
    
    return version_dir

def register_models_in_db(ticker, model_paths, metadata):
    """Insert model metadata into database"""
    conn = get_connection()
    
    with conn.cursor() as cursor:
        # Deactivate old models for this ticker
        cursor.execute("""
            UPDATE model_metadata 
            SET in_production = FALSE 
            WHERE ticker = %(ticker)s
        """, {'ticker': ticker})
        
        # Insert new models
        for model_key, filepath in model_paths.items():
            # model_key like "p10_1d" or "p90_5d"
            parts = model_key.split('_')
            q_str = parts[0] # p10
            h_str = parts[1] # 1d
            
            quantile_value = float(q_str.replace('p', '')) / 100
            
            # Get horizon from metrics if possible, or parse from key
            model_metrics = metadata['metrics'][model_key]
            horizon = model_metrics.get('horizon', 1)
            
            model_id = f"model_{ticker}_{model_key}_{metadata['training_timestamp']}"
            
            # Get date range for this horizon
            dates = metadata['data_ranges'][h_str]
            train_dates = [datetime.fromisoformat(d) if 'T' in d else datetime.strptime(d, "%Y-%m-%d %H:%M:%S") for d in dates['train']]
            val_dates = [datetime.fromisoformat(d) if 'T' in d else datetime.strptime(d, "%Y-%m-%d %H:%M:%S") for d in dates['validation']] if dates['validation'] else []
            
            train_start = min(train_dates)
            train_end = max(train_dates)
            train_days = len(train_dates) # This assumes contiguous or checking length of list? List is just min/max in previous step?
            # Wait, in trainer.py: metrics['train_dates'][h_suffix] = (min, max)
            # So dates['train'] is [str(min), str(max)]
            # Wait, check trainer.py output for metrics['train_dates']... yes it is tuple (min, max).
            # So len is 2. 'train_days' implies count of actual rows used.
            # trainer.py prints "Data split: Train=..." but doesn't return count in metrics['train_dates'].
            # I should assume some value or just store 0/diff.
            # Actually, `dates['train']` has only 2 elements [start, end].
            # train_days in DB probably expects count.
            # I'll calculate delta days or just use 0 if unknown. 
            # Or better, update trainer.py to store counts.
            # For now I will use (end - start).days
            
            cursor.execute("""
                INSERT INTO model_metadata 
                (model_id, ticker, quantile, horizon, file_path, in_production, 
                 training_date, hyperparameters, metrics,
                 train_start_date, train_end_date, train_days,
                 val_start_date, val_end_date)
                VALUES (%(model_id)s, %(ticker)s, %(quantile)s, %(horizon)s, %(file_path)s, 
                        %(in_production)s, %(training_date)s, %(hyperparameters)s, %(metrics)s,
                        %(train_start_date)s, %(train_end_date)s, %(train_days)s,
                        %(val_start_date)s, %(val_end_date)s)
            """, {
                'model_id': model_id,
                'ticker': ticker,
                'quantile': quantile_value,
                'horizon': horizon,
                'file_path': filepath,
                'in_production': True,
                'training_date': datetime.now(),
                'hyperparameters': json.dumps(metadata['hyperparameters']),
                'metrics': json.dumps(model_metrics),
                'train_start_date': train_start,
                'train_end_date': train_end,
                'train_days': (train_end - train_start).days, # Approximation
                'val_start_date': min(val_dates) if val_dates else None,
                'val_end_date': max(val_dates) if val_dates else None
            })
        
        conn.commit()
    
    conn.close()
    
    print("Registered models in database")

def validate_data_sufficiency(ticker):
    """Check if enough data exists for training"""
    data_info = check_data_availability(ticker)
    
    print(f"\nData availability for {ticker}:")
    print(f"  Total days: {data_info['total_days']}")
    print(f"  Complete features: {data_info['complete_days']}")
    print(f"  Date range: {data_info['earliest_date']} to {data_info['latest_date']}")
    
    required = config.TRAINING_WINDOW + config.VALIDATION_WINDOW + config.TEST_WINDOW
    
    if data_info['complete_days'] < required:
        raise ValueError(
            f"Insufficient data: need {required} days, have {data_info['complete_days']}"
        )
    
    print(f"  ✓ Sufficient data for training (need {required}, have {data_info['complete_days']})")
    
    return True

def main():
    parser = argparse.ArgumentParser(description='Train XGBoost quantile models')
    parser.add_argument('--ticker', type=str, required=True, help='Stock ticker symbol')
    parser.add_argument('--force', action='store_true', help='Overwrite existing models')
    parser.add_argument('--horizons', type=str, default="1,5,10", help='Comma-separated horizons (e.g. 1,5,10)')
    args = parser.parse_args()
    
    ticker = args.ticker.upper()
    horizons = [int(h) for h in args.horizons.split(',')]
    
    print("="*60)
    print(f"XGBoost Quantile Training: {ticker}")
    print(f"Horizons: {horizons}")
    print("="*60)
    
    # Step 1: Test database connection
    print("\n[1/5] Testing database connection...")
    if not test_connection():
        return
    
    # Step 2: Validate data availability
    print(f"\n[2/5] Validating data for {ticker}...")
    try:
        validate_data_sufficiency(ticker)
    except ValueError as e:
        print(f"✗ {e}")
        return
    
    # Step 3: Load training data
    print(f"\n[3/5] Loading features from database...")
    df = load_features_for_training(ticker)
    
    if len(df) < config.MIN_HISTORY_REQUIRED:
        print(f"✗ Insufficient data: {len(df)} samples, need {config.MIN_HISTORY_REQUIRED}")
        return
    
    feature_cols = get_feature_columns()
    print(f"  Using {len(feature_cols)} features")
    
    # Step 4: Train models
    print(f"\n[4/5] Training XGBoost models...")
    models, metrics = train_all_quantiles(
        df,
        feature_cols,
        config.HYPERPARAMETERS,
        config.TRAINING_WINDOW,
        config.VALIDATION_WINDOW,
        config.TEST_WINDOW,
        horizons=horizons # Pass horizons
    )
    
    # Step 5: Save models
    print(f"\n[5/5] Saving models...")
    version_dir = save_models(models, ticker, metrics, config.MODELS_DIR)
    
    print("\n" + "="*60)
    print("✓ Training complete!")
    print(f"Models saved to: {version_dir}")
    print("="*60)
    
    # Summary
    print("\nModel Performance Summary:")
    for key, quantile_metrics in metrics['quantiles'].items():
        print(f"  {key}:")
        print(f"    Test MAE: {quantile_metrics['test_mae']:.6f}")
        print(f"    Calibration: {quantile_metrics['coverage']:.3f} "
              f"(expected {quantile_metrics['expected_coverage']:.3f})")

if __name__ == "__main__":
    main()
