"""
Quantile Calibration Checker

Verifies that prediction intervals (p10, p50, p90) accurately reflect
realized distributions using empirical coverage formula.

Formula:
    Empirical Coverage = (1/N) Σ I(yi ∈ [Q_α_L, Q_α_U])

Where I() is the indicator function (1 if true, 0 if false).

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


# Expected coverage thresholds
COVERAGE_THRESHOLDS = {
    'p10': (0.08, 0.12),   # 10% ± 2%
    'p25': (0.23, 0.27),   # 25% ± 2%
    'p50': (0.45, 0.55),   # 50% ± 5%
    'p75': (0.73, 0.77),   # 75% ± 2%
    'p90': (0.88, 0.92)    # 90% ± 2%
}


class CalibrationChecker:
    """
    Check if predicted quantiles match realized empirical quantiles.
    
    For a properly calibrated p10 quantile, we expect:
    - 10% of actual outcomes to be below p10
    - 90% of actual outcomes to be above p10
    
    For 80% prediction interval [p10, p90]:
    - 80% of actual outcomes should fall within the interval
    """
    
    def __init__(self, db_conn: Optional[psycopg2.extensions.connection] = None):
        """
        Initialize calibration checker.
        
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
    
    def check_calibration(self, 
                          ticker: str,
                          horizon: int,
                          lookback_days: int = 90) -> Dict:
        """
        Check calibration of quantile predictions over recent period.
        
        Args:
            ticker: Stock symbol
            horizon: Model horizon (1, 5, or 10 days)
            lookback_days: Number of days to check (default 90)
            
        Returns:
            Dictionary with coverage metrics for each quantile
        """
        conn = self._get_connection()
        cursor = conn.cursor()
        
        # Fetch predictions with actual outcomes
        cutoff_date = (datetime.now() - timedelta(days=lookback_days)).strftime('%Y-%m-%d')
        
        cursor.execute(f"""
            SELECT prediction_date, target_date, p10, p50, p90, actual_return
            FROM "{DB_SCHEMA}".predictions
            WHERE ticker = %s 
              AND horizon = %s
              AND prediction_date >= %s
              AND actual_return IS NOT NULL
            ORDER BY prediction_date
        """, (ticker, horizon, cutoff_date))
        
        rows = cursor.fetchall()
        
        if not rows:
            return {
                'error': 'No predictions with actuals found',
                'ticker': ticker,
                'horizon': horizon
            }
        
        predictions = pd.DataFrame(rows, columns=['prediction_date', 'target_date', 'p10', 'p50', 'p90', 'actual_return'])
        
        # Calculate empirical coverage for each quantile
        results = {}
        
        # p10: Expected 10% of actuals below p10
        below_p10 = (predictions['actual_return'] < predictions['p10']).sum()
        results['p10'] = {
            'expected_coverage': 0.10,
            'actual_coverage': below_p10 / len(predictions),
            'num_below': below_p10,
            'total_samples': len(predictions)
        }
        
        # p50: Expected 50% of actuals below p50
        below_p50 = (predictions['actual_return'] < predictions['p50']).sum()
        results['p50'] = {
            'expected_coverage': 0.50,
            'actual_coverage': below_p50 / len(predictions),
            'num_below': below_p50,
            'total_samples': len(predictions)
        }
        
        # p90: Expected 90% of actuals below p90
        below_p90 = (predictions['actual_return'] < predictions['p90']).sum()
        results['p90'] = {
            'expected_coverage': 0.90,
            'actual_coverage': below_p90 / len(predictions),
            'num_below': below_p90,
            'total_samples': len(predictions)
        }
        
        # 80% prediction interval [p10, p90]
        within_range = ((predictions['actual_return'] >= predictions['p10']) & 
                        (predictions['actual_return'] <= predictions['p90'])).sum()
        results['interval_80'] = {
            'expected_coverage': 0.80,
            'actual_coverage': within_range / len(predictions),
            'num_within': within_range,
            'total_samples': len(predictions)
        }
        
        # Calculate calibration errors and status
        calibration_report = self._generate_calibration_report(
            ticker, horizon, results, datetime.now().strftime('%Y-%m-%d')
        )
        
        # Store in database
        self._store_calibration_report(calibration_report)
        conn.commit()
        
        return calibration_report
    
    def _generate_calibration_report(self, ticker: str, horizon: int, 
                                     results: Dict, check_date: str) -> Dict:
        """Generate calibration report with status classification."""
        report = {
            'ticker': ticker,
            'horizon': horizon,
            'check_date': check_date,
            'quantiles': {}
        }
        
        for quantile, metrics in results.items():
            if quantile == 'interval_80':
                continue
            
            expected = metrics['expected_coverage']
            actual = metrics['actual_coverage']
            error = actual - expected
            
            # Classify status
            if quantile in COVERAGE_THRESHOLDS:
                min_thresh, max_thresh = COVERAGE_THRESHOLDS[quantile]
                if min_thresh <= actual <= max_thresh:
                    status = 'OK'
                elif min_thresh - 0.01 <= actual <= max_thresh + 0.01:
                    status = 'WARNING'
                else:
                    status = 'ERROR'
            else:
                status = 'OK' if abs(error) <= 0.02 else 'WARNING' if abs(error) <= 0.03 else 'ERROR'
            
            report['quantiles'][quantile] = {
                'expected_coverage': expected,
                'actual_coverage': actual,
                'calibration_error': error,
                'num_samples': metrics['total_samples'],
                'status': status
            }
        
        return report
    
    def _store_calibration_report(self, report: Dict):
        """Store calibration report in database."""
        conn = self._get_connection()
        cursor = conn.cursor()
        
        for quantile, metrics in report['quantiles'].items():
            cursor.execute("""
                INSERT INTO calibration_reports
                (ticker, model_horizon, quantile_level, expected_coverage, 
                 actual_coverage, calibration_error, num_samples, check_date, status)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
            """, (
                report['ticker'],
                report['horizon'],
                quantile,
                metrics['expected_coverage'],
                metrics['actual_coverage'],
                metrics['calibration_error'],
                metrics['num_samples'],
                report['check_date'],
                metrics['status']
            ))
    
    def is_calibrated(self, report: Dict) -> Tuple[bool, List[str]]:
        """
        Check if model is properly calibrated.
        
        Returns:
            Tuple of (is_calibrated, list_of_errors)
        """
        errors = []
        
        for quantile, metrics in report.get('quantiles', {}).items():
            if metrics['status'] == 'ERROR':
                errors.append(
                    f"{quantile}: expected {metrics['expected_coverage']:.1%}, "
                    f"got {metrics['actual_coverage']:.1%} "
                    f"(error: {metrics['calibration_error']:+.1%})"
                )
        
        return len(errors) == 0, errors
    
    def recommend_recalibration(self, report: Dict) -> Dict:
        """
        Generate recalibration recommendations for miscalibrated quantiles.
        
        Returns:
            Dictionary with recommended parameter adjustments
        """
        recommendations = {}
        
        for quantile, metrics in report.get('quantiles', {}).items():
            if metrics['status'] in ['WARNING', 'ERROR']:
                error = metrics['calibration_error']
                expected = metrics['expected_coverage']
                actual = metrics['actual_coverage']
                
                # Recommendation logic
                if error > 0:
                    # Too pessimistic - catching more downside than expected
                    direction = "decrease"
                    reason = f"Catching {actual:.1%} of outcomes, should be {expected:.1%}. Too conservative."
                else:
                    # Too optimistic - missing downside
                    direction = "increase"
                    reason = f"Catching only {actual:.1%} of outcomes, should be {expected:.1%}. Too aggressive."
                
                recommendations[quantile] = {
                    'action': f'{direction} alpha parameter',
                    'reason': reason,
                    'current_error': error,
                    'severity': metrics['status']
                }
        
        return recommendations


if __name__ == '__main__':
    # Example usage
    checker = CalibrationChecker()
    
    with checker:
        report = checker.check_calibration(
            ticker='VCI',
            horizon=5,
            lookback_days=90
        )
        
        print("Calibration Report")
        print("=" * 60)
        print(f"Ticker: {report['ticker']}")
        print(f"Horizon: {report['horizon']} days")
        print(f"Check Date: {report['check_date']}")
        print()
        
        for quantile, metrics in report['quantiles'].items():
            print(f"{quantile}:")
            print(f"  Expected: {metrics['expected_coverage']:.1%}")
            print(f"  Actual: {metrics['actual_coverage']:.1%}")
            print(f"  Error: {metrics['calibration_error']:+.1%}")
            print(f"  Status: {metrics['status']}")
        
        is_cal, errors = checker.is_calibrated(report)
        print(f"\nCalibrated: {is_cal}")
        
        if not is_cal:
            print("\nCalibration Errors:")
            for error in errors:
                print(f"  - {error}")
            
            print("\nRecommendations:")
            recs = checker.recommend_recalibration(report)
            for quantile, rec in recs.items():
                print(f"  {quantile}: {rec['action']} - {rec['reason']}")
