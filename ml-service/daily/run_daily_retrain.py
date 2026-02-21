"""
Script to run daily model retraining checks for all active tickers.
"""
import sys
import os
import argparse
import logging
from datetime import datetime

# Add parent directory to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from data.loader import DataLoader
from daily.retrainer import Retrainer
from monitoring.alerter import alerter

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def main():
    parser = argparse.ArgumentParser(description='Run daily model retraining checks')
    parser.add_argument('--date', type=str, help='Target date (YYYY-MM-DD)', default=None)
    parser.add_argument('--force', action='store_true', help='Force retraining regardless of schedule')
    args = parser.parse_args()
    
    # Default to today
    target_date = args.date if args.date else datetime.now().strftime('%Y-%m-%d')
    logger.info(f"Starting daily retraining checks for {target_date}")
    
    try:
        tickers = DataLoader.get_active_tickers(target_date)
        logger.info(f"Found {len(tickers)} active tickers")
        
        retrainer = Retrainer()
        retrained_count = 0
        
        for ticker in tickers:
            try:
                should = args.force or retrainer.should_retrain(ticker)
                
                if should:
                    logger.info(f"Retraining models for {ticker}...")
                    if retrainer.retrain_models(ticker):
                        retrained_count += 1
                
            except Exception as e:
                logger.error(f"Error processing {ticker}: {e}")
                continue
        
        logger.info("Daily retraining checks completed")
        alerter.send_alert(f"Daily ML Retraining completed.\nChecked {len(tickers)} tickers. Retrained {retrained_count} models.\nTarget Date: {target_date}", level="INFO")
        
    except Exception as e:
        logger.error(f"Daily retraining failed: {e}")
        alerter.send_alert(f"Daily ML Retraining failed for {target_date}.\nError: {e}", level="CRITICAL")
        sys.exit(1)

if __name__ == "__main__":
    main()
