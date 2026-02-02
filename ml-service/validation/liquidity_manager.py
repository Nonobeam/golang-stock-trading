"""
Liquidity Management Module

Prevents execution slippage by enforcing position size limits based on
daily trading volume. Implements Vietnamese market liquidity constraints.

Position Cap Formula:
    Max Shares = Average Volume (20 days) × 0.01 (1%)

Author: ML Trading System
Created: 2026-02-02
"""

import numpy as np
import pandas as pd
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Tuple
import psycopg

from config import DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD, DB_SCHEMA


# Liquidity scoring thresholds
LIQUIDITY_TIERS = {
    10: 5_000_000,    # Highly liquid (> 5M shares/day)
    7: 1_000_000,     # Very liquid (1M - 5M)
    5: 500_000,       # Liquid (500K - 1M)
    3: 250_000,       # Moderately liquid (250K - 500K)
    1: 100_000        # Low liquidity (100K - 250K)
    # < 100K = untradeable, excluded from universe
}


class LiquidityManager:
    """
    Manage liquidity constraints for position sizing.
    
    Functions:
    - Calculate 20-day average volume
    - Enforce 1% position cap
    - Score liquidity from 1-10
    - Filter untradeable stocks (< 100K volume)
    - Recommend execution strategies for large orders
    """
    
    def __init__(self, db_conn: Optional[psycopg.Connection] = None):
        """
        Initialize liquidity manager.
        
        Args:
            db_conn: Optional database connection
        """
        self.db_conn = db_conn
        self._own_connection = db_conn is None
        self.volume_cache = {}  # Cache for average volumes
    
    def _get_connection(self):
        """Get or create database connection."""
        if self.db_conn is None:
            self.db_conn = psycopg.connect(
                host=DB_HOST,
                port=DB_PORT,
                dbname=DB_NAME,
                user=DB_USER,
                password=DB_PASSWORD
            )
        return self.db_conn
    
    def __enter__(self):
        self._get_connection()
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        if self._own_connection and self.db_conn:
            self.db_conn.close()
    
    def get_average_volume(self, ticker: str, lookback_days: int = 20) -> float:
        """
        Calculate average daily trading volume over lookback period.
        
        Args:
            ticker: Stock symbol
            lookback_days: Number of days to average (default 20)
            
        Returns:
            Average daily volume in shares
        """
        # Check cache first
        cache_key = f"{ticker}_{lookback_days}"
        if cache_key in self.volume_cache:
            return self.volume_cache[cache_key]
        
        conn = self._get_connection()
        cursor = conn.cursor()
        
        cursor.execute(f"""
            SELECT AVG(volume) as avg_volume
            FROM (
                SELECT volume
                FROM "{DB_SCHEMA}".daily_bars
                WHERE ticker = %s
                ORDER BY date DESC
                LIMIT %s
            ) recent
        """, (ticker, lookback_days))
        
        result = cursor.fetchone()
        avg_volume = float(result[0]) if result and result[0] else 0
        
        # Cache result
        self.volume_cache[cache_key] = avg_volume
        
        return avg_volume
    
    def calculate_position_cap(self, ticker: str, 
                                requested_shares: int,
                                lookback_days: int = 20) -> Dict:
        """
        Calculate maximum allowed position size based on liquidity.
        
        Position cap = 1% of average daily volume to minimize market impact.
        
        Args:
            ticker: Stock symbol
            requested_shares: Desired position size
            lookback_days: Days for volume average
            
        Returns:
            Dictionary with:
            - avg_volume: 20-day average volume
            - max_shares: Position cap (1% of volume)
            - recommended_shares: Min of requested and capped
            - is_capped: Whether position was reduced
            - warning: Warning message if capped
        """
        avg_volume = self.get_average_volume(ticker, lookback_days)
        max_shares = int(avg_volume * 0.01)  # 1% of daily volume
        
        is_capped = requested_shares > max_shares
        recommended_shares = min(requested_shares, max_shares)
        
        warning = None
        if is_capped:
            warning = (f"Liquidity cap: reduced from {requested_shares:,.0f} to "
                      f"{recommended_shares:,.0f} shares (1% of {avg_volume:,.0f} avg volume)")
        
        return {
            'ticker': ticker,
            'avg_volume': avg_volume,
            'max_shares': max_shares,
            'requested_shares': requested_shares,
            'recommended_shares': recommended_shares,
            'is_capped': is_capped,
            'warning': warning
        }
    
    def get_liquidity_score(self, ticker: str) -> int:
        """
        Assign liquidity score from 1-10 based on average volume.
        
        Scoring:
        - 10: > 5M shares/day (VNM, HPG, VIC)
        - 7: 1M - 5M shares/day
        - 5: 500K - 1M shares/day
        - 3: 250K - 500K shares/day
        - 1: 100K - 250K shares/day
        - 0: < 100K (untradeable)
        
        Args:
            ticker: Stock symbol
            
        Returns:
            Liquidity score (0-10)
        """
        avg_volume = self.get_average_volume(ticker)
        
        for score, threshold in sorted(LIQUIDITY_TIERS.items(), reverse=True):
            if avg_volume >= threshold:
                return score
        
        # Below 100K = untradeable
        return 0
    
    def filter_tradeable_universe(self, tickers: List[str], 
                                   min_volume: int = 100_000) -> Tuple[List[str], List[str]]:
        """
        Filter out illiquid stocks from trading universe.
        
        Args:
            tickers: List of stock symbols
            min_volume: Minimum average daily volume (default 100K)
            
        Returns:
            Tuple of (tradeable_tickers, excluded_tickers)
        """
        tradeable = []
        excluded = []
        
        for ticker in tickers:
            avg_volume = self.get_average_volume(ticker)
            if avg_volume >= min_volume:
                tradeable.append(ticker)
            else:
                excluded.append(ticker)
        
        return tradeable, excluded
    
    def recommend_execution_strategy(self, ticker: str, 
                                      total_shares: int,
                                      time_horizon_hours: int = 3) -> List[Dict]:
        """
        Recommend execution strategy for large orders to minimize market impact.
        
        Strategy: Split order into smaller chunks over time if position
        exceeds 1% of daily volume.
        
        Args:
            ticker: Stock symbol
            total_shares: Total shares to trade
            time_horizon_hours: Hours to spread execution (default 3)
            
        Returns:
            List of execution steps with recommended shares and timing
        """
        cap_info = self.calculate_position_cap(ticker, total_shares)
        
        if not cap_info['is_capped']:
            # Small order, execute immediately
            return [{
                'step': 1,
                'shares': total_shares,
                'time_offset_minutes': 0,
                'rationale': 'Order within liquidity cap, execute immediately'
            }]
        
        # Large order - split into 10 equal chunks over time window
        max_shares_per_order = cap_info['max_shares']
        num_orders = int(np.ceil(total_shares / max_shares_per_order))
        
        # Cap at 10 orders maximum
        if num_orders > 10:
            num_orders = 10
            shares_per_order = total_shares // num_orders
        else:
            shares_per_order = max_shares_per_order
        
        time_interval_minutes = (time_horizon_hours * 60) // num_orders
        
        execution_plan = []
        remaining_shares = total_shares
        
        for i in range(num_orders):
            shares_this_order = min(shares_per_order, remaining_shares)
            
            execution_plan.append({
                'step': i + 1,
                'shares': shares_this_order,
                'time_offset_minutes': i * time_interval_minutes,
                'rationale': f'Split large order to minimize market impact ({num_orders} total orders)'
            })
            
            remaining_shares -= shares_this_order
        
        return execution_plan
    
    def get_liquidity_report(self, tickers: List[str]) -> pd.DataFrame:
        """
        Generate liquidity report for multiple stocks.
        
        Args:
            tickers: List of stock symbols
            
        Returns:
            DataFrame with liquidity metrics
        """
        report_data = []
        
        for ticker in tickers:
            avg_volume = self.get_average_volume(ticker)
            liquidity_score = self.get_liquidity_score(ticker)
            max_position = int(avg_volume * 0.01)
            
            report_data.append({
                'ticker': ticker,
                'avg_volume_20d': avg_volume,
                'liquidity_score': liquidity_score,
                'max_position_shares': max_position,
                'tradeable': avg_volume >= 100_000
            })
        
        return pd.DataFrame(report_data).sort_values('avg_volume_20d', ascending=False)


if __name__ == '__main__':
    # Example usage
    manager = LiquidityManager()
    
    with manager:
        print("Liquidity Management Examples")
        print("=" * 60)
        
        # Example 1: Check VCI liquidity
        print("\n1. VCI Liquidity Analysis:")
        avg_vol = manager.get_average_volume('VCI')
        score = manager.get_liquidity_score('VCI')
        print(f"   Average Volume: {avg_vol:,.0f} shares/day")
        print(f"   Liquidity Score: {score}/10")
        
        # Example 2: Position cap
        print("\n2. Position Cap for 10,000 VCI shares:")
        cap = manager.calculate_position_cap('VCI', 10_000)
        print(f"   Requested: {cap['requested_shares']:,.0f}")
        print(f"   Max Allowed: {cap['max_shares']:,.0f}")
        print(f"   Recommended: {cap['recommended_shares']:,.0f}")
        if cap['warning']:
            print(f"   Warning: {cap['warning']}")
        
        # Example 3: Execution strategy
        print("\n3. Execution Strategy for Large Order:")
        strategy = manager.recommend_execution_strategy('VCI', 50_000, time_horizon_hours=2)
        for step in strategy:
            print(f"   Step {step['step']}: {step['shares']:,.0f} shares at T+{step['time_offset_minutes']}min")
            print(f"      {step['rationale']}")
