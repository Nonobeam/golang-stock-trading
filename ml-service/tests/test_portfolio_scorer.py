"""
Unit tests for portfolio.scorer module.
Tests composite score formula, weight sum correctness, and edge cases.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import config
config.DB_SCHEMA = "stock-trading"

import pytest
from portfolio.scorer import compute_scores, _liquidity_score, _momentum_quality_score


def _make_stock(ticker, vol_k=300, sector="Banking"):
    return {"ticker": ticker, "sector": sector, "exchange": "HOSE", "avg_daily_vol_k": vol_k}


def _make_preds(p10=0.01, p50=0.05, p90=0.10, conf=0.80):
    return {h: {"p10": p10, "p50": p50, "p90": p90, "confidence": conf} for h in (1, 5, 10)}


# ── Scoring weights must sum to 1.0 ───────────────────────────────

def test_weights_sum_to_one():
    cfg = config.PORTFOLIO_CONFIG
    total = (
        cfg["weight_return"] + cfg["weight_risk_adj"] +
        cfg["weight_liquidity"] + cfg["weight_floor"] + cfg["weight_momentum"]
    )
    assert abs(total - 1.0) < 1e-9, f"Weights do not sum to 1.0: {total}"


def test_horizon_weights_sum_to_one():
    cfg = config.PORTFOLIO_CONFIG
    total = cfg["horizon_weight_1d"] + cfg["horizon_weight_5d"] + cfg["horizon_weight_10d"]
    assert abs(total - 1.0) < 1e-9, f"Horizon weights do not sum to 1.0: {total}"


# ── Composite score range ─────────────────────────────────────────

def test_score_in_valid_range():
    stock = _make_stock("VCB")
    preds = _make_preds()
    fps   = {"VCB": 0.05}
    vol   = {"VCB": 300.0}
    scored = compute_scores([stock], {"VCB": preds}, fps, vol)
    assert len(scored) == 1
    score = scored[0]["composite_score"]
    assert 0.0 <= score <= 1.0, f"Score out of range: {score}"


def test_high_return_low_uncertainty_scores_higher():
    stock_a = _make_stock("AAA")
    stock_b = _make_stock("BBB")
    preds_a = {h: {"p10": 0.02, "p50": 0.08, "p90": 0.11, "confidence": 0.85} for h in (1, 5, 10)}
    preds_b = {h: {"p10": -0.04, "p50": 0.08, "p90": 0.25, "confidence": 0.85} for h in (1, 5, 10)}
    fps = {"AAA": 0.05, "BBB": 0.05}
    vol = {"AAA": 300.0, "BBB": 300.0}
    scored = compute_scores([stock_a, stock_b], {"AAA": preds_a, "BBB": preds_b}, fps, vol)
    scores = {s["ticker"]: s["composite_score"] for s in scored}
    assert scores["AAA"] > scores["BBB"], "Lower uncertainty should yield higher score"


# ── Liquidity tier ────────────────────────────────────────────────

def test_liquidity_tiers():
    assert _liquidity_score(600) == 1.00
    assert _liquidity_score(300) == 0.85
    assert _liquidity_score(150) == 0.65
    assert _liquidity_score(75)  == 0.40
    assert _liquidity_score(10)  == 0.20


# ── Momentum quality ─────────────────────────────────────────────

def test_momentum_quality_positive_p10():
    preds = {10: {"p10": 0.01, "p50": 0.05, "p90": 0.10, "confidence": 0.80}}
    assert _momentum_quality_score(preds) == 1.0


def test_momentum_quality_negative_p10():
    preds = {10: {"p10": -0.01, "p50": 0.05, "p90": 0.10, "confidence": 0.80}}
    assert _momentum_quality_score(preds) == 0.0


# ── Missing horizon fallback ──────────────────────────────────────

def test_missing_10d_falls_back_gracefully():
    stock = _make_stock("VCB")
    preds = {1: {"p10": 0.01, "p50": 0.05, "p90": 0.10, "confidence": 0.80},
             5: {"p10": 0.01, "p50": 0.05, "p90": 0.10, "confidence": 0.80}}
    fps = {"VCB": 0.05}
    vol = {"VCB": 300.0}
    scored = compute_scores([stock], {"VCB": preds}, fps, vol)
    assert len(scored) == 1
    assert 0.0 <= scored[0]["composite_score"] <= 1.0


# ── Score breakdown keys ──────────────────────────────────────────

def test_score_breakdown_keys_present():
    stock = _make_stock("VCB")
    preds = _make_preds()
    vol = {"VCB": 300.0}
    scored = compute_scores([stock], {"VCB": preds}, {"VCB": 0.05}, vol)
    bd = scored[0]["score_breakdown"]
    for key in ("return_score", "risk_adj_score", "liq_score", "floor_score", "momentum_score", "composite"):
        assert key in bd, f"Missing key: {key}"
