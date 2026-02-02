"""
Script to run daily outcome recording.
"""
import sys
import os
import argparse
from datetime import datetime
import logging

# Add parent directory to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from daily.outcome_recorder import OutcomeRecorder

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def main():
    parser = argparse.ArgumentParser(description='Run daily outcome recording')
    parser.add_argument('--date', type=str, help='Target date limit (YYYY-MM-DD)', default=None)
    args = parser.parse_args()
    
    if args.date:
        target_date = args.date
    else:
        target_date = datetime.now().strftime('%Y-%m-%d')
        
    logger.info(f"Starting outcome recording up to {target_date}")
    
    try:
        recorder = OutcomeRecorder()
        recorder.record_actual_outcomes(target_date)
        logger.info("Daily outcome recording completed")
        
    except Exception as e:
        logger.error(f"Outcome recording failed: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
