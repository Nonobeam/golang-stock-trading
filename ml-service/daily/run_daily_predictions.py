"""
Script to run daily predictions for all active tickers.
"""
import sys
import os
import argparse
from datetime import datetime
import logging

# Add parent directory to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from data.loader import DataLoader
from daily.prediction_generator import PredictionGenerator
from monitoring.alerter import alerter

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def main():
    parser = argparse.ArgumentParser(description='Run daily predictions')
    parser.add_argument('--date', type=str, help='Target date (YYYY-MM-DD)', default=None)
    args = parser.parse_args()
    
    # Default to all dates if none provided
    dates_to_process = []
    if args.date:
        dates_to_process = [args.date]
        logger.info(f"Starting prediction generation for specific date: {args.date}")
    else:
        dates_to_process = DataLoader.get_all_distinct_dates()
        logger.info(f"No date provided. Starting prediction generation for all {len(dates_to_process)} distinct dates in database.")
        
    try:
        generator = PredictionGenerator()
        total_processed_dates = 0
        
        for target_date in dates_to_process:
            logger.info(f"Processing predictions for {target_date}...")
            tickers = DataLoader.get_active_tickers(target_date)
            
            if not tickers:
                logger.warning(f"No active tickers found for {target_date}, skipping.")
                continue
                
            generator.generate_daily_predictions(tickers, target_date)
            total_processed_dates += 1
            
        logger.info(f"Daily predictions completed for {total_processed_dates} dates")
        if total_processed_dates > 0:
            alerter.send_alert(f"Daily Predictions generation completed for {total_processed_dates} dates.", level="INFO")
        
    except Exception as e:
        logger.error(f"Daily predictions failed: {e}")
        alerter.send_alert(f"Daily Predictions failed.\nError: {e}", level="CRITICAL")
        sys.exit(1)

if __name__ == "__main__":
    main()
