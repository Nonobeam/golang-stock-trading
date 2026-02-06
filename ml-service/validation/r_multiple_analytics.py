"""
R-Multiple Analytics Module

Calculates and tracks portfolio-level R-multiple statistics for performance evaluation.
Provides methods to aggregate R-multiples, analyze distribution, and track by signal type.
"""

import logging
from datetime import date, datetime, timedelta
from decimal import Decimal
from typing import Dict, List, Optional
import json
import psycopg
import numpy as np

logger = logging.getLogger(__name__)


class RMultipleAnalytics:
    """
    Analyzes R-multiple performance at portfolio level.
    
    R-multiple = (exit_price - entry_price) / (entry_price - stop_loss)
    
    Already calculated per position in positions.r_multiple.
    This class aggregates and analyzes across all positions.
    """
    
    def __init__(self, db_connection: Optional[psycopg.Connection] = None):
        """
        Initialize analytics with optional database connection.
        
        Args:
            db_connection: Re usable psycopg connection. If None, creates new connection per operation.
        """
        self.db_connection = db_connection
    
    def _get_connection(self) -> psycopg.Connection:
        """Get database connection (reuse or create new)."""
        if self.db_connection:
            return self.db_connection
        
        from ..db.connection import get_connection
        return get_connection()
    
    def calculate_portfolio_r_stats(self, user_id: int, lookback_days: int = 30) -> Dict:
        """
        Calculate portfolio-level R-multiple statistics over lookback period.
        
        Args:
            user_id: User identifier
            lookback_days: Number of days to look back (default: 30)
        
        Returns:
            Dict with:
                - avg_r_multiple: Average R across all trades
                - median_r_multiple: Median R
                - stddev_r_multiple: Standard deviation
                - best_r_multiple: Best trade
                - worst_r_multiple: Worst trade
                - win_rate: % of R > 0
                - total_trades: Count of closed positions
                - profitable_trades: Count of R > 0
        """
        conn = self._get_connection()
        
        try:
            with conn.cursor() as cursor:
                # Get all R-multiples for closed positions in lookback period
                cursor.execute("""
                    SELECT r_multiple
                    FROM positions
                    WHERE user_id = %s
                        AND is_closed = TRUE
                        AND r_multiple IS NOT NULL
                        AND exit_date >= CURRENT_DATE - INTERVAL '%s days'
                    ORDER BY exit_date DESC
                """, (user_id, lookback_days))
                
                rows = cursor.fetchall()
                
                if not rows:
                    logger.info(f"No closed positions with R-multiple for user {user_id} in last {lookback_days} days")
                    return {
                        'avg_r_multiple': None,
                        'median_r_multiple': None,
                        'stddev_r_multiple': None,
                        'best_r_multiple': None,
                        'worst_r_multiple': None,
                        'win_rate': None,
                        'total_trades': 0,
                        'profitable_trades': 0
                    }
                
                r_multiples = [float(row[0]) for row in rows]
                r_array = np.array(r_multiples)
                
                profitable = [r for r in r_multiples if r > 0]
                
                stats = {
                    'avg_r_multiple': float(np.mean(r_array)),
                    'median_r_multiple': float(np.median(r_array)),
                    'stddev_r_multiple': float(np.std(r_array)),
                    'best_r_multiple': float(np.max(r_array)),
                    'worst_r_multiple': float(np.min(r_array)),
                    'win_rate': len(profitable) / len(r_multiples),
                    'total_trades': len(r_multiples),
                    'profitable_trades': len(profitable)
                }
                
                logger.info(f"R-stats for user {user_id}: avg={stats['avg_r_multiple']:.2f}R, win_rate={stats['win_rate']*100:.1f}%")
                
                return stats
                
        finally:
            if not self.db_connection:
                conn.close()
    
    def get_r_distribution(self, user_id: int, lookback_days: int = 90) -> Dict[str, int]:
        """
        Get R-multiple distribution histogram data.
        
        Args:
            user_id: User identifier
            lookback_days: Number of days to analyze
        
        Returns:
            Dict with bins as keys and counts as values
            Example: {'-2R to -1R': 3, '-1R to 0R': 5, '0R to 1R': 8, ...}
        """
        conn = self._get_connection()
        
        try:
            with conn.cursor() as cursor:
                cursor.execute("""
                    SELECT r_multiple
                    FROM positions
                    WHERE user_id = %s
                        AND is_closed = TRUE
                        AND r_multiple IS NOT NULL
                        AND exit_date >= CURRENT_DATE - INTERVAL '%s days'
                """, (user_id, lookback_days))
                
                r_multiples = [float(row[0]) for row in cursor.fetchall()]
                
                if not r_multiples:
                    return {}
                
                # Define bins
                bins = {
                    '< -2R': 0,
                    '-2R to -1R': 0,
                    '-1R to 0R': 0,
                    '0R to 1R': 0,
                    '1R to 2R': 0,
                    '2R to 3R': 0,
                    '> 3R': 0
                }
                
                for r in r_multiples:
                    if r < -2:
                        bins['< -2R'] += 1
                    elif r < -1:
                        bins['-2R to -1R'] += 1
                    elif r < 0:
                        bins['-1R to 0R'] += 1
                    elif r < 1:
                        bins['0R to 1R'] += 1
                    elif r < 2:
                        bins['1R to 2R'] += 1
                    elif r < 3:
                        bins['2R to 3R'] += 1
                    else:
                        bins['> 3R'] += 1
                
                return bins
                
        finally:
            if not self.db_connection:
                conn.close()
    
    def get_r_by_signal_type(self, user_id: int, lookback_days: int = 30) -> Dict[str, Dict]:
        """
        Calculate R-multiple statistics grouped by signal type.
        
        Args:
            user_id: User identifier
            lookback_days: Number of days to analyze
        
        Returns:
            Dict mapping signal_type to stats
            Example: {
                'BUY_NEW': {'avg_r': 1.8, 'trades': 12, 'win_rate': 0.75},
                'BUY_MORE': {'avg_r': 2.6, 'trades': 5, 'win_rate': 0.80}
            }
        """
        conn = self._get_connection()
        
        try:
            with conn.cursor() as cursor:
                cursor.execute("""
                    SELECT signal_type, r_multiple
                    FROM positions
                    WHERE user_id = %s
                        AND is_closed = TRUE
                        AND r_multiple IS NOT NULL
                        AND signal_type IS NOT NULL
                        AND exit_date >= CURRENT_DATE - INTERVAL '%s days'
                """, (user_id, lookback_days))
                
                # Group by signal type
                signal_groups = {}
                for signal_type, r_multiple in cursor.fetchall():
                    if signal_type not in signal_groups:
                        signal_groups[signal_type] = []
                    signal_groups[signal_type].append(float(r_multiple))
                
                # Calculate stats per signal type
                results = {}
                for signal_type, r_list in signal_groups.items():
                    profitable = [r for r in r_list if r > 0]
                    
                    results[signal_type] = {
                        'avg_r': float(np.mean(r_list)),
                        'median_r': float(np.median(r_list)),
                        'trades': len(r_list),
                        'win_rate': len(profitable) / len(r_list) if r_list else 0,
                        'best_r': float(max(r_list)),
                        'worst_r': float(min(r_list))
                    }
                
                return results
                
        finally:
            if not self.db_connection:
                conn.close()
    
    def save_daily_r_statistics(self, user_id: int, calculation_date: date) -> None:
        """
        Calculate and save daily R-multiple statistics to database.
        
        Uses 30-day lookback for statistics calculation.
        
        Args:
            user_id: User identifier
            calculation_date: Date of calculation
        """
        # Calculate stats
        stats = self.calculate_portfolio_r_stats(user_id, lookback_days=30)
        
        if stats['total_trades'] == 0:
            logger.info(f"No trades to calculate R-stats for user {user_id}")
            return
        
        # Get signal type breakdown
        r_by_signal = self.get_r_by_signal_type(user_id, lookback_days=30)
        
        conn = self._get_connection()
        
        try:
            with conn.cursor() as cursor:
                cursor.execute("""
                    INSERT INTO r_multiple_statistics (
                        user_id, calculation_date,
                        avg_r_multiple, median_r_multiple, stddev_r_multiple,
                        best_r_multiple, worst_r_multiple,
                        win_rate, total_trades, profitable_trades,
                        r_by_signal_type
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
                    )
                    ON CONFLICT (user_id, calculation_date)
                    DO UPDATE SET
                        avg_r_multiple = EXCLUDED.avg_r_multiple,
                        median_r_multiple = EXCLUDED.median_r_multiple,
                        stddev_r_multiple = EXCLUDED.stddev_r_multiple,
                        best_r_multiple = EXCLUDED.best_r_multiple,
                        worst_r_multiple = EXCLUDED.worst_r_multiple,
                        win_rate = EXCLUDED.win_rate,
                        total_trades = EXCLUDED.total_trades,
                        profitable_trades = EXCLUDED.profitable_trades,
                        r_by_signal_type = EXCLUDED.r_by_signal_type
                """, (
                    user_id, calculation_date,
                    stats['avg_r_multiple'], stats['median_r_multiple'], stats['stddev_r_multiple'],
                    stats['best_r_multiple'], stats['worst_r_multiple'],
                    stats['win_rate'], stats['total_trades'], stats['profitable_trades'],
                    json.dumps(r_by_signal)
                ))
            
            conn.commit()
            logger.info(f"R-statistics saved for user {user_id} on {calculation_date}")
            
        except Exception as e:
            conn.rollback()
            logger.error(f"Failed to save R-statistics: {e}")
            raise
        finally:
            if not self.db_connection:
                conn.close()
    
    def get_r_statistics_history(self, user_id: int, days: int = 90) -> List[Dict]:
        """
        Get historical R-multiple statistics.
        
        Args:
            user_id: User identifier
            days: Number of days to retrieve
        
        Returns:
            List of dicts with historical statistics
        """
        conn = self._get_connection()
        
        try:
            with conn.cursor() as cursor:
                cursor.execute("""
                    SELECT 
                        calculation_date, avg_r_multiple, median_r_multiple,
                        best_r_multiple, worst_r_multiple, win_rate,
                        total_trades, profitable_trades, r_by_signal_type
                    FROM r_multiple_statistics
                    WHERE user_id = %s AND calculation_date >= CURRENT_DATE - INTERVAL '%s days'
                    ORDER BY calculation_date DESC
                """, (user_id, days))
                
                history = []
                for row in cursor.fetchall():
                    history.append({
                        'date': row[0],
                        'avg_r': float(row[1]) if row[1] else None,
                        'median_r': float(row[2]) if row[2] else None,
                        'best_r': float(row[3]) if row[3] else None,
                        'worst_r': float(row[4]) if row[4] else None,
                        'win_rate': float(row[5]) if row[5] else None,
                        'total_trades': row[6],
                        'profitable_trades': row[7],
                        'by_signal_type': row[8] if row[8] else {}
                    })
                
                return history
                
        finally:
            if not self.db_connection:
                conn.close()
    
    def generate_r_report(self, user_id: int, lookback_days: int = 30) -> str:
        """
        Generate markdown report of R-multiple performance.
        
        Args:
            user_id: User identifier
            lookback_days: Number of days to analyze
        
        Returns:
            Markdown-formatted report string
        """
        stats = self.calculate_portfolio_r_stats(user_id, lookback_days)
        
        if stats['total_trades'] == 0:
            return f"# R-Multiple Report (Last {lookback_days} Days)\n\nNo closed positions found."
        
        distribution = self.get_r_distribution(user_id, lookback_days)
        by_signal = self.get_r_by_signal_type(user_id, lookback_days)
        
        report = f"""# R-Multiple Report (Last {lookback_days} Days)

## Portfolio Statistics

- **Average R**: {stats['avg_r_multiple']:+.2f}R
- **Median R**: {stats['median_r_multiple']:+.2f}R
- **Std Dev**: {stats['stddev_r_multiple']:.2f}R
- **Win Rate**: {stats['win_rate']*100:.1f}% ({stats['profitable_trades']}/{stats['total_trades']} trades)
- **Best Trade**: {stats['best_r_multiple']:+.2f}R
- **Worst Trade**: {stats['worst_r_multiple']:+.2f}R

## Distribution

"""
        
        for bin_name, count in distribution.items():
            if count > 0:
                report += f"- **{bin_name}**: {count} trades\n"
        
        report += "\n## Performance by Signal Type\n\n"
        
        for signal_type,signal_stats in sorted(by_signal.items(), key=lambda x: x[1]['avg_r'], reverse=True):
            report += f"""
### {signal_type}
- Average R: {signal_stats['avg_r']:+.2f}R
- Win Rate: {signal_stats['win_rate']*100:.1f}% ({signal_stats['trades']} trades)
- Best: {signal_stats['best_r']:+.2f}R, Worst: {signal_stats['worst_r']:+.2f}R
"""
        
        return report
