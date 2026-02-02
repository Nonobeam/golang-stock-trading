"""
Position Manager - Portfolio State Management

Provides CRUD operations for active and closed positions,
integrating with ML signal generation and position sizing workflows.
"""

from typing import Dict, List, Optional
from datetime import datetime, date
import logging
from decimal import Decimal

from db.connection import get_connection

logger = logging.getLogger("position_manager")


class PositionManager:
    """
    Manages position data for trading workflows.
    
    Provides methods to:
    - Query active and closed positions
    - Create new position entries
    - Update position quantities with weighted average cost
    - Close positions with P&L calculation
    """
    
    def __init__(self, db_connection=None):
        """
        Initialize Position Manager.
        
        Args:
            db_connection: Optional database connection. If None, creates new connection per operation.
        """
        self.db_connection = db_connection
        
    def get_position(self, ticker: str, user_id: int = 1) -> Optional[Dict]:
        """
        Retrieve active position for a specific ticker.
        
        Args:
            ticker: Stock symbol
            user_id: User ID (default=1 for single-user system)
            
        Returns:
            Dictionary with position data or None if no active position exists
            Keys: id, symbol, entry_price, quantity, entry_date, stop_loss,
                  target_1, target_2, target_3, pnl, pnl_percent, signal_type, notes
        """
        conn = self.db_connection or get_connection()
        try:
            with conn.cursor() as cursor:
                cursor.execute("""
                    SELECT id, user_id, symbol, entry_date, entry_price, quantity,
                           stop_loss, target_1, target_2, target_3,
                           signal_type, score, notes, pnl, pnl_percent, r_multiple,
                           created_at, updated_at
                    FROM positions
                    WHERE symbol = %(ticker)s 
                      AND user_id = %(user_id)s 
                      AND is_closed = FALSE
                    LIMIT 1
                """, {'ticker': ticker, 'user_id': user_id})
                
                row = cursor.fetchone()
                if not row:
                    return None
                    
                return {
                    'id': row[0],
                    'user_id': row[1],
                    'symbol': row[2],
                    'entry_date': row[3],
                    'entry_price': float(row[4]) if row[4] else 0.0,
                    'quantity': row[5],
                    'stop_loss': float(row[6]) if row[6] else 0.0,
                    'target_1': float(row[7]) if row[7] else None,
                    'target_2': float(row[8]) if row[8] else None,
                    'target_3': float(row[9]) if row[9] else None,
                    'signal_type': row[10],
                    'score': row[11],
                    'notes': row[12],
                    'pnl': float(row[13]) if row[13] else None,
                    'pnl_percent': float(row[14]) if row[14] else None,
                    'r_multiple': float(row[15]) if row[15] else None,
                    'created_at': row[16],
                    'updated_at': row[17]
                }
        finally:
            if not self.db_connection:
                conn.close()
                
    def get_all_positions(self, user_id: int = 1) -> List[Dict]:
        """
        Retrieve all active positions for a user.
        
        Args:
            user_id: User ID (default=1)
            
        Returns:
            List of position dictionaries, ordered by symbol
        """
        conn = self.db_connection or get_connection()
        try:
            with conn.cursor() as cursor:
                cursor.execute("""
                    SELECT id, user_id, symbol, entry_date, entry_price, quantity,
                           stop_loss, target_1, target_2, target_3,
                           signal_type, score, notes, pnl, pnl_percent, r_multiple,
                           created_at, updated_at
                    FROM positions
                    WHERE user_id = %(user_id)s 
                      AND is_closed = FALSE
                    ORDER BY symbol ASC
                """, {'user_id': user_id})
                
                positions = []
                for row in cursor.fetchall():
                    positions.append({
                        'id': row[0],
                        'user_id': row[1],
                        'symbol': row[2],
                        'entry_date': row[3],
                        'entry_price': float(row[4]) if row[4] else 0.0,
                        'quantity': row[5],
                        'stop_loss': float(row[6]) if row[6] else 0.0,
                        'target_1': float(row[7]) if row[7] else None,
                        'target_2': float(row[8]) if row[8] else None,
                        'target_3': float(row[9]) if row[9] else None,
                        'signal_type': row[10],
                        'score': row[11],
                        'notes': row[12],
                        'pnl': float(row[13]) if row[13] else None,
                        'pnl_percent': float(row[14]) if row[14] else None,
                        'r_multiple': float(row[15]) if row[15] else None,
                        'created_at': row[16],
                        'updated_at': row[17]
                    })
                return positions
        finally:
            if not self.db_connection:
                conn.close()
                
    def get_position_for_signal(self, ticker: str, user_id: int = 1) -> Optional[Dict]:
        """
        Lightweight position query for signal generation.
        
        Returns only fields needed for signal logic to minimize data transfer.
        
        Args:
            ticker: Stock symbol
            user_id: User ID
            
        Returns:
            Simplified dictionary with keys: avg_price, quantity, stop_loss
            Returns None if no active position
        """
        conn = self.db_connection or get_connection()
        try:
            with conn.cursor() as cursor:
                cursor.execute("""
                    SELECT entry_price, quantity, stop_loss, target_1, target_2, target_3
                    FROM positions
                    WHERE symbol = %(ticker)s 
                      AND user_id = %(user_id)s 
                      AND is_closed = FALSE
                    LIMIT 1
                """, {'ticker': ticker, 'user_id': user_id})
                
                row = cursor.fetchone()
                if not row:
                    return None
                    
                return {
                    'avg_price': float(row[0]) if row[0] else 0.0,
                    'quantity': row[1],
                    'stop_loss': float(row[2]) if row[2] else 0.0,
                    'target_1': float(row[3]) if row[3] else None,
                    'target_2': float(row[4]) if row[4] else None,
                    'target_3': float(row[5]) if row[5] else None
                }
        finally:
            if not self.db_connection:
                conn.close()
                
    def add_position(self, user_id: int, ticker: str, shares: int, entry_price: float,
                    entry_date: str, stop_loss: float, target_1: float = None,
                    target_2: float = None, target_3: float = None,
                    signal_type: str = 'MANUAL', score: int = None, notes: str = None) -> str:
        """
        Create new position entry.
        
        Args:
            user_id: User ID
            ticker: Stock symbol
            shares: Number of shares
            entry_price: Entry price per share
            entry_date: Entry date (YYYY-MM-DD format)
            stop_loss: Stop loss price
            target_1: Optional first target price
            target_2: Optional second target price
            target_3: Optional third target price
            signal_type: Signal that generated position (default='MANUAL')
            score: Optional signal score
            notes: Optional position notes
            
        Returns:
            UUID of created position
        """
        if shares <= 0:
            raise ValueError(f"Shares must be positive, got {shares}")
        if entry_price <= 0:
            raise ValueError(f"Entry price must be positive, got {entry_price}")
            
        conn = self.db_connection or get_connection()
        try:
            with conn.cursor() as cursor:
                cursor.execute("""
                    INSERT INTO positions (
                        user_id, symbol, entry_date, entry_price, quantity,
                        stop_loss, target_1, target_2, target_3,
                        signal_type, score, notes, is_closed
                    ) VALUES (
                        %(user_id)s, %(ticker)s, %(entry_date)s, %(entry_price)s, %(quantity)s,
                        %(stop_loss)s, %(target_1)s, %(target_2)s, %(target_3)s,
                        %(signal_type)s, %(score)s, %(notes)s, FALSE
                    )
                    RETURNING id
                """, {
                    'user_id': user_id,
                    'ticker': ticker,
                    'entry_date': entry_date,
                    'entry_price': entry_price,
                    'quantity': shares,
                    'stop_loss': stop_loss,
                    'target_1': target_1,
                    'target_2': target_2,
                    'target_3': target_3,
                    'signal_type': signal_type,
                    'score': score,
                    'notes': notes
                })
                
                position_id = cursor.fetchone()[0]
            
            conn.commit()
            logger.info(f"Created position {position_id} for {ticker}: {shares} shares @ {entry_price}")
            return str(position_id)
        except Exception as e:
            conn.rollback()
            logger.error(f"Failed to create position for {ticker}: {e}")
            raise
        finally:
            if not self.db_connection:
                conn.close()
                
    def update_position_quantity(self, position_id: str, additional_shares: int,
                                 new_price: float = None) -> None:
        """
        Update position quantity with weighted average cost calculation.
        
        For additions (positive additional_shares): Recalculates entry_price using weighted average
        For reductions (negative additional_shares): Keeps entry_price unchanged
        
        Args:
            position_id: UUID of position to update
            additional_shares: Shares to add (positive) or remove (negative)
            new_price: Price per share for additions (required if additional_shares > 0)
            
        Raises:
            ValueError: If trying to remove more shares than available
        """
        if additional_shares == 0:
            return
            
        if additional_shares > 0 and (new_price is None or new_price <= 0):
            raise ValueError("new_price required and must be positive when adding shares")
            
        conn = self.db_connection or get_connection()
        try:
            with conn.cursor() as cursor:
                # Get current position
                cursor.execute("""
                    SELECT symbol, entry_price, quantity
                    FROM positions
                    WHERE id = %(position_id)s AND is_closed = FALSE
                """, {'position_id': position_id})
                
                row = cursor.fetchone()
                if not row:
                    raise ValueError(f"Position {position_id} not found or already closed")
                    
                ticker, current_avg_price, current_qty = row
                current_avg_price = float(current_avg_price)
                
                # Check if reduction is valid
                new_qty = current_qty + additional_shares
                if new_qty <= 0:
                    raise ValueError(
                        f"Cannot reduce position by {abs(additional_shares)} shares, "
                        f"only {current_qty} available"
                    )
                
                # Calculate new average price
                if additional_shares > 0:
                    # Adding shares - weighted average
                    total_cost = (current_qty * current_avg_price) + (additional_shares * new_price)
                    new_avg_price = total_cost / new_qty
                    
                    cursor.execute("""
                        UPDATE positions
                        SET quantity = %(new_qty)s,
                            entry_price = %(new_avg_price)s,
                            updated_at = CURRENT_TIMESTAMP
                        WHERE id = %(position_id)s
                    """, {
                        'new_qty': new_qty,
                        'new_avg_price': new_avg_price,
                        'position_id': position_id
                    })
                    
                    logger.info(
                        f"Added {additional_shares} shares to {ticker} @ {new_price}. "
                        f"New position: {new_qty} shares @ {new_avg_price:.2f} avg"
                    )
                else:
                    # Reducing shares - keep average price
                    cursor.execute("""
                        UPDATE positions
                        SET quantity = %(new_qty)s,
                            updated_at = CURRENT_TIMESTAMP
                        WHERE id = %(position_id)s
                    """, {'new_qty': new_qty, 'position_id': position_id})
                    
                    logger.info(
                        f"Reduced {ticker} by {abs(additional_shares)} shares. "
                        f"Remaining: {new_qty} @ {current_avg_price:.2f} avg"
                    )
            
            conn.commit()
        except Exception as e:
            conn.rollback()
            logger.error(f"Failed to update position {position_id}: {e}")
            raise
        finally:
            if not self.db_connection:
                conn.close()
                
    def close_position(self, position_id: str, exit_price: float,
                      exit_date: str, exit_reason: str) -> None:
        """
        Close position and calculate P&L metrics.
        
        Calculates:
        - P&L: (exit_price - entry_price) × quantity
        - P&L %: ((exit_price - entry_price) / entry_price) × 100
        - R-multiple: (exit_price - entry_price) / (entry_price - stop_loss)
        
        Args:
            position_id: UUID of position to close
            exit_price: Exit price per share
            exit_date: Exit date (YYYY-MM-DD format)
            exit_reason: Reason for exit (e.g., 'stop_loss_triggered', 'target_2_reached')
        """
        if exit_price <= 0:
            raise ValueError(f"Exit price must be positive, got {exit_price}")
            
        conn = self.db_connection or get_connection()
        try:
            with conn.cursor() as cursor:
                # Get position details
                cursor.execute("""
                    SELECT symbol, entry_price, quantity, stop_loss
                    FROM positions
                    WHERE id = %(position_id)s AND is_closed = FALSE
                """, {'position_id': position_id})
                
                row = cursor.fetchone()
                if not row:
                    raise ValueError(f"Position {position_id} not found or already closed")
                    
                ticker, entry_price, quantity, stop_loss = row
                entry_price = float(entry_price)
                stop_loss = float(stop_loss) if stop_loss else entry_price
                
                # Calculate P&L metrics
                pnl = quantity * (exit_price - entry_price)
                pnl_percent = ((exit_price - entry_price) / entry_price) * 100
                
                # R-multiple: profit per share / risk per share
                risk_per_share = entry_price - stop_loss if entry_price > stop_loss else 1.0
                r_multiple = (exit_price - entry_price) / risk_per_share if risk_per_share > 0 else 0.0
                
                # Close position
                cursor.execute("""
                    UPDATE positions
                    SET is_closed = TRUE,
                        exit_date = %(exit_date)s,
                        exit_price = %(exit_price)s,
                        exit_reason = %(exit_reason)s,
                        pnl = %(pnl)s,
                        pnl_percent = %(pnl_percent)s,
                        r_multiple = %(r_multiple)s,
                        updated_at = CURRENT_TIMESTAMP
                    WHERE id = %(position_id)s
                """, {
                    'exit_date': exit_date,
                    'exit_price': exit_price,
                    'exit_reason': exit_reason,
                    'pnl': pnl,
                    'pnl_percent': pnl_percent,
                    'r_multiple': r_multiple,
                    'position_id': position_id
                })
            
            conn.commit()
            logger.info(
                f"Closed {ticker} position: {quantity} shares @ {exit_price}. "
                f"P&L: {pnl:+,.0f} VND ({pnl_percent:+.2f}%), R-multiple: {r_multiple:+.2f}R"
            )
        except Exception as e:
            conn.rollback()
            logger.error(f"Failed to close position {position_id}: {e}")
            raise
        finally:
            if not self.db_connection:
                conn.close()
