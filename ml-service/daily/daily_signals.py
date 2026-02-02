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
from datetime import datetime, date
from typing import Dict, List
import json

# Add ml-service to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)) + '/..')

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
    report_lines.append("=" * 100)
    report_lines.append("")
    
    alerts = []
    
    for pos in positions:
        ticker = pos['symbol']
        report_lines.append(f"{'━' * 100}")
        report_lines.append(f"📊 {ticker}")
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
            
            pnl_symbol = "📈" if unrealized_pnl >= 0 else "📉"
            report_lines.append(f"Current: {format_number(current_price)} VND")
            report_lines.append(
                f"{pnl_symbol} Unrealized P&L: {format_number(unrealized_pnl)} VND "
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
                status = "✅" if current_price > stop_loss else "🚨 TRIGGERED"
                
                if current_price <= stop_loss:
                    alerts.append(f"🚨 {ticker}: STOP LOSS TRIGGERED at {format_number(stop_loss)}")
                
                report_lines.append(
                    f"  Stop Loss: {format_number(stop_loss)} VND ({distance_to_stop:+.2f}% from current) {status}"
                )
            
            # Target checks
            report_lines.append(f"\nProfit Targets:")
            for target_num, target_price in enumerate([target_1, target_2, target_3], 1):
                if target_price:
                    distance = ((target_price - current_price) / current_price) * 100
                    if current_price >= target_price:
                        status = f"🎯 REACHED"
                        alerts.append(f"🎯 {ticker}: Target {target_num} REACHED at {format_number(target_price)}")
                    else:
                        status = f"({distance:+.2f}% away)"
                    
                    report_lines.append(f"  T{target_num}: {format_number(target_price)} VND {status}")
        else:
            report_lines.append(f"❌ Current price not available")
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
                    user_id=user_id
                )
                
                signal = signal_dict['signal']
                strength = signal_dict['strength']
                reason = signal_dict['reason']
                
                # Format signal display
                signal_emoji = {
                    'BUY_NEW': '🟢',
                    'BUY_MORE': '🟦',
                    'SELL': '🔴',
                    'SELL_PARTIAL': '🟠',
                    'HOLD': '⚪',
                    'HOLD_NONE': '⚪'
                }.get(signal, '⚪')
                
                report_lines.append(f"\n{signal_emoji} SIGNAL: {signal} (strength: {strength:.2f})")
                report_lines.append(f"Reason: {reason}")
                
                # Add to alerts if action signal
                if signal in ['SELL', 'SELL_PARTIAL', 'BUY_MORE']:
                    alerts.append(f"{signal_emoji} {ticker}: {signal} - {reason}")
                
            except Exception as e:
                logger.error(f"Failed to generate signal for {ticker}: {e}")
                report_lines.append(f"\n❌ Signal generation failed: {str(e)}")
        else:
            report_lines.append(f"\n❌ Predictions not available for {report_date}")
            alerts.append(f"⚠️ {ticker}: No predictions available")
        
        report_lines.append("")
    
    # Summary section
    report_lines.append("=" * 100)
    report_lines.append("ALERTS SUMMARY")
    report_lines.append("=" * 100)
    
    if alerts:
        for alert in alerts:
            report_lines.append(alert)
    else:
        report_lines.append("✅ No alerts - all positions within normal parameters")
    
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
            print(f"✅ Report saved to {args.output}")
        else:
            print(report)
        
        conn.close()
        return 0
        
    except Exception as e:
        logger.error(f"Failed to generate daily report: {e}", exc_info=True)
        print(f"❌ Error: {str(e)}")
        return 1


if __name__ == '__main__':
    sys.exit(main())
