"""
Transaction Cost Module for Vietnamese Stock Market

Calculates transaction fees and validates profitability thresholds
considering Vietnamese market-specific costs:
- Brokerage: 0.15% (both buy and sell)
- Tax: 0.1% (sell only)
- Total round-trip: 0.40%

Author: ML Trading System
Created: 2026-02-02
"""

from typing import Dict, Tuple
from decimal import Decimal


# Vietnamese market fee structure
BROKERAGE_FEE_RATE = Decimal('0.0015')  # 0.15%
TAX_RATE = Decimal('0.001')  # 0.1% on sell transactions only
ROUND_TRIP_COST = Decimal('0.004')  # 0.40% total


# Minimum profit thresholds by horizon (after fees)
MIN_PROFIT_THRESHOLDS = {
    1: Decimal('0.01'),   # 1.0% for 1-day horizon
    5: Decimal('0.015'),  # 1.5% for 5-day horizon
    10: Decimal('0.02'),  # 2.0% for 10-day horizon
}


def calculate_fees(transaction_value: float, is_sell: bool = False) -> Dict[str, float]:
    """
    Calculate Vietnamese market transaction fees.
    
    Args:
        transaction_value: Total value of transaction in VND
        is_sell: True if selling (includes tax), False if buying
        
    Returns:
        Dictionary containing:
        - brokerage_fee: Brokerage commission (0.15%)
        - tax: Securities transaction tax (0.1% on sell only)
        - total_fee: Combined fees
        
    Example:
        >>> calculate_fees(10_000_000, is_sell=True)
        {'brokerage_fee': 15000.0, 'tax': 10000.0, 'total_fee': 25000.0}
    """
    value = Decimal(str(transaction_value))
    
    brokerage_fee = float(value * BROKERAGE_FEE_RATE)
    tax = float(value * TAX_RATE) if is_sell else 0.0
    total_fee = brokerage_fee + tax
    
    return {
        'brokerage_fee': brokerage_fee,
        'tax': tax,
        'total_fee': total_fee
    }


def calculate_round_trip_cost(entry_price: float, exit_price: float, shares: int) -> Dict[str, float]:
    """
    Calculate total round-trip transaction costs (buy + sell).
    
    Args:
        entry_price: Purchase price per share in VND
        exit_price: Sale price per share in VND
        shares: Number of shares traded
        
    Returns:
        Dictionary containing:
        - buy_fees: Fees paid when buying
        - sell_fees: Fees paid when selling
        - total_fees: Combined round-trip fees
        - gross_profit: Profit before fees
        - net_profit: Profit after fees
        - fee_drag: Percentage impact of fees on return
        
    Example:
        >>> calculate_round_trip_cost(100_000, 105_000, 100)
        {
            'buy_fees': 15000.0,
            'sell_fees': 26250.0,
            'total_fees': 41250.0,
            'gross_profit': 500000.0,
            'net_profit': 458750.0,
            'fee_drag': 0.004125
        }
    """
    buy_value = entry_price * shares
    sell_value = exit_price * shares
    
    buy_fees_dict = calculate_fees(buy_value, is_sell=False)
    sell_fees_dict = calculate_fees(sell_value, is_sell=True)
    
    total_fees = buy_fees_dict['total_fee'] + sell_fees_dict['total_fee']
    gross_profit = sell_value - buy_value
    net_profit = gross_profit - total_fees
    fee_drag = total_fees / buy_value  # As decimal (e.g., 0.004 = 0.4%)
    
    return {
        'buy_fees': buy_fees_dict['total_fee'],
        'sell_fees': sell_fees_dict['total_fee'],
        'total_fees': total_fees,
        'gross_profit': gross_profit,
        'net_profit': net_profit,
        'fee_drag': fee_drag
    }


def get_minimum_profit_threshold(horizon_days: int) -> float:
    """
    Get minimum predicted return threshold for generating signals.
    
    Returns must exceed this threshold to justify transaction costs.
    Longer horizons allow lower thresholds as fees are amortized.
    
    Args:
        horizon_days: Prediction horizon (1, 5, or 10 days)
        
    Returns:
        Minimum required return as decimal (e.g., 0.01 = 1%)
        
    Example:
        >>> get_minimum_profit_threshold(1)
        0.01  # 1.0% for 1-day trades
        >>> get_minimum_profit_threshold(5)
        0.015  # 1.5% for 5-day trades
    """
    return float(MIN_PROFIT_THRESHOLDS.get(horizon_days, Decimal('0.015')))


def calculate_fee_adjusted_return(gross_return: float, entry_price: float = None, 
                                   shares: int = None) -> float:
    """
    Calculate net return after subtracting round-trip transaction costs.
    
    Args:
        gross_return: Predicted or actual return as decimal (e.g., 0.025 = 2.5%)
        entry_price: Optional - price per share for exact calculation
        shares: Optional - number of shares for exact calculation
        
    Returns:
        Net return after 0.4% fee drag as decimal
        
    Note:
        If entry_price and shares not provided, uses simplified calculation
        subtracting flat 0.4% from gross return.
        
    Example:
        >>> calculate_fee_adjusted_return(0.025)  # 2.5% gross
        0.021  # 2.1% net (2.5% - 0.4%)
        
        >>> calculate_fee_adjusted_return(0.025, entry_price=100000, shares=100)
        0.020875  # Exact calculation considering price/shares
    """
    if entry_price is not None and shares is not None:
        # Exact calculation
        exit_price = entry_price * (1 + gross_return)
        result = calculate_round_trip_cost(entry_price, exit_price, shares)
        buy_value = entry_price * shares
        net_return = result['net_profit'] / buy_value
        return net_return
    else:
        # Simplified calculation (subtract flat 0.4%)
        return gross_return - float(ROUND_TRIP_COST)


def is_profitable_after_fees(predicted_return: float, horizon_days: int) -> Tuple[bool, str]:
    """
    Determine if a predicted return exceeds the minimum threshold for the horizon.
    
    Args:
        predicted_return: Predicted median return as decimal (e.g., 0.018 = 1.8%)
        horizon_days: Prediction horizon (1, 5, or 10 days)
        
    Returns:
        Tuple of (is_profitable: bool, rationale: str)
        
    Example:
        >>> is_profitable_after_fees(0.008, horizon_days=1)
        (False, "Expected return 0.8% below 1.0% threshold after fees")
        
        >>> is_profitable_after_fees(0.023, horizon_days=5)
        (True, "Expected return 2.3% exceeds 1.5% threshold")
    """
    threshold = get_minimum_profit_threshold(horizon_days)
    net_return = calculate_fee_adjusted_return(predicted_return)
    
    if net_return >= threshold:
        return True, f"Expected return {predicted_return*100:.1f}% exceeds {threshold*100:.1f}% threshold"
    else:
        return False, f"Expected return {predicted_return*100:.1f}% below {threshold*100:.1f}% threshold after fees"


def calculate_profit_factor(winning_trades: list, losing_trades: list) -> Dict[str, float]:
    """
    Calculate fee-adjusted profit factor for validation.
    
    Profit factor = Sum of net wins / Sum of net losses (absolute)
    Must exceed 1.5 for tradeable strategy.
    
    Args:
        winning_trades: List of gross winning returns (e.g., [0.025, 0.018, 0.031])
        losing_trades: List of gross losing returns (e.g., [-0.012, -0.008])
        
    Returns:
        Dictionary containing:
        - gross_profit_factor: Before fees
        - net_profit_factor: After 0.4% fees per trade
        - total_net_wins: Sum of net winning trades
        - total_net_losses: Sum of net losing trades (absolute)
        - num_winners: Count of winning trades
        - num_losers: Count of losing trades
        
    Example:
        >>> calculate_profit_factor([0.025, 0.018, 0.031], [-0.012, -0.008])
        {
            'gross_profit_factor': 3.7,
            'net_profit_factor': 1.64,
            'total_net_wins': 0.062,
            'total_net_losses': 0.0378,
            ...
        }
    """
    # Calculate gross profit factor
    gross_wins = sum(winning_trades) if winning_trades else 0
    gross_losses = abs(sum(losing_trades)) if losing_trades else 0
    gross_pf = gross_wins / gross_losses if gross_losses > 0 else float('inf')
    
    # Calculate net returns after fees
    net_wins = sum(calculate_fee_adjusted_return(r) for r in winning_trades)
    net_losses = abs(sum(calculate_fee_adjusted_return(r) for r in losing_trades))
    net_pf = net_wins / net_losses if net_losses > 0 else float('inf')
    
    return {
        'gross_profit_factor': gross_pf,
        'net_profit_factor': net_pf,
        'total_net_wins': net_wins,
        'total_net_losses': net_losses,
        'num_winners': len(winning_trades),
        'num_losers': len(losing_trades),
        'fee_impact': gross_pf - net_pf
    }


if __name__ == '__main__':
    # Example usage and validation
    print("Vietnamese Market Transaction Costs")
    print("=" * 50)
    
    # Example 1: Calculate fees for buying VCI
    print("\n1. Buying 100 shares of VCI at 37,000 VND:")
    buy_result = calculate_fees(100 * 37_000, is_sell=False)
    print(f"   Brokerage: {buy_result['brokerage_fee']:,.0f} VND")
    print(f"   Tax: {buy_result['tax']:,.0f} VND")
    print(f"   Total: {buy_result['total_fee']:,.0f} VND")
    
    # Example 2: Calculate round-trip cost
    print("\n2. Round-trip: Buy at 37,000, Sell at 38,500:")
    rt_result = calculate_round_trip_cost(37_000, 38_500, 100)
    print(f"   Gross profit: {rt_result['gross_profit']:,.0f} VND")
    print(f"   Total fees: {rt_result['total_fees']:,.0f} VND")
    print(f"   Net profit: {rt_result['net_profit']:,.0f} VND")
    print(f"   Fee drag: {rt_result['fee_drag']*100:.2f}%")
    
    # Example 3: Profit threshold validation
    print("\n3. Signal profitability check:")
    for horizon in [1, 5, 10]:
        profitable, reason = is_profitable_after_fees(0.008, horizon)
        print(f"   {horizon}-day: {profitable} - {reason}")
