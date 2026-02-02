"""
Pytest configuration and shared fixtures
"""
import pytest
import pandas as pd
import numpy as np
from datetime import datetime, timedelta
from db import get_connection

@pytest.fixture(scope="session")
def db_connection():
    """Database connection fixture"""
    conn = get_connection()
    yield conn
    conn.close()

@pytest.fixture
def sample_ticker():
    """Sample ticker symbol for testing"""
    return "TEST"

@pytest.fixture
def sample_dates():
    """Generate 500 consecutive trading dates"""
    start_date = datetime(2022, 1, 3)
    dates = []
    current = start_date
    
    while len(dates) < 600:
        if current.weekday() < 5:
            dates.append(current)
        current += timedelta(days=1)
    
    return dates

@pytest.fixture
def sample_daily_bars(db_connection, sample_ticker, sample_dates):
    """Insert sample daily bars into database"""
    with db_connection.cursor() as cursor:
        # Clean up existing test data
        cursor.execute("DELETE FROM daily_bars WHERE symbol = %(symbol)s", {'symbol': sample_ticker})
        cursor.execute("DELETE FROM features WHERE ticker = %(ticker)s", {'ticker': sample_ticker})
        cursor.execute("DELETE FROM model_metadata WHERE ticker = %(ticker)s", {'ticker': sample_ticker})
        
        # Generate realistic price data
        base_price = 100.0
        prices = [base_price]
        
        for i in range(1, 600):
            change = np.random.normal(0.001, 0.02)
            new_price = prices[-1] * (1 + change)
            prices.append(new_price)
        
        # Insert daily bars
        for date, price in zip(sample_dates, prices):
            high = price * (1 + abs(np.random.normal(0, 0.01)))
            low = price * (1 - abs(np.random.normal(0, 0.01)))
            open_price = price * (1 + np.random.normal(0, 0.005))
            volume = int(np.random.uniform(1000000, 5000000))
            turnover = price * volume
            
            cursor.execute("""
                INSERT INTO daily_bars (symbol, date, open, high, low, close, volume, turnover)
                VALUES (%(symbol)s, %(date)s, %(open)s, %(high)s, %(low)s, 
                        %(close)s, %(volume)s, %(turnover)s)
            """, {
                'symbol': sample_ticker,
                'date': date,
                'open': open_price,
                'high': high,
                'low': low,
                'close': price,
                'volume': volume,
                'turnover': turnover
            })
        
        db_connection.commit()
    
    yield sample_ticker
    
    # Cleanup
    with db_connection.cursor() as cursor:
        cursor.execute("DELETE FROM daily_bars WHERE symbol = %(symbol)s", {'symbol': sample_ticker})
        cursor.execute("DELETE FROM features WHERE ticker = %(ticker)s", {'ticker': sample_ticker})
        cursor.execute("DELETE FROM model_metadata WHERE ticker = %(ticker)s", {'ticker': sample_ticker})
        db_connection.commit()

@pytest.fixture
def sample_features(db_connection, sample_daily_bars, sample_dates):
    """Calculate and insert sample features"""
    from data.loader import get_feature_columns
    
    ticker = sample_daily_bars
    
    with db_connection.cursor() as cursor:
        # Get price data
        cursor.execute("""
            SELECT date, close, volume
            FROM daily_bars
            WHERE symbol = %(symbol)s
            ORDER BY date
        """, {'symbol': ticker})
        
        data = cursor.fetchall()
        closes = [float(d['close']) for d in data]
        dates = [d['date'] for d in data]
        
        # Calculate simple features for each day
        for i in range(len(dates)):
            features_complete = i >= 200
            
            return_1d = (closes[i] - closes[i-1]) / closes[i-1] if i > 0 else 0
            return_5d = (closes[i] - closes[i-5]) / closes[i-5] if i >= 5 else 0
            sma_20 = np.mean(closes[max(0, i-19):i+1]) if i >= 19 else closes[i]
            volatility_20 = np.std([
                (closes[j] - closes[j-1]) / closes[j-1] 
                for j in range(max(1, i-19), i+1)
            ]) if i >= 20 else 0
            
            cursor.execute("""
                INSERT INTO features (
                    ticker, date, 
                    return_1d, return_5d, return_20d, return_60d,
                    sma_5, sma_10, sma_20, sma_50, sma_200,
                    ema_12, ema_26,
                    rsi_14, macd, macd_signal, macd_hist,
                    bb_upper, bb_middle, bb_lower,
                    volatility_20d,
                    volume_ratio_20d,
                    atr_14,
                    features_complete
                ) VALUES (
                    %(ticker)s, %(date)s,
                    %(return_1d)s, %(return_5d)s, 0, 0,
                    %(sma_20)s, %(sma_20)s, %(sma_20)s, %(sma_20)s, %(sma_20)s,
                    %(sma_20)s, %(sma_20)s,
                    50, 0, 0, 0,
                    %(bb_upper)s, %(sma_20)s, %(bb_lower)s,
                    %(volatility_20)s,
                    1.0,
                    %(volatility_20)s,
                    %(features_complete)s
                )
            """, {
                'ticker': ticker,
                'date': dates[i],
                'return_1d': return_1d,
                'return_5d': return_5d,
                'sma_20': sma_20,
                'bb_upper': sma_20 * 1.02,
                'bb_lower': sma_20 * 0.98,
                'volatility_20': volatility_20,
                'features_complete': features_complete
            })
        
        db_connection.commit()
    
    return ticker