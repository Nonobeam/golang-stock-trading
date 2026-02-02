"""
Prediction Coverage Tracker

Daily monitoring of whether actual returns fall within predicted intervals
at the expected rate (80% for p10-p90 range).

Tracks coverage by horizon and detects systematic bias in predictions.

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


# Coverage thresholds
EXPECTED_COVERAGE = 0.80  # 80% for [p10, p90] interval
COVERAGE_LOWER_BOUND = 0.75
COVERAGE_UPPER_BOUND = 0.85


class CoverageTracker:
    """
    Track prediction interval coverage to ensure models are well-calibrated.
    
    A well-calibrated model should have:
    - 80% of actuals within [p10, p90] range
    - 10% of actuals below p10
    - 10% of actuals above p90
    
    Systematic bias indicators:
    - Too many below p10 → Model too optimistic
    - Too many above p90 → Model too pessimistic
    """
    
    def __init__(self, db_conn: Optional[psycopg2.extensions.connection] = None):
        """
        Initialize coverage tracker.
        
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
    
    def check_daily_coverage(self, check_date: str = None) -> Dict:
        """
        Check coverage for predictions that matured on a specific date.
        
        Args:
            check_date: Date to check (default: yesterday)
            
        Returns:
            Dictionary with coverage metrics by ticker and horizon
        """
        if check_date is None:
            check_date = (datetime.now() - timedelta(days=1)).strftime('%Y-%m-%d')
        
        conn = self._get_connection()
        cursor = conn.cursor()
        
        # Fetch predictions with actuals for the check date
        cursor.execute(f"""
            SELECT ticker, horizon, prediction_date, target_date,
                   p10, p50, p90, actual_return
            FROM "{DB_SCHEMA}".predictions
            WHERE target_date = %s
              AND actual_return IS NOT NULL
        """, (check_date,))
        
        predictions = pd.DataFrame(
            cursor.fetchall(),
            columns=['ticker', 'horizon', 'prediction_date', 'target_date',
                    'p10', 'p50', 'p90', 'actual_return']
        )
        
        if len(predictions) == 0:
            return {
                'check_date': check_date,
                'total_predictions': 0,
                'message': 'No predictions matured on this date'
            }
        
        # Calculate coverage for each ticker-horizon combination
        results = []
        
        for (ticker, horizon), group in predictions.groupby(['ticker', 'horizon']):
            within_range = (
                (group['actual_return'] >= group['p10']) & 
                (group['actual_return'] <= group['p90'])
            ).sum()
            
            below_p10 = (group['actual_return'] < group['p10']).sum()
            above_p90 = (group['actual_return'] > group['p90']).sum()
            total = len(group)
            
            coverage = within_range / total
            
            result = {
                'ticker': ticker,
                'horizon': horizon,
                'check_date': check_date,
                'total_predictions': total,
                'within_range': within_range,
                'below_p10': below_p10,
                'above_p90': above_p90,
                'coverage': coverage,
                'below_p10_rate': below_p10 / total,
                'above_p90_rate': above_p90 / total
            }
            
            results.append(result)
            
            # Store in database
            self._store_coverage(result)
        
        conn.commit()
        
        return {
            'check_date': check_date,
            'total_predictions': len(predictions),
            'by_ticker_horizon': results
        }
    
    def _store_coverage(self, metrics: Dict):
        """Store coverage metrics in database."""
        conn = self._get_connection()
        cursor = conn.cursor()
        
        # Note: This stores aggregate daily coverage
        # Individual prediction coverage is stored in prediction_coverage table
        # This is just for the summary metrics
        pass  # Implement if needed for aggregate storage
    
    def get_coverage_by_horizon(self, 
                                 ticker: str,
                                 horizon: int,
                                 lookback_days: int = 30) -> Dict:
        """
        Calculate rolling coverage rate for a ticker-horizon over recent period.
        
        Args:
            ticker: Stock symbol
            horizon: Prediction horizon
            lookback_days: Days to look back (default 30)
            
        Returns:
            Dictionary with rolling coverage metrics
        """
        conn = self._get_connection()
        cursor = conn.cursor()
        
        cutoff_date = (datetime.now() - timedelta(days=lookback_days)).strftime('%Y-%m-%d')
        
        cursor.execute(f"""
            SELECT prediction_date, target_date, p10, p50, p90, actual_return
            FROM "{DB_SCHEMA}".predictions
            WHERE ticker = %s
              AND horizon = %s
              AND target_date >= %s
              AND actual_return IS NOT NULL
            ORDER BY target_date
        """, (ticker, horizon, cutoff_date))
        
        predictions = pd.DataFrame(
            cursor.fetchall(),
            columns=['prediction_date', 'target_date', 'p10', 'p50', 'p90', 'actual_return']
        )
        
        if len(predictions) == 0:
            return {
                'ticker': ticker,
                'horizon': horizon,
                'error': 'No predictions with actuals in period'
            }
        
        # Calculate coverage metrics
        within_range = (
            (predictions['actual_return'] >= predictions['p10']) & 
            (predictions['actual_return'] <= predictions['p90'])
        ).sum()
        
        below_p10 = (predictions['actual_return'] < predictions['p10']).sum()
        above_p90 = (predictions['actual_return'] > predictions['p90']).sum()
        total = len(predictions)
        
        coverage = within_range / total
        
        return {
            'ticker': ticker,
            'horizon': horizon,
            'lookback_days': lookback_days,
            'total_predictions': total,
            'coverage': coverage,
            'within_range': within_range,
            'below_p10': below_p10,
            'below_p10_rate': below_p10 / total,
            'above_p90': above_p90,
            'above_p90_rate': above_p90 / total,
            'coverage_ok': COVERAGE_LOWER_BOUND <= coverage <= COVERAGE_UPPER_BOUND
        }
    
    def detect_systematic_bias(self, 
                                ticker: str,
                                horizon: int,
                                lookback_days: int = 30) -> Dict:
        """
        Detect if predictions have systematic directional bias.
        
        Bias Types:
        - Too optimistic: Overpredicting returns (many actuals below p10)
        - Too pessimistic: Underpredicting returns (many actuals above p90)
        - Well-calibrated: Roughly equal misses on both sides
        
        Args:
            ticker: Stock symbol
            horizon: Prediction horizon
            lookback_days: Days to analyze
            
        Returns:
            Dictionary with bias detection results
        """
        coverage_metrics = self.get_coverage_by_horizon(ticker, horizon, lookback_days)
        
        if 'error' in coverage_metrics:
            return coverage_metrics
        
        below_p10_rate = coverage_metrics['below_p10_rate']
        above_p90_rate = coverage_metrics['above_p90_rate']
        
        # Expected: 10% below p10, 10% above p90
        below_p10_excess = below_p10_rate - 0.10
        above_p90_excess = above_p90_rate - 0.10
        
        # Determine bias
        bias_type = 'NONE'
        bias_severity = 'NONE'
        recommendation = 'Model well-calibrated'
        
        if abs(below_p10_excess) > 0.05 or abs(above_p90_excess) > 0.05:
            if below_p10_excess > above_p90_excess:
                bias_type = 'TOO_OPTIMISTIC'
                bias_severity = 'SEVERE' if below_p10_excess > 0.10 else 'MODERATE'
                recommendation = (
                    f"Model predicting too high. Actual returns falling below p10 "
                    f"{below_p10_rate:.1%} of time (should be ~10%). "
                    f"Recommendation: Retrain with recent bearish data or add conservative bias."
                )
            else:
                bias_type = 'TOO_PESSIMISTIC'
                bias_severity = 'SEVERE' if above_p90_excess > 0.10 else 'MODERATE'
                recommendation = (
                    f"Model predicting too low. Actual returns exceeding p90 "
                    f"{above_p90_rate:.1%} of time (should be ~10%). "
                    f"Recommendation: Model missing upside, retrain or reduce pessimistic bias."
                )
        
        return {
            'ticker': ticker,
            'horizon': horizon,
            'bias_type': bias_type,
            'bias_severity': bias_severity,
            'below_p10_rate': below_p10_rate,
            'below_p10_excess': below_p10_excess,
            'above_p90_rate': above_p90_rate,
            'above_p90_excess': above_p90_excess,
            'recommendation': recommendation
        }
    
    def get_coverage_alerts(self, min_coverage: float = 0.75) -> List[Dict]:
        """
        Get list of ticker-horizon combinations with low coverage requiring attention.
        
        Args:
            min_coverage: Minimum acceptable coverage (default 0.75)
            
        Returns:
            List of alerts for low-coverage models
        """
        conn = self._get_connection()
        cursor = conn.cursor()
        
        # Get all active ticker-horizon combinations
        cursor.execute(f"""
            SELECT DISTINCT ticker, horizon
            FROM "{DB_SCHEMA}".predictions
            WHERE prediction_date >= CURRENT_DATE - INTERVAL '30 days'
        """)
        
        ticker_horizons = cursor.fetchall()
        
        alerts = []
        
        for ticker, horizon in ticker_horizons:
            coverage = self.get_coverage_by_horizon(ticker, horizon, lookback_days=30)
            
            if 'error' not in coverage and coverage['coverage'] < min_coverage:
                bias = self.detect_systematic_bias(ticker, horizon, lookback_days=30)
                
                alerts.append({
                    'ticker': ticker,
                    'horizon': horizon,
                    'coverage': coverage['coverage'],
                    'expected': EXPECTED_COVERAGE,
                    'shortfall': EXPECTED_COVERAGE - coverage['coverage'],
                    'bias_type': bias.get('bias_type', 'UNKNOWN'),
                    'recommendation': bias.get('recommendation', 'Review model')
                })
        
        return sorted(alerts, key=lambda x: x['coverage'])


if __name__ == '__main__':
    # Example usage
    tracker = CoverageTracker()
    
    with tracker:
        print("Coverage Tracking Examples")
        print("=" * 60)
        
        # Example 1: Check yesterday's coverage
        print("\n1. Yesterday's Coverage:")
        yesterday = (datetime.now() - timedelta(days=1)).strftime('%Y-%m-%d')
        daily = tracker.check_daily_coverage(yesterday)
        print(f"   Date: {daily['check_date']}")
        print(f"   Total Predictions: {daily['total_predictions']}")
        
        # Example 2: 30-day coverage for VCI 5-day
        print("\n2. VCI 5-day Coverage (30 days):")
        coverage = tracker.get_coverage_by_horizon('VCI', 5, lookback_days=30)
        if 'error' not in coverage:
            print(f"   Coverage: {coverage['coverage']:.1%}")
            print(f"   Below p10: {coverage['below_p10_rate']:.1%}")
            print(f"   Above p90: {coverage['above_p90_rate']:.1%}")
            print(f"   Status: {'✓ OK' if coverage['coverage_ok'] else '✗ LOW'}")
        
        # Example 3: Detect bias
        print("\n3. Systematic Bias Detection:")
        bias = tracker.detect_systematic_bias('VCI', 5)
        if 'error' not in bias:
            print(f"   Bias Type: {bias['bias_type']}")
            print(f"   Severity: {bias['bias_severity']}")
            print(f"   Recommendation: {bias['recommendation']}")
        
        # Example 4: Get alerts
        print("\n4. Coverage Alerts:")
        alerts = tracker.get_coverage_alerts(min_coverage=0.75)
        for alert in alerts[:3]:  # Show top 3
            print(f"   ⚠ {alert['ticker']} {alert['horizon']}-day: {alert['coverage']:.1%} coverage")
            print(f"      {alert['recommendation']}")
