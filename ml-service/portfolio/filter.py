"""
Hard-rule candidate filter.

Eliminates universe stocks that fail minimum quality requirements before scoring.
Returns a tuple of (candidates, audit_trail) so the report can explain exclusions.
"""
import logging
from typing import Dict, List, Tuple

import config

logger = logging.getLogger(__name__)

_CFG = config.PORTFOLIO_CONFIG


def filter_candidates(
    universe: List[Dict],
    predictions: Dict[str, Dict[int, Dict]],
    floor_probs: Dict[str, float],
    vol_map: Dict[str, float],
) -> Tuple[List[Dict], List[Dict]]:
    """
    Apply hard filter rules to the universe and return surviving candidates.

    Args:
        universe:      List of stock_universe rows (ticker, sector, exchange, ...).
        predictions:   {ticker: {horizon: {"p10": ..., "p50": ..., "p90": ..., "confidence": ...}}}
        floor_probs:   {ticker: floor_hit_probability}  (0.0 – 1.0)
        vol_map:       {ticker: avg_daily_volume_in_thousands} from daily_bars.

    Returns:
        (candidates, audit_trail)
        candidates   - Filtered list of universe rows that passed all rules.
        audit_trail  - List of dicts describing each stock and whether it passed (for report).
    """
    max_floor   = _CFG["max_floor_prob"]
    min_vol_k   = _CFG["min_daily_vol_k"]
    min_return  = _CFG["min_expected_return"]
    min_conf    = _CFG["min_confidence"]

    horizon_w = {
        1:  _CFG["horizon_weight_1d"],
        5:  _CFG["horizon_weight_5d"],
        10: _CFG["horizon_weight_10d"],
    }

    candidates: List[Dict] = []
    audit_trail: List[Dict] = []

    for stock in universe:
        ticker = stock["ticker"]
        eliminated = False
        reason = "PASS"

        # 1. Predictions must exist
        preds = predictions.get(ticker)
        if not preds:
            reason = "Eliminated: no predictions available"
            eliminated = True

        # 2. Floor-hit probability
        if not eliminated:
            fp = floor_probs.get(ticker, 0.0)
            if fp > max_floor:
                reason = f"Eliminated: floor_hit_probability {fp:.1%} > {max_floor:.0%} threshold"
                eliminated = True

        # 3. Average daily volume (from daily_bars via vol_map)
        if not eliminated:
            vol_k = vol_map.get(ticker, 0.0)
            if vol_k < min_vol_k:
                reason = f"Eliminated: avg_volume {vol_k}k < {min_vol_k}k threshold"
                eliminated = True

        # 4. Weighted p50 expected return
        if not eliminated:
            total_w = 0.0
            weighted_p50 = 0.0
            for h, w in horizon_w.items():
                if h in preds and preds[h]:
                    p50 = preds[h].get("p50", 0.0)
                    weighted_p50 += w * p50
                    total_w += w
            if total_w > 0:
                weighted_p50 /= total_w
            if weighted_p50 <= min_return:
                reason = (
                    f"Eliminated: expected return {weighted_p50:.2%} ≤ fee threshold {min_return:.2%}"
                )
                eliminated = True

        # 5. Model confidence
        if not eliminated:
            confs = [preds[h]["confidence"] for h in preds if preds[h] and "confidence" in preds[h]]
            avg_conf = sum(confs) / len(confs) if confs else 0.0
            if avg_conf < min_conf:
                reason = f"Eliminated: avg confidence {avg_conf:.2f} < {min_conf:.2f} threshold"
                eliminated = True

        audit_trail.append({
            "ticker":     ticker,
            "sector":     stock.get("sector", ""),
            "exchange":   stock.get("exchange", ""),
            "eliminated": eliminated,
            "reason":     reason,
        })

        if not eliminated:
            candidates.append(stock)

    logger.info(
        f"Filter: {len(candidates)} candidates from {len(universe)} universe stocks; "
        f"{len(universe) - len(candidates)} eliminated"
    )
    if len(candidates) < _CFG["portfolio_size"]:
        logger.warning(
            f"Only {len(candidates)} candidates survived filtering — fewer than target portfolio size "
            f"{_CFG['portfolio_size']}. Check ML predictions and thresholds."
        )
    return candidates, audit_trail
