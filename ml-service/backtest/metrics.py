"""
Backtest metrics module.
Calculates performance metrics for trading strategies and ML models.
"""

import numpy as np
import pandas as pd
from typing import Dict, List, Union

def calculate_sharpe_ratio(returns: pd.Series, risk_free_rate: float = 0.0) -> float:
    """
    Calculate annualized Sharpe Ratio.
    Assumes daily returns.
    
    Args:
        returns: Series of daily returns
        risk_free_rate: Annual risk-free rate (decimal)
        
    Returns:
        Sharpe ratio
    """
    if len(returns) < 2:
        return 0.0
        
    excess_returns = returns - (risk_free_rate / 252)
    mean_excess_return = excess_returns.mean()
    std_dev = excess_returns.std()
    
    if std_dev == 0:
        return 0.0
        
    # Annualize (sqrt(252) approx 15.87)
    return float((mean_excess_return / std_dev) * np.sqrt(252))

def calculate_max_drawdown(equity_curve: pd.Series) -> float:
    """
    Calculate Maximum Drawdown.
    
    Args:
        equity_curve: Series of equity values
        
    Returns:
        Maximum drawdown as a positive decimal (e.g., 0.20 for 20% drawdown)
    """
    if len(equity_curve) < 1:
        return 0.0
        
    # Calculate running max
    running_max = equity_curve.cummax()
    
    # Calculate drawdown
    drawdown = (equity_curve - running_max) / running_max
    
    # Max drawdown is the minimum (most negative) value
    max_dd = drawdown.min()
    
    # Return as positive value
    return abs(float(max_dd))

def calculate_win_rate(predictions: pd.Series, outcomes: pd.Series) -> float:
    """
    Calculate Win Rate (Directional Accuracy).
    
    Args:
        predictions: Series of predicted returns (or values)
        outcomes: Series of actual returns
        
    Returns:
        Win rate as decimal [0, 1]
    """
    if len(predictions) != len(outcomes) or len(predictions) == 0:
        return 0.0
    
    # Direction match: (pred > 0 and actual > 0) or (pred < 0 and actual < 0)
    # We can multiply signs
    matches = np.sign(predictions) == np.sign(outcomes)
    
    # Handle zero cases if necessary, but usually sign(0)=0
    
    return float(matches.mean())

def check_calibration(y_true: np.ndarray, p_pred: np.ndarray, quantile: float) -> float:
    """
    Check calibration for a specific quantile.
    
    Args:
        y_true: Actual values
        p_pred: Predicted quantile values
        quantile: Quantile level (e.g. 0.10)
        
    Returns:
        Observed frequency (should be close to quantile)
    """
    if len(y_true) == 0:
        return 0.0
        
    observed_freq = np.mean(y_true < p_pred)
    return float(observed_freq)

def generate_metrics_report(results_df: pd.DataFrame) -> Dict[str, float]:
    """
    Generate comprehensive metrics report from backtest results.
    
    Args:
        results_df: DataFrame with 'date', 'prediction', 'actual_return' keys.
                    Optional: 'equity' column if strategy simulated.
                    
    Returns:
        Dictionary of metrics
    """
    metrics = {}
    
    if 'actual_return' not in results_df.columns or 'prediction' not in results_df.columns:
        return metrics
    
    # Clean data
    df = results_df.dropna(subset=['actual_return', 'prediction'])
    
    if df.empty:
        return metrics
        
    # 1. Directional Accuracy
    metrics['win_rate'] = calculate_win_rate(df['prediction'], df['actual_return'])
    
    # 2. Information Coefficient (IC)
    metrics['ic'] = float(df['prediction'].corr(df['actual_return']))
    
    # 3. Error Metrics
    metrics['mae'] = float(np.mean(np.abs(df['prediction'] - df['actual_return'])))
    metrics['rmse'] = float(np.sqrt(np.mean((df['prediction'] - df['actual_return'])**2)))
    
    # 4. Strategy Simulation (assuming simple long-only if pred > 0)
    # This is a naive strategy for metric purposes
    strategy_returns = np.sign(df['prediction']) * df['actual_return']
    metrics['sharpe_ratio'] = calculate_sharpe_ratio(strategy_returns)
    
    # Equity curve for drawdown
    equity = (1 + strategy_returns).cumprod()
    metrics['max_drawdown'] = calculate_max_drawdown(equity)
    
    metrics['total_return'] = float(equity.iloc[-1] - 1) if not equity.empty else 0.0
    
    return metrics
