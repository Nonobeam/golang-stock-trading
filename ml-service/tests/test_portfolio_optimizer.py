"""
Unit tests for portfolio.optimizer module.
Tests constraint satisfaction, near-miss tagging, and fallback behaviour.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import config
config.DB_SCHEMA = "stock-trading"

import pandas as pd
import pytest
from portfolio.optimizer import optimize, _sector_cap_ok, _correlation_cap_ok


def _stock(ticker, sector="Banking", score=0.5):
    return {
        "ticker": ticker,
        "sector": sector,
        "exchange": "HOSE",
        "avg_daily_vol_k": 300,
        "composite_score": score,
        "score_breakdown": {},
    }


def _identity_corr(tickers):
    """Return zero-off-diagonal correlation matrix."""
    df = pd.DataFrame(0.0, index=tickers, columns=tickers)
    for t in tickers:
        df.loc[t, t] = 1.0
    return df


# ── Sector cap helper ──────────────────────────────────────────────

def test_sector_cap_ok_passes_when_within_limit():
    combo = [
        _stock("A", sector="Banking"),
        _stock("B", sector="Banking"),
        _stock("C", sector="Steel"),
        _stock("D", sector="Steel"),
        _stock("E", sector="Energy"),
    ]
    assert _sector_cap_ok(combo)


def test_sector_cap_ok_fails_when_3_from_same_sector():
    combo = [_stock("A"), _stock("B"), _stock("C"), _stock("D"), _stock("E")]  # all Banking
    assert not _sector_cap_ok(combo)


# ── Correlation cap helper ─────────────────────────────────────────

def test_correlation_cap_ok_zero_correlations():
    tickers = ["A", "B", "C"]
    combo   = [_stock(t) for t in tickers]
    corr_df = _identity_corr(tickers)
    assert _correlation_cap_ok(combo, corr_df)


def test_correlation_cap_ok_high_correlation():
    tickers = ["A", "B"]
    corr_df = pd.DataFrame(1.0, index=tickers, columns=tickers)  # perfect correlation
    combo   = [_stock(t) for t in tickers]
    assert not _correlation_cap_ok(combo, corr_df)


# ── Optimiser integration ─────────────────────────────────────────

def test_optimal_5_selected_from_10():
    """Should select the 5 highest-scoring stocks that pass constraints."""
    stocks = [
        _stock("A", "Banking",    score=0.9),
        _stock("B", "Steel",      score=0.85),
        _stock("C", "Energy",     score=0.80),
        _stock("D", "Technology", score=0.75),
        _stock("E", "Consumer",   score=0.70),
        _stock("F", "Banking",    score=0.65),  # second banking
        _stock("G", "Real Estate",score=0.60),
        _stock("H", "Aviation",   score=0.55),
        _stock("I", "Securities", score=0.50),
        _stock("J", "Chemicals",  score=0.45),
    ]
    all_tickers = [s["ticker"] for s in stocks]
    corr_df = _identity_corr(all_tickers)
    selected, near_misses, meta = optimize(stocks, corr_df)

    assert len(selected) == 5, f"Expected 5 selected, got {len(selected)}"
    assert len(near_misses) == 5
    assert meta["n_combos_evaluated"] > 0
    # All selected should be tagged is_selected=True
    assert all(s["is_selected"] for s in selected)
    # All near-misses should be tagged is_selected=False
    assert all(not nm["is_selected"] for nm in near_misses)


def test_fallback_when_fewer_than_5_candidates():
    stocks = [_stock("A", score=0.9), _stock("B", sector="Steel", score=0.8)]
    corr_df = _identity_corr(["A", "B"])
    selected, near_misses, meta = optimize(stocks, corr_df)
    assert len(selected) == 2, "Fallback: return all when < 5 candidates"
    assert len(near_misses) == 0


def test_all_stocks_same_sector_triggers_fallback():
    """All stocks from same sector — valid combos may be 0 due to sector cap."""
    stocks = [_stock(chr(65 + i)) for i in range(8)]  # 8 Banking stocks
    tickers = [s["ticker"] for s in stocks]
    corr_df = _identity_corr(tickers)
    selected, near_misses, meta = optimize(stocks, corr_df)
    # Selector falls back to top-k when no valid combo found
    assert len(selected) == config.PORTFOLIO_CONFIG["portfolio_size"]
