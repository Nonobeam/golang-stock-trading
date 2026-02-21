"""
Unit tests for portfolio.filter module.
Tests hard-rule elimination logic and boundary conditions.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from unittest.mock import patch
import config

# Override DB_SCHEMA so import of config doesn't fail in test env
config.DB_SCHEMA = "stock-trading"

import pytest
from portfolio.filter import filter_candidates


# ── Fixtures ──────────────────────────────────────────────────────

def _make_stock(ticker, sector="Banking", exchange="HOSE", vol_k=500):
    return {"ticker": ticker, "sector": sector, "exchange": exchange, "avg_daily_vol_k": vol_k}


def _make_preds(p10=0.01, p50=0.05, p90=0.10, conf=0.80):
    return {
        h: {"p10": p10, "p50": p50 * (1 + h * 0.1), "p90": p90, "confidence": conf}
        for h in (1, 5, 10)
    }


GOOD_STOCK = _make_stock("AAA")
GOOD_PREDS = _make_preds()
GOOD_FP    = {"AAA": 0.05}
GOOD_VOL   = {"AAA": 300.0}  # 300k shares/day


# ── Tests ─────────────────────────────────────────────────────────

def test_good_candidate_passes():
    candidates, audit = filter_candidates([GOOD_STOCK], {"AAA": GOOD_PREDS}, GOOD_FP, GOOD_VOL)
    assert len(candidates) == 1
    assert not audit[0]["eliminated"]


def test_no_predictions_eliminated():
    candidates, audit = filter_candidates([GOOD_STOCK], {}, GOOD_FP, GOOD_VOL)
    assert len(candidates) == 0
    assert audit[0]["eliminated"]
    assert "no predictions" in audit[0]["reason"].lower()


def test_floor_prob_at_boundary_passes():
    """floor_prob == max exactly should pass (> not >=)."""
    fp = {"AAA": config.PORTFOLIO_CONFIG["max_floor_prob"]}
    candidates, audit = filter_candidates([GOOD_STOCK], {"AAA": GOOD_PREDS}, fp, GOOD_VOL)
    assert len(candidates) == 1


def test_floor_prob_below_threshold_passes():
    fp = {"AAA": config.PORTFOLIO_CONFIG["max_floor_prob"] - 0.01}
    candidates, _ = filter_candidates([GOOD_STOCK], {"AAA": GOOD_PREDS}, fp, GOOD_VOL)
    assert len(candidates) == 1


def test_low_volume_eliminated():
    low_vol_map = {"AAA": float(config.PORTFOLIO_CONFIG["min_daily_vol_k"] - 1)}
    candidates, audit = filter_candidates([GOOD_STOCK], {"AAA": GOOD_PREDS}, GOOD_FP, low_vol_map)
    assert len(candidates) == 0
    assert "avg_volume" in audit[0]["reason"].lower()


def test_low_return_eliminated():
    low_return_preds = {
        h: {"p10": -0.01, "p50": 0.001, "p90": 0.01, "confidence": 0.80}
        for h in (1, 5, 10)
    }
    candidates, audit = filter_candidates([GOOD_STOCK], {"AAA": low_return_preds}, GOOD_FP, GOOD_VOL)
    assert len(candidates) == 0
    assert "expected return" in audit[0]["reason"].lower()


def test_low_confidence_eliminated():
    low_conf_preds = {
        h: {"p10": 0.01, "p50": 0.05, "p90": 0.10, "confidence": 0.40}
        for h in (1, 5, 10)
    }
    candidates, audit = filter_candidates([GOOD_STOCK], {"AAA": low_conf_preds}, GOOD_FP, GOOD_VOL)
    assert len(candidates) == 0
    assert "confidence" in audit[0]["reason"].lower()


def test_multiple_stocks_partial_pass():
    stocks = [
        _make_stock("AAA"),
        _make_stock("BBB"),
        _make_stock("CCC"),
    ]
    preds = {"AAA": GOOD_PREDS, "BBB": GOOD_PREDS, "CCC": GOOD_PREDS}
    fps   = {"AAA": 0.05, "BBB": 0.05, "CCC": 0.05}
    vol   = {"AAA": 300.0, "BBB": 5.0, "CCC": 300.0}  # BBB has low volume
    candidates, audit = filter_candidates(stocks, preds, fps, vol)
    assert len(candidates) == 2
    assert {c["ticker"] for c in candidates} == {"AAA", "CCC"}
