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
    
    # Default to today
    if args.date:
        target_date = args.date
    else:
        target_date = datetime.now().strftime('%Y-%m-%d')
        
    logger.info(f"Starting prediction generation for {target_date}")
    
    try:
        tickers = DataLoader.get_active_tickers(target_date)
        logger.info(f"Found {len(tickers)} active tickers for date {target_date}")
        
        generator = PredictionGenerator()
        generator.generate_daily_predictions(tickers, target_date)
        
        logger.info("Daily predictions completed")
        alerter.send_alert(f"✅ Daily Predictions generation completed.\nProcessed {len(tickers)} active tickers for {target_date}.", level="INFO")
        
    except Exception as e:
        logger.error(f"Daily predictions failed: {e}")
        alerter.send_alert(f"❌ Daily Predictions failed for {target_date}.\nError: {e}", level="CRITICAL")
        sys.exit(1)

if __name__ == "__main__":
    main()
