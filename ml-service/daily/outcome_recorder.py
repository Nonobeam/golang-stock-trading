"""
Daily outcome recorder module.
Handles recording actual outcomes for past predictions.
"""

import sys
import os
import logging
from datetime import datetime, date
from typing import Optional

# Add parent directory to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from db.connection import DatabaseConnection
from db.queries import GET_PENDING_PREDICTIONS, UPDATE_PREDICTION_OUTCOME
from utils.logging_config import setup_logging
from monitoring.alerter import alerter
from monitoring.metrics import PerformanceMonitor

# Configure logging
logger = setup_logging("outcome_recorder")

class OutcomeRecorder:
    """Records actual outcomes for predictions."""
    
    def record_actual_outcomes(self, target_date_limit: str) -> None:
        """
        Record actual outcomes for predictions where target_date <= target_date_limit.
        
        Args:
            target_date_limit: Latest date to process (YYYY-MM-DD)
        """
        conn = DatabaseConnection.get_connection()
        try:
            cursor = conn.cursor()
            
            # Fetch pending predictions
            cursor.execute(GET_PENDING_PREDICTIONS, (target_date_limit,))
            predictions = cursor.fetchall()
            logger.info(f"Found {len(predictions)} pending predictions to update")
            
            success_count = 0
            for row in predictions:
                ticker, pred_date, target_date, horizon, p10, p50, p90 = row
                
                # Convert date object to string if needed
                if isinstance(target_date, (datetime, date)):
                    target_date_str = target_date.strftime('%Y-%m-%d')
                else:
                    target_date_str = str(target_date)
                
                # Get actual return for ticker on target_date
                actual_return = self._get_actual_return(cursor, ticker, target_date_str)
                
                if actual_return is None:
                    # logger.warning(f"No actual return found for {ticker} on {target_date_str}")
                    continue
                    
                # Calculate errors
                # error = predicted - actual
                error_p10 = float(p10) - actual_return
                error_p50 = float(p50) - actual_return
                error_p90 = float(p90) - actual_return
                
                cursor.execute(UPDATE_PREDICTION_OUTCOME, (
                    actual_return, error_p10, error_p50, error_p90,
                    ticker, target_date, horizon
                ))
                success_count += 1
                
                # Simple alert check for this specific prediction
                # If error > 5%, trigger warning
                if abs(error_p50) > 0.05:
                     alerter.send_alert(f"High prediction error for {ticker} on {target_date_str}: {error_p50:.4f} (Pred: {p50}, Actual: {actual_return})", level="WARNING")
            
            conn.commit()
            logger.info(f"Updated outcomes for {success_count} predictions")
            cursor.close()
            
        except Exception as e:
            conn.rollback()
            logger.error(f"Error recording outcomes: {e}")
            raise
        finally:
            DatabaseConnection.return_connection(conn)

    def _get_actual_return(self, cursor, ticker: str, date: str) -> Optional[float]:
        """
        Get return_1d from features table for a specific date.
        
        Args:
            cursor: Database cursor
            ticker: Stock ticker
            date: Date string
            
        Returns:
            Return value or None
        """
        try:
            # We query the features table for return_1d
            # Assuming features are calculated for this date
            query = """
                SELECT return_1d 
                FROM "stock-trading".features 
                WHERE ticker = %s AND date = %s
            """
            cursor.execute(query, (ticker, date))
            row = cursor.fetchone()
            
            if row and row[0] is not None:
                return float(row[0])
            return None
            
        except Exception as e:
            logger.error(f"Error getting return for {ticker} on {date}: {e}")
            return None

if __name__ == "__main__":
    # Simple CLI
    import sys
    
    if len(sys.argv) > 1:
        date_limit = sys.argv[1]
    else:
        date_limit = datetime.now().strftime('%Y-%m-%d')
        
    recorder = OutcomeRecorder()
    recorder.record_actual_outcomes(date_limit)
