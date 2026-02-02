"""
Prediction inference engine.
Handles loading models and generating predictions.
"""

import pandas as pd
import numpy as np
import xgboost as xgb
from datetime import datetime
import os
from typing import Dict, Tuple, Optional

from models.trainer import ModelTrainer
from data.loader import DataLoader
from features.calculator import get_all_features
from db.connection import DatabaseConnection
from db.queries import GET_PRODUCTION_MODELS
from utils.logging_config import setup_logging
from monitoring.metrics import track_latency

# Configure logging
logger = setup_logging("predictor")


class Predictor:
    """Handles model inference for predictions."""
    
    def __init__(self):
        """Initialize predictor."""
        self.models = {}  # {ticker: {quantile: model}}
        self.model_metadata = {} # {ticker: {quantile: metadata_dict}}
        self.loader = DataLoader()
        
    def load_production_models(self, ticker: str) -> bool:
        """
        Load production models for a ticker.
        
        Args:
            ticker: Stock ticker symbol
            
        Returns:
            True if models loaded successfully
        """
        # Check if we have loaded models for at least one horizon
        if ticker in self.models and len(self.models[ticker]) > 0:
            # We assume if we have some models, we have loaded what was available
            # To force reload, create a new Predictor or clear .models
            logger.debug(f"Models for {ticker} already loaded")
            return True
        
        conn = DatabaseConnection.get_connection()
        try:
            cursor = conn.cursor()
            cursor.execute(GET_PRODUCTION_MODELS, (ticker,))
            rows = cursor.fetchall()
            cursor.close()
            
            logger.info(f"Found {len(rows)} model records in database for {ticker}")
            
            if len(rows) == 0:
                logger.warning(f"No models found for {ticker}")
                return False
            
            # Load models
            trainer = ModelTrainer(ticker)
            self.models[ticker] = {}
            self.model_metadata[ticker] = {}
            
            for row in rows:
                model_id = row['model_id']
                ticker_name = row['ticker']
                quantile = float(row['quantile'])
                horizon = int(row.get('horizon', 1)) # Default to 1 if missing
                file_path = row['file_path']
                
                if horizon not in self.models[ticker]:
                    self.models[ticker][horizon] = {}
                    self.model_metadata[ticker][horizon] = {}
                
                if not os.path.exists(file_path):
                    logger.error(f"Model file not found: {file_path}")
                    continue
                
                try:
                    model = trainer.load_model(file_path)
                    self.models[ticker][horizon][quantile] = model
                    self.model_metadata[ticker][horizon][quantile] = {
                        'model_id': model_id,
                        'file_path': file_path
                    }
                    logger.info(f"✓ Loaded {ticker} p{int(quantile*100)} {horizon}d model from {file_path}")
                except Exception as e:
                    logger.error(f"Failed to load model {file_path}: {e}")
                    return False
            
            # Verify we have complete sets (3 quantiles) for loaded horizons
            valid_horizons = []
            for h, quantiles in self.models[ticker].items():
                if len(quantiles) == 3:
                    valid_horizons.append(h)
                else:
                    logger.warning(f"Incomplete model set for {ticker} horizon {h}d: found {len(quantiles)} quantiles")
            
            if not valid_horizons:
                logger.error(f"No valid model sets (3 quantiles) found for {ticker}")
                del self.models[ticker]
                return False
                
            return True
            
        except Exception as e:
            logger.error(f"Error loading production models: {e}")
            return False
        finally:
            DatabaseConnection.return_connection(conn)
    
    def prepare_features_for_prediction(self, ticker: str, date: str) -> Optional[pd.DataFrame]:
        """
        Prepare features for prediction on a specific date.
        
        Args:
            ticker: Stock ticker symbol
            date: Date for prediction (YYYY-MM-DD)
            
        Returns:
            DataFrame with features, or None if insufficient data
        """
        # Load recent data (need 200+ days for features)
        df = self.loader.load_daily_bars_recent(ticker, days=250)
        
        if df.empty or len(df) < 200:
            logger.warning(f"Insufficient data for {ticker}")
            return None
        
        # Calculate features
        df = get_all_features(df)
        
        # Filter to requested date
        df['date'] = pd.to_datetime(df['date'])
        target_date = pd.to_datetime(date)
        df_date = df[df['date'] == target_date]
        
        if df_date.empty:
            # If exact date not found, use most recent
            logger.warning(f"Date {date} not found, using most recent")
            df_date = df.iloc[[-1]]
        
        # Get feature columns
        feature_cols = self.loader.get_feature_columns()
        features = df_date[feature_cols]
        
        # Check for NaN
        if features.isna().any().any():
            logger.warning("Features contain NaN values")
            features = features.fillna(0)
        
        return features
    
    def predict(self, ticker: str, features: pd.DataFrame) -> Dict[int, Dict[str, float]]:
        """
        Generate predictions using loaded models for all horizons.
        
        Args:
            ticker: Stock ticker symbol
            features: Feature matrix (single row)
            
        Returns:
            Dict[horizon, {p10, p50, p90, confidence}]
        """
        if ticker not in self.models:
            raise ValueError(f"Models not loaded for {ticker}")
        
        results = {}
        
        # Convert features to DMatrix for XGBoost Booster
        dmatrix = xgb.DMatrix(features)
        
        for horizon, models in self.models[ticker].items():
            if len(models) != 3:
                continue
                
            # Generate predictions
            p10 = float(models[0.10].predict(dmatrix)[0])
            p50 = float(models[0.50].predict(dmatrix)[0])
            p90 = float(models[0.90].predict(dmatrix)[0])
            
            # Validate ordering
            if not (p10 <= p50 <= p90):
                logger.warning(f"Quantile ordering violation for {ticker} {horizon}d: p10={p10}, p50={p50}, p90={p90}")
                # Force ordering
                p10 = min(p10, p50, p90)
                p90 = max(p10, p50, p90)
                p50 = np.median([p10, p50, p90])
            
            # Calculate precision score based on prediction range
            # Narrower range = higher precision
            prediction_range = p90 - p10
            precision_score = 1.0 - min(1.0, prediction_range / 0.20)
            
            # Get calibration score from model metadata (default 0.70 for new models)
            calibration_score = self.model_metadata[ticker][horizon].get(0.50, {}).get('calibration_score', 0.70)
            
            # Final confidence = precision × calibration
            confidence = precision_score * calibration_score
            
            # Ensure confidence in [0, 1]
            confidence = max(0.0, min(1.0, confidence))
            
            results[horizon] = {
                'p10': p10,
                'p50': p50,
                'p90': p90,
                'confidence': confidence,
                'precision': precision_score  # Include for debugging/transparency
            }
        
        return results
    
    @track_latency
    def predict_for_date(self, ticker: str, date: str) -> Dict[int, Dict[str, float]]:
        """
        Complete prediction workflow for a date.
        
        Args:
            ticker: Stock ticker symbol
            date: Date for prediction (YYYY-MM-DD)
            
        Returns:
            Dict[horizon, {p10, p50, p90, confidence}]
        """
        # Load models if not already loaded
        if not self.load_production_models(ticker):
            raise ValueError(f"Failed to load models for {ticker}")
        
        # Prepare features
        features = self.prepare_features_for_prediction(ticker, date)
        if features is None:
            raise ValueError(f"Failed to prepare features for {ticker} on {date}")
        
        # Generate prediction
        return self.predict(ticker, features)
    
    def calculate_confidence(self, p90: float, p10: float, p50: float) -> float:
        """
        Calculate prediction confidence score.
        """
        uncertainty = p90 - p10
        if abs(p50) > 1e-6:
            confidence = 1.0 - min(1.0, uncertainty / abs(p50))
        else:
            confidence = 0.5
        
        return max(0.0, min(1.0, confidence))

    def get_model_version(self, ticker: str, horizon: int = 1) -> str:
        """
        Get version identifier for the loaded models.
        
        Args:
            ticker: Stock ticker symbol
            horizon: Prediction horizon
            
        Returns:
            Model version string
        """
        if ticker not in self.model_metadata:
            return "unknown"
        
        if horizon not in self.model_metadata[ticker]:
            return "unknown"
            
        # Use p50 model ID as the primary version identifier
        meta = self.model_metadata[ticker][horizon].get(0.50)
        if meta:
            return meta.get('model_id', 'unknown')
        return "unknown"
