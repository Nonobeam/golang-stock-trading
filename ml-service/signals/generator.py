
"""
Signal Generation Logic based on Multi-Horizon Predictions.
"""
import pandas as pd
import numpy as np
import json
from typing import Dict, Tuple, Optional
from datetime import datetime
import logging

from db.connection import get_connection
from position_manager.manager import PositionManager

logger = logging.getLogger("signal_generator")

class SignalGenerator:
    """Generates trading signals from ML predictions with optional position awareness."""
    
    def __init__(self, user_id: int = 1):
        """
        Initialize Signal Generator.
        
        Args:
            user_id: User ID for position queries (default=1 for single-user system)
        """
        # Configuration thresholds (could be moved to config.py)
        self.user_id = user_id
        self.MIN_CONFIDENCE = 0.60
        self.MIN_RETURN_BUY = 0.02   # 2% expected return
        self.MAX_DOWNSIDE_RISK = -0.05 # -5% max acceptable p10
        self.SELL_THRESHOLD = -0.01  # Sell if expected return drops below -1%
        self.PROFIT_TAKE_THRESHOLD = 0.05  # 5% unrealized profit triggers taking some profit
        
        
    def generate_signal(self, ticker: str, predictions: Dict[int, Dict[str, float]], 
                       current_price: float = None, db_connection=None, user_id: int = None) -> Tuple[str, float, str]:
        """
        Generate Buy/Sell/Hold signal based on multi-horizon predictions.
        
        Position-aware: If db_connection provided, loads position data and generates
        context-aware signals (BUY_NEW vs BUY_MORE, stop-loss checking, etc.)
        
        Args:
            ticker: Stock symbol
            predictions: Dictionary of horizon -> prediction dict
            current_price: Current market price (required for stop-loss/target checking)
            db_connection: Optional database connection for position queries
            user_id: Optional user ID (uses self.user_id if not provided)
            
        Returns:
            Tuple of (Signal, Strength, Reason)
            Signal: "BUY_NEW", "BUY_MORE", "SELL", "SELL_PARTIAL", "HOLD", "HOLD_NONE"
            Strength: 0.0 to 1.0
            Reason: Explanation string
        """
        if not predictions:
            return "HOLD", 0.0, "No predictions available"
        
        # Load position if db_connection provided
        position = None
        if db_connection:
            uid = user_id if user_id is not None else self.user_id
            pm = PositionManager(db_connection)
            position = pm.get_position_for_signal(ticker, uid)
            
        # 1. Check stop-loss if position exists and current_price available
        if position and current_price and position.get('stop_loss'):
            if current_price <= position['stop_loss']:
                return "SELL", 1.0, f"STOP LOSS TRIGGERED at {position['stop_loss']:,.0f}"
                
        # 2. Check target levels if position exists and current_price available
        if position and current_price:
            target_3 = position.get('target_3')
            target_2 = position.get('target_2')
            target_1 = position.get('target_1')
            
            if target_3 and current_price >= target_3:
                return "SELL", 1.0, f"Target 3 reached at {target_3:,.0f} - Close entire position"
            elif target_2 and current_price >= target_2:
                return "SELL_PARTIAL", 0.9, f"Target 2 reached at {target_2:,.0f} - Sell 1/3 of position"
            elif target_1 and current_price >= target_1:
                return "SELL_PARTIAL", 0.8, f"Target 1 reached at {target_1:,.0f} - Sell 1/3 of position"
            
        # 3. Aggregate Confidence
        confidences = [p.get('confidence', 0.5) for p in predictions.values()]
        avg_confidence = sum(confidences) / len(confidences) if confidences else 0
        
        if avg_confidence < self.MIN_CONFIDENCE:
            signal_type = "HOLD" if position else "HOLD_NONE"
            return signal_type, 0.0, f"Low confidence: {avg_confidence:.2f} < {self.MIN_CONFIDENCE}"
            
        # 4. Extract Key Metrics (focus on 10d and 5d)
        pred_10d = predictions.get(10)
        pred_5d = predictions.get(5)
        pred_1d = predictions.get(1)
        
        # Need at least 5d or 10d for trend
        if not pred_10d and not pred_5d:
            signal_type = "HOLD" if position else "HOLD_NONE"
            return signal_type, 0.0, "Missing multi-horizon forecasts"
            
        primary_pred = pred_10d if pred_10d else pred_5d
        horizon_used = 10 if pred_10d else 5
        
        p50 = primary_pred['p50']
        p10 = primary_pred['p10']
        p90 = primary_pred['p90']
        
        # 5. Calculate unrealized P&L if position exists and current_price available
        unrealized_pnl_pct = None
        if position and current_price and position.get('avg_price'):
            avg_price = position['avg_price']
            if avg_price > 0:
                unrealized_pnl_pct = ((current_price - avg_price) / avg_price)
        
        # 6. Decision Logic
        
        # SELL logic for existing positions with profit
        if position and unrealized_pnl_pct is not None:
            if unrealized_pnl_pct > self.PROFIT_TAKE_THRESHOLD and p50 < 0.01:
                # Have 5%+ profit and weak forward outlook - take profit
                sell_strength = min(1.0, unrealized_pnl_pct / 0.10)
                return "SELL", sell_strength, f"Take profit: {unrealized_pnl_pct:.1%} gain with weak {horizon_used}d outlook"
        
        # Buy Logic - differentiate BUY_NEW vs BUY_MORE
        if p50 > self.MIN_RETURN_BUY:
            if p10 > self.MAX_DOWNSIDE_RISK:
                # Check consistency if 1d available
                if pred_1d and pred_1d['p50'] < -0.01:
                    signal_type = "HOLD" if position else "HOLD_NONE"
                    return signal_type, 0.0, "Short-term pull back detected despite long-term growth"
                
                # Calculate strength based on return magnitude
                return_strength = min(1.0, p50 / 0.10) # 10% return = max strength
                
                if position:
                    # For BUY_MORE, check if already at target
                    if current_price and position.get('target_1') and current_price >= position['target_1']:
                        return "HOLD", 0.0, "Price already at T1 - don't add to position"
                    return "BUY_MORE", return_strength, f"Strong {horizon_used}d outlook: {p50:.1%} return - Add to position"
                else:
                    return "BUY_NEW", return_strength, f"Strong {horizon_used}d outlook: {p50:.1%} return - Initiate position"
            else:
                signal_type = "HOLD" if position else "HOLD_NONE"
                return signal_type, 0.0, f"High risk: p10 {p10:.1%} < {self.MAX_DOWNSIDE_RISK:.1%}"
                
        # SELL logic for negative outlook (applies to positions only)
        if position and p50 < self.SELL_THRESHOLD:
            sell_strength = min(1.0, abs(p50) / 0.05)
            return "SELL", sell_strength, f"Negative outlook: {p50:.1%} return"
            
        # Default HOLD
        signal_type = "HOLD" if position else "HOLD_NONE"
        return signal_type, 0.0, "Neutral outlook"

    def save_signal(self, ticker: str, date: str, signal: str, strength: float, reason: str, metadata: Dict = None):
        """Save generated signal to database."""
        conn = get_connection()
        try:
            with conn.cursor() as cursor:
                # Pass metadata dict directly - psycopg will serialize to JSONB
                metadata_dict = metadata if metadata else {}
                
                cursor.execute("""
                    INSERT INTO "stock-trading".signals (
                        ticker, signal_date, signal, strength, reason, metadata, created_at
                    ) VALUES (
                        %(ticker)s, %(date)s, %(signal)s, %(strength)s, %(reason)s, %(metadata)s, NOW()
                    )
                    ON CONFLICT (ticker, signal_date) 
                    DO UPDATE SET
                        signal = EXCLUDED.signal,
                        strength = EXCLUDED.strength,
                        reason = EXCLUDED.reason,
                        metadata = EXCLUDED.metadata,
                        created_at = NOW()
                """, {
                    'ticker': ticker,
                    'date': date,
                    'signal': signal,
                    'strength': strength,
                    'reason': reason,
                    'metadata': metadata_dict  # Pass dict directly for JSONB
                })
            conn.commit()
            logger.info(f"Saved signal {signal} for {ticker} on {date}")
            return True
        except Exception as e:
            logger.error(f"Failed to save signal: {e}")
            return False
        finally:
            conn.close()
    
    
    def generate_and_save_signal(self, ticker: str, predictions: Dict[int, Dict[str, float]], 
                                 date: str, current_price: float = None, 
                                 db_connection=None, user_id: int = None) -> Tuple[Dict, bool]:
        """
        Combined method for production use - generate and save signal in single call.
        
        Args:
            ticker: Stock symbol
            predictions: Dictionary of horizon -> prediction dict
            date: Date in YYYY-MM-DD format
            current_price: Optional current market price for position checks
            db_connection: Optional DB connection (uses get_connection if None, enables position-aware signals)
            user_id: Optional user ID (uses self.user_id if not provided)
        
        Returns:
            Tuple of (signal_dict, save_success)
            signal_dict contains: signal, strength, reason, position_exists, position_details
            save_success is True if database save succeeded
        """
        # Generate signal (position-aware if db_connection provided)
        try:
            uid = user_id if user_id is not None else self.user_id
            signal, strength, reason = self.generate_signal(
                ticker, predictions, current_price, db_connection, uid
            )
            
            # Load position for metadata (if not already loaded)
            position = None
            if db_connection:
                pm = PositionManager(db_connection)
                position = pm.get_position_for_signal(ticker, uid)
            
            signal_dict = {
                'signal': signal,
                'strength': strength,
                'reason': reason,
                'ticker': ticker,
                'date': date,
                'position_exists': position is not None
            }
            
            # Build metadata for storage
            metadata = {
                'predictions': predictions,
                'generated_at': datetime.now().isoformat(),
                'position_exists': position is not None
            }
            
            # Add position details to metadata if available
            if position:
                metadata['current_quantity'] = position.get('quantity')
                metadata['avg_price'] = position.get('avg_price')
                metadata['stop_loss_price'] = position.get('stop_loss')
                
                if current_price and position.get('avg_price'):
                    unrealized_pnl_pct = ((current_price - position['avg_price']) / position['avg_price']) * 100
                    metadata['unrealized_pnl_pct'] = round(unrealized_pnl_pct, 2)
                    
                # Add next target to metadata
                if position.get('target_1') and (not current_price or current_price < position['target_1']):
                    metadata['next_target_price'] = position['target_1']
                elif position.get('target_2') and (not current_price or current_price < position['target_2']):
                    metadata['next_target_price'] = position['target_2']
                elif position.get('target_3') and (not current_price or current_price < position['target_3']):
                    metadata['next_target_price'] = position['target_3']
            
            # Save to database
            save_success = self.save_signal(ticker, date, signal, strength, reason, metadata)
            
            if not save_success:
                logger.warning(f"Signal generated but save failed for {ticker} on {date}")
            
            return signal_dict, save_success
            
        except Exception as e:
            logger.error(f"Failed to generate signal for {ticker}: {e}")
            # Return hold signal if generation fails
            error_signal = {
                'signal': 'HOLD',
                'strength': 0.0,
                'reason': f'Error: {str(e)}',
                'ticker': ticker,
                'date': date,
                'position_exists': False
            }
            return error_signal, False

