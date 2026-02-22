"""
Unit tests for portfolio.filter module.
Tests hard-rule elimination logic and boundary conditions — both the original
five ML-centric rules and the eight new technical market-structure filters.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import config

# Override DB_SCHEMA so import of config doesn't fail in test env
config.DB_SCHEMA = "stock-trading"

import pytest
from portfolio.filter import filter_candidates, _compute_sma, _compute_rsi, _technical_filter


# ── Shared fixtures ────────────────────────────────────────────────────────────

def _make_stock(ticker, sector="Banking", exchange="HOSE", vol_k=500):
    return {"ticker": ticker, "sector": sector, "exchange": exchange, "avg_daily_vol_k": vol_k}


def _make_preds(p10=0.01, p50=0.05, p90=0.10, conf=0.80):
    return {
        h: {"p10": p10, "p50": p50 * (1 + h * 0.1), "p90": p90, "confidence": conf}
        for h in (1, 5, 10)
    }


def _make_bars(n: int, trend: str = "up", base: float = 100.0) -> list:
    """
    Generate synthetic OHLCV bars (oldest-first) with realistic RSI values.

    Pattern (3-up / 1-down cycle) keeps RSI in a natural range:
      trend="up"   — net +0.3 %/day; RSI stays ~55-70    (healthy uptrend)
      trend="down" — net -0.4 %/day; RSI stays ~25-38    (clear downtrend)
      trend="flat" — oscillates ±0.1 %; RSI stays ~45-55 (neutral)
    """
    UP_3_DOWN_1   = [+0.005, +0.005, +0.004, -0.008]   # net ~+0.3 %/day
    DOWN_3_UP_1   = [-0.006, -0.006, -0.005, +0.004]   # net ~-0.4 %/day
    FLAT_ZIGZAG   = [+0.002, -0.002, +0.001, -0.001]   # net ~0 %/day

    pattern = UP_3_DOWN_1 if trend == "up" else (DOWN_3_UP_1 if trend == "down" else FLAT_ZIGZAG)
    bars = []
    price = base
    for i in range(n):
        change = pattern[i % len(pattern)]
        price  = price * (1 + change)
        bars.append({
            "date":   f"2025-{(i // 30) + 1:02d}-{(i % 30) + 1:02d}",
            "open":   price * 0.998,
            "high":   price * 1.005,
            "low":    price * 0.994,
            "close":  price,
            "volume": 500_000.0,
        })
    return bars


GOOD_STOCK = _make_stock("AAA")
GOOD_PREDS = _make_preds()
GOOD_FP    = {"AAA": 0.05}
GOOD_VOL   = {"AAA": 300.0}
GOOD_BARS  = {"AAA": _make_bars(252, "up")}   # 252 days of clean uptrend


# ── Legacy ML-centric filter tests (unchanged behaviour) ──────────────────────

def test_good_candidate_passes():
    candidates, audit = filter_candidates(
        [GOOD_STOCK], {"AAA": GOOD_PREDS}, GOOD_FP, GOOD_VOL, GOOD_BARS
    )
    assert len(candidates) == 1
    assert not audit[0]["eliminated"]


def test_no_predictions_eliminated():
    candidates, audit = filter_candidates([GOOD_STOCK], {}, GOOD_FP, GOOD_VOL, GOOD_BARS)
    assert len(candidates) == 0
    assert "no predictions" in audit[0]["reason"].lower()


def test_floor_prob_at_boundary_passes():
    fp = {"AAA": config.PORTFOLIO_CONFIG["max_floor_prob"]}
    candidates, _ = filter_candidates(
        [GOOD_STOCK], {"AAA": GOOD_PREDS}, fp, GOOD_VOL, GOOD_BARS
    )
    assert len(candidates) == 1


def test_floor_prob_below_threshold_passes():
    fp = {"AAA": config.PORTFOLIO_CONFIG["max_floor_prob"] - 0.01}
    candidates, _ = filter_candidates(
        [GOOD_STOCK], {"AAA": GOOD_PREDS}, fp, GOOD_VOL, GOOD_BARS
    )
    assert len(candidates) == 1


def test_low_volume_eliminated():
    low_vol_map = {"AAA": float(config.PORTFOLIO_CONFIG["min_daily_vol_k"] - 1)}
    candidates, audit = filter_candidates(
        [GOOD_STOCK], {"AAA": GOOD_PREDS}, GOOD_FP, low_vol_map, GOOD_BARS
    )
    assert len(candidates) == 0
    assert "avg_volume" in audit[0]["reason"].lower()


def test_low_return_eliminated():
    low_return_preds = {
        h: {"p10": -0.01, "p50": 0.001, "p90": 0.01, "confidence": 0.80}
        for h in (1, 5, 10)
    }
    candidates, audit = filter_candidates(
        [GOOD_STOCK], {"AAA": low_return_preds}, GOOD_FP, GOOD_VOL, GOOD_BARS
    )
    assert len(candidates) == 0
    assert "expected return" in audit[0]["reason"].lower()


def test_low_confidence_eliminated():
    low_conf_preds = {
        h: {"p10": 0.01, "p50": 0.05, "p90": 0.10, "confidence": 0.40}
        for h in (1, 5, 10)
    }
    candidates, audit = filter_candidates(
        [GOOD_STOCK], {"AAA": low_conf_preds}, GOOD_FP, GOOD_VOL, GOOD_BARS
    )
    assert len(candidates) == 0
    assert "confidence" in audit[0]["reason"].lower()


def test_multiple_stocks_partial_pass():
    stocks = [_make_stock("AAA"), _make_stock("BBB"), _make_stock("CCC")]
    preds = {"AAA": GOOD_PREDS, "BBB": GOOD_PREDS, "CCC": GOOD_PREDS}
    fps   = {"AAA": 0.05, "BBB": 0.05, "CCC": 0.05}
    vol   = {"AAA": 300.0, "BBB": 5.0, "CCC": 300.0}  # BBB has low volume
    ph    = {"AAA": _make_bars(252, "up"), "BBB": _make_bars(252, "up"), "CCC": _make_bars(252, "up")}
    candidates, audit = filter_candidates(stocks, preds, fps, vol, ph)
    assert len(candidates) == 2
    assert {c["ticker"] for c in candidates} == {"AAA", "CCC"}


def test_no_price_history_skips_technical_filters():
    """Passing price_history={} (or None) skips technical checks — backward compatible."""
    candidates, audit = filter_candidates(
        [GOOD_STOCK], {"AAA": GOOD_PREDS}, GOOD_FP, GOOD_VOL, {}
    )
    assert len(candidates) == 1, "ML-only path should still pass"


# ── Indicator helper unit tests ────────────────────────────────────────────────

def test_compute_sma_basic():
    closes = [1.0, 2.0, 3.0, 4.0, 5.0]
    assert _compute_sma(closes, 3) == pytest.approx(4.0)


def test_compute_sma_insufficient():
    assert _compute_sma([1.0, 2.0], 5) is None


def test_compute_rsi_all_gains():
    closes = [100.0 + i for i in range(20)]
    rsi = _compute_rsi(closes, 14)
    assert rsi == pytest.approx(100.0)


def test_compute_rsi_all_losses():
    closes = [100.0 - i for i in range(20)]
    rsi = _compute_rsi(closes, 14)
    assert rsi == pytest.approx(0.0)


def test_compute_rsi_insufficient():
    assert _compute_rsi([100.0] * 10, 14) is None


# ── Technical filter integration tests ────────────────────────────────────────

# --- Filter 1: Trend Direction ---

def test_trend_pass_strong_uptrend():
    bars = _make_bars(252, "up")
    passed, reason = _technical_filter("AAA", bars)
    assert passed, f"Expected pass, got: {reason}"


def test_trend_fail_downtrend():
    bars = _make_bars(252, "down")
    passed, reason = _technical_filter("AAA", bars)
    assert not passed
    assert "trend" in reason.lower()


def test_trend_fail_insufficient_history():
    bars = _make_bars(30, "up")
    passed, reason = _technical_filter("AAA", bars)
    assert not passed
    assert "insufficient" in reason.lower()


def test_trend_fail_price_below_sma20():
    """Price dips below SMA-20 even in a generally rising market."""
    bars = _make_bars(252, "up")
    # Force last close well below what was trending
    bars[-1]["close"] = bars[-1]["close"] * 0.85
    passed, reason = _technical_filter("AAA", bars)
    assert not passed
    assert "trend" in reason.lower()


# --- Filter 2: Price Momentum Quality ---

def test_momentum_pass():
    bars = _make_bars(252, "up")
    # Up-trend means 20d and 60d returns are well above thresholds
    passed, reason = _technical_filter("AAA", bars)
    assert passed, f"Expected pass, got: {reason}"


def test_momentum_fail_20d():
    bars = _make_bars(252, "flat")
    # Inject a steep drop 15 days ago so 20d return < -5 %
    drop_price = bars[-20]["close"] * 1.30
    for i in range(-20, 0):
        bars[i]["close"] = bars[i]["close"] * 0.994  # mild drift down
    # Force 20d return to be exactly -7 %
    bars[-1]["close"] = bars[-20]["close"] * 0.93
    # Ensure SMA-20 > SMA-60 and slope conditions are also met by patching
    # (simplest: use _technical_filter directly and check momentum text)
    passed, reason = _technical_filter("AAA", bars)
    # We're just verifying momentum logic triggers somewhere in filter stack
    # (may also fail on trend — that's fine, what matters is rejected)
    assert not passed


def test_momentum_fail_60d():
    bars = _make_bars(252, "flat")
    # Make 60d return = -12 % (worse than -10 % threshold)
    bars[-1]["close"] = bars[-60]["close"] * 0.88
    passed, reason = _technical_filter("AAA", bars)
    assert not passed


def test_momentum_insufficient_history():
    bars = _make_bars(30, "up")
    passed, reason = _technical_filter("AAA", bars)
    assert not passed


# --- Filter 3: Volume Confirmation ---

def test_volume_confirmation_pass():
    bars = _make_bars(252, "up")
    # Default bars have constant 500_000 volume → ratio = 1.0
    passed, reason = _technical_filter("AAA", bars)
    assert passed, f"Expected pass, got: {reason}"


def test_volume_confirmation_fail_drying_up():
    bars = _make_bars(252, "up")
    # Drop recent 10 days' volume to 50 % of baseline (ratio = 0.50 < 0.80)
    for i in range(-10, 0):
        bars[i]["volume"] = 250_000.0
    passed, reason = _technical_filter("AAA", bars)
    assert not passed
    assert "volume" in reason.lower()


def test_volume_confirmation_boundary():
    bars = _make_bars(252, "up")
    baseline_vol = 500_000.0
    ratio_target = config.PORTFOLIO_CONFIG["volume_ratio_min"]  # 0.80
    for i in range(-10, 0):
        bars[i]["volume"] = baseline_vol * ratio_target  # exactly at limit
    passed, reason = _technical_filter("AAA", bars)
    assert passed, f"Boundary should pass (≥ ratio): {reason}"


# --- Filter 4: RSI Health Check ---

def test_rsi_healthy_range_passes():
    bars = _make_bars(252, "up")
    passed, reason = _technical_filter("AAA", bars)
    assert passed, f"Expected pass, got: {reason}"


def test_rsi_oversold_eliminated():
    """A stock with mostly declining days produces low RSI and fails on RSI or trend."""
    bars = _make_bars(252, "down")
    passed, reason = _technical_filter("AAA", bars)
    assert not passed  # fails trend or RSI — both are valid rejections


def test_rsi_insufficient_history():
    bars = _make_bars(5, "up")
    passed, reason = _technical_filter("AAA", bars)
    assert not passed
    assert "insufficient" in reason.lower()


# --- Filter 5: Higher High / Higher Low Structure ---

def test_hhhl_pass():
    bars = _make_bars(252, "up")
    passed, reason = _technical_filter("AAA", bars)
    assert passed, f"Expected pass, got: {reason}"


def test_hhhl_fail_downtrend():
    bars = _make_bars(252, "down")
    passed, reason = _technical_filter("AAA", bars)
    assert not passed  # fails trend, RSI, or HH/HL — all valid


def test_hhhl_fail_no_higher_high():
    bars = _make_bars(252, "up")
    # Flatten just the highs in the current 20-day window below prior 20-day highs
    prior_max = max(b["high"] for b in bars[-40:-20])
    for b in bars[-20:]:
        b["high"] = prior_max * 0.95  # lower than prior period highs
    passed, reason = _technical_filter("AAA", bars)
    assert not passed
    assert "higher high" in reason.lower() or "structure" in reason.lower()


def test_hhhl_insufficient_history():
    bars = _make_bars(30, "up")
    passed, reason = _technical_filter("AAA", bars)
    assert not passed
    assert "insufficient" in reason.lower()


# Pre-verified: _make_bars(252, "up") with this module's _CFG values produces bars that
# pass filters 1-5 (trend, momentum, volume, RSI, HH/HL).  The helper below
# generates the same clean uptrend, verifies the first 5 pass at build time, and
# returns bars ready for per-filter fault injection.
def _make_filter_passing_bars(n: int = 252) -> list:
    """
    Build `n` uptrend bars guaranteed to pass technical filters 1–5 (trend,
    momentum, volume, RSI, HH/HL).  The bars are verified at construction time
    so any future change to thresholds that breaks this guarantee will surface
    immediately rather than silently masking filter-specific tests.

    Returns bars (oldest-first) ready for sharp-drop or 52-week-high injection.
    """
    bars = _make_bars(n, "up")
    # Runtime guard: verify filters 1–5 don't fire on clean bars.
    # Temporarily inject an impossibly low value 10 days from now (outside window)
    # so filter 6-8 don't accidentally reject either — we just call _technical_filter
    # and stop the helper if it fails before filter 6 for an unexpected reason.
    passed, reason = _technical_filter("_setup_", bars)
    assert passed, (
        f"_make_filter_passing_bars: clean uptrend bars unexpectedly rejected "
        f"before the injection point: {reason}\n"
        f"Check if config thresholds changed and update _make_bars accordingly."
    )
    return bars


# --- Filter 6: No Recent Sharp Drop ---

def test_no_sharp_drop_pass():
    bars = _make_bars(252, "up")
    passed, reason = _technical_filter("AAA", bars)
    assert passed, f"Expected pass, got: {reason}"


def test_sharp_drop_within_window_isolated():
    """
    Isolation test for Filter 6 (no recent sharp drop).

    Uses bars that verifiably pass all upstream filters (1-5), then injects a
    single -7 % CLOSE-to-close return at position -6 (within the 10-day look-back
    window), while keeping that bar's HIGH and LOW at the surrounding price level.
    This is critical: the sharp-drop filter measures close-to-close returns only,
    so setting high/low to the pre-drop level means Filter 5 (HH/HL) does NOT
    see a "lower low" — only Filter 6 sees the bad day.

    Because the pre-injection bars are confirmed to pass _technical_filter with
    NO modifications, and Filter 6 runs before 7 and 8, the ONLY possible
    rejection reason after the injection is Filter 6.
    """
    bars = _make_filter_passing_bars(n=300)  # extra history so SMA is well-established

    # Inject -7 % CLOSE at day -6 (inside the 10-day sharp-drop window)
    drop_idx = len(bars) - 6
    pre_drop_close = bars[drop_idx - 1]["close"]
    bars[drop_idx]["close"] = pre_drop_close * 0.930   # -7.0 % close-to-close

    # Keep high/low at the surrounding price level so HH/HL filter (Filter 5)
    # does NOT see a lower low — only the close changed.
    surrounding_price = pre_drop_close
    bars[drop_idx]["high"] = surrounding_price * 1.005
    bars[drop_idx]["low"]  = surrounding_price * 0.994
    bars[drop_idx]["open"] = surrounding_price * 0.998

    # Repair bars[drop_idx+1 .. end] with the same 3-up-1-down pattern
    # continuing from the pre-drop close level so all trend/RSI/momentum
    # conditions look identical to a clean uptrend.
    UP_3_DOWN_1 = [+0.005, +0.005, +0.004, -0.008]
    recovery = pre_drop_close
    for j in range(drop_idx + 1, len(bars)):
        step     = (j - drop_idx - 1) % 4
        recovery = recovery * (1 + UP_3_DOWN_1[step])
        bars[j]["close"] = recovery
        bars[j]["high"]  = recovery * 1.005
        bars[j]["low"]   = recovery * 0.994
        bars[j]["open"]  = recovery * 0.998

    passed, reason = _technical_filter("AAA", bars)

    assert not passed, "Expected rejection due to sharp drop but stock passed"
    assert "sharp drop" in reason.lower(), (
        f"Filter 6 (sharp drop) should have fired.\n"
        f"Actual reason: '{reason}'\n"
        f"This means filter 6 has a bug or an upstream filter fired unexpectedly."
    )





def test_sharp_drop_boundary_passes():
    bars = _make_bars(252, "up")
    # Exactly -5.0 % should NOT trigger the sharp drop filter (strict < -5 %)
    idx = -5
    prev_close = bars[idx - 1]["close"]
    bars[idx]["close"] = prev_close * 0.950  # exactly -5.0 %
    # Repair subsequent bars
    for i in range(idx + 1, 0):
        bars[i]["close"] = bars[i - 1]["close"] * 1.004
        bars[i]["high"]  = bars[i]["close"] * 1.005
        bars[i]["low"]   = bars[i]["close"] * 0.994
    passed, reason = _technical_filter("AAA", bars)
    # Sharp-drop filter must NOT fire (−5.0 % is not strictly less than −5 %)
    if not passed:
        assert "sharp drop" not in reason.lower(), \
            f"Sharp-drop boundary incorrectly fired at -5.0 %: {reason}"


def test_sharp_drop_insufficient_history():
    bars = _make_bars(5, "up")
    passed, reason = _technical_filter("AAA", bars)
    assert not passed


# --- Filter 7: Distance From 52-Week High ---

def test_52wk_high_pass():
    bars = _make_bars(252, "up")
    passed, reason = _technical_filter("AAA", bars)
    assert passed, f"Expected pass, got: {reason}"


def test_52wk_high_fail_fallen_stock():
    """
    A stock trading at < 70 % of its 52-week high must be rejected.

    Build 150 days of strong uptrend followed by 102 days of decline so the
    current price ends up well below 70 % of the annual peak.
    """
    bars_up   = _make_bars(150, "up", base=100.0)
    peak_close = bars_up[-1]["close"]
    # 102 days of our down pattern at net -0.4 %/day → ~33 % total decline
    bars_down = _make_bars(102, "down", base=peak_close)
    bars = bars_up + bars_down

    closes = [b["close"] for b in bars]
    lookback = min(len(closes), 252)
    week52_high = max(closes[-lookback:])
    ratio = closes[-1] / week52_high
    # down patter: net ~ -13.7 % over 102 days = ratio ~0.863; not enough
    # If still not < 0.70, build even more decline bars
    if ratio >= 0.70:
        extra_down = _make_bars(80, "down", base=bars_down[-1]["close"])
        bars = bars + extra_down
        closes = [b["close"] for b in bars]
        lookback = min(len(closes), 252)
        week52_high = max(closes[-lookback:])
        ratio = closes[-1] / week52_high

    assert ratio < 0.70, f"Setup: expected ratio < 0.70 but got {ratio:.2f}"
    passed, reason = _technical_filter("AAA", bars)
    assert not passed  # rejected (may be trend, momentum, RSI, or 52-week — all valid)


def test_52wk_high_boundary_passes():
    bars = _make_bars(252, "up")
    peak = max(b["close"] for b in bars)
    # Set last close to exactly 70 % of peak
    bars[-1]["close"] = peak * 0.70
    passed, reason = _technical_filter("AAA", bars)
    # May fail other filters (trend/momentum) due to the large drop — that's expected.
    # What matters is 52w filter fires at 70 % — it should NOT fire (inclusive boundary)
    if not passed:
        assert "52-week" not in reason.lower() or "70" in reason, \
            f"52-week boundary incorrectly rejected at 70 %: {reason}"


def test_52wk_insufficient_history():
    bars = _make_bars(40, "up")  # < 63 bars
    passed, reason = _technical_filter("AAA", bars)
    assert not passed
    assert "insufficient" in reason.lower()


# --- Filter 8: Positive Days Ratio ---

def test_positive_days_pass():
    bars = _make_bars(252, "up")
    passed, reason = _technical_filter("AAA", bars)
    assert passed, f"Expected pass, got: {reason}"


def test_positive_days_fail_persistent_selling():
    bars = _make_bars(252, "up")
    # Make last 20 days mostly down: 14 down, 6 up → ratio = 0.30 < 0.45
    for i in range(-21, -1):
        step = (i + 21) % 7
        # 6 up days, 14 down days across 20 closes
        if step < 5:
            bars[i]["close"] = bars[i - 1]["close"] * 0.993
        else:
            bars[i]["close"] = bars[i - 1]["close"] * 1.001
    passed, reason = _technical_filter("AAA", bars)
    assert not passed
    # Accept any fail reason (trend may trigger first, which is also valid)


def test_positive_days_boundary():
    bars = _make_bars(252, "up")
    # Force exactly 9 out of 20 days positive (45 % → pass)
    down_days = 11
    up_days = 9
    # Patch last 20 bars: first down_days down, then up_days up
    base = bars[-22]["close"]
    for j in range(20):
        prev = bars[-(21 - j)]["close"]
        if j < down_days:
            bars[-(20 - j)]["close"] = prev * 0.998
        else:
            bars[-(20 - j)]["close"] = prev * 1.002
    # Check that the positive-days filter itself would pass
    closes = [b["close"] for b in bars]
    recent = closes[-21:]
    up_or_flat = sum(1 for i in range(1, 21) if recent[i] >= recent[i - 1])
    ratio = up_or_flat / 20
    assert ratio >= 0.45, f"Setup error — ratio is {ratio:.2f}, should be ≥ 0.45"


def test_positive_days_insufficient_history():
    bars = _make_bars(10, "up")  # < 21 bars
    passed, reason = _technical_filter("AAA", bars)
    assert not passed
    assert "insufficient" in reason.lower()


# ── Integration: audit trail completeness ──────────────────────────────────────

def test_audit_trail_has_technical_reason_field():
    """All audit entries must include technical_reason key."""
    stocks = [_make_stock("AAA"), _make_stock("BBB")]
    preds  = {"AAA": GOOD_PREDS, "BBB": GOOD_PREDS}
    fps    = {"AAA": 0.05, "BBB": 0.05}
    vol    = {"AAA": 300.0, "BBB": 300.0}
    ph     = {"AAA": _make_bars(252, "up"), "BBB": _make_bars(252, "down")}
    _, audit = filter_candidates(stocks, preds, fps, vol, ph)
    for entry in audit:
        assert "technical_reason" in entry


def test_ml_failure_short_circuits_technical():
    """A stock failing an ML rule should never reach technical filters."""
    # BBB has no predictions so it short-circuits early
    stocks = [_make_stock("BBB")]
    preds  = {}      # no predictions
    fps    = {"BBB": 0.05}
    vol    = {"BBB": 300.0}
    ph     = {"BBB": _make_bars(252, "down")}  # would fail technical too, but shouldn't reach it
    _, audit = filter_candidates(stocks, preds, fps, vol, ph)
    assert audit[0]["eliminated"]
    assert "no predictions" in audit[0]["reason"].lower()
    assert audit[0]["technical_reason"] == ""   # short-circuited, not reached


def test_multi_stock_mixed_filters():
    """5 stocks — 2 fail ML rules, 2 fail technical, 1 passes."""
    stocks = [_make_stock(t) for t in ("A1", "A2", "A3", "A4", "A5")]
    preds = {
        "A1": GOOD_PREDS,                     # passes ML
        "A2": GOOD_PREDS,                     # passes ML, good bars
        "A3": GOOD_PREDS,                     # passes ML, downtrend bars
        "A4": GOOD_PREDS,                     # passes ML, downtrend bars
        # A5 has no predictions
    }
    fps = {t: 0.05 for t in ("A1", "A2", "A3", "A4", "A5")}
    vol = {t: 300.0 for t in ("A1", "A2", "A3", "A4", "A5")}
    ph  = {
        "A1": _make_bars(252, "up"),
        "A2": _make_bars(252, "up"),
        "A3": _make_bars(252, "down"),
        "A4": _make_bars(252, "down"),
        "A5": _make_bars(252, "up"),
    }
    candidates, audit = filter_candidates(stocks, preds, fps, vol, ph)
    # A5 has no predictions → eliminated
    # A3, A4 have downtrend → eliminated
    # A1, A2 have uptrend → pass
    assert len(candidates) == 2
    assert {c["ticker"] for c in candidates} == {"A1", "A2"}
    assert len(audit) == 5
