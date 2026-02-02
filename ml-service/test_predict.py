#!/usr/bin/env python3
"""
Quick test script to debug model loading without gRPC.

Usage:
    python test_predict.py VCI
    python test_predict.py HPG
"""

import sys
import os
from datetime import datetime

# Add current directory to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from inference.predictor import Predictor
from utils.logging_config import setup_logging

# Configure logging
logger = setup_logging("test_predict")

def main():
    if len(sys.argv) < 2:
        print("Usage: python test_predict.py <TICKER>")
        print("Example: python test_predict.py VCI")
        sys.exit(1)
    
    ticker = sys.argv[1].upper()
    date = datetime.now().strftime("%Y-%m-%d")
    
    logger.info("="*60)
    logger.info(f"Testing prediction for {ticker} on {date}")
    logger.info("="*60)
    
    try:
        # Create predictor
        predictor = Predictor()
        logger.info("✓ Predictor created")
        
        # Try to load models
        logger.info(f"\nAttempting to load models for {ticker}...")
        success = predictor.load_production_models(ticker)
        
        if not success:
            logger.error(f"\n✗ Failed to load models for {ticker}")
            logger.error("Check the logs above for detailed path information")
            sys.exit(1)
        
        logger.info(f"\n✓ Successfully loaded models for {ticker}")
        
        # Try prediction
        logger.info(f"\nAttempting prediction...")
        p10, p50, p90, confidence = predictor.predict_for_date(ticker, date)
        
        # Display results
        logger.info("\n" + "="*60)
        logger.info("PREDICTION RESULTS")
        logger.info("="*60)
        logger.info(f"Ticker: {ticker}")
        logger.info(f"Date: {date}")
        logger.info(f"p10 (pessimistic): {p10:+.4f} ({p10*100:+.2f}%)")
        logger.info(f"p50 (expected):    {p50:+.4f} ({p50*100:+.2f}%)")
        logger.info(f"p90 (optimistic):  {p90:+.4f} ({p90*100:+.2f}%)")
        logger.info(f"Confidence: {confidence:.2%}")
        logger.info("="*60)
        logger.info("✓ Test completed successfully")
        
    except Exception as e:
        logger.exception(f"\n✗ Test failed with error: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
