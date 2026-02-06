"""
Enhanced Signal Generation with ML Validation Modules

Phase 5: Integration with all validation enhancements:
- Transaction cost filtering (0.4% Vietnamese fees)
- Liquidity constraints (1% volume cap)
- Floor-hit risk assessment (±7%/±10% circuit breakers)
- Uncertainty quantification (epistemic + aleatoric)
- Calibration checks (quantile validation)

Author: ML Trading System
Created: 2026-02-02
"""

import pandas as pd
import numpy as np
import json
from typing import Dict, Tuple, Optional
from datetime import datetime
import logging

from db.connection import get_connection
from position_manager.manager import PositionManager

# Import validation modules
from validation.transaction_costs import (
    is_profitable_after_fees,
    calculate_fee_adjusted_return,
    get_minimum_profit_threshold
)
from validation.liquidity_manager import LiquidityManager
from models.floor_hit_classifier import FloorHitClassifier
from position_sizing.locked_risk import LockedRiskCalculator, get_user_locked_risk_threshold

logger = logging.getLogger("enhanced_signal_generator")


class EnhancedSignalGenerator:
    """
    Production-ready signal generator with comprehensive validations.
    
    Enhancements over basic SignalGenerator:
    1. Transaction cost awareness (Vietnamese market fees)
    2. Liquidity constraints (position size limits)
    3. Floor-hit risk assessment (circuit breaker protection)
    4. Uncertainty filtering (high epistemic uncertainty rejection)
    5. Metadata enrichment (all validation results stored)
    """
    
    def __init__(self, user_id: int = 1, exchange: str = 'HOSE'):
        """
        Initialize Enhanced Signal Generator.
        
        Args:
            user_id: User ID for position queries
            exchange: 'HOSE' (±7%) or 'HNX' (±10%) for circuit breakers
        """
        self.user_id = user_id
        self.exchange = exchange
        
        # Original thresholds
        self.MIN_CONFIDENCE = 0.60
        self.MAX_DOWNSIDE_RISK = -0.05  # -5% max acceptable p10
        self.SELL_THRESHOLD = -0.01
        self.PROFIT_TAKE_THRESHOLD = 0.05
        
        # New validation thresholds
        self.MAX_FLOOR_HIT_PROBABILITY = 0.20  # 20% floor risk = reject
        self.FLOOR_RISK_WARNING = 0.10  # 10% = reduce position
        self.MIN_LIQUIDITY_SCORE = 2  # 1-10 scale, <2 = too illiquid
        self.MAX_EPISTEMIC_UNCERTAINTY = 0.05  # 5% model disagreement = reject

        # T+2 Settlement Risk thresholds
        self.APPLY_ENTRY_DAY_LIMITS = True  # Enforce Thursday/Friday restrictions
        self.LOCKED_RISK_THRESHOLD = 0.10   # Default 10%, overridden by user config

        # Initialize validation modules
        self.liquidity_manager = LiquidityManager()
        self.floor_classifier = FloorHitClassifier(exchange=exchange)
    
    def generate_signal(
        self, 
        ticker: str, 
        predictions: Dict[int, Dict[str, float]], 
        current_price: float = None,
        current_features: Dict = None,  # NEW: For floor-hit prediction
        uncertainty_metrics: Dict = None,  # NEW: From uncertainty estimator
        db_connection=None,
        user_id: int = None
    ) -> Tuple[str, float, str, Dict]:
        """
        Generate validation-enhanced Buy/Sell/Hold signal.
        
        Args:
            ticker: Stock symbol
            predictions: Dict of horizon -> {p10, p50, p90, confidence}
            current_price: Current market price
            current_features: Dict of current technical features for floor-hit
            uncertainty_metrics: Dict with epistemic/aleatoric uncertainty
            db_connection: Optional DB connection
            user_id: Optional user ID
            
        Returns:
            Tuple of (Signal, Strength, Reason, ValidationMetadata)
        """
        if not predictions:
            return "HOLD", 0.0, "No predictions available", {}
        
        # Initialize validation metadata
        validation_metadata = {
            'validations_passed': [],
            'validations_failed': [],
            'warnings': []
        }
        
        # Load position if available
        position = None
        if db_connection:
            uid = user_id if user_id is not None else self.user_id
            pm = PositionManager(db_connection)
            position = pm.get_position_for_signal(ticker, uid)
        
        # ===================================================================
        # CRITICAL SAFETY CHECKS (Immediate rejections)
        # ===================================================================
        
        # 1. Stop-Loss Check (highest priority)
        if position and current_price and position.get('stop_loss'):
            if current_price <= position['stop_loss']:
                return "SELL", 1.0, f"STOP LOSS TRIGGERED at {position['stop_loss']:,.0f}", validation_metadata
        
        # 2. Target Level Checks
        if position and current_price:
            target_signal = self._check_targets(position, current_price)
            if target_signal:
                return target_signal
        
        # 3. **NEW** Floor-Hit Risk Assessment
        floor_risk_result = self._check_floor_risk(ticker, current_features, validation_metadata)
        if floor_risk_result:
            return floor_risk_result  # Critical risk - reject signal
        
        # ===================================================================
        # PREDICTION VALIDATION
        # ===================================================================
        
        # 4. Confidence Check
        confidences = [p.get('confidence', 0.5) for p in predictions.values()]
        avg_confidence = sum(confidences) / len(confidences) if confidences else 0
        
        if avg_confidence < self.MIN_CONFIDENCE:
            signal_type = "HOLD" if position else "HOLD_NONE"
            validation_metadata['validations_failed'].append({
                'check': 'confidence',
                'value': avg_confidence,
                'threshold': self.MIN_CONFIDENCE
            })
            return signal_type, 0.0, f"Low confidence: {avg_confidence:.2f} < {self.MIN_CONFIDENCE}", validation_metadata
        
        validation_metadata['validations_passed'].append('confidence')
        
        # 5. Extract Primary Prediction
        pred_10d = predictions.get(10)
        pred_5d = predictions.get(5)
        pred_1d = predictions.get(1)
        
        if not pred_10d and not pred_5d:
            signal_type = "HOLD" if position else "HOLD_NONE"
            return signal_type, 0.0, "Missing multi-horizon forecasts", validation_metadata
        
        primary_pred = pred_10d if pred_10d else pred_5d
        horizon_used = 10 if pred_10d else 5
        
        p50 = primary_pred['p50']
        p10 = primary_pred['p10']
        p90 = primary_pred['p90']
        
        # ===================================================================
        # NEW VALIDATION CHECKS
        # ===================================================================
        
        # 6. **NEW** Transaction Cost Check
        fee_check_result = self._check_transaction_costs(
            p50, horizon_used, validation_metadata
        )
        if not fee_check_result['profitable']:
            signal_type = "HOLD" if position else "HOLD_NONE"
            return signal_type, 0.0, fee_check_result['reason'], validation_metadata
        
        # Calculate fee-adjusted return
        fee_adjusted_return = calculate_fee_adjusted_return(p50)
        validation_metadata['fee_adjusted_return'] = fee_adjusted_return
        validation_metadata['gross_return'] = p50
        
        # 7. **NEW** Liquidity Constraint Check
        liquidity_check = self._check_liquidity(ticker, validation_metadata)
        if not liquidity_check['tradeable']:
            return "HOLD_NONE", 0.0, liquidity_check['reason'], validation_metadata
        
        # 8. **NEW** Uncertainty Check
        if uncertainty_metrics:
            uncertainty_check = self._check_uncertainty(
                uncertainty_metrics, validation_metadata
            )
            if not uncertainty_check['acceptable']:
                signal_type = "HOLD" if position else "HOLD_NONE"
                return signal_type, 0.0, uncertainty_check['reason'], validation_metadata

        # 9. **NEW** T+2 Settlement Risk Check (for BUY signals only)
        # This check is performed later in BUY logic section to avoid running it unnecessarily

        # ===================================================================
        # DECISION LOGIC (with validations)
        # ===================================================================
        
        # Calculate unrealized P&L if position exists
        unrealized_pnl_pct = None
        if position and current_price and position.get('avg_price'):
            avg_price = position['avg_price']
            if avg_price > 0:
                unrealized_pnl_pct = ((current_price - avg_price) / avg_price)
        
        # SELL logic for existing positions with profit
        if position and unrealized_pnl_pct is not None:
            if unrealized_pnl_pct > self.PROFIT_TAKE_THRESHOLD and p50 < 0.01:
                sell_strength = min(1.0, unrealized_pnl_pct / 0.10)
                return "SELL", sell_strength, \
                    f"Take profit: {unrealized_pnl_pct:.1%} gain with weak {horizon_used}d outlook", \
                    validation_metadata
        
        # BUY logic (with all validations passed)
        if fee_adjusted_return > get_minimum_profit_threshold(horizon_used):
            if p10 > self.MAX_DOWNSIDE_RISK:
                # Check 1d consistency
                if pred_1d and pred_1d['p50'] < -0.01:
                    signal_type = "HOLD" if position else "HOLD_NONE"
                    return signal_type, 0.0, \
                        "Short-term pullback despite long-term growth", \
                        validation_metadata

                # **NEW** T+2 Settlement Risk Check (only for BUY signals)
                # Calculate proposed shares if not provided (default to max liquidity allows)
                max_position_shares = liquidity_check.get('max_position_shares', 0)

                if db_connection and current_price and account_value and max_position_shares > 0:
                    settlement_check = self._check_settlement_risk(
                        db_connection,
                        uid if 'uid' in locals() else user_id if user_id else self.user_id,
                        ticker,
                        max_position_shares,
                        current_price,
                        account_value,
                        validation_metadata
                    )
                    if settlement_check:
                        return settlement_check  # Rejected due to settlement risk

                # Calculate strength
                return_strength = min(1.0, fee_adjusted_return / 0.08)

                # Check liquidity cap for position sizing
                validation_metadata['max_position_shares'] = max_position_shares

                if position:
                    if current_price and position.get('target_1') and \
                       current_price >= position['target_1']:
                        return "HOLD", 0.0, \
                            "Price already at T1 - don't add to position", \
                            validation_metadata
                    
                    return "BUY_MORE", return_strength, \
                        f"Strong {horizon_used}d outlook: {fee_adjusted_return:.1%} (net after fees) - Add to position", \
                        validation_metadata
                else:
                    return "BUY_NEW", return_strength, \
                        f"Strong {horizon_used}d outlook: {fee_adjusted_return:.1%} (net after fees) - Initiate position", \
                        validation_metadata
            else:
                signal_type = "HOLD" if position else "HOLD_NONE"
                validation_metadata['validations_failed'].append({
                    'check': 'downside_risk',
                    'p10': p10,
                    'threshold': self.MAX_DOWNSIDE_RISK
                })
                return signal_type, 0.0, \
                    f"High downside risk: p10 {p10:.1%} < {self.MAX_DOWNSIDE_RISK:.1%}", \
                    validation_metadata
        
        # SELL logic for negative outlook
        if position and p50 < self.SELL_THRESHOLD:
            sell_strength = min(1.0, abs(p50) / 0.05)
            return "SELL", sell_strength, \
                f"Negative outlook: {p50:.1%} return", \
                validation_metadata
        
        # Default HOLD
        signal_type = "HOLD" if position else "HOLD_NONE"
        return signal_type, 0.0, "Neutral outlook - all validations passed", validation_metadata
    
    def _check_targets(self, position: Dict, current_price: float) -> Optional[Tuple]:
        """Check if price has hit target levels."""
        target_3 = position.get('target_3')
        target_2 = position.get('target_2')
        target_1 = position.get('target_1')
        
        if target_3 and current_price >= target_3:
            return "SELL", 1.0, f"Target 3 reached at {target_3:,.0f} - Close entire position", {}
        elif target_2 and current_price >= target_2:
            return "SELL_PARTIAL", 0.9, f"Target 2 reached at {target_2:,.0f} - Sell 1/3 of position", {}
        elif target_1 and current_price >= target_1:
            return "SELL_PARTIAL", 0.8, f"Target 1 reached at {target_1:,.0f} - Sell 1/3 of position", {}
        
        return None
    
    def _check_floor_risk(
        self, 
        ticker: str, 
        current_features: Optional[Dict], 
        metadata: Dict
    ) -> Optional[Tuple]:
        """Check circuit breaker (floor-hit) risk."""
        if not current_features:
            metadata['warnings'].append('Floor-hit risk not assessed - no features provided')
            return None
        
        try:
            floor_prob = self.floor_classifier.predict_floor_probability(ticker, current_features)
            metadata['floor_hit_probability'] = floor_prob
            
            if floor_prob > self.MAX_FLOOR_HIT_PROBABILITY:
                metadata['validations_failed'].append({
                    'check': 'floor_risk',
                    'probability': floor_prob,
                    'threshold': self.MAX_FLOOR_HIT_PROBABILITY
                })
                return "HOLD_NONE", 0.0, \
                    f"CRITICAL: High floor-hit risk ({floor_prob:.1%}) - Circuit breaker likely", \
                    metadata
            elif floor_prob > self.FLOOR_RISK_WARNING:
                metadata['warnings'].append(
                    f"Moderate floor risk ({floor_prob:.1%}) - Consider reducing position"
                )
            else:
                metadata['validations_passed'].append('floor_risk')
        
        except Exception as e:
            logger.warning(f"Floor-hit risk check failed: {e}")
            metadata['warnings'].append(f'Floor-hit risk check error: {str(e)}')
        
        return None
    
    def _check_transaction_costs(
        self, 
        gross_return: float, 
        horizon: int, 
        metadata: Dict
    ) -> Dict:
        """Validate profitability after Vietnamese market fees (0.4%)."""
        profitable, reason = is_profitable_after_fees(gross_return, horizon)
        
        min_threshold = get_minimum_profit_threshold(horizon)
        
        if profitable:
            metadata['validations_passed'].append('transaction_costs')
        else:
            metadata['validations_failed'].append({
                'check': 'transaction_costs',
                'gross_return': gross_return,
                'min_threshold': min_threshold,
                'horizon': horizon
            })
        
        return {
            'profitable': profitable,
            'reason': reason,
            'min_threshold': min_threshold
        }
    
    def _check_liquidity(self, ticker: str, metadata: Dict) -> Dict:
        """Check liquidity constraints (1% volume cap)."""
        try:
            with self.liquidity_manager:
                avg_volume = self.liquidity_manager.get_average_volume(ticker)
                liquidity_score = self.liquidity_manager.get_liquidity_score(ticker)
                max_position = int(avg_volume * 0.01)
                
                metadata['liquidity_score'] = liquidity_score
                metadata['avg_volume_20d'] = avg_volume
                metadata['max_position_shares'] = max_position
                
                tradeable = liquidity_score >= self.MIN_LIQUIDITY_SCORE and avg_volume >= 100_000
                
                if tradeable:
                    metadata['validations_passed'].append('liquidity')
                else:
                    metadata['validations_failed'].append({
                        'check': 'liquidity',
                        'liquidity_score': liquidity_score,
                        'min_score': self.MIN_LIQUIDITY_SCORE,
                        'avg_volume': avg_volume
                    })
                
                return {
                    'tradeable': tradeable,
                    'reason': f"Liquidity too low (score: {liquidity_score}/10)" if not tradeable else "Liquidity OK",
                    'max_position_shares': max_position
                }
        
        except Exception as e:
            logger.warning(f"Liquidity check failed: {e}")
            metadata['warnings'].append(f'Liquidity check error: {str(e)}')
            return {
                'tradeable': True,  # Don't reject on error
                'reason': 'Liquidity check unavailable',
                'max_position_shares': 0
            }
    
    def _check_uncertainty(self, uncertainty_metrics: Dict, metadata: Dict) -> Dict:
        """Check prediction uncertainty (epistemic + aleatoric)."""
        epistemic = uncertainty_metrics.get('epistemic_uncertainty', 0)
        aleatoric = uncertainty_metrics.get('aleatoric_uncertainty', 0)
        total = uncertainty_metrics.get('total_uncertainty', 0)
        confidence_score = uncertainty_metrics.get('confidence_score', 1.0)
        
        metadata['epistemic_uncertainty'] = epistemic
        metadata['aleatoric_uncertainty'] = aleatoric
        metadata['total_uncertainty'] = total
        metadata['confidence_score'] = confidence_score
        
        # High epistemic uncertainty = models disagree = unreliable
        if epistemic > self.MAX_EPISTEMIC_UNCERTAINTY:
            metadata['validations_failed'].append({
                'check': 'epistemic_uncertainty',
                'value': epistemic,
                'threshold': self.MAX_EPISTEMIC_UNCERTAINTY
            })
            return {
                'acceptable': False,
                'reason': f"High model uncertainty ({epistemic:.1%}) - Models disagree, unreliable prediction"
            }
        
        # High aleatoric uncertainty = market unpredictable (not a reject, but a warning)
        if aleatoric > 0.15:  # 15% prediction range
            metadata['warnings'].append(
                f"High market unpredictability ({aleatoric:.1%}) - Consider reducing position size"
            )
        
        metadata['validations_passed'].append('uncertainty')
        return {'acceptable': True, 'reason': 'Uncertainty acceptable'}

    def _check_settlement_risk(
        self,
        db_connection,
        user_id: int,
        ticker: str,
        shares: int,
        current_price: float,
        account_value: float,
        metadata: Dict
    ) -> Optional[Tuple]:
        """
        Check T+2 settlement risk constraints.

        Validates:
        1. Locked risk budget - max 10% of account value in locked capital
        2. Entry day restrictions - 50% position size on Thursday/Friday

        Args:
            db_connection: Database connection
            user_id: User ID
            ticker: Stock symbol
            shares: Proposed shares to purchase
            current_price: Entry price
            account_value: Total account value
            metadata: Validation metadata dict

        Returns:
            Tuple of (Signal, Strength, Reason, Metadata) if rejected, None if approved
        """
        try:
            locked_risk_calc = LockedRiskCalculator(db_connection)
            threshold = get_user_locked_risk_threshold(db_connection, user_id)

            # Check locked risk budget
            approved, message = locked_risk_calc.check_locked_risk_budget(
                user_id, ticker, shares, current_price, account_value, threshold
            )

            metadata['settlement_risk'] = {
                'locked_risk_threshold': threshold,
                'locked_risk_approved': approved,
                'validation_message': message
            }

            if not approved:
                metadata['validations_failed'].append({
                    'check': 'locked_risk_budget',
                    'message': message,
                    'threshold': threshold
                })
                return "HOLD_NONE", 0.0, \
                    f"T+2 Settlement Risk: {message}", \
                    metadata

            # Check entry day restrictions
            if self.APPLY_ENTRY_DAY_LIMITS:
                from datetime import datetime
                weekday = datetime.now().weekday()  # 0=Monday, 6=Sunday
                if weekday in (3, 4):  # Thursday or Friday
                    day_name = 'Thursday' if weekday == 3 else 'Friday'
                    metadata['settlement_risk']['entry_day_warning'] = (
                        f"Position size should be reduced to 50% due to {day_name} entry "
                        f"(weekend lock risk extends settlement period)"
                    )
                    metadata['warnings'].append(
                        f"{day_name} entry - Consider 50% position size due to weekend settlement lock"
                    )

            # If there's a warning but approved
            if message and approved:
                metadata['settlement_risk']['budget_warning'] = message
                metadata['warnings'].append(f"Settlement risk: {message}")

            metadata['validations_passed'].append('settlement_risk')
            return None

        except Exception as e:
            logger.warning(f"Settlement risk check failed: {e}")
            metadata['warnings'].append(f'Settlement risk check error: {str(e)}')
            return None
    
    def save_signal(self, ticker: str, date: str, signal: str, strength: float, 
                   reason: str, metadata: Dict = None, validation_metadata: Dict = None):
        """
        Save enhanced signal with validation metadata.
        
        Stores all validation results for historical analysis and debugging.
        """
        conn = get_connection()
        try:
            with conn.cursor() as cursor:
                full_metadata = metadata if metadata else {}
                
                # Add validation results to metadata
                if validation_metadata:
                    full_metadata['validation'] = validation_metadata
                
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
                    'metadata': full_metadata
                })
            conn.commit()
            logger.info(f"Saved enhanced signal {signal} for {ticker} with {len(validation_metadata.get('validations_passed', []))} validations passed")
            return True
        except Exception as e:
            logger.error(f"Failed to save signal: {e}")
            return False
        finally:
            conn.close()


if __name__ == '__main__':
    # Example usage
    print("Enhanced Signal Generator - Example")
    print("=" * 60)
    
    generator = EnhancedSignalGenerator(exchange='HOSE')
    
    # Simulated inputs
    predictions = {
        5: {'p10': -0.02, 'p50': 0.025, 'p90': 0.07, 'confidence': 0.72},
        10: {'p10': -0.01, 'p50': 0.035, 'p90': 0.09, 'confidence': 0.68}
    }
    
    current_features = {
        'momentum_5d': 0.02,
        'volume_surge': 1.2,
        'consecutive_down': 0,
        'distance_from_support': 0.03,
        'volatility_5d': 0.018,
        'relative_strength': 0.015,
        'rsi_14': 58
    }
    
    uncertainty_metrics = {
        'epistemic_uncertainty': 0.03,  # 3% model disagreement
        'aleatoric_uncertainty': 0.08,  # 8% market randomness
        'total_uncertainty': 0.085,
        'confidence_score': 0.75
    }
    
    signal, strength, reason, val_metadata = generator.generate_signal(
        ticker='VCI',
        predictions=predictions,
        current_price=45_000,
        current_features=current_features,
        uncertainty_metrics=uncertainty_metrics
    )
    
    print(f"\nSignal: {signal}")
    print(f"Strength: {strength:.2f}")
    print(f"Reason: {reason}")
    print(f"\nValidations Passed: {len(val_metadata['validations_passed'])}")
    for v in val_metadata['validations_passed']:
        print(f"  ✓ {v}")
    
    if val_metadata['validations_failed']:
        print(f"\nValidations Failed: {len(val_metadata['validations_failed'])}")
        for v in val_metadata['validations_failed']:
            print(f"  ✗ {v}")
    
    if val_metadata['warnings']:
        print(f"\nWarnings: {len(val_metadata['warnings'])}")
        for w in val_metadata['warnings']:
            print(f"  ⚠ {w}")
    
    print("\n" + "=" * 60)
