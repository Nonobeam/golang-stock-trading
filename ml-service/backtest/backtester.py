"""
Backtester evaluation module.
Handles walk-forward backtesting of XGBoost models.
"""

import pandas as pd
import numpy as np
from datetime import datetime, timedelta
import logging
import sys
import os
from typing import Dict, Any, List

# Add parent directory to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from models.trainer import ModelTrainer
from data.loader import DataLoader
from backtest.metrics import generate_metrics_report, check_calibration
from features.calculator import get_all_features

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class Backtester:
    """Handles walk-forward backtesting simulation."""
    
    def __init__(self, ticker: str):
        self.ticker = ticker
        self.loader = DataLoader()
        self.results = []
        
    def walk_forward_backtest(self, start_date: str, end_date: str, 
                             training_window: int = 500,
                             retrain_days: int = 30) -> pd.DataFrame:
        """
        Execute walk-forward backtest.
        
        Args:
            start_date: Backtest start date (YYYY-MM-DD)
            end_date: Backtest end date (YYYY-MM-DD)
            training_window: Number of days to use for training
            retrain_days: How often to retrain model (e.g. every 30 days)
            
        Returns:
            DataFrame with backtest results
        """
        logger.info(f"Starting walk-forward backtest for {self.ticker} from {start_date} to {end_date}")
        
        # 1. Load data
        # We need data from (start - training_window) to end_date
        # Estimate robust lookback buffer (extra 100 days for features like SMA200)
        lookback_buffer = 100
        total_days = (datetime.strptime(end_date, '%Y-%m-%d') - datetime.strptime(start_date, '%Y-%m-%d')).days
        total_lookback = training_window + lookback_buffer + total_days + 50
        
        logger.info(f"Loading {total_lookback} days of data...")
        df_raw = self.loader.load_daily_bars_recent(self.ticker, days=total_lookback)
        
        if df_raw.empty:
            raise ValueError(f"No data found for {self.ticker}")
            
        # 2. Calculate features
        logger.info("Calculating features...")
        df_features = get_all_features(df_raw)
        
        # 3. Create target (next day return)
        df_features['target'] = df_features['close'].pct_change().shift(-1)
        
        # Ensure 'date' is datetime
        df_features['date'] = pd.to_datetime(df_features['date'])
        
        # Filter simulation period
        sim_start = pd.to_datetime(start_date)
        sim_end = pd.to_datetime(end_date)
        
        sim_dates = df_features[(df_features['date'] >= sim_start) & (df_features['date'] <= sim_end)]['date'].tolist()
        sim_dates.sort()
        
        if not sim_dates:
            logger.error("No trading dates found in the specified range")
            return pd.DataFrame()
        
        logger.info(f"Simulation will cover {len(sim_dates)} trading days")
        
        trainer = ModelTrainer(self.ticker)
        current_models = {}
        last_train_date = None
        
        predictions = []
        
        for i, current_date in enumerate(sim_dates):
            current_date_str = current_date.strftime('%Y-%m-%d')
            
            # Check if retraining is needed
            # Retrain if:
            # 1. No models yet
            # 2. Passed retrain_days threshold
            # 3. Last training involved data too old
            days_since_train = (current_date - last_train_date).days if last_train_date else 9999
            
            if days_since_train >= retrain_days:
                logger.info(f"Retraining models at {current_date_str}...")
                
                # Prepare TRAINING data: Up to yesterday (current_date - 1 day)
                # Actually, in walk-forward, "current_date" is the prediction date.
                # We assume we are at the evening of 'current_date' predicting for 'current_date+1'??
                # Wait: 'target' shifted -1 means 'target' at row T is return from T to T+1.
                # So if we are at row T, we have features at T, and we want to predict 'target' at T.
                # Training data should be prior to T.
                
                train_mask = (df_features['date'] < current_date) & \
                             (df_features['date'] >= current_date - timedelta(days=training_window))
                
                train_df = df_features[train_mask].dropna(subset=['target'])
                
                if len(train_df) < 100:
                    logger.warning(f"Insufficient training data at {current_date_str} ({len(train_df)} rows), skipping date")
                    continue
                
                feature_cols = trainer.get_feature_columns()
                X_train = train_df[feature_cols]
                y_train = train_df['target']
                
                # Train models
                current_models[0.10] = trainer.train_quantile_model(X_train, y_train, 0.10)
                current_models[0.50] = trainer.train_quantile_model(X_train, y_train, 0.50)
                current_models[0.90] = trainer.train_quantile_model(X_train, y_train, 0.90)
                
                last_train_date = current_date
            
            # Make prediction for current row
            row = df_features[df_features['date'] == current_date]
            if row.empty:
                continue
                
            feature_cols = trainer.get_feature_columns()
            X_pred = row[feature_cols]
            
            # Generate predictions
            p10 = float(current_models[0.10].predict(X_pred)[0])
            p50 = float(current_models[0.50].predict(X_pred)[0])
            p90 = float(current_models[0.90].predict(X_pred)[0])
            
            actual = float(row['target'].values[0]) if not pd.isna(row['target'].values[0]) else None
            
            # Validate ordering
            p10, p90 = min(p10, p90), max(p10, p90)
            
            predictions.append({
                'date': current_date_str,
                'ticker': self.ticker,
                'p10': p10,
                'p50': p50,
                'p90': p90,
                'prediction': p50, # Use p50 as primary point estimate
                'actual_return': actual
            })
            
            # Basic progress log
            if i % 10 == 0:
                logger.info(f"Processed {i+1}/{len(sim_dates)} days")
                
        # Compile results
        self.results = pd.DataFrame(predictions)
        
        # Calculate calibration checks locally
        if not self.results.empty and 'actual_return' in self.results.columns:
            valid_res = self.results.dropna(subset=['actual_return'])
            p10_cal = check_calibration(valid_res['actual_return'].values, valid_res['p10'].values, 0.10)
            p90_cal = check_calibration(valid_res['actual_return'].values, valid_res['p90'].values, 0.90)
            logger.info(f"Backtest Complete. Calibration: p10={p10_cal:.4f}, p90={p90_cal:.4f}")
            
        return self.results
