"""
Performance metrics and monitoring module.
"""
import time
import functools
import logging
from typing import Dict, Any, Optional
from datetime import datetime, timedelta
import pandas as pd

from db.connection import DatabaseConnection
from utils.logging_config import setup_logging

logger = setup_logging("monitoring.metrics")

def track_latency(func):
    """Decorator to track and log function latency."""
    @functools.wraps(func)
    def wrapper(*args, **kwargs):
        start_time = time.time()
        try:
            result = func(*args, **kwargs)
            return result
        finally:
            end_time = time.time()
            duration_ms = (end_time - start_time) * 1000
            logger.info(f"Latency - {func.__name__}: {duration_ms:.2f}ms")
    return wrapper

class PerformanceMonitor:
    """Methods to retrieve and calculate system performance metrics."""
    
    @staticmethod
    def get_model_accuracy(ticker: str, days: int = 30) -> Dict[str, float]:
        """
        Calculate model accuracy (MAE) over the last N days.
        """
        conn = DatabaseConnection.get_connection()
        try:
            # Query recent outcomes
            query = """
                SELECT 
                    AVG(ABS(error_p10)) as mae_p10,
                    AVG(ABS(error_p50)) as mae_p50,
                    AVG(ABS(error_p90)) as mae_p90,
                    COUNT(*) as count
                FROM "stock-trading".predictions
                WHERE ticker = %s 
                  AND target_date >= CURRENT_DATE - INTERVAL '%s days'
                  AND actual_return IS NOT NULL
            """
            cursor = conn.cursor()
            cursor.execute(query, (ticker, days))
            row = cursor.fetchone()
            cursor.close()
            
            if not row or row[3] == 0:
                logger.warning(f"No accuracy data found for {ticker} in last {days} days")
                return {}
                
            return {
                'mae_p10': float(row[0]) if row[0] else 0.0,
                'mae_p50': float(row[1]) if row[1] else 0.0,
                'mae_p90': float(row[2]) if row[2] else 0.0,
                'sample_size': int(row[3])
            }
            
        except Exception as e:
            logger.error(f"Error calculating accuracy for {ticker}: {e}")
            return {}
        finally:
            DatabaseConnection.return_connection(conn)
            
    @staticmethod
    def check_calibration(ticker: str, days: int = 90) -> Dict[str, float]:
        """
        Check quantile calibration (p10 should have ~10% actuals below it).
        """
        conn = DatabaseConnection.get_connection()
        try:
            query = """
                SELECT 
                    COUNT(*) as total,
                    SUM(CASE WHEN actual_return < p10 THEN 1 ELSE 0 END) as below_p10,
                    SUM(CASE WHEN actual_return > p90 THEN 1 ELSE 0 END) as above_p90
                FROM "stock-trading".predictions
                WHERE ticker = %s 
                  AND target_date >= CURRENT_DATE - INTERVAL '%s days'
                  AND actual_return IS NOT NULL
            """
            cursor = conn.cursor()
            cursor.execute(query, (ticker, days))
            row = cursor.fetchone()
            cursor.close()
            
            if not row or row[0] == 0:
                return {}
                
            total = row[0]
            below_p10 = row[1]
            above_p90 = row[2]
            
            return {
                'p10_calibration': below_p10 / total, # Target 0.10
                'p90_calibration': above_p90 / total, # Target 0.10 (above p90 means 10% tail)
                'sample_size': total
            }
            
        except Exception as e:
            logger.error(f"Error checking calibration for {ticker}: {e}")
            return {}
        finally:
            DatabaseConnection.return_connection(conn)
