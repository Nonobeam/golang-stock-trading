"""
Feature calculator module.
Computes 40+ technical indicators from OHLC data for ML model inputs.
"""

import pandas as pd
import numpy as np
from typing import Dict, Any


def calculate_returns(df: pd.DataFrame, periods: list = [1, 5, 20, 60]) -> pd.DataFrame:
    """
    Calculate returns over various periods.
    
    Args:
        df: DataFrame with 'close' column
        periods: List of periods to calculate returns for
        
    Returns:
        DataFrame with return columns added
    """
    for period in periods:
        df[f'return_{period}d'] = df['close'].pct_change(period)
    return df


def calculate_targets(df: pd.DataFrame) -> pd.DataFrame:
    """
    Calculate target returns for multiple horizons (1d, 5d, 10d).
    These are FUTURE returns used for training labels.
    
    Args:
        df: DataFrame with 'close' column
        
    Returns:
        DataFrame with target_return_Xd columns added
    """
    # 1-day target (next day return)
    df['target_return_1d'] = df['close'].pct_change(1).shift(-1)
    
    # 5-day target (return 5 trading days from now)
    df['target_return_5d'] = df['close'].pct_change(5).shift(-5)
    
    # 10-day target (return 10 trading days from now)
    df['target_return_10d'] = df['close'].pct_change(10).shift(-10)
    
    return df


def calculate_moving_averages(df: pd.DataFrame, windows: list = [5, 10, 20, 50, 200]) -> pd.DataFrame:
    """
    Calculate Simple Moving Averages.
    
    Args:
        df: DataFrame with 'close' column
        windows: List of window sizes
        
    Returns:
        DataFrame with SMA columns added
    """
    for window in windows:
        df[f'sma_{window}'] = df['close'].rolling(window=window).mean()
    return df


def calculate_ema(df: pd.DataFrame, spans: list = [12, 26]) -> pd.DataFrame:
    """
    Calculate Exponential Moving Averages.
    
    Args:
        df: DataFrame with 'close' column
        spans: List of span values
        
    Returns:
        DataFrame with EMA columns added
    """
    for span in spans:
        df[f'ema_{span}'] = df['close'].ewm(span=span, adjust=False).mean()
    return df


def calculate_rsi(df: pd.DataFrame, periods: list = [14, 28]) -> pd.DataFrame:
    """
    Calculate Relative Strength Index.
    
    Args:
        df: DataFrame with 'close' column
        periods: List of RSI periods
        
    Returns:
        DataFrame with RSI columns added
    """
    for period in periods:
        delta = df['close'].diff()
        gain = (delta.where(delta > 0, 0)).rolling(window=period).mean()
        loss = (-delta.where(delta < 0, 0)).rolling(window=period).mean()
        rs = gain / loss
        df[f'rsi_{period}'] = 100 - (100 / (1 + rs))
    return df


def calculate_macd(df: pd.DataFrame, fast: int = 12, slow: int = 26, signal: int = 9) -> pd.DataFrame:
    """
    Calculate MACD (Moving Average Convergence Divergence).
    
    Args:
        df: DataFrame with 'close' column
        fast: Fast EMA period
        slow: Slow EMA period
        signal: Signal line period
        
    Returns:
        DataFrame with MACD columns added
    """
    ema_fast = df['close'].ewm(span=fast, adjust=False).mean()
    ema_slow = df['close'].ewm(span=slow, adjust=False).mean()
    df['macd'] = ema_fast - ema_slow
    df['macd_signal'] = df['macd'].ewm(span=signal, adjust=False).mean()
    df['macd_hist'] = df['macd'] - df['macd_signal']
    return df


def calculate_bollinger_bands(df: pd.DataFrame, window: int = 20, std: float = 2.0) -> pd.DataFrame:
    """
    Calculate Bollinger Bands.
    
    Args:
        df: DataFrame with 'close' column
        window: Rolling window size
        std: Number of standard deviations
        
    Returns:
        DataFrame with Bollinger Band columns added
    """
    df['bb_middle'] = df['close'].rolling(window=window).mean()
    rolling_std = df['close'].rolling(window=window).std()
    df['bb_upper'] = df['bb_middle'] + (rolling_std * std)
    df['bb_lower'] = df['bb_middle'] - (rolling_std * std)
    df['bb_width'] = (df['bb_upper'] - df['bb_lower']) / df['bb_middle']
    return df


def calculate_volume_features(df: pd.DataFrame) -> pd.DataFrame:
    """
    Calculate volume-based features.
    
    Args:
        df: DataFrame with 'volume' and 'close' columns
        
    Returns:
        DataFrame with volume feature columns added
    """
    # Volume ratios
    df['volume_ratio_5d'] = df['volume'] / df['volume'].rolling(window=5).mean()
    df['volume_ratio_20d'] = df['volume'] / df['volume'].rolling(window=20).mean()
    
    # Volume trend
    volume_ma_short = df['volume'].rolling(window=5).mean()
    volume_ma_long = df['volume'].rolling(window=20).mean()
    df['volume_trend'] = (volume_ma_short - volume_ma_long) / volume_ma_long
    
    # On-Balance Volume (OBV)
    df['obv'] = (np.sign(df['close'].diff()) * df['volume']).fillna(0).cumsum()
    
    # Turnover ratios (if turnover column exists)
    if 'turnover' in df.columns:
        df['turnover_ratio_5d'] = df['turnover'] / df['turnover'].rolling(window=5).mean()
        df['turnover_ratio_20d'] = df['turnover'] / df['turnover'].rolling(window=20).mean()
    else:
        df['turnover_ratio_5d'] = None
        df['turnover_ratio_20d'] = None
    
    return df


def calculate_volatility(df: pd.DataFrame, windows: list = [5, 20]) -> pd.DataFrame:
    """
    Calculate volatility metrics.
    
    Args:
        df: DataFrame with 'close', 'high', 'low' columns
        windows: List of window sizes
        
    Returns:
        DataFrame with volatility columns added
    """
    # Rolling standard deviation of returns
    returns = df['close'].pct_change()
    for window in windows:
        df[f'volatility_{window}d'] = returns.rolling(window=window).std()
    
    # Average True Range (ATR)
    high_low = df['high'] - df['low']
    high_close = np.abs(df['high'] - df['close'].shift())
    low_close = np.abs(df['low'] - df['close'].shift())
    true_range = pd.concat([high_low, high_close, low_close], axis=1).max(axis=1)
    df['atr_14'] = true_range.rolling(window=14).mean()
    
    # Coefficient of Variation
    df['coefficient_variation'] = df['close'].rolling(window=20).std() / df['close'].rolling(window=20).mean()
    
    return df


def calculate_price_ratios(df: pd.DataFrame) -> pd.DataFrame:
    """
    Calculate price ratio features.
    
    Args:
        df: DataFrame with price columns
        
    Returns:
        DataFrame with price ratio columns added
    """
    # Price to moving averages
    if 'sma_20' in df.columns:
        df['price_to_sma20'] = df['close'] / df['sma_20']
    if 'sma_50' in df.columns:
        df['price_to_sma50'] = df['close'] / df['sma_50']
    if 'sma_200' in df.columns:
        df['price_to_sma200'] = df['close'] / df['sma_200']
    
    # Range to close ratio
    df['range_to_close'] = (df['high'] - df['low']) / df['close']
    
    return df


def get_all_features(df: pd.DataFrame) -> pd.DataFrame:
    """
    Calculate all features for the dataset.
    
    Args:
        df: DataFrame with OHLC data (columns: open, high, low, close, volume, turnover)
        
    Returns:
        DataFrame with all feature columns added
    """
    # Make a copy to avoid modifying original
    df = df.copy()
    
    # Calculate all features
    df = calculate_returns(df, periods=[1, 5, 20, 60])
    df = calculate_moving_averages(df, windows=[5, 10, 20, 50, 200])
    df = calculate_ema(df, spans=[12, 26])
    df = calculate_rsi(df, periods=[14, 28])
    df = calculate_macd(df)
    df = calculate_bollinger_bands(df, window=20, std=2.0)
    df = calculate_volume_features(df)
    df = calculate_volatility(df, windows=[5, 20])
    df = calculate_price_ratios(df)
    
    return df


def validate_features(df: pd.DataFrame) -> tuple[bool, list]:
    """
    Validate that all required features are present and have valid values.
    
    Args:
        df: DataFrame with calculated features
        
    Returns:
        Tuple of (all_valid: bool, missing_features: list)
    """
    required_features = [
        'return_1d', 'return_5d', 'return_20d', 'return_60d',
        'sma_5', 'sma_10', 'sma_20', 'sma_50', 'sma_200',
        'ema_12', 'ema_26',
        'rsi_14', 'rsi_28',
        'macd', 'macd_signal', 'macd_hist',
        'bb_upper', 'bb_middle', 'bb_lower', 'bb_width',
        'volume_ratio_5d', 'volume_ratio_20d', 'volume_trend', 'obv',
        'volatility_5d', 'volatility_20d', 'atr_14', 'coefficient_variation',
        'price_to_sma20', 'price_to_sma50', 'price_to_sma200', 'range_to_close'
    ]
    
    missing_features = [feat for feat in required_features if feat not in df.columns]
    all_valid = len(missing_features) == 0
    
    return all_valid, missing_features


def get_feature_dict(df: pd.DataFrame, index: int) -> Dict[str, Any]:
    """
    Extract feature dictionary for a specific row.
    
    Args:
        df: DataFrame with all features
        index: Row index to extract
        
    Returns:
        Dictionary of feature names and values
    """
    feature_cols = [col for col in df.columns if col not in ['ticker', 'date', 'open', 'high', 'low', 'close', 'volume', 'turnover']]
    
    features = {}
    for col in feature_cols:
        value = df.iloc[index][col]
        # Convert NaN to None
        features[col] = None if pd.isna(value) else float(value)
    
    return features
