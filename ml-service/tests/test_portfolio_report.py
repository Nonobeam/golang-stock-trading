"""
Unit tests for portfolio.report module.
Tests message splitting, report structure, and section completeness.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import config
config.DB_SCHEMA = "stock-trading"

import pandas as pd
import pytest
from portfolio.report import build_report, TELEGRAM_MAX_CHARS, _split_messages


# ── Fixtures ──────────────────────────────────────────────────────

def _stock(ticker, rank, sector="Banking", score=0.80):
    return {
        "ticker":           ticker,
        "sector":           sector,
        "exchange":         "HOSE",
        "avg_daily_vol_k":  300,
        "composite_score":  score,
        "score_breakdown":  {
            "return_score": 0.75, "risk_adj_score": 0.70, "liq_score": 0.85,
            "floor_score": 0.90, "momentum_score": 1.0, "composite": score,
            "weighted_p50": 0.06, "floor_prob": 0.05, "avg_daily_vol_k": 300,
            "predictions": {
                10: {"p10": 0.01, "p50": 0.06, "p90": 0.10, "confidence": 0.80}
            },
        },
        "rank":             rank,
        "is_selected":      True,
        "selection_reason": f"Rank {rank}: composite_score={score:.4f}",
    }


SELECTED = [_stock(t, i + 1, s) for i, (t, s) in enumerate([
    ("VCB", "Banking"), ("HPG", "Steel"), ("FPT", "Technology"), ("MWG", "Consumer"), ("GAS", "Energy")
])]

NEAR_MISSES = [_stock("VPB", 6, "Banking", score=0.65)]

AUDIT_TRAIL = [{"ticker": "LOW", "sector": "Pharma", "eliminated": True, "reason": "Eliminated: avg_volume too low"}]

COMPARISON = {
    "overlap_count": 2,
    "overlap_tickers": ["VCB", "HPG"],
    "exits": [],
    "new_entries": ["FPT", "MWG", "GAS"],
    "no_positions": False,
}

TICKERS = [s["ticker"] for s in SELECTED]
CORR_DF = pd.DataFrame(0.3, index=TICKERS, columns=TICKERS)
for t in TICKERS:
    CORR_DF.loc[t, t] = 1.0

META = {"n_candidates": 35, "n_combos_evaluated": 52360, "n_combos_valid": 4200}


# ── Tests ─────────────────────────────────────────────────────────

def test_report_returns_list_of_strings():
    msgs = build_report("2026-02-23", SELECTED, NEAR_MISSES, AUDIT_TRAIL, COMPARISON, CORR_DF, META)
    assert isinstance(msgs, list)
    assert all(isinstance(m, str) for m in msgs)
    assert len(msgs) >= 1


def test_all_selected_tickers_appear_in_report():
    msgs = build_report("2026-02-23", SELECTED, NEAR_MISSES, AUDIT_TRAIL, COMPARISON, CORR_DF, META)
    combined = "\n".join(msgs)
    for s in SELECTED:
        assert s["ticker"] in combined, f"{s['ticker']} missing from report"


def test_report_contains_week_date():
    msgs = build_report("2026-02-23", SELECTED, NEAR_MISSES, AUDIT_TRAIL, COMPARISON, CORR_DF, META)
    combined = "\n".join(msgs)
    assert "2026-02-23" in combined


def test_report_contains_diversification_section():
    msgs = build_report("2026-02-23", SELECTED, NEAR_MISSES, AUDIT_TRAIL, COMPARISON, CORR_DF, META)
    combined = "\n".join(msgs)
    assert "DIVERSIFICATION" in combined or "correlation" in combined.lower()


def test_each_message_under_telegram_limit():
    msgs = build_report("2026-02-23", SELECTED, NEAR_MISSES, AUDIT_TRAIL, COMPARISON, CORR_DF, META)
    for i, msg in enumerate(msgs):
        assert len(msg) <= TELEGRAM_MAX_CHARS, f"Message {i} exceeds {TELEGRAM_MAX_CHARS} chars: {len(msg)}"


def test_split_messages_single_chunk():
    msgs = _split_messages("Short text", "2026-02-23")
    assert msgs == ["Short text"]


def test_split_messages_splits_long_text():
    long_text = ("A" * 3000 + "\n\n") * 2  # 8004 chars
    msgs = _split_messages(long_text, "2026-02-23")
    assert len(msgs) >= 2
    for m in msgs:
        assert len(m) <= TELEGRAM_MAX_CHARS


def test_empty_selected_report_doesnt_crash():
    msgs = build_report("2026-02-23", [], [], AUDIT_TRAIL, COMPARISON, CORR_DF, META)
    assert isinstance(msgs, list)
    assert len(msgs) >= 1
