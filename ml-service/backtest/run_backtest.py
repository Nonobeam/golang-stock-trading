"""
Script to run historical backtests.
"""
import sys
import os
import argparse
import logging
from datetime import datetime, timedelta
import pandas as pd

# Add parent directory to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from backtest.backtester import Backtester
from backtest.metrics import generate_metrics_report

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def main():
    parser = argparse.ArgumentParser(description='Run historical backtest')
    parser.add_argument('--ticker', type=str, required=True, help='Ticker to backtest')
    parser.add_argument('--days', type=int, default=100, help='Number of days to backtest')
    parser.add_argument('--end_date', type=str, default=None, help='End date (YYYY-MM-DD)')
    parser.add_argument('--retrain_days', type=int, default=30, help='Retraining interval (days)')
    parser.add_argument('--output', type=str, default='backtest_results.csv', help='Output CSV file')
    
    args = parser.parse_args()
    
    # Calculate dates
    if args.end_date:
        end_date = args.end_date
    else:
        end_date = datetime.now().strftime('%Y-%m-%d')
        
    end_dt = datetime.strptime(end_date, '%Y-%m-%d')
    start_dt = end_dt - timedelta(days=args.days)
    start_date = start_dt.strftime('%Y-%m-%d')
    
    logger.info(f"Running backtest for {args.ticker}")
    logger.info(f"Period: {start_date} -> {end_date} ({args.days} days)")
    
    try:
        backtester = Backtester(args.ticker)
        results = backtester.walk_forward_backtest(
            start_date=start_date,
            end_date=end_date,
            retrain_days=args.retrain_days
        )
        
        if results.empty:
            logger.error("Backtest produced no results")
            sys.exit(1)
            
        # Metrics
        metrics = generate_metrics_report(results)
        
        print("\n" + "="*50)
        print(f"BACKTEST RESULTS: {args.ticker}")
        print("="*50)
        print(f"Period:       {start_date} to {end_date}")
        print(f"Trading Days: {len(results)}")
        print(f"Win Rate:     {metrics.get('win_rate', 0):.2%}")
        print(f"IC:           {metrics.get('ic', 0):.4f}")
        print(f"Sharpe Ratio: {metrics.get('sharpe_ratio', 0):.4f}")
        print(f"Max Drawdown: {metrics.get('max_drawdown', 0):.2%}")
        print(f"Total Return: {metrics.get('total_return', 0):.2%}")
        print(f"MAE:          {metrics.get('mae', 0):.6f}")
        print("="*50 + "\n")
        
        # Save results
        results.to_csv(args.output, index=False)
        logger.info(f"Results saved to {args.output}")
        
    except Exception as e:
        logger.error(f"Backtest failed: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

if __name__ == "__main__":
    main()
