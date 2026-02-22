"""
Floor-Hit Probability Classifier

Predicts probability of hitting circuit breaker price limits in
Vietnamese market using binary classification.

Circuit Breaker Rules:
- HOSE (Ho Chi Minh Stock Exchange): ±7% daily limit
- HNX (Hanoi Stock Exchange): ±10% daily limit

When a stock hits floor price, all bids are removed and liquidity
disappears, trapping positions until sentiment recovers.

Author: ML Trading System
Created: 2026-02-02
"""

import numpy as np
import pandas as pd
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Tuple
import xgboost as xgb
import psycopg
from sklearn.model_selection import train_test_split
from sklearn.metrics import classification_report, roc_auc_score

from config import DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD, DB_SCHEMA, BASE_DIR


# Circuit breaker limits by exchange
CIRCUIT_BREAKER_LIMITS = {
    'HOSE': 0.07,   # ±7%
    'HNX': 0.10     # ±10%
}


class FloorHitClassifier:
    """
    Binary classifier to predict floor/ceiling hit probability.
    
    Features:
    - momentum_5d: 5-day price change %
    - volume_surge: current_volume / avg_volume_20d
    - consecutive_down_days: streak of negative returns
    - distance_from_support: (price - support_level) / support_level
    - vn_index_momentum: Market momentum
    - relative_strength: stock vs market momentum
    """
    
    def __init__(self, 
                 exchange: str = 'HOSE',
                 db_conn: Optional[psycopg.Connection] = None):
        """
        Initialize floor-hit classifier.
        
        Args:
            exchange: 'HOSE' or 'HNX' for appropriate circuit breaker limit
            db_conn: Optional database connection
        """
        self.exchange = exchange
        self.circuit_limit = CIRCUIT_BREAKER_LIMITS[exchange]
        self.db_conn = db_conn
        self._own_connection = db_conn is None
        
        self.floor_model = None
        self.ceiling_model = None
        self.models_dir = BASE_DIR / 'models' / 'saved' / 'floor_hit'
        self.models_dir.mkdir(parents=True, exist_ok=True)
    
    def _get_connection(self):
        """Get or create database connection."""
        if self.db_conn is None:
            self.db_conn = psycopg.connect(
                host=DB_HOST,
                port=DB_PORT,
                dbname=DB_NAME,
                user=DB_USER,
                password=DB_PASSWORD
            )
        return self.db_conn
    
    def __enter__(self):
        self._get_connection()
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        if self._own_connection and self.db_conn:
            self.db_conn.close()
    
    def prepare_training_data(self, ticker: str, 
                               start_date: str = '2023-01-01',
                               end_date: str = None) -> pd.DataFrame:
        """
        Prepare training data with features and binary labels.
        
        Label Definition:
        - hit_floor = 1 if next_day_return <= -circuit_limit
        - hit_ceiling = 1 if next_day_return >= +circuit_limit
        
        Args:
            ticker: Stock symbol
            start_date: Training data start date
            end_date: Training data end date (default: today)
            
        Returns:
            DataFrame with features and labels
        """
        if end_date is None:
            end_date = datetime.now().strftime('%Y-%m-%d')
        
        conn = self._get_connection()
        cursor = conn.cursor()
        
        # Fetch features and price data
        cursor.execute(f"""
            SELECT 
                f.date,
                f.return_1d,
                f.return_5d,
                f.return_20d,
                f.volume_ratio_5d,
                f.volume_ratio_20d,
                f.volatility_5d,
                f.rsi_14,
                f.sma_5,
                f.sma_20,
                d.close as current_price,
                LEAD(d.close, 1) OVER (ORDER BY f.date) as next_price
            FROM "{DB_SCHEMA}".features f
            JOIN "{DB_SCHEMA}".daily_bars d ON f.ticker = d.ticker AND f.date = d.date
            WHERE f.ticker = %s
              AND f.date BETWEEN %s AND %s
              AND f.features_complete = TRUE
            ORDER BY f.date
        """, (ticker, start_date, end_date))
        
        data = pd.DataFrame(cursor.fetchall(), 
                           columns=['date', 'return_1d', 'return_5d', 'return_20d',
                                   'volume_ratio_5d', 'volume_ratio_20d', 'volatility_5d',
                                   'rsi_14', 'sma_5', 'sma_20', 'current_price', 'next_price'])
        
        # Calculate next day return
        data['next_day_return'] = (data['next_price'] - data['current_price']) / data['current_price']
        
        # Create binary labels
        data['hit_floor'] = (data['next_day_return'] <= -self.circuit_limit).astype(int)
        data['hit_ceiling'] = (data['next_day_return'] >= self.circuit_limit).astype(int)
        
        # Engineer features
        data['momentum_5d'] = data['return_5d']
        data['volume_surge'] = data['volume_ratio_5d']
        
        # Consecutive down days
        data['consecutive_down'] = 0
        for i in range(1, len(data)):
            if data.loc[i, 'return_1d'] < 0:
                data.loc[i, 'consecutive_down'] = data.loc[i-1, 'consecutive_down'] + 1
        
        # Distance from support (using SMA20 as support)
        data['distance_from_support'] = (data['current_price'] - data['sma_20']) / data['sma_20']
        
        # Relative strength (stock vs its own 20-day average)
        data['relative_strength'] = data['return_5d'] - data['return_20d']
        
        # Drop rows with missing data
        data = data.dropna()
        
        return data
    
    def train(self, ticker: str, start_date: str = '2023-01-01', 
              end_date: str = None):
        """
        Train binary classifiers for floor and ceiling hits.
        
        Args:
            ticker: Stock symbol to train on
            start_date: Training data start
            end_date: Training data end
        """
        # Prepare data
        data = self.prepare_training_data(ticker, start_date, end_date)
        
        # Select features
        feature_cols = ['momentum_5d', 'volume_surge', 'consecutive_down',
                       'distance_from_support', 'volatility_5d', 'relative_strength',
                       'rsi_14']
        X = data[feature_cols]
        
        # Train floor classifier
        y_floor = data['hit_floor']
        X_train, X_val, y_train, y_val = train_test_split(
            X, y_floor, test_size=0.2, random_state=42, stratify=y_floor
        )
        
        # Handle class imbalance with scale_pos_weight
        scale_pos_weight = (y_floor == 0).sum() / (y_floor == 1).sum()
        
        self.floor_model = xgb.XGBClassifier(
            max_depth=4,
            learning_rate=0.05,
            n_estimators=100,
            scale_pos_weight=scale_pos_weight,
            random_state=42,
            eval_metric='auc'
        )
        
        self.floor_model.fit(X_train, y_train, 
                            eval_set=[(X_val, y_val)],
                            verbose=False)
        
        # Train ceiling classifier
        y_ceiling = data['hit_ceiling']
        X_train, X_val, y_train, y_val = train_test_split(
            X, y_ceiling, test_size=0.2, random_state=42, stratify=y_ceiling
        )
        
        scale_pos_weight = (y_ceiling == 0).sum() / (y_ceiling == 1).sum()
        
        self.ceiling_model = xgb.XGBClassifier(
            max_depth=4,
            learning_rate=0.05,
            n_estimators=100,
            scale_pos_weight=scale_pos_weight,
            random_state=42,
            eval_metric='auc'
        )
        
        self.ceiling_model.fit(X_train, y_train,
                              eval_set=[(X_val, y_val)],
                              verbose=False)
        
        # Save models
        self.floor_model.save_model(str(self.models_dir / f'{ticker}_floor.json'))
        self.ceiling_model.save_model(str(self.models_dir / f'{ticker}_ceiling.json'))
        
        print(f"Trained floor-hit classifiers for {ticker}")
        print(f"Floor hits in data: {data['hit_floor'].sum()} / {len(data)} ({data['hit_floor'].mean():.2%})")
        print(f"Ceiling hits in data: {data['hit_ceiling'].sum()} / {len(data)} ({data['hit_ceiling'].mean():.2%})")
    
    def predict_floor_probability(self, ticker: str, features: Dict) -> float:
        """
        Predict probability of hitting floor limit.
        
        Args:
            ticker: Stock symbol
            features: Dictionary of feature values
            
        Returns:
            Probability of hitting floor (0-1)
        """
        if self.floor_model is None:
            # Try to load saved model
            model_path = self.models_dir / f'{ticker}_floor.json'
            if model_path.exists():
                self.floor_model = xgb.XGBClassifier()
                self.floor_model.load_model(str(model_path))
            else:
                raise ValueError(f"Floor model for {ticker} not trained")
        
        # Convert features to DataFrame
        feature_cols = ['momentum_5d', 'volume_surge', 'consecutive_down',
                       'distance_from_support', 'volatility_5d', 'relative_strength',
                       'rsi_14']
        X = pd.DataFrame([{k: features.get(k, 0) for k in feature_cols}])
        
        # Get probability of class 1 (floor hit)
        prob = self.floor_model.predict_proba(X)[0][1]
        
        return float(prob)
    
    def predict_ceiling_probability(self, ticker: str, features: Dict) -> float:
        """
        Predict probability of hitting ceiling limit.
        
        Args:
            ticker: Stock symbol
            features: Dictionary of feature values
            
        Returns:
            Probability of hitting ceiling (0-1)
        """
        if self.ceiling_model is None:
            model_path = self.models_dir / f'{ticker}_ceiling.json'
            if model_path.exists():
                self.ceiling_model = xgb.XGBClassifier()
                self.ceiling_model.load_model(str(model_path))
            else:
                raise ValueError(f"Ceiling model for {ticker} not trained")
        
        feature_cols = ['momentum_5d', 'volume_surge', 'consecutive_down',
                       'distance_from_support', 'volatility_5d', 'relative_strength',
                       'rsi_14']
        X = pd.DataFrame([{k: features.get(k, 0) for k in feature_cols}])
        
        prob = self.ceiling_model.predict_proba(X)[0][1]
        
        return float(prob)
    
    def store_prediction(self, ticker: str, prediction_date: str,
                        floor_prob: float, ceiling_prob: float):
        """Store floor-hit probability prediction in database."""
        conn = self._get_connection()
        cursor = conn.cursor()
        
        cursor.execute("""
            INSERT INTO floor_hit_probabilities
            (ticker, exchange, prediction_date, floor_probability, ceiling_probability)
            VALUES (%s, %s, %s, %s, %s)
            ON CONFLICT (ticker, prediction_date) DO UPDATE SET
                floor_probability   = EXCLUDED.floor_probability,
                ceiling_probability = EXCLUDED.ceiling_probability,
                exchange            = EXCLUDED.exchange
        """, (ticker, self.exchange, prediction_date, floor_prob, ceiling_prob))
        
        conn.commit()


if __name__ == '__main__':
    # Example usage
    classifier = FloorHitClassifier(exchange='HOSE')
    
    with classifier:
        # Train on VCI
        print("Training floor-hit classifier for VCI...")
        classifier.train('VCI', start_date='2023-01-01', end_date='2025-12-31')
        
        # Predict current floor risk
        test_features = {
            'momentum_5d': -0.042,  # Down 4.2% over 5 days
            'volume_surge': 1.8,     # 80% above average volume
            'consecutive_down': 2,   # 2 consecutive down days
            'distance_from_support': -0.05,  # 5% below SMA20
            'volatility_5d': 0.03,
            'relative_strength': -0.02,
            'rsi_14': 35
        }
        
        floor_prob = classifier.predict_floor_probability('VCI', test_features)
        ceiling_prob = classifier.predict_ceiling_probability('VCI', test_features)
        
        print(f"\nFloor-Hit Probability: {floor_prob:.2%}")
        print(f"Ceiling-Hit Probability: {ceiling_prob:.2%}")
        
        if floor_prob > 0.20:
            print(f"⚠ WARNING: High floor risk ({floor_prob:.2%}) - consider reducing position")
        if floor_prob > 0.40:
            print(f"🚨 ALERT: Critical floor risk ({floor_prob:.2%}) - recommend exit")
