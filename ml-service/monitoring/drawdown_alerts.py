"""
Monitoring and Alerting Module

Extends existing alerter with drawdown-based risk alerts.
"""

import logging
from typing import Optional
from decimal import Decimal

logger = logging.getLogger(__name__)


def check_drawdown_alert(user_id: int, db_connection=None) -> None:
    """
    Check portfolio drawdown and send alerts if thresholds are crossed.
    
    Alerting Thresholds:
    - WARNING: -8% drawdown (approaching half-sizing threshold)
    - CRITICAL: -10% drawdown (position sizes reduced to 50%)
    - EMERGENCY: -15% drawdown (trading stopped)
    
    Args:
        user_id: User identifier
        db_connection: Optional database connection
    """
    from validation.portfolio_metrics import PortfolioEquityTracker
    from position_sizing.drawdown_manager import DrawdownManager
    
    tracker = PortfolioEquityTracker(db_connection=db_connection)
    current_equity, peak_equity, drawdown = tracker.calculate_drawdown(user_id)
    
    drawdown_pct = drawdown * 100
    
    # Define threshold levels
    THRESHOLD_WARNING = -8.0
    THRESHOLD_CRITICAL = -10.0
    THRESHOLD_EMERGENCY = -15.0
    
    if drawdown_pct <= THRESHOLD_EMERGENCY:
        log_and_alert(
            level="EMERGENCY",
            user_id=user_id,
            message=(
                f"🔴 EMERGENCY: Portfolio drawdown at {drawdown_pct:.2f}%\\n"
                f"Trading has been STOPPED to preserve capital.\\n"
                f"Current equity: {current_equity:,.0f} VND (Peak: {peak_equity:,.0f} VND)\\n"
                f"No new positions will be opened until drawdown recovers above -10%."
            )
        )
    
    elif drawdown_pct <= THRESHOLD_CRITICAL:
        log_and_alert(
            level="CRITICAL",
            user_id=user_id,
            message=(
                f"🟠 CRITICAL: Portfolio drawdown at {drawdown_pct:.2f}%\\n"
                f"Position sizes have been REDUCED to 50% to manage risk.\\n"
                f"Current equity: {current_equity:,.0f} VND (Peak: {peak_equity:,.0f} VND)\\n"
                f"Review open positions and consider closing weak performers."
            )
        )
    
    elif drawdown_pct <= THRESHOLD_WARNING:
        log_and_alert(
            level="WARNING",
            user_id=user_id,
            message=(
                f"⚠️ WARNING: Portfolio drawdown at {drawdown_pct:.2f}%\\n"
                f"Approaching -10% threshold where position sizes will be reduced.\\n"
                f"Current equity: {current_equity:,.0f} VND (Peak: {peak_equity:,.0f} VND)\\n"
                f"Monitor positions closely."
            )
        )
    
    else:
        # No alert needed
        logger.info(f"Drawdown at {drawdown_pct:.2f}% for user {user_id} - within acceptable range")


def log_and_alert(level: str, user_id: int, message: str) -> None:
    """
    Log alert and send notification.
    
    Args:
        level: Alert level (WARNING, CRITICAL, EMERGENCY)
        user_id: User identifier
        message: Alert message
    """
    # Log to console/file
    if level == "EMERGENCY":
        logger.critical(f"[User {user_id}] {message}")
    elif level == "CRITICAL":
        logger.error(f"[User {user_id}] {message}")
    else:
        logger.warning(f"[User {user_id}] {message}")
    
    # TODO: Integrate with existing notification system
    # Example:
    # from monitoring.notifier import send_alert
    # send_alert(user_id=user_id, level=level, message=message)
    
    print(f"\\n{'='*80}\\n{message}\\n{'='*80}\\n")
