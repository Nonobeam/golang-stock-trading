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
                    predictions = self.predictor.predict_for_date(ticker, date)
                    model_version = self.predictor.get_model_version(ticker)
                    
                    # Target date is the next trading day. 
                    # Ideally we should use a trading calendar, but for now assuming T+1
                    target_date = (datetime.strptime(date, '%Y-%m-%d') + timedelta(days=1)).strftime('%Y-%m-%d')
                    
                    for horizon, preds in predictions.items():
                        cursor.execute(SAVE_PREDICTION, (
                            ticker, date, target_date, horizon,
                            preds['p10'], preds['p50'], preds['p90'], preds['confidence'], model_version
                        ))
                    conn.commit()
                    success_count += 1
                    
                    if 1 in predictions:
                        logger.info(f"Generated prediction for {ticker} on {date}: p50={predictions[1]['p50']:.4f}, conf={predictions[1]['confidence']:.2f}")
                    else:
                        logger.info(f"Generated predictions for {ticker} on {date}")
                    
                except Exception as e:
                    conn.rollback() # Rollback transaction for this ticker
                    msg = f"Error predicting for {ticker} on {date}: {e}"
                    logger.error(msg)
                    alerter.send_alert(msg, level="CRITICAL")
                    
            cursor.close()
            logger.info(f"Generated predictions for {success_count}/{len(tickers)} tickers")
            
        finally:
            DatabaseConnection.return_connection(conn)

    def run_universe_predictions(self, date: str) -> dict:
        """
        Generate predictions for all active stocks in the stock_universe table.

        Loads the active universe from the database and calls generate_daily_predictions
        for all tickers. Tickers that fail prediction are logged but do not abort the run.

        Args:
            date: Date in YYYY-MM-DD format to generate predictions for.

        Returns:
            Dict with keys 'success' (count) and 'failed' (list of ticker strings).
        """
        from db.connection import get_connection

        conn = get_connection()
        try:
            with conn.cursor() as cur:
                cur.execute(
                    'SELECT ticker FROM "stock-trading".stock_universe WHERE is_active = TRUE ORDER BY ticker'
                )
                rows = cur.fetchall()
            universe_tickers = [row["ticker"] for row in rows]
        except Exception as e:
            logger.error(f"Failed to load universe for batch predictions: {e}")
            return {"success": 0, "failed": []}
        finally:
            conn.close()

        if not universe_tickers:
            logger.warning("No active tickers found in stock_universe")
            return {"success": 0, "failed": []}

        logger.info(f"Running universe predictions for {len(universe_tickers)} stocks on {date}")

        success_tickers: list = []
        failed_tickers:  list = []

        for ticker in universe_tickers:
            try:
                self.generate_daily_predictions([ticker], date)
                success_tickers.append(ticker)
            except Exception as e:
                logger.error(f"Universe prediction failed for {ticker} on {date}: {e}")
                failed_tickers.append(ticker)

        result = {"success": len(success_tickers), "failed": failed_tickers}
        logger.info(
            f"Universe predictions complete: {result['success']}/{len(universe_tickers)} succeeded, "
            f"{len(failed_tickers)} failed"
        )
        return result


if __name__ == "__main__":
    # Simple CLI
    import sys
    if len(sys.argv) < 3:
        print("Usage: python prediction_generator.py <ticker> <date>")
        print("       python prediction_generator.py --universe")
        sys.exit(1)
        
    if len(sys.argv) == 3:
        ticker = sys.argv[1]
        date = sys.argv[2]
        generator = PredictionGenerator()
        generator.generate_daily_predictions([ticker], date)

    elif len(sys.argv) == 2 and sys.argv[1] == "--universe":
        # Run predictions for all active universe stocks
        date_today = datetime.now().strftime("%Y-%m-%d")
        generator = PredictionGenerator()
        result = generator.run_universe_predictions(date_today)
        print(f"Universe predictions: {result}")
