"""
Stock universe loader.

Loads the curated 50-stock universe from the `stock_universe` database table.
Returns a list of dicts and helper sector/volume maps.
"""
import logging
from typing import Dict, List, Optional

from db.connection import get_connection

logger = logging.getLogger(__name__)


def load_universe(active_only: bool = True) -> List[Dict]:
    """
    Load stock universe from database.

    Args:
        active_only: If True (default), return only tickers where is_active = TRUE.

    Returns:
        List of dicts with keys: ticker, sector, exchange, is_active, notes.
    """
    conn = get_connection()
    try:
        with conn.cursor() as cur:
            if active_only:
                cur.execute(
                    'SELECT ticker, sector, exchange, is_active, notes '
                    'FROM "stock-trading".stock_universe '
                    'WHERE is_active = TRUE '
                    'ORDER BY ticker'
                )
            else:
                cur.execute(
                    'SELECT ticker, sector, exchange, is_active, notes '
                    'FROM "stock-trading".stock_universe '
                    'ORDER BY ticker'
                )
            rows = cur.fetchall()
        logger.info(f"Loaded {len(rows)} {'active ' if active_only else ''}universe stocks")
        return rows
    except Exception as e:
        logger.error(f"Failed to load stock universe: {e}")
        raise
    finally:
        conn.close()


def get_sector_map(universe: Optional[List[Dict]] = None) -> Dict[str, str]:
    """
    Build ticker -> sector mapping.

    Args:
        universe: Pre-loaded universe list; if None, loads from DB.

    Returns:
        Dict of {ticker: sector}.
    """
    rows = universe if universe is not None else load_universe()
    return {row["ticker"]: row["sector"] for row in rows}


def get_volume_map(tickers: List[str], lookback_days: int = 90, date: Optional[str] = None) -> Dict[str, float]:
    """
    Compute average daily volume (in thousands) per ticker from daily_bars.

    Args:
        tickers:      List of ticker symbols to compute volume for.
        lookback_days: Number of calendar days to look back.
        date:         Reference date (YYYY-MM-DD); defaults to today.

    Returns:
        Dict {ticker: avg_daily_volume_in_thousands}. Missing tickers default to 0.
    """
    import pandas as pd

    ref_date = date or pd.Timestamp.now().strftime("%Y-%m-%d")
    conn = get_connection()
    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                SELECT ticker, AVG(volume) / 1000.0 AS avg_vol_k
                FROM "stock-trading".daily_bars
                WHERE ticker = ANY(%(tickers)s)
                  AND date >= (%(ref_date)s::date - %(days)s)
                  AND date <= %(ref_date)s::date
                GROUP BY ticker
                """,
                {"tickers": tickers, "ref_date": ref_date, "days": lookback_days},
            )
            rows = cur.fetchall()
    except Exception as e:
        logger.error(f"Failed to compute volume from daily_bars: {e}")
        return {t: 0.0 for t in tickers}
    finally:
        conn.close()

    vol_map = {row["ticker"]: float(row["avg_vol_k"] or 0) for row in rows}
    # Fill missing tickers with 0
    for t in tickers:
        if t not in vol_map:
            vol_map[t] = 0.0
    logger.info(f"Computed avg daily volume from daily_bars for {len(vol_map)} tickers")
    return vol_map



def get_ticker_list(universe: Optional[List[Dict]] = None) -> List[str]:
    """Return sorted list of tickers from the universe."""
    rows = universe if universe is not None else load_universe()
    return [row["ticker"] for row in rows]
