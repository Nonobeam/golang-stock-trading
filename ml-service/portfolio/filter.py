"""
Hard-rule candidate filter.

Eliminates universe stocks that fail minimum quality requirements before scoring.
Returns a tuple of (candidates, audit_trail) so the report can explain exclusions.

Two layers of checks run in order:
  1. ML-centric rules (predictions present, floor-hit probability, volume, expected
     return, model confidence).
  2. Technical / market-structure rules (trend, momentum, volume confirmation, RSI,
     higher-high/lower-low structure, no recent sharp drop, distance from 52-week
     high, positive-days ratio).

Stocks short-circuit on the first failing rule so cheaper checks are reached first.
"""
import logging
from typing import Dict, List, Optional, Tuple

import config

logger = logging.getLogger(__name__)

_CFG = config.PORTFOLIO_CONFIG


# ──────────────────────────────────────────────────────────────────────────────
# Indicator helpers
# ──────────────────────────────────────────────────────────────────────────────

def _compute_sma(closes: List[float], period: int) -> Optional[float]:
    """Return the SMA of the last `period` values, or None if insufficient data."""
    if len(closes) < period:
        return None
    return sum(closes[-period:]) / period


def _compute_rsi(closes: List[float], period: int = 14) -> Optional[float]:
    """
    Compute RSI using Wilder smoothing.

    Requires at least period + 1 close prices.
    Returns None if insufficient data.
    """
    if len(closes) < period + 1:
        return None

    deltas = [closes[i] - closes[i - 1] for i in range(1, len(closes))]
    gains  = [d if d > 0 else 0.0 for d in deltas]
    losses = [-d if d < 0 else 0.0 for d in deltas]

    # Seed with simple average of first `period` values
    avg_gain = sum(gains[:period]) / period
    avg_loss = sum(losses[:period]) / period

    # Wilder smoothing for remaining values
    for g, l in zip(gains[period:], losses[period:]):
        avg_gain = (avg_gain * (period - 1) + g) / period
        avg_loss = (avg_loss * (period - 1) + l) / period

    if avg_loss == 0:
        return 100.0
    rs = avg_gain / avg_loss
    return 100.0 - (100.0 / (1.0 + rs))


# ──────────────────────────────────────────────────────────────────────────────
# Technical filter (8 rules)
# ──────────────────────────────────────────────────────────────────────────────

def _technical_filter(
    ticker: str,
    bars: List[Dict],
) -> Tuple[bool, str]:
    """
    Apply the eight technical market-structure filters to a stock.

    Args:
        ticker: Ticker symbol (used only for log messages).
        bars:   List of OHLCV dicts ordered oldest-first, each with keys:
                date, open, high, low, close, volume.

    Returns:
        (passed, reason) — reason is "" when passed, a descriptive message when failed.
    """
    if not bars:
        return False, "No price history available"

    closes  = [b["close"]  for b in bars]
    highs   = [b["high"]   for b in bars]
    lows    = [b["low"]    for b in bars]
    volumes = [b["volume"] for b in bars]

    # ── Filter 1: Trend Direction ────────────────────────────────────────────
    sma_short = _CFG["trend_sma_short"]   # 20
    sma_long  = _CFG["trend_sma_long"]    # 60
    slope_days = _CFG["trend_slope_days"]  # 10

    if len(closes) < sma_long:
        return False, "Insufficient history for trend filter"

    sma20 = _compute_sma(closes, sma_short)
    sma60 = _compute_sma(closes, sma_long)
    price = closes[-1]

    if sma20 is None or sma60 is None:
        return False, "Insufficient history for trend filter"

    if sma20 <= sma60:
        return False, f"Eliminated: trend — SMA-{sma_short} ({sma20:.4f}) ≤ SMA-{sma_long} ({sma60:.4f})"

    if price < sma20:
        return False, f"Eliminated: trend — price ({price:.4f}) < SMA-{sma_short} ({sma20:.4f})"

    if len(closes) >= sma_short + slope_days:
        sma20_prev = _compute_sma(closes[:-slope_days], sma_short)
        if sma20_prev is not None and sma20 < sma20_prev:
            return False, (
                f"Eliminated: trend — SMA-{sma_short} curling down "
                f"({sma20:.4f} < {sma20_prev:.4f} {slope_days}d ago)"
            )

    # ── Filter 2: Price Momentum Quality ────────────────────────────────────
    m20_min = _CFG["momentum_20d_min"]  # -0.05
    m60_min = _CFG["momentum_60d_min"]  # -0.10

    if len(closes) < 61:
        return False, "Insufficient history for momentum filter"

    ret20 = (closes[-1] / closes[-20]) - 1 if closes[-20] else 0.0
    ret60 = (closes[-1] / closes[-60]) - 1 if closes[-60] else 0.0

    if ret20 <= m20_min:
        return False, f"Eliminated: 20d return {ret20:.1%} ≤ {m20_min:.0%} threshold"

    if ret60 <= m60_min:
        return False, f"Eliminated: 60d return {ret60:.1%} ≤ {m60_min:.0%} threshold"

    # ── Filter 3: Volume Confirmation ────────────────────────────────────────
    vol_short = _CFG["volume_short_window"]  # 10
    vol_long  = _CFG["volume_long_window"]   # 30
    vol_ratio_min = _CFG["volume_ratio_min"] # 0.80

    if len(volumes) < vol_long:
        return False, "Insufficient history for volume filter"

    avg_vol_short = sum(volumes[-vol_short:]) / vol_short
    avg_vol_long  = sum(volumes[-vol_long:])  / vol_long

    if avg_vol_long > 0:
        vol_ratio = avg_vol_short / avg_vol_long
        if vol_ratio < vol_ratio_min:
            return False, (
                f"Eliminated: volume declining — "
                f"{vol_short}d/30d ratio {vol_ratio:.2f} < {vol_ratio_min:.2f}"
            )

    # ── Filter 4: RSI Health Check ───────────────────────────────────────────
    rsi_period = _CFG["rsi_period"]  # 14
    rsi_lower  = _CFG["rsi_lower"]   # 35
    rsi_upper  = _CFG["rsi_upper"]   # 75

    if len(closes) < rsi_period + 1:
        return False, "Insufficient history for RSI filter"

    rsi = _compute_rsi(closes, rsi_period)
    if rsi is None:
        return False, "Insufficient history for RSI filter"

    if rsi < rsi_lower:
        return False, f"Eliminated: RSI oversold — RSI-{rsi_period} {rsi:.1f} < {rsi_lower}"

    if rsi > rsi_upper:
        return False, f"Eliminated: RSI overbought — RSI-{rsi_period} {rsi:.1f} > {rsi_upper}"

    # ── Filter 5: Higher High / Higher Low ───────────────────────────────────
    hhhl_window = _CFG["hhhl_window"]  # 20

    if len(highs) < hhhl_window * 2 or len(lows) < hhhl_window * 2:
        return False, "Insufficient history for HH/HL filter"

    prior_high = max(highs[-hhhl_window * 2:-hhhl_window])
    curr_high  = max(highs[-hhhl_window:])
    prior_low  = min(lows[-hhhl_window * 2:-hhhl_window])
    curr_low   = min(lows[-hhhl_window:])

    if curr_high <= prior_high and curr_low <= prior_low:
        return False, (
            f"Eliminated: structure — no higher high ({curr_high:.4f} ≤ {prior_high:.4f}) "
            f"and no higher low ({curr_low:.4f} ≤ {prior_low:.4f})"
        )
    if curr_high <= prior_high:
        return False, (
            f"Eliminated: structure — no higher high "
            f"({curr_high:.4f} ≤ {prior_high:.4f})"
        )
    if curr_low <= prior_low:
        return False, (
            f"Eliminated: structure — no higher low "
            f"({curr_low:.4f} ≤ {prior_low:.4f})"
        )

    # ── Filter 6: No Recent Sharp Drop ───────────────────────────────────────
    drop_window    = _CFG["sharp_drop_window"]    # 10
    drop_threshold = _CFG["sharp_drop_threshold"] # -0.05

    if len(closes) < drop_window + 1:
        return False, "Insufficient history for sharp-drop filter"

    recent_closes = closes[-(drop_window + 1):]
    for i in range(1, len(recent_closes)):
        if recent_closes[i - 1] > 0:
            daily_ret = (recent_closes[i] / recent_closes[i - 1]) - 1
            if daily_ret < drop_threshold:
                return False, (
                    f"Eliminated: sharp drop — {daily_ret:.1%} single-day decline "
                    f"within last {drop_window} days"
                )

    # ── Filter 7: Distance From 52-Week High ─────────────────────────────────
    high52w_lookback  = _CFG["high52w_lookback"]   # 252
    high52w_ratio_min = _CFG["high52w_ratio_min"]  # 0.70

    if len(closes) < 63:
        return False, "Insufficient history for 52-week high filter"

    lookback = min(len(closes), high52w_lookback)
    week52_high = max(closes[-lookback:])
    if week52_high > 0:
        ratio = closes[-1] / week52_high
        if ratio < high52w_ratio_min:
            return False, (
                f"Eliminated: 52-week distance — price is {ratio:.1%} of "
                f"52-week high ({week52_high:.4f}), below {high52w_ratio_min:.0%} threshold"
            )

    # ── Filter 8: Positive Days Ratio ────────────────────────────────────────
    posdays_window    = _CFG["posdays_window"]    # 20
    posdays_ratio_min = _CFG["posdays_ratio_min"] # 0.45

    if len(closes) < posdays_window + 1:
        return False, "Insufficient history for positive-days filter"

    recent = closes[-(posdays_window + 1):]
    up_or_flat = sum(1 for i in range(1, len(recent)) if recent[i] >= recent[i - 1])
    ratio = up_or_flat / posdays_window
    if ratio < posdays_ratio_min:
        return False, (
            f"Eliminated: positive-days ratio {ratio:.0%} "
            f"({up_or_flat}/{posdays_window} days) < {posdays_ratio_min:.0%} threshold"
        )

    return True, ""


# ──────────────────────────────────────────────────────────────────────────────
# Public API
# ──────────────────────────────────────────────────────────────────────────────

def filter_candidates(
    universe: List[Dict],
    predictions: Dict[str, Dict[int, Dict]],
    floor_probs: Dict[str, float],
    vol_map: Dict[str, float],
    price_history: Optional[Dict[str, List[Dict]]] = None,
) -> Tuple[List[Dict], List[Dict]]:
    """
    Apply hard filter rules to the universe and return surviving candidates.

    Args:
        universe:       List of stock_universe rows (ticker, sector, exchange, ...).
        predictions:    {ticker: {horizon: {"p10": ..., "p50": ..., "p90": ..., "confidence": ...}}}
        floor_probs:    {ticker: floor_hit_probability}  (0.0 – 1.0)
        vol_map:        {ticker: avg_daily_volume_in_thousands} from daily_bars.
        price_history:  {ticker: [OHLCV bar dicts, oldest-first]} for technical filters.
                        Pass None or {} to skip technical filters (backward-compatible).

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

    ph = price_history or {}

    candidates: List[Dict] = []
    audit_trail: List[Dict] = []

    for stock in universe:
        ticker = stock["ticker"]
        eliminated = False
        reason = "PASS"
        technical_reason = ""

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

        # 6–13. Technical / market-structure filters
        if not eliminated and ph:
            bars = ph.get(ticker, [])
            tech_pass, tech_reason = _technical_filter(ticker, bars)
            if not tech_pass:
                reason = tech_reason
                technical_reason = tech_reason
                eliminated = True

        audit_trail.append({
            "ticker":           ticker,
            "sector":           stock.get("sector", ""),
            "exchange":         stock.get("exchange", ""),
            "eliminated":       eliminated,
            "reason":           reason,
            "technical_reason": technical_reason,
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
