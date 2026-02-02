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
    
    # Default to today
    if args.date:
        target_date = args.date
    else:
        target_date = datetime.now().strftime('%Y-%m-%d')
        
    logger.info(f"Starting feature update for {target_date}")
    
    try:
        tickers = DataLoader.get_active_tickers()
        logger.info(f"Found {len(tickers)} active tickers")
        
        FeatureUpdater.update_all_tickers(tickers, target_date)
        
        logger.info("Daily feature update completed")
        
    except Exception as e:
        logger.error(f"Daily update failed: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
