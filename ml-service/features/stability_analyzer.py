"""
Feature Stability Analyzer

Tracks feature importance over time to identify stable vs noisy predictors.

Stable features maintain consistent importance across retraining periods,
while unstable features show high variance indicating noise or regime changes.

Uses coefficient of variation (CV) for stability metric:
    CV = std(importance) / mean(importance)

Low CV (<0.3) indicates stable, reliable features.

Author: ML Trading System
Created: 2026-02-02
"""

import numpy as np
import pandas as pd
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Tuple
import psycopg2
from psycopg2.extras import execute_values

from ..config import DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD, DB_SCHEMA


# Stability thresholds
STABLE_CV_THRESHOLD = 0.3  # CV < 0.3 considered stable
STABLE_RANK_CHANGE_THRESHOLD = 3  # Max rank position change for stable feature


class FeatureStabilityAnalyzer:
    """
    Analyze feature importance stability over time.
    
    Purpose:
    - Identify core features that consistently predict well
    - Detect noisy features with unstable importance
    - Support adaptive feature selection during volatile periods
    
    Metrics:
    - Mean importance (gain-based from XGBoost)
    - Std dev of importance
    - Coefficient of variation (CV = std/mean)
    - Rank position changes over time
    """
    
    def __init__(self, db_conn: Optional[psycopg2.extensions.connection] = None):
        """
        Initialize feature stability analyzer.
        
        Args:
            db_conn: Optional database connection
        """
        self.db_conn = db_conn
        self._own_connection = db_conn is None
    
    def _get_connection(self):
        """Get or create database connection."""
        if self.db_conn is None:
            self.db_conn = psycopg2.connect(
                host=DB_HOST,
                port=DB_PORT,
                database=DB_NAME,
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
    
    def record_feature_importance(self,
                                   ticker: str,
                                   horizon: int,
                                   feature_importance: Dict[str, float],
                                   training_date: str = None):
        """
        Record feature importance from a training session.
        
        Args:
            ticker: Stock symbol
            horizon: Model horizon
            feature_importance: Dict of {feature_name: importance_gain}
            training_date: Date of training (default: today)
        """
        if training_date is None:
            training_date = datetime.now().strftime('%Y-%m-%d')
        
        conn = self._get_connection()
        cursor = conn.cursor()
        
        # Sort features by importance and assign ranks
        sorted_features = sorted(
            feature_importance.items(),
            key=lambda x: x[1],
            reverse=True
        )
        
        # Insert records
        for rank, (feature_name, importance_gain) in enumerate(sorted_features, start=1):
            cursor.execute("""
                INSERT INTO feature_stability
                (ticker, model_horizon, feature_name, importance_gain, 
                 rank_position, training_date)
                VALUES (%s, %s, %s, %s, %s, %s)
            """, (ticker, horizon, feature_name, importance_gain, rank, training_date))
        
        conn.commit()
        
        print(f"Recorded importance for {len(feature_importance)} features "
              f"({ticker}, horizon={horizon}, date={training_date})")
    
    def calculate_stability_metrics(self,
                                     ticker: str,
                                     horizon: int,
                                     feature_name: str,
                                     lookback_months: int = 6) -> Dict:
        """
        Calculate stability metrics for a specific feature.
        
        Args:
            ticker: Stock symbol
            horizon: Model horizon
            feature_name: Name of feature to analyze
            lookback_months: Months of history to analyze
            
        Returns:
            Dictionary with stability metrics
        """
        conn = self._get_connection()
        cursor = conn.cursor()
        
        cutoff_date = (datetime.now() - timedelta(days=lookback_months*30)).strftime('%Y-%m-%d')
        
        cursor.execute("""
            SELECT training_date, importance_gain, rank_position
            FROM feature_stability
            WHERE ticker = %s
              AND model_horizon = %s
              AND feature_name = %s
              AND training_date >= %s
            ORDER BY training_date
        """, (ticker, horizon, feature_name, cutoff_date))
        
        rows = cursor.fetchall()
        
        if not rows:
            return {
                'feature_name': feature_name,
                'error': 'No historical data found'
            }
        
        data = pd.DataFrame(rows, columns=['training_date', 'importance_gain', 'rank_position'])
        
        # Calculate stability metrics
        importance_values = data['importance_gain'].values
        rank_values = data['rank_position'].values
        
        mean_importance = np.mean(importance_values)
        std_importance = np.std(importance_values)
        cv_importance = std_importance / mean_importance if mean_importance > 0 else float('inf')
        
        mean_rank = np.mean(rank_values)
        max_rank_change = np.max(np.abs(np.diff(rank_values)))
        
        # Classify stability
        is_stable_importance = cv_importance < STABLE_CV_THRESHOLD
        is_stable_rank = max_rank_change <= STABLE_RANK_CHANGE_THRESHOLD
        is_stable = is_stable_importance and is_stable_rank
        
        return {
            'ticker': ticker,
            'horizon': horizon,
            'feature_name': feature_name,
            'num_observations': len(data),
            'mean_importance': float(mean_importance),
            'std_importance': float(std_importance),
            'cv_importance': float(cv_importance),
            'mean_rank': float(mean_rank),
            'max_rank_change': int(max_rank_change),
            'is_stable': is_stable,
            'stability_class': 'STABLE' if is_stable else 'UNSTABLE'
        }
    
    def get_core_feature_set(self,
                              ticker: str,
                              horizon: int,
                              min_importance: float = 0.01,
                              lookback_months: int = 6) -> List[str]:
        """
        Get list of stable core features for reliable prediction.
        
        Core features are those with:
        - CV < 0.3 (stable importance)
        - Mean importance > min_importance
        - Rank changes <= 3 positions
        
        Args:
            ticker: Stock symbol
            horizon: Model horizon
            min_importance: Minimum mean importance threshold
            lookback_months: Months to analyze
            
        Returns:
            List of stable feature names
        """
        conn = self._get_connection()
        cursor = conn.cursor()
        
        # Get all features seen in the period
        cutoff_date = (datetime.now() - timedelta(days=lookback_months*30)).strftime('%Y-%m-%d')
        
        cursor.execute("""
            SELECT DISTINCT feature_name
            FROM feature_stability
            WHERE ticker = %s
              AND model_horizon = %s
              AND training_date >= %s
        """, (ticker, horizon, cutoff_date))
        
        all_features = [row[0] for row in cursor.fetchall()]
        
        # Calculate stability for each feature
        core_features = []
        
        for feature in all_features:
            metrics = self.calculate_stability_metrics(ticker, horizon, feature, lookback_months)
            
            if ('error' not in metrics and 
                metrics['is_stable'] and 
                metrics['mean_importance'] >= min_importance):
                core_features.append(feature)
        
        return core_features
    
    def recommend_feature_selection(self,
                                      ticker: str,
                                      horizon: int,
                                      market_regime: str = 'NORMAL') -> Dict:
        """
        Recommend feature selection strategy based on market regime.
        
        During high volatility or regime changes, use only core stable features.
        During normal conditions, use all features.
        
        Args:
            ticker: Stock symbol
            horizon: Model horizon
            market_regime: 'NORMAL', 'HIGH_VOL', or 'REGIME_CHANGE'
            
        Returns:
            Dictionary with recommendation
        """
        core_features = self.get_core_feature_set(ticker, horizon)
        
        if market_regime in ['HIGH_VOL', 'REGIME_CHANGE']:
            recommendation = 'USE_CORE_ONLY'
            rationale = (
                f"Market in {market_regime} regime. Use only {len(core_features)} "
                f"stable core features to avoid noise from unstable predictors."
            )
            features = core_features
        else:
            recommendation = 'USE_ALL'
            rationale = (
                f"Market in NORMAL regime. Use all features. "
                f"{len(core_features)} core features provide stability baseline."
            )
            features = None  # Use all
        
        return {
            'ticker': ticker,
            'horizon': horizon,
            'market_regime': market_regime,
            'recommendation': recommendation,
            'rationale': rationale,
            'core_features': core_features,
            'recommended_features': features
        }
    
    def get_stability_report(self,
                              ticker: str,
                              horizon: int,
                              lookback_months: int = 6) -> pd.DataFrame:
        """
        Generate comprehensive stability report for all features.
        
        Args:
            ticker: Stock symbol
            horizon: Model horizon
            lookback_months: Months to analyze
            
        Returns:
            DataFrame with stability metrics for all features
        """
        conn = self._get_connection()
        cursor = conn.cursor()
        
        cutoff_date = (datetime.now() - timedelta(days=lookback_months*30)).strftime('%Y-%m-%d')
        
        cursor.execute("""
            SELECT DISTINCT feature_name
            FROM feature_stability
            WHERE ticker = %s
              AND model_horizon = %s
              AND training_date >= %s
        """, (ticker, horizon, cutoff_date))
        
        all_features = [row[0] for row in cursor.fetchall()]
        
        report_data = []
        
        for feature in all_features:
            metrics = self.calculate_stability_metrics(ticker, horizon, feature, lookback_months)
            if 'error' not in metrics:
                report_data.append(metrics)
        
        df = pd.DataFrame(report_data)
        
        if len(df) > 0:
            df = df.sort_values('mean_importance', ascending=False)
        
        return df


if __name__ == '__main__':
    # Example usage
    analyzer = FeatureStabilityAnalyzer()
    
    with analyzer:
        print("Feature Stability Analysis Examples")
        print("=" * 60)
        
        # Example 1: Record feature importance (simulated)
        print("\n1. Recording Feature Importance:")
        sample_importance = {
            'return_5d': 0.25,
            'volume_ratio_5d': 0.18,
            'rsi_14': 0.15,
            'volatility_20d': 0.12,
            'macd': 0.10,
            'sma_20': 0.08,
            'bb_width': 0.07,
            'obv': 0.05
        }
        # analyzer.record_feature_importance('VCI', 5, sample_importance)
        print("   (Would record 8 features)")
        
        # Example 2: Calculate stability for a feature
        print("\n2. Feature Stability Metrics:")
        # metrics = analyzer.calculate_stability_metrics('VCI', 5, 'return_5d')
        # print(f"   Feature: {metrics['feature_name']}")
        # print(f"   Mean Importance: {metrics['mean_importance']:.3f}")
        # print(f"   CV: {metrics['cv_importance']:.3f}")
        # print(f"   Stability: {metrics['stability_class']}")
        print("   (Example output)")
        
        # Example 3: Get core features
        print("\n3. Core Feature Set:")
        # core = analyzer.get_core_feature_set('VCI', 5)
        # print(f"   Core Features ({len(core)}): {', '.join(core)}")
        print("   (Would return list of stable features)")
        
        # Example 4: Feature selection recommendation
        print("\n4. Feature Selection Recommendation:")
        # rec = analyzer.recommend_feature_selection('VCI', 5, market_regime='HIGH_VOL')
        # print(f"   Regime: {rec['market_regime']}")
        # print(f"   Recommendation: {rec['recommendation']}")
        # print(f"   Rationale: {rec['rationale']}")
        print("   (Would provide regime-based recommendation)")
