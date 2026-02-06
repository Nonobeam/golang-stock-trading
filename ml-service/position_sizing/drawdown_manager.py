"""
Drawdown Manager - Portfolio Risk Control

Manages drawdown-based position sizing adjustments to preserve capital during losing streaks.
"""

import logging
from typing import Tuple
from decimal import Decimal

logger = logging.getLogger(__name__)


class DrawdownManager:
    """
    Manages portfolio drawdown-based risk controls.
    
    Adjusts position sizing based on portfolio drawdown from peak equity:
    - Drawdown > -5%: Normal trading (multiplier = 1.0)
    - Drawdown -10% to -15%: Half position sizes (multiplier = 0.5)
    - Drawdown < -15%: Stop all new trades (multiplier = 0.0)
    """
    
    # Drawdown thresholds
    THRESHOLD_WARNING = -0.10  # -10%: Reduce position sizes
    THRESHOLD_CRITICAL = -0.15  # -15%: Stop trading
    THRESHOLD_RECOVERY = -0.05  # -5%: Back to normal
    
    # Position size multipliers
    MULTIPLIER_NORMAL = 1.0
    MULTIPLIER_HALF = 0.5
    MULTIPLIER_STOP = 0.0
    
    def __init__(self, db_connection=None):
        """
        Initialize drawdown manager.
        
        Args:
            db_connection: Optional database connection for equity queries
        """
        self.db_connection = db_connection
    
    def get_drawdown_multiplier(self, user_id: int) -> float:
        """
        Get position sizing multiplier based on current portfolio drawdown.
        
        Args:
            user_id: User identifier
        
        Returns:
            Multiplier (0.0, 0.5, or 1.0) to apply to position sizes
        """
        from validation.portfolio_metrics import PortfolioEquityTracker
        
        tracker = PortfolioEquityTracker(db_connection=self.db_connection)
        current_equity, peak_equity, drawdown = tracker.calculate_drawdown(user_id)
        
        # Determine multiplier based on drawdown
        if drawdown <= self.THRESHOLD_CRITICAL:
            # Drawdown >= -15%: STOP TRADING
            multiplier = self.MULTIPLIER_STOP
            logger.warning(f"CRITICAL drawdown {drawdown*100:.2f}% for user {user_id}: Trading STOPPED")
        
        elif drawdown <= self.THRESHOLD_WARNING:
            # Drawdown -10% to -15%: HALF POSITION SIZES
            multiplier = self.MULTIPLIER_HALF
            logger.warning(f"WARNING drawdown {drawdown*100:.2f}% for user {user_id}: Position sizes REDUCED to 50%")
        
        else:
            # Drawdown > -10%: NORMAL TRADING
            multiplier = self.MULTIPLIER_NORMAL
            if drawdown < -0.05:
                logger.info(f"Minor drawdown {drawdown*100:.2f}% for user {user_id}: Normal trading continues")
        
        return multiplier
    
    def check_trading_allowed(self, user_id: int) -> Tuple[bool, str]:
        """
        Check if trading is allowed based on current drawdown.
        
        Args:
            user_id: User identifier
        
        Returns:
            Tuple of (is_allowed: bool, reason: str)
        """
        from validation.portfolio_metrics import PortfolioEquityTracker
        
        tracker = PortfolioEquityTracker(db_connection=self.db_connection)
        current_equity, peak_equity, drawdown = tracker.calculate_drawdown(user_id)
        
        if drawdown <= self.THRESHOLD_CRITICAL:
            return (False, f"Trading stopped: Portfolio drawdown at {drawdown*100:.2f}% (threshold: {self.THRESHOLD_CRITICAL*100:.1f}%)")
        
        elif drawdown <= self.THRESHOLD_WARNING:
            return (True, f"Trading with reduced sizes: Portfolio drawdown at {drawdown*100:.2f}% (50% position sizes)")
        
        else:
            return (True, f"Normal trading: Portfolio drawdown at {drawdown*100:.2f}%")
    
    def get_drawdown_status(self, user_id: int) -> dict:
        """
        Get comprehensive drawdown status for monitoring and logging.
        
        Args:
            user_id: User identifier
        
        Returns:
            Dict with drawdown metrics and status
        """
        from validation.portfolio_metrics import PortfolioEquityTracker
        
        tracker = PortfolioEquityTracker(db_connection=self.db_connection)
        current_equity, peak_equity, drawdown = tracker.calculate_drawdown(user_id)
        
        multiplier = self.get_drawdown_multiplier(user_id)
        is_allowed, reason = self.check_trading_allowed(user_id)
        
        # Determine risk level
        if drawdown <= self.THRESHOLD_CRITICAL:
            risk_level = "EMERGENCY"
        elif drawdown <= self.THRESHOLD_WARNING:
            risk_level = "WARNING"
        elif drawdown < -0.05:
            risk_level = "CAUTION"
        else:
            risk_level = "NORMAL"
        
        return {
            'user_id': user_id,
            'current_equity': float(current_equity),
            'peak_equity': float(peak_equity),
            'drawdown': float(drawdown),
            'drawdown_percent': f"{drawdown*100:.2f}%",
            'multiplier': multiplier,
            'trading_allowed': is_allowed,
            'risk_level': risk_level,
            'reason': reason,
            'thresholds': {
                'warning': f"{self.THRESHOLD_WARNING*100:.0f}%",
                'critical': f"{self.THRESHOLD_CRITICAL*100:.0f}%"
            }
        }
    
    def should_skip_signal_generation(self, user_id: int) -> Tuple[bool, str]:
        """
        Determine if signal generation should be skipped entirely.
        
        Args:
            user_id: User identifier
        
        Returns:
            Tuple of (should_skip: bool, reason: str)
        """
        is_allowed, reason = self.check_trading_allowed(user_id)
        
        if not is_allowed:
            return (True, reason)
        
        return (False, "Signal generation allowed")
