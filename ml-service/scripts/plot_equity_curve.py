#!/usr/bin/env python3
"""
Plot Portfolio Equity Curve

Visualizes equity curve, peak equity, and drawdown periods.
Requires matplotlib.

Usage:
    python plot_equity_curve.py --user-id 1 --days 90
"""

import sys
import os
import argparse
from datetime import datetime, timedelta

# Add parent to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)) + '/..')

from validation.portfolio_metrics import PortfolioEquityTracker


def plot_equity_curve(user_id: int, days: int = 90, output_path: str = None):
    """
    Generate equity curve visualization.
    
    Args:
        user_id: User identifier
        days: Number of days to plot
        output_path: Output file path (default: logs/equity_curve_{user_id}.png)
    """
    try:
        import matplotlib.pyplot as plt
        import matplotlib.dates as mdates
    except ImportError:
        print("Error: matplotlib not installed. Install with: pip install matplotlib")
        return
    
    # Get equity history
    tracker = PortfolioEquityTracker()
    history = tracker.get_equity_history(user_id, days)
    
    if not history:
        print(f"No equity data found for user {user_id}")
        return
    
    # Extract data
    dates = [h['date'] for h in history]
    equity = [h['total_equity'] for h in history]
    peak = [h['peak_equity'] for h in history]
    drawdown = [h['drawdown'] * 100 for h in history]  # Convert to percentage
    
    # Create figure with two subplots
    fig, (ax1, ax2) = plt.subplots(2, 1, figsize=(14, 10), sharex=True)
    
    # Plot 1: Equity Curve
    ax1.plot(dates, equity, label='Total Equity', linewidth=2, color='#2E86AB')
    ax1.plot(dates, peak, label='Peak Equity', linewidth=1.5, linestyle='--', color='#06A77D', alpha=0.7)
    ax1.axhline(y=peak[-1] * 0.90, color='orange', linestyle=':', label='-10% Warning', alpha=0.5)
    ax1.axhline(y=peak[-1] * 0.85, color='red', linestyle=':', label='-15% Critical', alpha=0.5)
    
    ax1.set_ylabel('Equity (VND)', fontsize=12)
    ax1.set_title(f'Portfolio Equity Curve - User {user_id}', fontsize=14, fontweight='bold')
    ax1.legend(loc='best')
    ax1.grid(True, alpha=0.3)
    ax1.yaxis.set_major_formatter(plt.FuncFormatter(lambda x, p: f'{x/1e6:.1f}M'))
    
    # Plot 2: Drawdown
    colors = ['red' if dd < -10 else 'orange' if dd < -5 else 'green' for dd in drawdown]
    ax2.bar(dates, drawdown, color=colors, alpha=0.7, label='Drawdown')
    ax2.axhline(y=-10, color='orange', linestyle='--', label='-10% Threshold', alpha=0.7)
    ax2.axhline(y=-15, color='red', linestyle='--', label='-15% Threshold', alpha=0.7)
    
    ax2.set_xlabel('Date', fontsize=12)
    ax2.set_ylabel('Drawdown (%)', fontsize=12)
    ax2.set_title('Drawdown from Peak', fontsize=12, fontweight='bold')
    ax2.legend(loc='best')
    ax2.grid(True, alpha=0.3)
    
    # Format x-axis
    ax2.xaxis.set_major_formatter(mdates.DateFormatter('%Y-%m-%d'))
    ax2.xaxis.set_major_locator(mdates.WeekdayLocator(interval=2))
    plt.xticks(rotation=45)
    
    # Tight layout
    plt.tight_layout()
    
    # Save figure
    if not output_path:
        os.makedirs('logs', exist_ok=True)
        output_path = f'logs/equity_curve_{user_id}.png'
    
    plt.savefig(output_path, dpi=150, bbox_inches='tight')
    print(f"✅ Equity curve saved to {output_path}")
    print(f"📊 Latest equity: {equity[-1]:,.0f} VND")
    print(f"📈 Peak equity: {peak[-1]:,.0f} VND")
    print(f"📉 Current drawdown: {drawdown[-1]:.2f}%")
    
    plt.close()


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Plot portfolio equity curve')
    parser.add_argument('--user-id', type=int, default=1, help='User ID')
    parser.add_argument('--days', type=int, default=90, help='Number of days to plot')
    parser.add_argument('--output', type=str, default=None, help='Output file path')
    
    args = parser.parse_args()
    plot_equity_curve(args.user_id, args.days, args.output)
