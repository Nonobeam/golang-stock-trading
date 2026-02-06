"""
Portfolio Equity Tracking Module

Calculates and tracks daily portfolio equity for drawdown-based risk management.
Provides methods to calculate current equity, peak equity, and drawdown percentage.
"""

import logging
from datetime import date, datetime
from decimal import Decimal
from typing import Dict, Optional, Tuple
import psycopg

logger = logging.getLogger(__name__)


class PortfolioEquityTracker:
    """
    Tracks portfolio equity over time for drawdown calculations.
    
    Formula:
        total_equity = open_positions_market_value + closed_positions_total_pnl + cash_balance
        
    Where:
        open_positions_market_value = Σ(quantity × current_price) for all open positions
        closed_positions_total_pnl = Σ(pnl) for all closed positions
        cash_balance = initial_capital - Σ(invested_capital_open_positions)
    """
    
    def __init__(self, db_connection: Optional[psycopg.Connection] = None):
        """
        Initialize tracker with optional database connection.
        
        Args:
            db_connection: Reusable psycopg connection. If None, creates new connection per operation.
        """
        self.db_connection = db_connection
        
    def _get_connection(self) -> psycopg.Connection:
        """Get database connection (reuse or create new)."""
        if self.db_connection:
            return self.db_connection
        
        # Import here to avoid circular dependency
        from ..db.connection import get_connection
        return get_connection()
    
    def calculate_current_equity(self, user_id: int, current_prices: Optional[Dict[str, float]] = None) -> Dict[str, Decimal]:
        """
        Calculate current portfolio equity components.
        
        Args:
            user_id: User identifier
            current_prices: Optional dict of {symbol: current_price}. If None, uses entry prices.
        
        Returns:
            Dict with:
                - total_equity: Total portfolio value
                - open_positions_value: Market value of open positions
                - closed_pnl: Total P&L from closed positions
                - cash_balance: Remaining cash
                - details: List of open position details
        """
        conn = self._get_connection()
        
        try:
            with conn.cursor() as cursor:
                # Get user's initial capital
                cursor.execute("""
                    SELECT initial_capital FROM user_config WHERE user_id = %s
                """, (user_id,))
                row = cursor.fetchone()
                if not row:
                    raise ValueError(f"User {user_id} not found in user_config")
                
                initial_capital = Decimal(str(row[0]))
                
                # Calculate open positions value
                cursor.execute("""
                    SELECT symbol, entry_price, quantity
                    FROM positions
                    WHERE user_id = %s AND is_closed = FALSE
                """, (user_id,))
                
                open_positions = cursor.fetchall()
                open_value = Decimal('0')
                invested_capital = Decimal('0')
                position_details = []
                
                for symbol, entry_price, quantity in open_positions:
                    entry_price = Decimal(str(entry_price))
                    current_price = Decimal(str(current_prices.get(symbol, entry_price))) if current_prices else entry_price
                    
                    position_value = current_price * quantity
                    position_cost = entry_price * quantity
                    
                    open_value += position_value
                    invested_capital += position_cost
                    
                    position_details.append({
                        'symbol': symbol,
                        'quantity': quantity,
                        'entry_price': float(entry_price),
                        'current_price': float(current_price),
                        'market_value': float(position_value),
                        'unrealized_pnl': float(position_value - position_cost)
                    })
                
                # Calculate total P&L from closed positions
                cursor.execute("""
                    SELECT COALESCE(SUM(pnl), 0) as total_pnl
                    FROM positions
                    WHERE user_id = %s AND is_closed = TRUE
                """, (user_id,))
                
                closed_pnl = Decimal(str(cursor.fetchone()[0]))
                
                # Cash balance = initial capital - invested in open positions
                cash_balance = initial_capital - invested_capital
                
                # Total equity
                total_equity = open_value + closed_pnl + cash_balance
                
                logger.info(f"Equity calculated for user {user_id}: {total_equity:,.0f} VND")
                
                return {
                    'total_equity': total_equity,
                    'open_positions_value': open_value,
                    'closed_pnl': closed_pnl,
                    'cash_balance': cash_balance,
                    'initial_capital': initial_capital,
                    'details': position_details
                }
                
        finally:
            if not self.db_connection:
                conn.close()
    
    def get_peak_equity(self, user_id: int) -> Decimal:
        """
        Get historical peak equity for user (running maximum).
        
        Args:
            user_id: User identifier
        
        Returns:
            Peak equity value. If no history, returns user's initial capital.
        """
        conn = self._get_connection()
        
        try:
            with conn.cursor() as cursor:
                cursor.execute("""
                    SELECT MAX(peak_equity) as peak
                    FROM portfolio_equity_snapshots
                    WHERE user_id = %s
                """, (user_id,))
                
                row = cursor.fetchone()
                peak = row[0] if row and row[0] else None
                
                if peak:
                    return Decimal(str(peak))
                
                # No historical data - use initial capital as peak
                cursor.execute("""
                    SELECT initial_capital FROM user_config WHERE user_id = %s
                """, (user_id,))
                
                row = cursor.fetchone()
                if not row:
                    raise ValueError(f"User {user_id} not found")
                
                return Decimal(str(row[0]))
                
        finally:
            if not self.db_connection:
                conn.close()
    
    def calculate_drawdown(self, user_id: int, current_equity: Optional[Decimal] = None) -> Tuple[Decimal, Decimal, Decimal]:
        """
        Calculate current drawdown from peak equity.
        
        Formula:
            drawdown = (current_equity - peak_equity) / peak_equity
        
        Args:
            user_id: User identifier
            current_equity: Optional current equity. If None, calculates it.
        
        Returns:
            Tuple of (current_equity, peak_equity, drawdown_percentage)
            
        Example:
            Peak: 100,000,000 VND, Current: 88,000,000 VND
            Drawdown: (88M - 100M) / 100M = -0.12 (-12%)
        """
        if current_equity is None:
            equity_data = self.calculate_current_equity(user_id)
            current_equity = equity_data['total_equity']
        
        peak_equity = self.get_peak_equity(user_id)
        
        if peak_equity <= 0:
            logger.warning(f"Peak equity is {peak_equity} for user {user_id}, defaulting drawdown to 0")
            return current_equity, peak_equity, Decimal('0')
        
        drawdown = (current_equity - peak_equity) / peak_equity
        
        logger.info(f"Drawdown for user {user_id}: {drawdown*100:.2f}%")
        
        return current_equity, peak_equity, drawdown
    
    def save_equity_snapshot(self, user_id: int, snapshot_date: date, equity_data: Optional[Dict] = None) -> None:
        """
        Save daily equity snapshot to database.
        
        Args:
            user_id: User identifier
            snapshot_date: Date of snapshot
            equity_data: Optional pre-calculated equity data. If None, calculates it.
        """
        if equity_data is None:
            equity_data = self.calculate_current_equity(user_id)
        
        current_equity = equity_data['total_equity']
        peak_equity = self.get_peak_equity(user_id)
        
        # Update peak if current exceeds historical max
        if current_equity > peak_equity:
            peak_equity = current_equity
        
        # Calculate drawdown
        drawdown = (current_equity - peak_equity) / peak_equity if peak_equity > 0 else Decimal('0')
        
        conn = self._get_connection()
        
        try:
            with conn.cursor() as cursor:
                cursor.execute("""
                    INSERT INTO portfolio_equity_snapshots (
                        user_id, snapshot_date, total_equity, peak_equity, current_drawdown,
                        open_positions_value, closed_pnl, cash_balance
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s, %s, %s
                    )
                    ON CONFLICT (user_id, snapshot_date)
                    DO UPDATE SET
                        total_equity = EXCLUDED.total_equity,
                        peak_equity = EXCLUDED.peak_equity,
                        current_drawdown = EXCLUDED.current_drawdown,
                        open_positions_value = EXCLUDED.open_positions_value,
                        closed_pnl = EXCLUDED.closed_pnl,
                        cash_balance = EXCLUDED.cash_balance
                """, (
                    user_id, snapshot_date, current_equity, peak_equity, drawdown,
                    equity_data['open_positions_value'], equity_data['closed_pnl'], equity_data['cash_balance']
                ))
            
            conn.commit()
            logger.info(f"Equity snapshot saved for user {user_id} on {snapshot_date}: {current_equity:,.0f} VND (drawdown: {drawdown*100:.2f}%)")
            
        except Exception as e:
            conn.rollback()
            logger.error(f"Failed to save equity snapshot: {e}")
            raise
        finally:
            if not self.db_connection:
                conn.close()
    
    def get_equity_history(self, user_id: int, days: int = 90) -> list:
        """
        Get historical equity snapshots for user.
        
        Args:
            user_id: User identifier
            days: Number of days to retrieve
        
        Returns:
            List of dicts with snapshot data
        """
        conn = self._get_connection()
        
        try:
            with conn.cursor() as cursor:
                cursor.execute("""
                    SELECT 
                        snapshot_date, total_equity, peak_equity, current_drawdown,
                        open_positions_value, closed_pnl, cash_balance
                    FROM portfolio_equity_snapshots
                    WHERE user_id = %s AND snapshot_date >= CURRENT_DATE - INTERVAL '%s days'
                    ORDER BY snapshot_date DESC
                """, (user_id, days))
                
                history = []
                for row in cursor.fetchall():
                    history.append({
                        'date': row[0],
                        'total_equity': float(row[1]),
                        'peak_equity': float(row[2]),
                        'drawdown': float(row[3]),
                        'open_value': float(row[4]) if row[4] else 0,
                        'closed_pnl': float(row[5]) if row[5] else 0,
                        'cash': float(row[6]) if row[6] else 0
                    })
                
                return history
                
        finally:
            if not self.db_connection:
                conn.close()
