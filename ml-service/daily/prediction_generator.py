"""
Daily prediction generator module.
Handles generating and storing predictions for daily operations.
"""

import sys
import os
import logging
from datetime import datetime, timedelta
from typing import List

# Add parent directory to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from inference.predictor import Predictor
from db.connection import DatabaseConnection
from db.queries import SAVE_PREDICTION
from utils.logging_config import setup_logging
from monitoring.alerter import alerter

# Configure logging
logger = setup_logging("prediction_generator")

class PredictionGenerator:
    """Generates daily predictions for active tickers."""
    
    def __init__(self):
        self.predictor = Predictor()
        
    def generate_daily_predictions(self, tickers: List[str], date: str) -> None:
        """
        Generate and save predictions for a list of tickers.
        
        Args:
            tickers: List of ticker symbols
            date: Date to generate predictions FROM (using data up to this date)
        """
        conn = DatabaseConnection.get_connection()
        try:
            cursor = conn.cursor()
            success_count = 0
            
            for ticker in tickers:
                try:
                    # Predict for the given date (using data up to 'date')
                    p10, p50, p90, confidence = self.predictor.predict_for_date(ticker, date)
                    model_version = self.predictor.get_model_version(ticker)
                    
                    # Target date is the next trading day. 
                    # Ideally we should use a trading calendar, but for now assuming T+1
                    target_date = (datetime.strptime(date, '%Y-%m-%d') + timedelta(days=1)).strftime('%Y-%m-%d')
                    
                    cursor.execute(SAVE_PREDICTION, (
                        ticker, date, target_date, 1, # horizon 1 day
                        p10, p50, p90, confidence, model_version
                    ))
                    conn.commit()
                    success_count += 1
                    logger.info(f"Generated prediction for {ticker} on {date}: p50={p50:.4f}, conf={confidence:.2f}")
                    
                except Exception as e:
                    conn.rollback() # Rollback transaction for this ticker
                    msg = f"Error predicting for {ticker} on {date}: {e}"
                    logger.error(msg)
                    alerter.send_alert(msg, level="CRITICAL")
                    
            cursor.close()
            logger.info(f"Generated predictions for {success_count}/{len(tickers)} tickers")
            
        finally:
            DatabaseConnection.return_connection(conn)

if __name__ == "__main__":
    # Simple CLI
    import sys
    if len(sys.argv) < 3:
        print("Usage: python prediction_generator.py <ticker> <date>")
        sys.exit(1)
        
    ticker = sys.argv[1]
    date = sys.argv[2]
    
    generator = PredictionGenerator()
    generator.generate_daily_predictions([ticker], date)
