import xgboost as xgb
import numpy as np
from sklearn.metrics import mean_absolute_error
import pandas as pd
import os

def split_train_val_test(df, train_size, val_size, test_size):
    """
    Time-series split: train, validation, test
    
    Args:
        df: Full dataset sorted by date
        train_size: Number of days for training
        val_size: Number of days for validation
        test_size: Number of days for testing
        
    Returns:
        train_df, val_df, test_df
    """
    total_needed = train_size + val_size + test_size
    
    if len(df) < total_needed:
        raise ValueError(f"Not enough data. Have {len(df)}, need {total_needed}")
    
    # Use most recent data
    df = df.iloc[-total_needed:]
    
    train_df = df.iloc[:train_size]
    val_df = df.iloc[train_size:train_size+val_size]
    test_df = df.iloc[train_size+val_size:]
    
    return train_df, val_df, test_df

def train_quantile_model(X_train, y_train, X_val, y_val, quantile, hyperparams):
    """
    Train single XGBoost quantile regression model
    
    Args:
        X_train: Training features
        y_train: Training targets
        X_val: Validation features
        y_val: Validation targets
        quantile: Quantile to predict (0.10, 0.50, 0.90)
        hyperparams: Model hyperparameters dict
        
    Returns:
        Trained model, validation MAE
    """
    print(f"  Training quantile {quantile} model...")
    
    # Create DMatrix for XGBoost
    dtrain = xgb.DMatrix(X_train, label=y_train)
    dval = xgb.DMatrix(X_val, label=y_val)
    
    # Configure quantile objective
    params = hyperparams.copy()
    params['objective'] = 'reg:quantileerror'
    params['quantile_alpha'] = quantile
    params['eval_metric'] = 'mae'
    
    # Extract n_estimators for num_boost_round (xgb.train doesn't accept n_estimators in params)
    num_boost_round = params.pop('n_estimators', 200)
    
    # Train with early stopping
    evals = [(dtrain, 'train'), (dval, 'val')]
    
    model = xgb.train(
        params,
        dtrain,
        num_boost_round=num_boost_round,
        evals=evals,
        early_stopping_rounds=50,
        verbose_eval=False
    )
    
    # Validate
    y_pred = model.predict(dval)
    mae = mean_absolute_error(y_val, y_pred)
    
    print(f"    Validation MAE: {mae:.6f}")
    
    return model, mae

def calculate_quantile_calibration(model, X, y, quantile):
    """
    Check if quantile model is well-calibrated
    
    For p10: ~10% of actual values should be below prediction
    For p90: ~10% of actual values should be above prediction
    """
    dtest = xgb.DMatrix(X)
    y_pred = model.predict(dtest)
    
    if quantile == 0.10:
        coverage = np.mean(y < y_pred)
        expected = 0.10
    elif quantile == 0.90:
        coverage = np.mean(y > y_pred)
        expected = 0.10
    else:  # median
        coverage = np.mean(y < y_pred)
        expected = 0.50
    
    return coverage, expected

def train_all_quantiles(df, feature_cols, hyperparams, train_size, val_size, test_size, horizons=[1]):
    """
    Train all three quantile models (p10, p50, p90) for each horizon.
    
    Returns:
        dict of models, dict of metrics
    """
    models = {}
    metrics = {
        'train_dates': {}, # per horizon
        'val_dates': {},
        'test_dates': {},
        'quantiles': {},
        'quantile_ordering_valid': {}
    }
    
    for horizon in horizons:
        print(f"\n--- Training Horizon: {horizon} days ---")
        
        # Determine target column
        # Try specific column first, fall back to generic 'target_return' for 1d if needed
        target_col = f'target_return_{horizon}d'
        if target_col not in df.columns:
            if horizon == 1 and 'target_return' in df.columns:
                target_col = 'target_return'
            else:
                print(f"  Skipping horizon {horizon}: column {target_col} not found")
                continue
                
        # Filter data for this horizon (drop NaNs in target)
        df_horizon = df.dropna(subset=[target_col]).copy()
        
        try:
            # Split data
            train_df, val_df, test_df = split_train_val_test(df_horizon, train_size, val_size, test_size)
            
            print(f"  Data split: Train={len(train_df)}, Val={len(val_df)}, Test={len(test_df)}")
            
            # Record date ranges
            h_suffix = f"{horizon}d"
            metrics['train_dates'][h_suffix] = (train_df['date'].min(), train_df['date'].max())
            metrics['val_dates'][h_suffix] = (val_df['date'].min(), val_df['date'].max())
            metrics['test_dates'][h_suffix] = (test_df['date'].min(), test_df['date'].max())
            
            X_train = train_df[feature_cols].values
            y_train = train_df[target_col].values
            X_val = val_df[feature_cols].values
            y_val = val_df[target_col].values
            X_test = test_df[feature_cols].values
            y_test = test_df[target_col].values
            
            horizon_models = {}
            
            # Train each quantile model
            for quantile in [0.10, 0.50, 0.90]:
                model, val_mae = train_quantile_model(
                    X_train, y_train, X_val, y_val, quantile, hyperparams
                )
                
                # Test set evaluation
                dtest = xgb.DMatrix(X_test)
                y_pred_test = model.predict(dtest)
                test_mae = mean_absolute_error(y_test, y_pred_test)
                
                # Calibration check
                coverage, expected = calculate_quantile_calibration(model, X_test, y_test, quantile)
                
                quantile_name = f"p{int(quantile*100)}_{horizon}d"
                models[quantile_name] = model
                horizon_models[f"p{int(quantile*100)}"] = model # For ordering check
                
                metrics['quantiles'][quantile_name] = {
                    'val_mae': float(val_mae),
                    'test_mae': float(test_mae),
                    'coverage': float(coverage),
                    'expected_coverage': float(expected),
                    'calibration_error': abs(coverage - expected),
                    'horizon': horizon
                }
                
                print(f"  {quantile_name} - Test MAE: {test_mae:.6f}, "
                      f"Coverage: {coverage:.2f} (expected {expected:.2f})")
            
            # Verify quantile ordering on test set for this horizon
            dtest = xgb.DMatrix(X_test)
            pred_p10 = horizon_models['p10'].predict(dtest)
            pred_p50 = horizon_models['p50'].predict(dtest)
            pred_p90 = horizon_models['p90'].predict(dtest)
            
            ordering_valid = np.all((pred_p10 <= pred_p50) & (pred_p50 <= pred_p90))
            metrics['quantile_ordering_valid'][h_suffix] = bool(ordering_valid)
            
            if not ordering_valid:
                print(f"  WARNING: Quantile ordering violated for horizon {horizon}!")
            else:
                print(f"  ✓ Quantile ordering verified for horizon {horizon}")
                
        except ValueError as e:
            print(f"  Error training horizon {horizon}: {e}")
            continue
    
    return models, metrics

class ModelTrainer:
    """
    Wrapper for model operations.
    Used by caching/prediction logic to load models consistently.
    """
    def __init__(self, ticker: str):
        self.ticker = ticker
        
    def load_model(self, file_path: str) -> xgb.Booster:
        """
        Load XGBoost model from file.
        
        Args:
            file_path: Path to model file
            
        Returns:
            XGBoost Booster object
        """
        if not os.path.exists(file_path):
            raise FileNotFoundError(f"Model file not found: {file_path}")
            
        model = xgb.Booster()
        model.load_model(file_path)
        return model

    def train_all_models(self, days=1000):
        from data.loader import load_features_for_training, get_feature_columns
        df = load_features_for_training(self.ticker)
        
        if len(df) < 200:
            raise ValueError(f"Not enough data for {self.ticker}, found {len(df)} rows")
            
        feature_cols = get_feature_columns()
        hyperparams = {'n_estimators': 200, 'max_depth': 5}
        
        actual_days = min(len(df), days)
        test_size = max(int(actual_days * 0.1), 1)
        val_size = max(int(actual_days * 0.1), 1)
        train_size = actual_days - val_size - test_size
        
        models, metrics = train_all_quantiles(
            df, feature_cols, hyperparams, 
            train_size, val_size, test_size, horizons=[1]
        )
        
        results = {}
        for q in ['p10', 'p50', 'p90']:
            q_model_key = f"{q}_1d"
            if q_model_key in models:
                results[q] = {
                    'model': models[q_model_key],
                    'mae': metrics['quantiles'][q_model_key]['val_mae']
                }
                
        if '1d' in metrics['train_dates']:
            t_start, t_end = metrics['train_dates']['1d']
            v_start, v_end = metrics['val_dates']['1d']
            results['train_dates'] = {'start': t_start, 'end': t_end, 'days': train_size}
            results['val_dates'] = {'start': v_start, 'end': v_end}
            
        return results

    def save_model(self, model, quantile, timestamp, filepath):
        os.makedirs(os.path.dirname(filepath), exist_ok=True)
        model.save_model(filepath)
