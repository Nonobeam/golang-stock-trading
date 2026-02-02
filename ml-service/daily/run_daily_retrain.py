"""
Script to run daily model retraining checks for all active tickers.
"""
import sys
import os
import argparse
import logging

# Add parent directory to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from data.loader import DataLoader
from daily.retrainer import Retrainer

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def main():
    parser = argparse.ArgumentParser(description='Run daily model retraining checks')
    parser.add_argument('--force', action='store_true', help='Force retraining regardless of schedule')
    args = parser.parse_args()
    
    logger.info("Starting daily retraining checks")
    
    try:
        tickers = DataLoader.get_active_tickers()
        logger.info(f"Found {len(tickers)} active tickers")
        
        retrainer = Retrainer()
        
        for ticker in tickers:
            try:
                should = args.force or retrainer.should_retrain(ticker)
                
                if should:
                    logger.info(f"Retraining models for {ticker}...")
                    retrainer.retrain_models(ticker)
                
            except Exception as e:
                logger.error(f"Error processing {ticker}: {e}")
                continue
        
        logger.info("Daily retraining checks completed")
        
    except Exception as e:
        logger.error(f"Daily retraining failed: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
