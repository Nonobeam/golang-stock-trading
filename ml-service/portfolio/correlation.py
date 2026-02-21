"""
Pairwise Pearson correlation matrix builder.

Fetches 90 calendar days of daily close prices from `daily_bars` for the
universe of stocks, computes daily returns, and returns a correlation matrix
as a nested dict and as a pandas DataFrame.
"""
import logging
from typing import Dict, List, Optional, Tuple

import numpy as np
import pandas as pd

from db.connection import get_connection
import config

logger = logging.getLogger(__name__)

_CFG = config.PORTFOLIO_CONFIG


def build_correlation_matrix(
    tickers: List[str],
    lookback_days: int = 90,
    date: Optional[str] = None,
) -> Tuple[pd.DataFrame, Dict[str, str]]:
    """
    Build pairwise Pearson correlation matrix from daily returns.

    Args:
        tickers:       List of ticker symbols to include.
        lookback_days: Number of calendar days to look back (default 90 ≈ 63 trading days).
        date:          Reference date in YYYY-MM-DD format; defaults to today if None.

    Returns:
        (corr_df, warnings) where:
            corr_df  - DataFrame[ticker x ticker] of Pearson correlations (NaN filled with default).
            warnings - Dict[ticker_pair_str, message] for pairs with insufficient history.
    """
    if not tickers:
        return pd.DataFrame(), {}

    ref_date = date or pd.Timestamp.now().strftime("%Y-%m-%d")

    conn = get_connection()
    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                SELECT symbol AS ticker, date, close
                FROM "stock-trading".daily_bars
                WHERE symbol = ANY(%(tickers)s)
                  AND date >= (%(ref_date)s::date - %(days)s)
                  AND date <= %(ref_date)s::date
                ORDER BY symbol, date
                """,
                {"tickers": tickers, "ref_date": ref_date, "days": lookback_days},
            )
            rows = cur.fetchall()
    except Exception as e:
        logger.error(f"Failed to fetch daily_bars for correlation: {e}")
        raise
    finally:
        conn.close()

    if not rows:
        logger.warning("No daily_bars data found for universe correlation matrix")
        return _default_corr_matrix(tickers), {}

    # Build close price pivot: date x ticker
    df = pd.DataFrame(rows)
    pivot = df.pivot(index="date", columns="ticker", values="close").sort_index()

    # Daily log returns
    returns = np.log(pivot / pivot.shift(1)).dropna(how="all")

    # Track which pairs have insufficient overlap, apply default correlation
    warnings: Dict[str, str] = {}
    min_obs = _CFG.get("min_history_days", 60) // 3  # ~30 trading-day minimum
    default_corr = _CFG.get("default_unknown_corr", 0.50)

    corr = returns.corr(method="pearson")

    # Extend corr to include tickers that had no data at all
    for t in tickers:
        if t not in corr.columns:
            corr[t] = default_corr
            corr.loc[t] = default_corr
            corr.loc[t, t] = 1.0
            warnings[t] = f"No daily_bars data for {t} — using default correlation {default_corr}"

    # Fill NaN pairs (insufficient overlap) with default correlation
    for t1 in corr.index:
        for t2 in corr.columns:
            if pd.isna(corr.loc[t1, t2]):
                pair_key = f"{t1}/{t2}"
                pair_returns = returns[[t1, t2]].dropna() if (t1 in returns and t2 in returns) else pd.DataFrame()
                n_obs = len(pair_returns)
                if n_obs < min_obs:
                    msg = f"Insufficient overlap ({n_obs} days) between {t1} and {t2} — using default {default_corr}"
                    warnings[pair_key] = msg
                    logger.warning(msg)
                corr.loc[t1, t2] = default_corr
                corr.loc[t2, t1] = default_corr

    # Ensure diagonal is 1.0
    for t in corr.index:
        if t in corr.columns:
            corr.loc[t, t] = 1.0

    logger.info(
        f"Built correlation matrix for {len(corr)} tickers from {len(pivot)} days of data; "
        f"{len(warnings)} warnings"
    )
    return corr, warnings


def get_pair_corr(corr_df: pd.DataFrame, t1: str, t2: str) -> float:
    """
    Safely retrieve the correlation between two tickers.

    Returns default_unknown_corr if either ticker is missing from the matrix.
    """
    default = _CFG.get("default_unknown_corr", 0.50)
    try:
        return float(corr_df.loc[t1, t2])
    except KeyError:
        return default


def _default_corr_matrix(tickers: List[str]) -> pd.DataFrame:
    """Create a default correlation matrix (identity diagonal + 0.5 off-diagonal)."""
    default = _CFG.get("default_unknown_corr", 0.50)
    df = pd.DataFrame(default, index=tickers, columns=tickers)
    for t in tickers:
        df.loc[t, t] = 1.0
    return df
