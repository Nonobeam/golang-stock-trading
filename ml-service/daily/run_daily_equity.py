#!/usr/bin/env python3
"""
Daily Equity Snapshot Script

Runs end-of-day equity snapshots and R-multiple statistics for all active users.

Usage:
    python run_daily_equity.py [--date YYYY-MM-DD]
"""

import sys
import os
import argparse
from datetime import date
import logging

# Add parent to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)) + '/..')

from db.connection import get_connection
from validation.portfolio_metrics import PortfolioEquityTracker
from validation.r_multiple_analytics import RMultipleAnalytics
from monitoring.drawdown_alerts import check_drawdown_alert

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("daily_equity")


def run_daily_equity(snapshot_date: str = None):
    """
    Execute daily equity snapshot and analytics for all users.
    
    Args:
        snapshot_date: Date for snapshot (default: today)
    """
    if not snapshot_date:
        snapshot_date = date.today()
    else:
        snapshot_date = date.fromisoformat(snapshot_date)
    
    logger.info(f"Running daily equity snapshot for {snapshot_date}")
    
    conn = get_connection()
    
    try:
        # Get all active users
        with conn.cursor() as cursor:
            cursor.execute("SELECT user_id FROM user_config")
            users = [row[0] for row in cursor.fetchall()]
        
        if not users:
            logger.warning("No users found in user_config")
            return
        
        logger.info(f"Processing {len(users)} users")
        
        # Process each user
        for user_id in users:
            logger.info(f"Processing user {user_id}")
            
            try:
                # 1. Save equity snapshot
                tracker = PortfolioEquityTracker(db_connection=conn)
                tracker.save_equity_snapshot(user_id, snapshot_date)
                logger.info(f"✅ Equity snapshot saved for user {user_id}")
                
                # 2. Calculate and save R-multiple statistics
                analytics = RMultipleAnalytics(db_connection=conn)
                analytics.save_daily_r_statistics(user_id, snapshot_date)
                logger.info(f"✅ R-multiple statistics saved for user {user_id}")
                
                # 3. Check for drawdown alerts
                check_drawdown_alert(user_id, db_connection=conn)
                logger.info(f"✅ Drawdown check completed for user {user_id}")
                
            except Exception as e:
                logger.error(f"Failed to process user {user_id}: {e}", exc_info=True)
                continue
        
        logger.info("Daily equity snapshot completed successfully")
        print(f"\n✅ Daily equity snapshot completed for {len(users)} users on {snapshot_date}")
        
    finally:
        conn.close()


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Run daily equity snapshots')
    parser.add_argument('--date', type=str, default=None, help='Snapshot date (YYYY-MM-DD, default: today)')
    
    args = parser.parse_args()
    run_daily_equity(args.date)
