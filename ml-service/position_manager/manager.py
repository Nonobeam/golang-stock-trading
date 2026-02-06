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
                
    def calculate_average_cost(self, ticker: str, user_id: int = 1) -> Optional[float]:
        """
        Calculate weighted average cost from all position entries.
        
        This method queries the position_entries table to compute the true
        average cost across all purchases of a stock.
        
        Args:
            ticker: Stock symbol
            user_id: User ID (default=1)
            
        Returns:
            Weighted average cost per share, or None if no entries exist
            
        Formula:
            avg_cost = SUM(shares × price) / SUM(shares)
        """
        conn = self.db_connection or get_connection()
        try:
            with conn.cursor() as cursor:
                cursor.execute("""
                    SELECT 
                        SUM(shares_purchased * entry_price) as total_cost,
                        SUM(shares_purchased) as total_shares
                    FROM position_entries
                    WHERE ticker = %(ticker)s 
                      AND user_id = %(user_id)s
                """, {'ticker': ticker, 'user_id': user_id})
                
                row = cursor.fetchone()
                if not row or row[0] is None or row[1] is None or row[1] == 0:
                    return None
                    
                total_cost = float(row[0])
                total_shares = int(row[1])
                
                avg_cost = total_cost / total_shares
                logger.debug(f"Average cost for {ticker}: {avg_cost:.2f} ({total_shares} shares)")
                return avg_cost
        finally:
            if not self.db_connection:
                conn.close()
                
    def get_position_entries(self, ticker: str, user_id: int = 1) -> List[Dict]:
        """
        Retrieve all purchase entries for a ticker.
        
        Returns transaction history showing each individual purchase,
        useful for entry quality analysis and detailed position reporting.
        
        Args:
            ticker: Stock symbol
            user_id: User ID (default=1)
            
        Returns:
            List of entry dictionaries ordered by entry_date DESC
            Each dict contains: entry_id, entry_date, entry_price, shares_purchased,
                              entry_fee_paid, transaction_type
        """
        conn = self.db_connection or get_connection()
        try:
            with conn.cursor() as cursor:
                cursor.execute("""
                    SELECT entry_id, entry_date, entry_price, shares_purchased,
                           entry_fee_paid, transaction_type, created_at
                    FROM position_entries
                    WHERE ticker = %(ticker)s 
                      AND user_id = %(user_id)s
                    ORDER BY entry_date DESC
                """, {'ticker': ticker, 'user_id': user_id})
                
                entries = []
                for row in cursor.fetchall():
                    entries.append({
                        'entry_id': str(row[0]),
                        'entry_date': row[1],
                        'entry_price': float(row[2]),
                        'shares_purchased': row[3],
                        'entry_fee_paid': float(row[4]),
                        'transaction_type': row[5],
                        'created_at': row[6]
                    })
                return entries
        finally:
            if not self.db_connection:
                conn.close()
                
    def check_buying_capacity(self, ticker: str, current_price: float, 
                              account_value: float, user_id: int = 1) -> Dict:
        """
        Check remaining buying capacity for a ticker based on multiple constraints.
        
        Enforces three capacity limits:
        1. Portfolio allocation: Maximum 20% of account value per position
        2. Liquidity constraint: Maximum 1% of 20-day average daily volume
        3. Total risk constraint: Total position risk ≤ 2% of account value
        
        Args:
            ticker: Stock symbol
            current_price: Current market price
            account_value: Total account value
            user_id: User ID (default=1)
            
        Returns:
            Dictionary with capacity information:
            {
                'at_limit': bool,              # True if any limit reached
                'current_position_value': float,
                'max_position_value': float,    # 20% of account
                'remaining_value_capacity': float,
                'current_shares': int,
                'max_shares_liquidity': int,    # 1% of 20-day avg volume
                'remaining_share_capacity': int,
                'max_buyable_shares': int,      # Minimum of all constraints
                'limit_reason': str,            # Which limit is binding
                'total_risk': float,            # Current + proposed risk
                'max_risk_allowed': float       # 2% of account value
            }
        """
        conn = self.db_connection or get_connection()
        try:
            # Get current position
            position = self.get_position(ticker, user_id)
            current_shares = position['quantity'] if position else 0
            current_avg_price = position['entry_price'] if position else 0.0
            stop_loss = position['stop_loss'] if position else None
            
            # Get first entry price for stop-loss calculation
            first_entry_price = current_avg_price  # Default to avg if only one entry
            if position:
                with conn.cursor() as cursor:
                    cursor.execute("""
                        SELECT entry_price 
                        FROM position_entries
                        WHERE ticker = %(ticker)s AND user_id = %(user_id)s
                        ORDER BY entry_date ASC
                        LIMIT 1
                    """, {'ticker': ticker, 'user_id': user_id})
                    row = cursor.fetchone()
                    if row:
                        first_entry_price = float(row[0])
            
            # 1. Portfolio Allocation Limit (20% max)
            current_position_value = current_shares * current_price
            max_position_value = account_value * 0.20
            remaining_value_capacity = max_position_value - current_position_value
            
            # 2. Liquidity Limit (1% of 20-day avg daily volume)
            with conn.cursor() as cursor:
                cursor.execute("""
                    SELECT AVG(volume) as avg_volume
                    FROM (
                        SELECT volume
                        FROM market_data
                        WHERE symbol = %(ticker)s
                        ORDER BY date DESC
                        LIMIT 20
                    ) recent_volume
                """, {'ticker': ticker})
                row = cursor.fetchone()
                avg_daily_volume = int(row[0]) if row and row[0] else 1000000  # Default fallback
            
            max_shares_liquidity = int(avg_daily_volume * 0.01)
            remaining_share_capacity = max_shares_liquidity - current_shares
            
            # 3. Total Risk Limit (2% of account value)
            # Risk is calculated from first entry price, not average cost
            max_risk_allowed = account_value * 0.02
            
            # Current risk based on stop-loss from first entry
            if stop_loss and current_shares > 0:
                current_risk = current_shares * (first_entry_price - stop_loss)
            else:
                current_risk = 0.0
            
            # Calculate max additional shares based on risk
            # For new shares: risk_per_share = (current_price - stop_loss)
            if stop_loss and current_price > stop_loss:
                risk_per_share = current_price - stop_loss
                remaining_risk_capacity = max_risk_allowed - current_risk
                max_shares_by_risk = int(remaining_risk_capacity / risk_per_share) if risk_per_share > 0 else 0
            else:
                max_shares_by_risk = remaining_share_capacity  # No stop-loss constraint
            
            # Determine binding constraint
            max_shares_by_value = int(remaining_value_capacity / current_price) if current_price > 0 else 0
            
            # Take minimum across all constraints
            max_buyable_shares = max(0, min(
                max_shares_by_value,
                remaining_share_capacity,
                max_shares_by_risk
            ))
            
            # Determine which limit is binding
            at_limit = max_buyable_shares == 0
            if at_limit:
                if remaining_value_capacity <= 0:
                    limit_reason = "portfolio_allocation_20pct"
                elif remaining_share_capacity <= 0:
                    limit_reason = "liquidity_1pct_volume"
                elif max_shares_by_risk <= 0:
                    limit_reason = "total_risk_2pct"
                else:
                    limit_reason = "position_at_capacity_limit"
            else:
                limit_reason = None
            
            return {
                'at_limit': at_limit,
                'current_position_value': current_position_value,
                'max_position_value': max_position_value,
                'remaining_value_capacity': remaining_value_capacity,
                'current_shares': current_shares,
                'max_shares_liquidity': max_shares_liquidity,
                'remaining_share_capacity': remaining_share_capacity,
                'max_buyable_shares': max_buyable_shares,
                'limit_reason': limit_reason,
                'total_risk': current_risk,
                'max_risk_allowed': max_risk_allowed,
                'avg_daily_volume': avg_daily_volume
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
                
    def update_position_after_buy(self, ticker: str, new_shares: int, new_price: float,
                                   entry_date: str, user_id: int = 1,
                                   transaction_type: str = 'BUY_MORE') -> None:
        """
        Record new share purchase and recalculate weighted average cost.
        
        This method:
        1. Inserts entry into position_entries table
        2. Database trigger auto-updates positions table with new average cost
        3. Updates aggregate tracking fields (total_entries, total_fees_paid, etc.)
        
        Args:
            ticker: Stock symbol
            new_shares: Number of shares purchased
            new_price: Price per share for this purchase
            entry_date: Purchase date (YYYY-MM-DD format)
            user_id: User ID (default=1)
            transaction_type: 'BUY_NEW' or 'BUY_MORE' (default='BUY_MORE')
            
        Raises:
            ValueError: If shares or price are invalid
        """
        if new_shares <= 0:
            raise ValueError(f"Shares must be positive, got {new_shares}")
        if new_price <= 0:
            raise ValueError(f"Price must be positive, got {new_price}")
            
        # Calculate entry fee (0.15% of purchase value)
        purchase_value = new_shares * new_price
        entry_fee = purchase_value * 0.0015
        
        conn = self.db_connection or get_connection()
        try:
            with conn.cursor() as cursor:
                # Insert entry into position_entries
                # Trigger will automatically update positions table
                cursor.execute("""
                    INSERT INTO position_entries (
                        user_id, ticker, entry_date, entry_price,
                        shares_purchased, entry_fee_paid, transaction_type
                    ) VALUES (
                        %(user_id)s, %(ticker)s, %(entry_date)s, %(entry_price)s,
                        %(shares)s, %(entry_fee)s, %(transaction_type)s
                    )
                    RETURNING entry_id
                """, {
                    'user_id': user_id,
                    'ticker': ticker,
                    'entry_date': entry_date,
                    'entry_price': new_price,
                    'shares': new_shares,
                    'entry_fee': entry_fee,
                    'transaction_type': transaction_type
                })
                
                entry_id = cursor.fetchone()[0]
                
            conn.commit()
            
            # Log the update
            avg_cost = self.calculate_average_cost(ticker, user_id)
            logger.info(
                f"Recorded {transaction_type} for {ticker}: {new_shares} shares @ {new_price:.2f}. "
                f"Entry ID: {entry_id}, Fee: {entry_fee:.2f} VND, New avg cost: {avg_cost:.2f}"
            )
            
        except Exception as e:
            conn.rollback()
            logger.error(f"Failed to record entry for {ticker}: {e}")
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
                
    def partial_exit_position(self, position_id: str, shares_to_sell: int,
                              exit_price: float, exit_date: str,
                              exit_reason: str = 'PARTIAL_EXIT') -> Dict:
        """
        Sell a portion of a position with proportional fee allocation.
        
        This method:
        1. Validates that shares_to_sell < total shares
        2. Calculates proportional entry fees: (shares_sold / total_shares) × total_fees_paid
        3. Calculates exit fee: sale_value × 0.0025 (0.15% + 0.10% tax)
        4. Computes fee-adjusted P&L
        5. Updates position with reduced quantity and proportionally reduced fees
        6. Keeps average cost unchanged (entries remain immutable)
        
        Args:
            position_id: UUID of position
            shares_to_sell: Number of shares to sell (must be < total quantity)
            exit_price: Exit price per share
            exit_date: Exit date (YYYY-MM-DD format)
            exit_reason: Reason for partial exit (default='PARTIAL_EXIT')
            
        Returns:
            Dictionary with partial exit details:
            {
                'shares_sold': int,
                'remaining_shares': int,
                'avg_cost': float,
                'gross_proceeds': float,
                'proportional_entry_fees': float,
                'exit_fee': float,
                'total_fees': float,
                'realized_pnl': float,
                'pnl_percent': float
            }
            
        Raises:
            ValueError: If trying to sell all or more shares (use close_position instead)
        """
        if shares_to_sell <= 0:
            raise ValueError(f"Shares to sell must be positive, got {shares_to_sell}")
        if exit_price <= 0:
            raise ValueError(f"Exit price must be positive, got {exit_price}")
            
        conn = self.db_connection or get_connection()
        try:
            with conn.cursor() as cursor:
                # Get position details including  aggregated fees
                cursor.execute("""
                    SELECT symbol, user_id, entry_price, quantity, total_fees_paid
                    FROM positions
                    WHERE id = %(position_id)s AND is_closed = FALSE
                """, {'position_id': position_id})
                
                row = cursor.fetchone()
                if not row:
                    raise ValueError(f"Position {position_id} not found or already closed")
                    
                ticker, user_id, avg_cost, total_shares, total_fees_paid = row
                avg_cost = float(avg_cost)
                total_fees_paid = float(total_fees_paid) if total_fees_paid else 0.0
                
                # Validate partial exit
                if shares_to_sell >= total_shares:
                    raise ValueError(
                        f"Cannot partially exit {shares_to_sell} shares from position of {total_shares}. "
                        f"Use close_position() to exit all shares."
                    )
                
                remaining_shares = total_shares - shares_to_sell
                
                # Calculate proportional entry fees
                proportional_entry_fees = total_fees_paid * (shares_to_sell / total_shares)
                
                # Calculate exit fee (0.25% = 0.15% broker + 0.10% tax)
                sale_value = shares_to_sell * exit_price
                exit_fee = sale_value * 0.0025
                
                # Calculate fee-adjusted P&L
                cost_basis = avg_cost * shares_to_sell
                gross_proceeds = sale_value
                total_fees = proportional_entry_fees + exit_fee
                realized_pnl = gross_proceeds - cost_basis - total_fees
                pnl_percent = (realized_pnl / (cost_basis + proportional_entry_fees)) * 100
                
                # Update position with reduced quantity and fees
                remaining_fees = total_fees_paid - proportional_entry_fees
                
                cursor.execute("""
                    UPDATE positions
                    SET quantity = %(remaining_shares)s,
                        total_fees_paid = %(remaining_fees)s,
                        updated_at = CURRENT_TIMESTAMP
                    WHERE id = %(position_id)s
                """, {
                    'remaining_shares': remaining_shares,
                    'remaining_fees': remaining_fees,
                    'position_id': position_id
                })
                
            conn.commit()
            
            result = {
                'shares_sold': shares_to_sell,
                'remaining_shares': remaining_shares,
                'avg_cost': avg_cost,
                'gross_proceeds': gross_proceeds,
                'proportional_entry_fees': proportional_entry_fees,
                'exit_fee': exit_fee,
                'total_fees': total_fees,
                'realized_pnl': realized_pnl,
                'pnl_percent': pnl_percent
            }
            
            logger.info(
                f"Partial exit {ticker}: Sold {shares_to_sell}/{total_shares} shares @ {exit_price:.2f}. "
                f"Remaining: {remaining_shares} @ {avg_cost:.2f} avg. "
                f"P&L: {realized_pnl:+,.0f} VND ({pnl_percent:+.2f}%)"
            )
            
            return result
            
        except Exception as e:
            conn.rollback()
            logger.error(f"Failed to partial exit position {position_id}: {e}")
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
                # Get position details including total fees
                cursor.execute("""
                    SELECT symbol, entry_price, quantity, stop_loss, total_fees_paid
                    FROM positions
                    WHERE id = %(position_id)s AND is_closed = FALSE
                """, {'position_id': position_id})
                
                row = cursor.fetchone()
                if not row:
                    raise ValueError(f"Position {position_id} not found or already closed")
                    
                ticker, entry_price, quantity, stop_loss, total_fees_paid = row
                entry_price = float(entry_price)
                stop_loss = float(stop_loss) if stop_loss else entry_price
                total_fees_paid = float(total_fees_paid) if total_fees_paid else 0.0
                
                # Calculate exit fee (0.25% = 0.15% broker + 0.10% tax)
                exit_value = quantity * exit_price
                exit_fee = exit_value * 0.0025
                
                # Calculate fee-adjusted P&L metrics
                cost_basis = quantity * entry_price
                gross_proceeds = exit_value
                total_fees = total_fees_paid + exit_fee
                pnl = gross_proceeds - cost_basis - total_fees
                pnl_percent = (pnl / (cost_basis + total_fees_paid)) * 100 if (cost_basis + total_fees_paid) > 0 else 0.0
                
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
    
    def snapshot_portfolio_equity(self, user_id: int, snapshot_date: date) -> None:
        """
        Create daily equity snapshot for portfolio drawdown tracking.
        
        Args:
            user_id: User ID
            snapshot_date: Date of snapshot
        """
        from validation.portfolio_metrics import PortfolioEquityTracker
        
        tracker = PortfolioEquityTracker(db_connection=self.db_connection)
        tracker.save_equity_snapshot(user_id, snapshot_date)
        logger.info(f"Portfolio equity snapshot created for user {user_id} on {snapshot_date}")
