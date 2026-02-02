import os
import sys
import pandas as pd
from datetime import datetime
from typing import List

# Add parent directory to path
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

from db.connection import get_connection
from db.queries import LOAD_DAILY_BARS, SAVE_FEATURES
from features.calculator import get_all_features, validate_features, calculate_targets

def backfill_features(ticker: str):
    print(f"Starting backfill for {ticker}...")
    
    conn = get_connection()
    
    # 1. Load all daily bars
    with conn.cursor() as cursor:
        print(f"Loading daily bars for {ticker}...")
        # We don't use date range to get everything
        cursor.execute("""
            SELECT symbol as ticker, date, open, high, low, close, volume, turnover
            FROM "stock-trading".daily_bars
            WHERE symbol = %s
            ORDER BY date ASC
        """, (ticker,))
        data = cursor.fetchall()
        
    if not data:
        print(f"No daily bars found for {ticker}")
        conn.close()
        return

    df = pd.DataFrame(data)
    print(f"Loaded {len(df)} bars")

    # 1.5 Convert Decimal columns to float/numeric
    numeric_cols = ['open', 'high', 'low', 'close', 'volume', 'turnover']
    for col in numeric_cols:
        if col in df.columns:
            df[col] = pd.to_numeric(df[col], errors='coerce')
    
    # Ensure date is datetime
    df['date'] = pd.to_datetime(df['date'])
    df = df.sort_values('date')

    # 2. Calculate features
    print("Calculating features...")
    df_features = get_all_features(df)
    df_features = calculate_targets(df_features) # Calculate multi-horizon targets
    
    # Filter for rows where features could be calculated (avoid first few rows with NaNs)
    # Actually we can save everything and mark features_complete = FALSE if missing
    
    # 3. Save features
    print(f"Saving features for {len(df_features)} dates...")
    
    feature_cols = [
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

    with conn.cursor() as cursor:
        for i, row in df_features.iterrows():
            # Check if features are complete for this row
            # validate_features takes a whole dataframe, we need for one row
            row_df = df_features.iloc[[i]]
            is_complete, missing = validate_features(row_df)
            
            # Prepare values for SAVE_FEATURES
            # Order must match SAVE_FEATURES in db/queries.py
            # ticker, date, features..., features_complete, feature_version
            
            values = [
                ticker,
                row['date'],
            ]
            
            for col in feature_cols:
                val = row[col]
                values.append(None if pd.isna(val) else float(val))
            
            values.append(is_complete)
            values.append('v1.0') # feature_version
            
            cursor.execute(SAVE_FEATURES, tuple(values))
            
            if i % 100 == 0:
                print(f"  Processed {i}/{len(df_features)}...")
        
        conn.commit()
    
    print(f"Backfill complete for {ticker}!")
    conn.close()

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python backfill_features.py <ticker>")
        sys.exit(1)
    
    ticker = sys.argv[1].upper()
    backfill_features(ticker)
