#!/usr/bin/env python3
"""
Daily Position Monitoring and Signal Generation

Monitors all active positions and generates position-aware trading signals.
Checks stop-loss and target levels, calculates unrealized P&L, and produces
a comprehensive daily report.

Usage:
    python daily_signals.py [--user-id USER_ID] [--output FILEPATH]
    
Example:
    python daily_signals.py --user-id 1 --output reports/daily_2026-02-02.txt
"""

import sys
import os
import argparse
import logging
from datetime import datetime, date, timedelta
from typing import Dict, List
import json

# Add ml-service to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)) + '/..')

from monitoring.alerter import alerter

from db.connection import get_connection
from position_manager.manager import PositionManager
from signals.generator import SignalGenerator

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("daily_signals")


def get_current_price(conn, ticker: str, target_date: str = None) -> float:
    """
    Fetch current/latest market price for a ticker.
    
    Args:
        conn: Database connection
        ticker: Stock symbol
        target_date: Optional date (defaults to today)
        
    Returns:
        Close price or None if not available
    """
    try:
        with conn.cursor() as cursor:
            if target_date:
                cursor.execute("""
                    SELECT close
                    FROM daily_bars
                    WHERE ticker = %(ticker)s 
                      AND date <= %(date)s
                    ORDER BY date DESC
                    LIMIT 1
                """, {'ticker': ticker, 'date': target_date})
            else:
                cursor.execute("""
                    SELECT close
                    FROM daily_bars
                    WHERE ticker = %(ticker)s
                    ORDER BY date DESC
                    LIMIT 1
                """, {'ticker': ticker})
            
            row = cursor.fetchone()
            return float(row[0]) if row else None
    except Exception as e:
        logger.error(f"Failed to fetch price for {ticker}: {e}")
        return None


def get_predictions(conn, ticker: str, target_date: str) -> Dict[int, Dict]:
    """
    Fetch multi-horizon predictions for a ticker.
    
    Args:
        conn: Database connection
        ticker: Stock symbol
        target_date: Prediction date
        
    Returns:
        Dictionary of horizon -> prediction dict
    """
    try:
        with conn.cursor() as cursor:
            cursor.execute("""
                SELECT horizon, p10, p50, p90, confidence
                FROM predictions
                WHERE ticker = %(ticker)s
                  AND prediction_date = %(date)s
                ORDER BY horizon
            """, {'ticker': ticker, 'date': target_date})
            
            predictions = {}
            for row in cursor.fetchall():
                horizon, p10, p50, p90, confidence = row
                predictions[horizon] = {
                    'p10': float(p10),
                    'p50': float(p50),
                    'p90': float(p90),
                    'confidence': float(confidence) if confidence else 0.5
                }
            
            return predictions if predictions else None
    except Exception as e:
        logger.error(f"Failed to fetch predictions for {ticker}: {e}")
        return None


def format_number(num, decimals=0):
    """Format number with thousand separators."""
    if num is None:
        return "N/A"
    return f"{num:,.{decimals}f}"


def generate_daily_report(conn, user_id: int = 1, report_date: str = None) -> str:
    """
    Generate comprehensive daily position monitoring report.
    
    Args:
        conn: Database connection
        user_id: User ID
        report_date: Optional date (defaults to today)
        
    Returns:
        Formatted report string
    """
    if not report_date:
        report_date = date.today().isoformat()
    
    logger.info(f"Generating daily report for user {user_id} on {report_date}")
    
    # ===== DRAWDOWN RISK MANAGEMENT =====
    # Check current drawdown before generating signals
    from position_sizing.drawdown_manager import DrawdownManager
    from validation.portfolio_metrics import PortfolioEquityTracker
    
    drawdown_mgr = DrawdownManager(db_connection=conn)
    drawdown_status = drawdown_mgr.get_drawdown_status(user_id)
    
    logger.info(f"Portfolio Status: {drawdown_status['risk_level']} - {drawdown_status['drawdown_percent']} drawdown")
    logger.info(f"Trading allowed: {drawdown_status['trading_allowed']} - Multiplier: {drawdown_status['multiplier']}")
    
    # Snapshot previous day's equity
    tracker = PortfolioEquityTracker(db_connection=conn)
    try:
        yesterday = date.fromisoformat(report_date) - timedelta(days=1)
        tracker.save_equity_snapshot(user_id, yesterday)
        logger.info(f"Equity snapshot saved for {yesterday}")
    except Exception as e:
        logger.warning(f"Failed to save equity snapshot: {e}")
        
    # Calculate current equity for capacity checks
    account_value = 0.0
    try:
        equity_data = tracker.calculate_current_equity(user_id)
        account_value = float(equity_data['total_equity'])
        logger.info(f"Current account value for capacity checks: {format_number(account_value)} VND")
    except Exception as e:
        logger.error(f"Failed to calculate current equity: {e}")
    
    # Initialize managers
    pm = PositionManager(conn)
    sg = SignalGenerator(user_id=user_id)
    
    # Get all active positions
    positions = pm.get_all_positions(user_id)
    
    if not positions:
        return f"No active positions for user {user_id} on {report_date}"
    
    # Build report
    report_lines = []
    report_lines.append("=" * 100)
    report_lines.append(f"DAILY POSITION REPORT - {report_date}")
    report_lines.append(f"User ID: {user_id} | Active Positions: {len(positions)}")
    report_lines.append(f"Total Equity: {format_number(account_value)} VND")
    
    # Add drawdown risk status
    report_lines.append(f"Portfolio Status: {drawdown_status['risk_level']} | Drawdown: {drawdown_status['drawdown_percent']} | Position Sizing: {int(drawdown_status['multiplier']*100)}%")
    report_lines.append("=" * 100)
    report_lines.append("")
    
    alerts = []
    
    for pos in positions:
        ticker = pos['symbol']
        report_lines.append(f"{'━' * 100}")
        report_lines.append(f"{ticker}")
        report_lines.append(f"{'━' * 100}")
        
        # Position details
        entry_price = pos['entry_price']
        quantity = pos['quantity']
        entry_date = pos['entry_date']
        
        report_lines.append(f"Entry: {format_number(entry_price)} VND × {quantity} shares on {entry_date}")
        
        # Get current price
        current_price = get_current_price(conn, ticker, report_date)
        
        if current_price:
            # Calculate unrealized P&L
            unrealized_pnl = quantity * (current_price - entry_price)
            unrealized_pnl_pct = ((current_price - entry_price) / entry_price) * 100
            
            report_lines.append(f"Current: {format_number(current_price)} VND")
            report_lines.append(
                f"Unrealized P&L: {format_number(unrealized_pnl)} VND "
                f"({unrealized_pnl_pct:+.2f}%)"
            )
            
            # Risk management levels
            stop_loss = pos.get('stop_loss')
            target_1 = pos.get('target_1')
            target_2 = pos.get('target_2')
            target_3 = pos.get('target_3')
            
            report_lines.append(f"\nRisk Management:")
            
            # Stop-loss check
            if stop_loss:
                distance_to_stop = ((current_price - stop_loss) / current_price) * 100
                status = "OK" if current_price > stop_loss else "TRIGGERED"
                
                if current_price <= stop_loss:
                    alerts.append(f"{ticker}: STOP LOSS TRIGGERED at {format_number(stop_loss)}")
                
                report_lines.append(
                    f"  Stop Loss: {format_number(stop_loss)} VND ({distance_to_stop:+.2f}% from current) {status}"
                )
            
            # Target checks
            report_lines.append(f"\nProfit Targets:")
            for target_num, target_price in enumerate([target_1, target_2, target_3], 1):
                if target_price:
                    distance = ((target_price - current_price) / current_price) * 100
                    if current_price >= target_price:
                        status = "REACHED"
                        alerts.append(f"{ticker}: Target {target_num} REACHED at {format_number(target_price)}")
                    else:
                        status = f"({distance:+.2f}% away)"
                    
                    report_lines.append(f"  T{target_num}: {format_number(target_price)} VND {status}")

            # Capacity Check
            if account_value > 0:
                report_lines.append(f"\nCapacity Status:")
                capacity = pm.check_buying_capacity(ticker, current_price, account_value, user_id)
                
                if capacity['at_limit']:
                    report_lines.append(f"  AT LIMIT: {capacity['limit_reason']}")
                    report_lines.append(f"  Max Buying Capability: 0 shares")
                else:
                    max_shares = capacity['max_buyable_shares']
                    rem_val = capacity['remaining_value_capacity']
                    report_lines.append(f"  Within Limits")
                    report_lines.append(f"  Buying Capacity: {max_shares} shares (~{format_number(rem_val)} VND)")
                    
                # Show utilization
                alloc_pct = (capacity['current_position_value'] / account_value) * 100
                alloc_limit = 20.0 # 20% limit
                
                status_note = ""
                if alloc_pct >= 18.0 and not capacity['at_limit']:
                    status_note = " (NEAR LIMIT)"
                    
                report_lines.append(f"  Portfolio Allocation: {alloc_pct:.1f}% / {alloc_limit}%{status_note}")

        else:
            report_lines.append(f"Current price not available")
            current_price = None
        
        # Get predictions
        predictions = get_predictions(conn, ticker, report_date)
        
        if predictions:
            report_lines.append(f"\nML Predictions:")
            for horizon in sorted(predictions.keys()):
                pred = predictions[horizon]
                report_lines.append(
                    f"  {horizon}d: {pred['p50']:+.1%} return "
                    f"(confidence: {pred['confidence']:.0%}, risk: {pred['p10']:+.1%} to {pred['p90']:+.1%})"
                )
            
            # Generate signal
            try:
                signal_dict, save_success = sg.generate_and_save_signal(
                    ticker=ticker,
                    predictions=predictions,
                    date=report_date,
                    current_price=current_price,
                    db_connection=conn,
                    user_id=user_id,
                    account_value=account_value
                )
                
                signal = signal_dict['signal']
                strength = signal_dict['strength']
                reason = signal_dict['reason']
                
                report_lines.append(f"\nSIGNAL: {signal} (strength: {strength:.2f})")
                report_lines.append(f"Reason: {reason}")
                
                # Add to alerts if action signal
                if signal in ['SELL', 'SELL_PARTIAL', 'BUY_MORE']:
                    alerts.append(f"{ticker}: {signal} - {reason}")
                
            except Exception as e:
                logger.error(f"Failed to generate signal for {ticker}: {e}")
                report_lines.append(f"\nSignal generation failed: {str(e)}")
        else:
            report_lines.append(f"\nPredictions not available for {report_date}")
            alerts.append(f"{ticker}: No predictions available")
        
        report_lines.append("")
    
    # Summary section
    report_lines.append("=" * 100)
    report_lines.append("ALERTS SUMMARY")
    report_lines.append("=" * 100)
    
    if alerts:
        for alert in alerts:
            report_lines.append(alert)
    else:
        report_lines.append("No alerts - all positions within normal parameters")
    
    report_lines.append("")
    report_lines.append("=" * 100)
    report_lines.append(f"Report generated at {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    report_lines.append("=" * 100)
    
    return "\n".join(report_lines)


def main():
    """Main entry point for daily signals script."""
    parser = argparse.ArgumentParser(description='Generate daily position monitoring report')
    parser.add_argument('--user-id', type=int, default=1, help='User ID (default: 1)')
    parser.add_argument('--date', type=str, default=None, help='Report date (YYYY-MM-DD, default: today)')
    parser.add_argument('--output', type=str, default=None, help='Output file path (default: print to stdout)')
    
    args = parser.parse_args()
    
    try:
        # Connect to database
        conn = get_connection()
        logger.info("Connected to database")
        
        # Generate report
        report = generate_daily_report(conn, args.user_id, args.date)
        
        # Output report
        if args.output:
            os.makedirs(os.path.dirname(args.output), exist_ok=True)
            with open(args.output, 'w', encoding='utf-8') as f:
                f.write(report)
            logger.info(f"Report saved to {args.output}")
            print(f"Report saved to {args.output}")
        else:
            print(report)
            
        # Send Telegram summary
        alerts_summary = _extract_alerts_summary(report)
        alerter.send_alert(f"Daily Signals generation completed for {args.date or date.today().isoformat()}.\n\n*Alerts Summary:*\n{alerts_summary}", level="INFO")
        
        conn.close()
        return 0
        
    except Exception as e:
        logger.error(f"Failed to generate daily report: {e}", exc_info=True)
        print(f"Error: {str(e)}")
        alerter.send_alert(f"Daily Signals generation failed.\nError: {e}", level="CRITICAL")
        return 1

def _extract_alerts_summary(report_text: str) -> str:
    """Extract just the alerts portion from the full report."""
    try:
        parts = report_text.split("ALERTS SUMMARY")
        if len(parts) > 1:
            alerts_section = parts[1].split("=" * 100)[1].strip()
            # Remove the "Report generated at..." footer if present
            footer_idx = alerts_section.find("Report generated at")
            if footer_idx > 0:
                alerts_section = alerts_section[:footer_idx].strip()
            return alerts_section
    except:
        pass
    return "Could not parse alerts summary."

if __name__ == '__main__':
    sys.exit(main())
