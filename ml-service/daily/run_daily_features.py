"""
Script to run daily feature updates for all active tickers.
"""
import sys
import os
import argparse
from datetime import datetime
import logging

# Add parent directory to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from data.loader import DataLoader
from daily.feature_updater import FeatureUpdater

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def main():
    parser = argparse.ArgumentParser(description='Run daily feature updates')
    parser.add_argument('--date', type=str, help='Target date (YYYY-MM-DD)', default=None)
    args = parser.parse_args()
    # Default to all dates if none provided
    dates_to_process = []
    if args.date:
        dates_to_process = [args.date]
        logger.info(f"Starting feature update for specific date: {args.date}")
    else:
        dates_to_process = DataLoader.get_all_distinct_dates()
        logger.info(f"No date provided. Starting feature update for all {len(dates_to_process)} distinct dates in database.")
        
    try:
        for target_date in dates_to_process:
            logger.info(f"Processing features for {target_date}...")
            tickers = DataLoader.get_active_tickers(target_date)
            
            if not tickers:
                logger.warning(f"No active tickers found for {target_date}, skipping.")
                continue
                
            FeatureUpdater.update_all_tickers(tickers, target_date)
            
        logger.info("Daily feature update completed for all target dates")
        
    except Exception as e:
        logger.error(f"Daily update failed: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
