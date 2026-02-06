#!/usr/bin/env python3
"""
Generate R-Multiple Performance Report

Creates markdown report with R-multiple statistics and analysis.

Usage:
    python generate_r_report.py --user-id 1 --days 30
"""

import sys
import os
import argparse
from datetime import date

# Add parent to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)) + '/..')

from validation.r_multiple_analytics import RMultipleAnalytics


def generate_r_report(user_id: int, lookback_days: int = 30, output_path: str = None):
    """
    Generate R-multiple performance report.
    
    Args:
        user_id: User identifier
        lookback_days: Number of days to analyze
        output_path: Output file path (default: logs/r_multiple_report_{date}.md)
    """
    analytics = RMultipleAnalytics()
    
    # Generate report
    report = analytics.generate_r_report(user_id, lookback_days)
    
    # Save to file
    if not output_path:
        os.makedirs('logs', exist_ok=True)
        output_path = f'logs/r_multiple_report_{date.today().isoformat()}.md'
    
    with open(output_path, 'w', encoding='utf-8') as f:
        f.write(report)
    
    print(f"✅ R-multiple report saved to {output_path}")
    print("\n" + "="*80)
    print(report)
    print("="*80)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Generate R-multiple performance report')
    parser.add_argument('--user-id', type=int, default=1, help='User ID')
    parser.add_argument('--days', type=int, default=30, help='Lookback period in days')
    parser.add_argument('--output', type=str, default=None, help='Output file path')
    
    args = parser.parse_args()
    generate_r_report(args.user_id, args.days, args.output)
