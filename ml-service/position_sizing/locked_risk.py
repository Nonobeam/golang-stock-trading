"""
Locked Risk Calculator for T+2 Settlement

Calculates worst-case floor-hit risk for shares in settlement period.
Vietnamese market: shares purchased cannot be sold for 3 trading days (T+0, T+1, T+2 -> sellable on T+3).
"""

import psycopg3
from typing import Dict, Tuple, Optional
from datetime import datetime


class LockedRiskCalculator:
    """Calculates and validates locked capital risk during T+2 settlement period."""

    # Exchange-specific risk multipliers (floor limit + margin for slippage/fees)
    EXCHANGE_RISK_MULTIPLIERS = {
        'HOSE': 0.20,   # 7% floor + margin = 20% worst case
        'HNX': 0.30,    # 10% floor + margin = 30% worst case
        'UPCOM': 0.40,  # 15% floor + margin = 40% worst case
    }

    def __init__(self, db_conn: psycopg3.Connection):
        """
        Initialize locked risk calculator.

        Args:
            db_conn: PostgreSQL database connection
        """
        self.db = db_conn

    def get_exchange_risk_multiplier(self, exchange: str) -> float:
        """
        Get risk multiplier for an exchange.

        Args:
            exchange: Exchange code (HOSE, HNX, UPCOM)

        Returns:
            Risk multiplier (0.20 for HOSE, 0.30 for HNX, 0.40 for UPCOM)
        """
        return self.EXCHANGE_RISK_MULTIPLIERS.get(exchange, 0.20)

    def calculate_locked_risk(self, shares: int, price: float, exchange: str) -> float:
        """
        Calculate worst-case locked risk for shares in settlement.

        Args:
            shares: Number of shares
            price: Entry price per share
            exchange: Exchange code

        Returns:
            Locked risk amount in VND
        """
        capital = shares * price
        multiplier = self.get_exchange_risk_multiplier(exchange)
        return capital * multiplier

    def get_total_locked_risk(self, user_id: int) -> float:
        """
        Calculate total locked risk across all user's locked positions.

        Args:
            user_id: User ID

        Returns:
            Total locked risk in VND
        """
        with self.db.cursor() as cur:
            cur.execute("""
                SELECT COALESCE(
                    SUM(
                        CASE
                            WHEN exchange = 'HOSE' THEN locked_capital * 0.20
                            WHEN exchange = 'HNX' THEN locked_capital * 0.30
                            WHEN exchange = 'UPCOM' THEN locked_capital * 0.40
                            ELSE locked_capital * 0.20
                        END
                    ),
                    0
                )
                FROM positions
                WHERE user_id = %s
                    AND settlement_status IN ('LOCKED_T0', 'LOCKED_T1', 'LOCKED_T2')
                    AND is_closed = FALSE
            """, (user_id,))

            result = cur.fetchone()
            return float(result[0]) if result else 0.0

    def get_locked_risk_budget(self, user_id: int, account_value: float, threshold: float) -> Dict[str, float]:
        """
        Get locked risk budget status.

        Args:
            user_id: User ID
            account_value: Total account value
            threshold: Locked risk threshold (e.g., 0.10 for 10%)

        Returns:
            Dict with total_locked_risk, max_allowed, available, used_percent
        """
        total_locked_risk = self.get_total_locked_risk(user_id)
        max_allowed = account_value * threshold
        available = max(0, max_allowed - total_locked_risk)
        used_percent = (total_locked_risk / max_allowed * 100) if max_allowed > 0 else 0

        return {
            'total_locked_risk': total_locked_risk,
            'max_allowed': max_allowed,
            'available': available,
            'used_percent': used_percent,
            'threshold_percent': threshold * 100
        }

    def check_locked_risk_budget(
        self,
        user_id: int,
        ticker: str,
        shares: int,
        price: float,
        account_value: float,
        threshold: float = 0.10
    ) -> Tuple[bool, Optional[str]]:
        """
        Check if a purchase fits within locked risk budget.

        Args:
            user_id: User ID
            ticker: Stock ticker
            shares: Number of shares to purchase
            price: Entry price
            account_value: Total account value
            threshold: Locked risk threshold (default 10%)

        Returns:
            Tuple of (approved, rejection_message)
            - approved: True if purchase is allowed
            - rejection_message: Reason if rejected, None if approved
        """
        # Get exchange from ticker (simplified - should use lookup table in production)
        exchange = self._get_exchange_from_ticker(ticker)

        # Calculate locked risk for new purchase
        new_locked_risk = self.calculate_locked_risk(shares, price, exchange)

        # Get current locked risk
        current_locked_risk = self.get_total_locked_risk(user_id)

        # Check against threshold
        max_allowed = account_value * threshold
        total_after_purchase = current_locked_risk + new_locked_risk

        if total_after_purchase > max_allowed:
            message = (
                f"Locked risk budget exceeded: "
                f"current {current_locked_risk:,.0f} VND + "
                f"new {new_locked_risk:,.0f} VND = "
                f"{total_after_purchase:,.0f} VND > "
                f"max {max_allowed:,.0f} VND ({threshold*100:.0f}% of account)"
            )
            return False, message

        # Warn if approaching threshold (80%)
        if total_after_purchase > max_allowed * 0.8:
            warning = (
                f"WARNING: Locked risk will be {total_after_purchase/max_allowed*100:.0f}% "
                f"of threshold ({total_after_purchase:,.0f} / {max_allowed:,.0f} VND)"
            )
            return True, warning

        return True, None

    def _get_exchange_from_ticker(self, ticker: str) -> str:
        """
        Infer exchange from ticker symbol.

        This is a simplified heuristic. In production, use actual ticker-to-exchange mapping.

        Args:
            ticker: Stock ticker

        Returns:
            Exchange code (HOSE, HNX, or UPCOM)
        """
        # TODO: Replace with actual exchange lookup from database
        # For now, default to HOSE (most common)
        return 'HOSE'

    def calculate_max_shares_for_budget(
        self,
        user_id: int,
        ticker: str,
        price: float,
        account_value: float,
        threshold: float = 0.10
    ) -> int:
        """
        Calculate maximum shares purchasable within locked risk budget.

        Args:
            user_id: User ID
            ticker: Stock ticker
            price: Entry price per share
            account_value: Total account value
            threshold: Locked risk threshold

        Returns:
            Maximum number of shares (rounded down to lot size of 100)
        """
        # Get available budget
        budget_info = self.get_locked_risk_budget(user_id, account_value, threshold)
        available = budget_info['available']

        if available <= 0:
            return 0

        # Get exchange and multiplier
        exchange = self._get_exchange_from_ticker(ticker)
        multiplier = self.get_exchange_risk_multiplier(exchange)

        # Calculate max capital we can lock
        max_capital = available / multiplier

        # Convert to shares
        max_shares = int(max_capital / price)

        # Round down to lot size (100 shares in Vietnamese market)
        lot_size = 100
        return (max_shares // lot_size) * lot_size


def get_user_locked_risk_threshold(db_conn: psycopg3.Connection, user_id: int) -> float:
    """
    Get user's configured locked risk threshold from database.

    Args:
        db_conn: Database connection
        user_id: User ID

    Returns:
        Locked risk threshold (default 0.10 if not set)
    """
    with db_conn.cursor() as cur:
        cur.execute("""
            SELECT COALESCE(locked_risk_threshold, 0.10)
            FROM user_config
            WHERE user_id = %s
        """, (user_id,))

        result = cur.fetchone()
        return float(result[0]) if result else 0.10
