"""
Daily feature updater module.
Handles calculating and updating features for daily operations.
"""

import pandas as pd
from datetime import datetime, timedelta
from typing import List, Optional
import logging
import sys
import os

# Add parent directory to path to allow importing from other modules
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from db.connection import DatabaseConnection
from data.loader import DataLoader
from features.calculator import get_all_features, get_feature_dict, validate_features
from utils.logging_config import setup_logging
from monitoring.metrics import track_latency
from monitoring.alerter import alerter

# Configure logging
logger = setup_logging("feature_updater")

class FeatureUpdater:
    """
    Handles daily feature updates for tickers.
    """
    
    @staticmethod
    @track_latency
    def update_features_for_date(ticker: str, date: str) -> bool:
        """
        Calculate and store features for a specific ticker and date.
        
        Args:
            ticker: Stock ticker symbol
            date: Target date in YYYY-MM-DD format
            
        Returns:
            bool: True if successful, False otherwise
        """
        try:
            # We need enough history for the longest window (200-day SMA)
            # Fetching 1.5 years (550 days) to be safe with trading days vs calendar days
            target_date = datetime.strptime(date, '%Y-%m-%d')
            start_date = (target_date - timedelta(days=550)).strftime('%Y-%m-%d')
            
            # Load OHLC data
            # get_daily_bars returns data inclusive of start and end date
            df = DataLoader.load_daily_bars(ticker, start_date, date)
            
            if df.empty:
                logger.warning(f"No data found for {ticker} ending {date}")
                return False
                
            # Check if the last row is indeed the target date
            # It might not be if the market was closed on 'date'
            last_date = df.iloc[-1]['date']
            if isinstance(last_date, pd.Timestamp):
                last_date_str = last_date.strftime('%Y-%m-%d')
            else:
                last_date_str = str(last_date)
                
            if last_date_str != date:
                logger.warning(f"Data available up to {last_date_str}, but requested {date}")
                # We stop here because we strictly requested features for 'date'
                return False

            # Calculate all features
            # This computes features for the whole window, but we only need the last row
            # get_all_features expects a dataframe with OHLCV data
            df_features = get_all_features(df)
            
            # Extract features for the target date (last row) using get_feature_dict
            # get_feature_dict handles NaN/None conversion
            feature_dict = get_feature_dict(df_features, -1)
            
            # Validate features
            # Create a localized dataframe for validation of just the last row
            last_row_df = df_features.iloc[[-1]] 
            valid, missing = validate_features(last_row_df)
            
            if not valid:
                msg = f"Missing features for {ticker} on {date}: {missing}"
                logger.error(msg)
                alerter.send_alert(msg, level="WARNING")
                # Depending on policy, we might still save partial data or fail
                return False

            # Save to database
            # feature_version allows us to track calculation logic changes
            DataLoader.save_features(ticker, date, feature_dict, feature_version='v1.0')
            logger.info(f"Successfully updated features for {ticker} on {date}")
            return True
            
        except Exception as e:
            logger.error(f"Error updating features for {ticker} on {date}: {e}")
            return False

    @staticmethod
    def update_all_tickers(tickers: List[str], date: str) -> None:
        """
        Update features for a list of tickers for a specific date.
        
        Args:
            tickers: List of ticker symbols
            date: Target date in YYYY-MM-DD format
        """
        success_count = 0
        for ticker in tickers:
            if FeatureUpdater.update_features_for_date(ticker, date):
                success_count += 1
        
        logger.info(f"Updated features for {success_count}/{len(tickers)} tickers on {date}")

if __name__ == "__main__":
    # Simple CLI for testing
    import sys
    if len(sys.argv) < 3:
        print("Usage: python feature_updater.py <ticker> <date>")
        sys.exit(1)
        
    ticker = sys.argv[1]
    date = sys.argv[2]
    
    FeatureUpdater.update_features_for_date(ticker, date)
