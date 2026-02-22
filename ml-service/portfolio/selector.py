"""
Weekly portfolio selection orchestrator.

Runs the full pipeline end-to-end:
  Universe → Predictions → Filter → Score → Optimise → Compare → Report → Save → Send Telegram
"""
import json
import logging
import os
from datetime import date, datetime, timedelta
from typing import Dict, List, Optional, Tuple

import requests

from db.connection import get_connection
from portfolio.universe import load_universe, get_sector_map, get_ticker_list, get_volume_map
from portfolio.correlation import build_correlation_matrix
from portfolio.filter import filter_candidates
from portfolio.scorer import compute_scores
from portfolio.optimizer import optimize
from portfolio.comparator import load_current_positions, compare
from portfolio.report import build_report
import config

logger = logging.getLogger(__name__)


# ──────────────────────────────────────────────────────────────────
# Helpers
# ──────────────────────────────────────────────────────────────────

def _get_monday(ref_date: str) -> str:
    """Return the Monday of the week containing ref_date (YYYY-MM-DD)."""
    d = date.fromisoformat(ref_date)
    monday = d - timedelta(days=d.weekday())
    return monday.isoformat()


def _load_predictions(tickers: List[str], pred_date: str) -> Dict[str, Dict[int, Dict]]:
    """
    Load multi-horizon predictions for all tickers for a given prediction_date.

    Returns:
        {ticker: {horizon: {p10, p50, p90, confidence}}}
    """
    conn = get_connection()
    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                SELECT ticker, horizon, p10, p50, p90, confidence
                FROM "stock-trading".predictions
                WHERE ticker = ANY(%(tickers)s)
                  AND prediction_date = %(pred_date)s
                ORDER BY ticker, horizon
                """,
                {"tickers": tickers, "pred_date": pred_date},
            )
            rows = cur.fetchall()
    except Exception as e:
        logger.error(f"Failed to load predictions for {pred_date}: {e}")
        return {}
    finally:
        conn.close()

    result: Dict[str, Dict[int, Dict]] = {}
    for row in rows:
        t = row["ticker"]
        h = row["horizon"]
        if t not in result:
            result[t] = {}
        result[t][h] = {
            "p10":        float(row["p10"]        or 0),
            "p50":        float(row["p50"]        or 0),
            "p90":        float(row["p90"]        or 0),
            "confidence": float(row["confidence"] or 0),
        }
    logger.info(f"Loaded predictions for {len(result)}/{len(tickers)} tickers on {pred_date}")
    return result


def _load_floor_probs(tickers: List[str], pred_date: str) -> Dict[str, float]:
    """
    Load floor-hit probabilities from the floor_hit_probabilities table.

    Reads the most recent row per ticker on or before pred_date.
    Falls back to a p10-based proxy for tickers not yet in the table
    (e.g. before the first /train all run populates it).
    """
    floor_probs: Dict[str, float] = {}

    try:
        conn = get_connection()
        try:
            with conn.cursor() as cur:
                cur.execute(
                    """
                    SELECT DISTINCT ON (ticker)
                        ticker, floor_probability
                    FROM floor_hit_probabilities
                    WHERE ticker = ANY(%(tickers)s)
                      AND prediction_date <= %(pred_date)s
                    ORDER BY ticker, prediction_date DESC
                    """,
                    {"tickers": tickers, "pred_date": pred_date},
                )
                for row in cur.fetchall():
                    floor_probs[row["ticker"]] = float(row["floor_probability"] or 0.0)
        finally:
            conn.close()
    except Exception as e:
        logger.warning(f"Could not read floor_hit_probabilities table: {e} — using proxy for all tickers")

    # Proxy fallback for tickers missing from the table
    missing = [t for t in tickers if t not in floor_probs]
    if missing:
        logger.warning(
            f"{len(missing)} tickers not in floor_hit_probabilities table "
            f"(run /train all to populate). Using p10 proxy for: {missing[:5]}{'...' if len(missing) > 5 else ''}"
        )
        preds = _load_predictions(missing, pred_date)
        for t in missing:
            p1d = preds.get(t, {}).get(1, {})
            p10_1d = float(p1d.get("p10", 0.0)) if p1d else 0.0
            floor_probs[t] = min(1.0, abs(p10_1d) / 0.07) if p10_1d < 0 else 0.0

    return floor_probs


def _save_selections(
    week_start: str,
    selected: List[Dict],
    near_misses: List[Dict],
) -> None:
    """Persist selected and near-miss stocks to weekly_portfolio_selection."""
    conn = get_connection()
    rows_to_insert = selected + near_misses
    try:
        with conn.cursor() as cur:
            for stock in rows_to_insert:
                cur.execute(
                    """
                    INSERT INTO "stock-trading".weekly_portfolio_selection
                        (week_start, ticker, composite_score, score_breakdown,
                         rank, is_selected, selection_reason, created_at)
                    VALUES (%(week_start)s, %(ticker)s, %(composite_score)s,
                            %(score_breakdown)s, %(rank)s, %(is_selected)s,
                            %(selection_reason)s, NOW())
                    ON CONFLICT (week_start, ticker) DO UPDATE SET
                        composite_score  = EXCLUDED.composite_score,
                        score_breakdown  = EXCLUDED.score_breakdown,
                        rank             = EXCLUDED.rank,
                        is_selected      = EXCLUDED.is_selected,
                        selection_reason = EXCLUDED.selection_reason,
                        created_at       = NOW()
                    """,
                    {
                        "week_start":       week_start,
                        "ticker":           stock["ticker"],
                        "composite_score":  stock.get("composite_score", 0),
                        "score_breakdown":  json.dumps(stock.get("score_breakdown", {})),
                        "rank":             stock.get("rank", 99),
                        "is_selected":      stock.get("is_selected", False),
                        "selection_reason": stock.get("selection_reason", ""),
                    },
                )
        conn.commit()
        logger.info(
            f"Saved {len(selected)} selected + {len(near_misses)} near-miss stocks "
            f"for week {week_start}"
        )
    except Exception as e:
        conn.rollback()
        logger.error(f"Failed to save weekly selections: {e}")
        raise
    finally:
        conn.close()


def _send_telegram(messages: List[str]) -> None:
    """Send a list of message strings to the configured Telegram chat."""
    bot_token = os.environ.get("TELEGRAM_BOT_TOKEN", "")
    chat_id   = os.environ.get("TELEGRAM_CHAT_ID", "")
    if not bot_token or not chat_id:
        logger.error("TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID not set — cannot send messages")
        return

    url = f"https://api.telegram.org/bot{bot_token}/sendMessage"
    for msg in messages:
        try:
            resp = requests.post(
                url,
                json={"chat_id": chat_id, "text": msg, "parse_mode": "Markdown"},
                timeout=15,
            )
            resp.raise_for_status()
            logger.info(f"Telegram message sent ({len(msg)} chars)")
        except Exception as e:
            logger.error(f"Failed to send Telegram message: {e}")


# ──────────────────────────────────────────────────────────────────
# Main entry point
# ──────────────────────────────────────────────────────────────────

def run(pred_date: Optional[str] = None, user_id: int = 1, skip_telegram: bool = False) -> List[str]:
    """
    Run the full weekly portfolio selection pipeline.

    Args:
        pred_date:      Date (YYYY-MM-DD) to use for predictions. Defaults to today.
        user_id:        User ID to load current positions for. Defaults to 1.
        skip_telegram:  If True, do NOT send messages to Telegram (caller handles delivery).

    Returns:
        List of Telegram message strings.
    """
    run_date = pred_date or date.today().isoformat()
    week_start = _get_monday(run_date)
    logger.info(f"Starting weekly portfolio selection for week {week_start} (pred_date={run_date})")

    # 1. Load universe
    universe = load_universe(active_only=True)
    tickers  = get_ticker_list(universe)
    logger.info(f"Universe: {len(tickers)} active stocks")

    # 2. Load predictions
    predictions = _load_predictions(tickers, run_date)
    if len(predictions) < 10:
        msg = (
            f"Portfolio scan aborted: insufficient predictions available "
            f"({len(predictions)}/50). Check ML pipeline."
        )
        logger.error(msg)
        if not skip_telegram:
            _send_telegram([msg])
        return [msg]

    # 3. Load floor probabilities (proxy from p10)
    floor_probs = _load_floor_probs(tickers, run_date)

    # 4. Load avg daily volume from daily_bars (90-day window)
    vol_map = get_volume_map(tickers, lookback_days=90, date=run_date)

    # 5. Filter
    candidates, audit_trail = filter_candidates(universe, predictions, floor_probs, vol_map)

    # 6. Score
    scored = compute_scores(candidates, predictions, floor_probs, vol_map)

    # 7. Build correlation matrix
    corr_df, corr_warnings = build_correlation_matrix(tickers, lookback_days=90, date=run_date)
    if corr_warnings:
        for w in list(corr_warnings.values())[:5]:
            logger.warning(w)

    # 8. Optimise
    selected, near_misses, optimizer_meta = optimize(scored, corr_df)
    optimizer_meta["n_candidates"] = len(candidates)

    # 9. Compare against current positions
    current_positions = load_current_positions(user_id=user_id)
    scored_score_map  = {s["ticker"]: s["composite_score"] for s in scored}
    comparison = compare(selected, current_positions, scored_score_map)

    # 10. Build report
    messages = build_report(
        week_start=week_start,
        selected=selected,
        near_misses=near_misses,
        audit_trail=audit_trail,
        comparison=comparison,
        corr_df=corr_df,
        optimizer_meta=optimizer_meta,
        run_date=run_date,
    )

    # 11. Save to database
    _save_selections(week_start, selected, near_misses)

    # 12. Send Telegram (unless caller handles delivery)
    if not skip_telegram:
        _send_telegram(messages)

    logger.info(f"Weekly portfolio selection complete for week {week_start}")
    return messages
