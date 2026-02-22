"""
Pairwise Pearson correlation matrix builder — overlapping-window method.

Algorithm
---------
1.  Fetch up to 5 years of daily close prices from ``daily_bars`` for all
    universe tickers.  Each ticker may have a different start date.

2.  For every pair (A, B) find the dates where *both* stocks have data
    (the overlapping window).

3.  Apply one of three treatments based on the overlap length:

    < corr_min_overlap_days  (default 90 trading days)
        → "unknown" pair: assign corr_unknown_penalty (default 0.60).
          This is below the 0.70 rejection cap but high enough to discourage
          pairing these stocks unless necessary.

    corr_min_overlap_days … corr_full_trust_days  (90 → 252 trading days)
        → "short overlap" pair: compute raw Pearson, then blend toward
          the neutral value (0.50) proportionally:
              trust = (overlap - min) / (full - min)   # 0 → 1
              blended_corr = trust * raw_corr + (1 - trust) * 0.50

    ≥ corr_full_trust_days  (default 252 trading days, ≈ 1 year)
        → "full trust" pair: cap the window at the most-recent
          corr_full_trust_days and use the raw Pearson correlation.
          This keeps the estimate current and avoids letting 2019-2020
          regime differences dominate Vietnamese market relationships.

The matrix is symmetric.  All diagonals are 1.0.  Tickers that have no
daily-bars rows at all get a full-row/column set to corr_unknown_penalty.
"""

import logging
from typing import Dict, List, Optional, Tuple

import numpy as np
import pandas as pd

from db.connection import get_connection
import config

logger = logging.getLogger(__name__)

_CFG = config.PORTFOLIO_CONFIG

# How far back we fetch from the DB (calendar days).
# 5 years ≈ 1 825 calendar days; a stock listed in 2022 will simply have
# fewer rows — the algorithm handles that gracefully.
_FETCH_LOOKBACK_CALENDAR_DAYS = 1_825


def build_correlation_matrix(
    tickers: List[str],
    lookback_days: int = 90,   # kept for API compatibility; ignored internally
    date: Optional[str] = None,
) -> Tuple[pd.DataFrame, Dict[str, str]]:
    """
    Build pairwise Pearson correlation matrix using the overlapping-window method.

    Args:
        tickers:       List of ticker symbols to include.
        lookback_days: Ignored (kept for backward-compatibility with callers).
                       The algorithm always fetches up to 5 years and caps at
                       corr_full_trust_days per pair.
        date:          Reference date in YYYY-MM-DD format; defaults to today.

    Returns:
        (corr_df, warnings)
        corr_df  – DataFrame[ticker × ticker] of adjusted Pearson correlations.
        warnings – Dict[pair_key_str, human-readable message] for any pair that
                   received a non-raw treatment (unknown or discounted).
    """
    if not tickers:
        return pd.DataFrame(), {}

    min_overlap   = _CFG.get("corr_min_overlap_days", 90)
    full_trust    = _CFG.get("corr_full_trust_days",  252)
    unknown_pen   = _CFG.get("corr_unknown_penalty",  0.60)
    neutral       = _CFG.get("default_unknown_corr",  0.50)

    ref_date = date or pd.Timestamp.now().strftime("%Y-%m-%d")

    # ── 1. Fetch returns ────────────────────────────────────────────────
    returns = _fetch_returns(tickers, ref_date, _FETCH_LOOKBACK_CALENDAR_DAYS)

    # Initialise output matrix
    n = len(tickers)
    corr_values = pd.DataFrame(np.nan, index=tickers, columns=tickers, dtype=float)
    warnings: Dict[str, str] = {}

    # ── 2. Build per-pair correlations ──────────────────────────────────
    for i in range(n):
        t1 = tickers[i]
        corr_values.loc[t1, t1] = 1.0

        for j in range(i + 1, n):
            t2 = tickers[j]
            pair_key = f"{t1}/{t2}"

            c, msg = _pair_correlation(
                returns, t1, t2,
                min_overlap, full_trust, unknown_pen, neutral,
            )
            corr_values.loc[t1, t2] = c
            corr_values.loc[t2, t1] = c
            if msg:
                warnings[pair_key] = msg
                logger.debug(f"corr [{pair_key}]: {msg}")

    # ── 3. Handle tickers with NO data at all ───────────────────────────
    for t in tickers:
        if t not in returns.columns or returns[t].dropna().empty:
            msg = (
                f"{t}: no daily_bars data in the last "
                f"{_FETCH_LOOKBACK_CALENDAR_DAYS} calendar days — "
                f"using unknown penalty {unknown_pen}"
            )
            warnings[t] = msg
            logger.warning(msg)
            corr_values.loc[t, :] = unknown_pen
            corr_values.loc[:, t] = unknown_pen
            corr_values.loc[t, t] = 1.0

    n_warnings = len(warnings)
    logger.info(
        f"Built correlation matrix for {n} tickers; "
        f"{n - n_warnings} pairs fully trusted, {n_warnings} pair/ticker warnings"
    )
    return corr_values, warnings


# ──────────────────────────────────────────────────────────────────────
# Internal helpers
# ──────────────────────────────────────────────────────────────────────

def _fetch_returns(
    tickers: List[str],
    ref_date: str,
    lookback_calendar: int,
) -> pd.DataFrame:
    """
    Fetch daily log-returns from daily_bars.

    Returns a DataFrame[date × ticker] where missing dates for a ticker
    are NaN (not forward-filled) so the overlapping-window logic works
    correctly.
    """
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
                {"tickers": tickers, "ref_date": ref_date, "days": lookback_calendar},
            )
            rows = cur.fetchall()
    except Exception as e:
        logger.error(f"Failed to fetch daily_bars for correlation: {e}")
        raise
    finally:
        conn.close()

    if not rows:
        logger.warning("No daily_bars data found — returning empty returns DataFrame")
        return pd.DataFrame(index=pd.DatetimeIndex([]), columns=tickers, dtype=float)

    df = pd.DataFrame(rows)
    pivot = (
        df.pivot(index="date", columns="ticker", values="close")
        .sort_index()
        .astype(float)
    )

    # Daily log returns; NaN where data starts (first row per ticker)
    returns = np.log(pivot / pivot.shift(1))
    return returns


def _pair_correlation(
    returns: pd.DataFrame,
    t1: str,
    t2: str,
    min_overlap: int,
    full_trust: int,
    unknown_pen: float,
    neutral: float,
) -> Tuple[float, str]:
    """
    Compute adjusted correlation for a single (t1, t2) pair.

    Returns (correlation_value, warning_message).
    warning_message is empty string when the pair has full trust.
    """
    # Pull series for both tickers; drop rows where either is NaN (overlapping window)
    if t1 not in returns.columns or t2 not in returns.columns:
        msg = f"One or both tickers not in daily_bars — using unknown penalty {unknown_pen}"
        return unknown_pen, msg

    overlapping = returns[[t1, t2]].dropna()
    n_overlap = len(overlapping)

    # ── Case A: Truly unknown — too little shared history ──────────────
    if n_overlap < min_overlap:
        msg = (
            f"Only {n_overlap} overlapping trading days (need ≥{min_overlap}) "
            f"— using conservative penalty {unknown_pen}"
        )
        return unknown_pen, msg

    # ── Case B / C: Enough history — cap to most-recent full_trust days ─
    if n_overlap > full_trust:
        overlapping = overlapping.iloc[-full_trust:]
        n_overlap = full_trust

    raw_corr = float(overlapping[t1].corr(overlapping[t2]))
    if np.isnan(raw_corr):
        # Zero-variance series (e.g. stock halted); treat as unknown
        msg = f"NaN correlation (zero-variance returns) — using penalty {unknown_pen}"
        return unknown_pen, msg

    # ── Case B: Short overlap — blend toward neutral ───────────────────
    if n_overlap < full_trust:
        trust = (n_overlap - min_overlap) / (full_trust - min_overlap)   # 0 → 1
        blended = trust * raw_corr + (1.0 - trust) * neutral
        msg = (
            f"{n_overlap} overlap days — blended {raw_corr:.3f} "
            f"× {trust:.2f} toward neutral {neutral} → {blended:.3f}"
        )
        return blended, msg

    # ── Case C: Full trust — return raw correlation ────────────────────
    return raw_corr, ""


def get_pair_corr(corr_df: pd.DataFrame, t1: str, t2: str) -> float:
    """
    Safely retrieve the correlation between two tickers from the matrix.

    Returns corr_unknown_penalty if either ticker is missing.
    """
    fallback = _CFG.get("corr_unknown_penalty", 0.60)
    try:
        val = float(corr_df.loc[t1, t2])
        return val if not np.isnan(val) else fallback
    except KeyError:
        return fallback


def _default_corr_matrix(tickers: List[str]) -> pd.DataFrame:
    """Create a default correlation matrix (1.0 diagonal, corr_unknown_penalty off-diagonal)."""
    pen = _CFG.get("corr_unknown_penalty", 0.60)
    df = pd.DataFrame(pen, index=tickers, columns=tickers, dtype=float)
    for t in tickers:
        df.loc[t, t] = 1.0
    return df
