
"""
Position Sizing using Fixed Fractional Approach.
"""

class PositionSizer:
    """Calculates optimal position size using fixed fractional approach with confidence and horizon adjustments."""
    
    def __init__(self, base_fraction=0.10, max_allocation=0.20):
        """
        Initialize position sizer.
        
        Args:
            base_fraction: Base allocation per position (default 10%)
            max_allocation: Maximum allocation cap per stock (default 20%)
        """
        self.base_fraction = base_fraction
        self.max_allocation = max_allocation
        
    def calculate_size(self, prediction_dict: dict, horizon: int = 5) -> float:
        """
        Calculate position size fraction using fixed fractional approach.
        
        Formula: f = f_base × m_confidence × m_horizon
        
        Args:
            prediction_dict: Dict with 'p10', 'p50', 'p90' keys
            horizon: Forecast horizon in days (1, 5, or 10)
        
        Returns:
            Float representing fraction of capital to allocate (0.0 to max_allocation)
        """
        p10 = prediction_dict.get('p10', 0.0)
        p50 = prediction_dict.get('p50', 0.0)
        p90 = prediction_dict.get('p90', 0.0)
        
        # Input validation
        if p50 <= 0:
            return 0.0  # Don't buy stocks with negative expected return
            
        if p10 >= p90:
            raise ValueError("Quantiles inverted - check model")
            
        prediction_range = p90 - p10
        
        if prediction_range < 0.001:  # Range under 0.1%
            return 0.01  # Minimum position size of 1% for zero variance case
        
        # Confidence multiplier based on prediction range
        if prediction_range < 0.05:  # Less than 5%
            confidence_multiplier = 1.5
        elif prediction_range <= 0.15:  # Between 5% and 15%
            confidence_multiplier = 1.0
        else:  # Greater than 15%
            confidence_multiplier = 0.5
        
        # Horizon multiplier
        horizon_multipliers = {
            1: 0.8,   # 1-day predictions
            5: 1.0,   # 5-day predictions
            10: 1.2   # 10-day predictions
        }
        horizon_multiplier = horizon_multipliers.get(horizon, 1.0)
        
        # Calculate position fraction
        position_fraction = self.base_fraction * confidence_multiplier * horizon_multiplier
        
        # Apply maximum cap
        final_allocation = min(position_fraction, self.max_allocation)
        
        return final_allocation
    
    def calculate_shares(self, account_value: float, price: float, 
                         prediction_dict: dict, horizon: int = 5) -> int:
        """
        Calculate number of shares to buy.
        
        Args:
            account_value: Total account value in VND
            price: Current stock price in VND
            prediction_dict: Dict with 'p10', 'p50', 'p90' keys
            horizon: Forecast horizon in days
        
        Returns:
            Integer number of shares to buy
        """
        position_fraction = self.calculate_size(prediction_dict, horizon)
        position_size_vnd = account_value * position_fraction
        shares = int(position_size_vnd / price)  # Round down to integer
        
        return shares
    
    def calculate_position_change(self, ticker: str, account_value: float, 
                                  current_price: float, prediction_dict: dict,
                                  db_connection=None, user_id: int = 1,
                                  horizon: int = 5, tolerance: float = 0.10) -> dict:
        """
        Calculate recommended position change for portfolio-aware sizing.
        
        Compares current position size to optimal allocation and recommends
        BUY_MORE or SELL_PARTIAL if difference exceeds tolerance.
        
        Args:
            ticker: Stock symbol
            account_value: Total account value in VND (sum of positions + cash)
            current_price: Current market price
            prediction_dict: Dict with 'p10', 'p50', 'p90' keys
            db_connection: Optional database connection for position queries
            user_id: User ID (default=1)
            horizon: Forecast horizon in days
            tolerance: Tolerance for position mismatch (default 10%)
        
        Returns:
            Dict with keys:
                - action: 'BUY_MORE', 'SELL_PARTIAL', 'HOLD'
                - current_shares: Current position size
                - recommended_shares: Optimal position size
                - delta_shares: Shares to buy (positive) or sell (negative)
                - current_allocation: Current % of portfolio
                - recommended_allocation: Optimal % of portfolio
                - reason: Explanation string
        """
        from position_manager.manager import PositionManager
        
        # Get current position if db_connection provided
        current_shares = 0
        if db_connection:
            pm = PositionManager(db_connection)
            position = pm.get_position(ticker, user_id)
            if position:
                current_shares = position['quantity']
                
                # Check if price already at first target - don't recommend buying more
                target_1 = position.get('target_1')
                if target_1 and current_price >= target_1:
                    return {
                        'action': 'HOLD',
                        'current_shares': current_shares,
                        'recommended_shares': current_shares,
                        'delta_shares': 0,
                        'current_allocation': (current_shares * current_price) / account_value,
                        'recommended_allocation': (current_shares * current_price) / account_value,
                        'reason': 'Price at/above Target 1 - do not add to position'
                    }
        
        # Calculate optimal position size
        recommended_shares = self.calculate_shares(
            account_value, current_price, prediction_dict, horizon
        )
        
        # Calculate allocations
        current_value = current_shares * current_price
        recommended_value = recommended_shares * current_price
        
        current_allocation = current_value / account_value if account_value > 0 else 0
        recommended_allocation = recommended_value / account_value if account_value > 0 else 0
        
        # Determine action based on tolerance
        delta_shares = recommended_shares - current_shares
        allocation_diff = abs(recommended_allocation - current_allocation)
        
        if allocation_diff <= tolerance:
            # Within tolerance - no action
            return {
                'action': 'HOLD',
                'current_shares': current_shares,
                'recommended_shares': recommended_shares,
                'delta_shares': 0,
                'current_allocation': current_allocation,
                'recommended_allocation': recommended_allocation,
                'reason': f'Current allocation {current_allocation:.1%} within {tolerance:.0%} of optimal {recommended_allocation:.1%}'
            }
        elif delta_shares > 0:
            # Recommended position is larger - buy more
            return {
                'action': 'BUY_MORE',
                'current_shares': current_shares,
                'recommended_shares': recommended_shares,
                'delta_shares': delta_shares,
                'current_allocation': current_allocation,
                'recommended_allocation': recommended_allocation,
                'reason': f'Underweight by {allocation_diff:.1%} - recommend buying {delta_shares} shares'
            }
        else:
            # Recommended position is smaller - sell partial
            return {
                'action': 'SELL_PARTIAL',
                'current_shares': current_shares,
                'recommended_shares': recommended_shares,
                'delta_shares': delta_shares,  # negative value
                'current_allocation': current_allocation,
                'recommended_allocation': recommended_allocation,
                'reason': f'Overweight by {allocation_diff:.1%} - recommend selling {abs(delta_shares)} shares'
            }

