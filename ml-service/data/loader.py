import pandas as pd
from db import get_connection

def load_features_for_training(ticker, min_date=None):
    """
    Load features from database for a specific ticker
    
    Args:
        ticker: Stock symbol
        min_date: Optional minimum date filter
        
    Returns:
        DataFrame with features and target
    """
    conn = get_connection()
    
    query = """
    SELECT 
        f.date,
        f.ticker,
        f.return_1d, f.return_5d, f.return_20d, f.return_60d,
        f.sma_5, f.sma_10, f.sma_20, f.sma_50, f.sma_200,
        f.ema_12, f.ema_26,
        f.rsi_14,
        f.macd, f.macd_signal, f.macd_hist,
        f.bb_upper, f.bb_middle, f.bb_lower,
        f.bb_upper, f.bb_middle, f.bb_lower,
        f.volatility_20d,
        f.volume_ratio_20d,
        f.atr_14,
        f.features_complete,
        db.close as current_close,
        LEAD(db.close, 1) OVER (ORDER BY f.date) as next_close
    FROM features f
    JOIN daily_bars db ON f.ticker = db.symbol AND f.date = db.date
    WHERE f.ticker = %(ticker)s
        AND f.features_complete = TRUE
    """
    
    params = {'ticker': ticker}
    
    if min_date:
        query += " AND f.date >= %(min_date)s"
        params['min_date'] = min_date
    
    query += " ORDER BY f.date ASC"
    
    # psycopg3 uses different parameter style
    with conn.cursor() as cursor:
        cursor.execute(query, params)
        columns = [desc[0] for desc in cursor.description]
        data = cursor.fetchall()
    
    conn.close()
    
    # Convert to DataFrame (data is list of dicts)
    df = pd.DataFrame(data)
    
    # Remove duplicate columns if any
    df = df.loc[:, ~df.columns.duplicated()]
    
    # Convert Decimal columns to float
    for col in df.columns:
        if col not in ['date', 'ticker']:
            df[col] = pd.to_numeric(df[col], errors='coerce')

    # Calculate target (next day return)
    df['target_return'] = (df['next_close'] - df['current_close']) / df['current_close']
    
    # Drop rows where target is null (last day)
    df = df.dropna(subset=['target_return'])
    
    print(f"Loaded {len(df)} samples for {ticker}")
    
    return df

def get_feature_columns():
    """Return list of feature column names"""
    return [
        'return_1d', 'return_5d', 'return_20d', 'return_60d',
        'sma_5', 'sma_10', 'sma_20', 'sma_50', 'sma_200',
        'ema_12', 'ema_26',
        'rsi_14',
        'macd', 'macd_signal', 'macd_hist',
        'bb_upper', 'bb_middle', 'bb_lower',
        'bb_upper', 'bb_middle', 'bb_lower',
        'volatility_20d',
        'volume_ratio_20d',
        'atr_14'
    ]

def check_data_availability(ticker):
    """Check how many days of data available for ticker"""
    conn = get_connection()
    
    query = """
    SELECT COUNT(*) as total_days,
           COUNT(*) FILTER (WHERE features_complete = TRUE) as complete_days,
           MIN(date) as earliest_date,
           MAX(date) as latest_date
    FROM features
    WHERE ticker = %(ticker)s
    """
    
    with conn.cursor() as cursor:
        cursor.execute(query, {'ticker': ticker})
        result = cursor.fetchone()
    
    conn.close()
    
    return result
    

class DataLoader:
    """
    Wrapper class for data loading operations.
    Used by the Predictor and other components.
    """
    def __init__(self):
        pass
        
    def load_daily_bars_recent(self, ticker, days=250):
        """
        Load recent daily bars for feature calculation.
        
        Args:
            ticker: Stock symbol
            days: Number of days to load (default: 250)
            
        Returns:
            DataFrame with OHLCV data
        """
        conn = get_connection()
        query = """
        SELECT date, open, high, low, close, volume
        FROM daily_bars 
        WHERE symbol = %(ticker)s
        ORDER BY date DESC
        LIMIT %(days)s
        """
        
        with conn.cursor() as cursor:
            cursor.execute(query, {'ticker': ticker, 'days': days})
            columns = [desc[0] for desc in cursor.description]
            data = cursor.fetchall()
        
        conn.close()
        
        if not data:
            return pd.DataFrame()
            
        df = pd.DataFrame(data, columns=columns)
        
        # Convert types
        for col in ['open', 'high', 'low', 'close', 'volume']:
            df[col] = pd.to_numeric(df[col])
            
        df['date'] = pd.to_datetime(df['date'])
        
        # Sort ascending by date
        df = df.sort_values('date')
        
        return df
        
    @staticmethod
    def load_daily_bars(ticker, start_date, end_date):
        """
        Load daily bars for a date range.
        
        Args:
            ticker: Stock symbol
            start_date: Start date string
            end_date: End date string
            
        Returns:
            DataFrame with OHLCV data
        """
        from db.connection import get_connection
        from db.queries import LOAD_DAILY_BARS
        
        conn = get_connection()
        try:
            with conn.cursor() as cursor:
                cursor.execute(LOAD_DAILY_BARS, (ticker, start_date, end_date))
                columns = [desc[0] for desc in cursor.description]
                data = cursor.fetchall()
                
            if not data:
                return pd.DataFrame()
                
            df = pd.DataFrame(data, columns=columns)
            
            # Convert types
            for col in ['open', 'high', 'low', 'close', 'volume']:
                df[col] = pd.to_numeric(df[col])
                
            df['date'] = pd.to_datetime(df['date'])
            
            return df
        finally:
            conn.close()

    @staticmethod
    def save_features(ticker, date, feature_dict, feature_version='v1.0'):
        """
        Save calculated features to database.
        
        Args:
            ticker: Stock symbol
            date: Target date string
            feature_dict: Dictionary of calculated features
            feature_version: Version string for the feature set
        """
        from db.connection import get_connection
        from db.queries import SAVE_FEATURES
        
        # Define the exact order of columns matching the SQL query
        columns = [
            'return_1d', 'return_5d', 'return_20d', 'return_60d',
            'sma_5', 'sma_10', 'sma_20', 'sma_50', 'sma_200',
            'ema_12', 'ema_26',
            'rsi_14', 'rsi_28', 'macd', 'macd_signal', 'macd_hist',
            'bb_upper', 'bb_middle', 'bb_lower', 'bb_width',
            'volume_ratio_5d', 'volume_ratio_20d', 'volume_trend', 'obv',
            'turnover_ratio_5d', 'turnover_ratio_20d',
            'volatility_5d', 'volatility_20d', 'atr_14', 'coefficient_variation',
            'price_to_sma20', 'price_to_sma50', 'price_to_sma200', 'range_to_close',
            'target_return_5d', 'target_return_10d'
        ]
        
        # Build params tuple in correct order, using None for missing keys
        params = [ticker, date]
        for col in columns:
            val = feature_dict.get(col)
            # Convert numpy NaN/Inf to None for psycopg
            if pd.isna(val) or val == float('inf') or val == float('-inf'):
                params.append(None)
            else:
                params.append(val)
                
        # Append features_complete flag and version
        params.extend([True, feature_version])
        
        conn = get_connection()
        try:
            with conn.cursor() as cursor:
                cursor.execute(SAVE_FEATURES, tuple(params))
            conn.commit()
        except Exception as e:
            conn.rollback()
            raise Exception(f"Failed to save features: {e}")
        finally:
            conn.close()

    @staticmethod
    def get_feature_columns():
        """Wrapper for get_feature_columns function"""
        return get_feature_columns()

    @staticmethod
    def get_active_tickers(target_date=None):
        """Get all active tickers from the last 7 days (relative to target_date)"""
        from db.connection import get_connection
        from db.queries import GET_ALL_ACTIVE_TICKERS
        
        conn = get_connection()
        try:
            with conn.cursor() as cursor:
                cursor.execute(GET_ALL_ACTIVE_TICKERS, (target_date,))
                data = cursor.fetchall()
            return [row['ticker'] for row in data]
        finally:
            conn.close()

    @staticmethod
    def get_all_distinct_dates():
        """Get all distinct dates from the daily_bars table"""
        from db.connection import get_connection
        from db.queries import GET_ALL_DISTINCT_DATES
        
        conn = get_connection()
        try:
            with conn.cursor() as cursor:
                cursor.execute(GET_ALL_DISTINCT_DATES)
                data = cursor.fetchall()
            return [row['date'].strftime('%Y-%m-%d') for row in data]
        finally:
            conn.close()

    @staticmethod
    def get_latest_prediction_date():
        """Get the latest prediction_date available in predictions table"""
        from db.connection import get_connection
        from db.queries import GET_LATEST_PREDICTION_DATE
        
        conn = get_connection()
        try:
            with conn.cursor() as cursor:
                cursor.execute(GET_LATEST_PREDICTION_DATE)
                row = cursor.fetchone()
                if row and row['max']:
                    return row['max'].strftime('%Y-%m-%d')
                return None
        finally:
            conn.close()