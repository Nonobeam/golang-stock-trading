"""
Composite scorer for the weekly portfolio selection.

Computes a [0, 1] composite score per candidate using five weighted components.
All weights are read from config.PORTFOLIO_CONFIG so they can be tuned without
touching this file.
"""
import logging
from typing import Dict, List, Optional, Tuple

import config

logger = logging.getLogger(__name__)

_CFG = config.PORTFOLIO_CONFIG

# Liquidity tiers based on avg_daily_vol_k
_LIQUIDITY_TIERS: List[Tuple[int, float]] = [
    (500,  1.00),   # >= 500k shares/day -> score 1.0
    (200,  0.85),
    (100,  0.65),
    (50,   0.40),
    (0,    0.20),
]


def _liquidity_score(avg_daily_vol_k: int) -> float:
    """Map daily volume (thousands) to a [0, 1] liquidity score."""
    for threshold, score in _LIQUIDITY_TIERS:
        if avg_daily_vol_k >= threshold:
            return score
    return 0.10


def _weighted_return_score(predictions: Dict[int, Dict]) -> Tuple[float, float]:
    """
    Compute the horizon-weighted p50 return and normalise to [0, 1] score.

    Returns (weighted_p50_raw, normalised_score).
    Normalises assuming >= 10% weighted return = score 1.0.
    """
    w1  = _CFG["horizon_weight_1d"]
    w5  = _CFG["horizon_weight_5d"]
    w10 = _CFG["horizon_weight_10d"]

    available_weight = 0.0
    weighted_p50 = 0.0
    for h, w in [(1, w1), (5, w5), (10, w10)]:
        p = predictions.get(h)
        if p:
            weighted_p50 += w * p.get("p50", 0.0)
            available_weight += w

    if available_weight > 0:
        weighted_p50 /= available_weight  # re-normalise if horizons missing

    score = min(1.0, max(0.0, weighted_p50 / 0.10))  # 10% return = max
    return weighted_p50, score


def _risk_adjusted_score(predictions: Dict[int, Dict]) -> float:
    """
    Penalise wide p90-p10 uncertainty spread relative to p50.

    Uses the primary horizon (10d preferred, then 5d).
    Score = 1 - normalised_spread, where spread = (p90-p10) / max(abs(p50), 0.01).
    Clamped to [0, 1].
    """
    pred = predictions.get(10) or predictions.get(5) or predictions.get(1)
    if not pred:
        return 0.5  # neutral if no data
    p10 = pred.get("p10", 0.0)
    p50 = pred.get("p50", 0.0)
    p90 = pred.get("p90", 0.0)
    spread = p90 - p10
    reference = max(abs(p50), 0.01)
    normalised = min(1.0, spread / (reference * 4))  # spread > 4x p50 = worst case
    return max(0.0, 1.0 - normalised)


def _floor_penalty_score(floor_prob: float) -> float:
    """
    Map floor-hit probability to a [0, 1] score.

    Score = 1 - (prob / max_floor_prob).
    Stocks with prob = 0 score 1.0; at the hard cutoff (20%) score 0.0.
    """
    max_fp = _CFG["max_floor_prob"]
    return max(0.0, 1.0 - (floor_prob / max_fp))


def _momentum_quality_score(predictions: Dict[int, Dict]) -> float:
    """
    Award 1.0 if the pessimistic scenario (p10) for the 10d horizon is positive; else 0.0.

    Falls back to 5d if 10d not available.
    """
    pred = predictions.get(10) or predictions.get(5)
    if not pred:
        return 0.0
    return 1.0 if pred.get("p10", -1.0) > 0.0 else 0.0


def compute_scores(
    candidates: List[Dict],
    predictions: Dict[str, Dict[int, Dict]],
    floor_probs: Dict[str, float],
    vol_map: Dict[str, float],
) -> List[Dict]:
    """
    Compute composite score for each candidate.

    Args:
        candidates:  Filtered universe rows (output of filter.filter_candidates).
        predictions: {ticker: {horizon: {p10, p50, p90, confidence}}}.
        floor_probs: {ticker: floor_hit_probability}.
        vol_map:     {ticker: avg_daily_volume_in_thousands} from daily_bars.

    Returns:
        List of candidate dicts enriched with 'composite_score' and 'score_breakdown',
        sorted descending by composite_score.
    """
    w_ret  = _CFG["weight_return"]
    w_risk = _CFG["weight_risk_adj"]
    w_liq  = _CFG["weight_liquidity"]
    w_flo  = _CFG["weight_floor"]
    w_mom  = _CFG["weight_momentum"]

    scored = []
    for stock in candidates:
        ticker = stock["ticker"]
        preds  = predictions.get(ticker, {})
        fp     = floor_probs.get(ticker, 0.0)
        vol_k  = vol_map.get(ticker, 0.0)

        weighted_p50, return_score   = _weighted_return_score(preds)
        risk_adj_score               = _risk_adjusted_score(preds)
        liq_score                    = _liquidity_score(vol_k)
        floor_score                  = _floor_penalty_score(fp)
        momentum_score               = _momentum_quality_score(preds)

        composite = (
            w_ret  * return_score   +
            w_risk * risk_adj_score +
            w_liq  * liq_score      +
            w_flo  * floor_score    +
            w_mom  * momentum_score
        )

        breakdown = {
            "return_score":    round(return_score,    4),
            "risk_adj_score":  round(risk_adj_score,  4),
            "liq_score":       round(liq_score,       4),
            "floor_score":     round(floor_score,     4),
            "momentum_score":  round(momentum_score,  4),
            "composite":       round(composite,       6),
            "weighted_p50":    round(weighted_p50,    6),
            "floor_prob":      round(fp,              4),
            "avg_daily_vol_k": vol_k,
            "predictions":     preds,
        }

        enriched = dict(stock)
        enriched["composite_score"] = composite
        enriched["score_breakdown"] = breakdown
        enriched["avg_daily_vol_k"] = vol_k
        scored.append(enriched)

    scored.sort(key=lambda x: x["composite_score"], reverse=True)
    logger.info(f"Scored {len(scored)} candidates; top ticker: {scored[0]['ticker'] if scored else 'none'}")
    return scored
