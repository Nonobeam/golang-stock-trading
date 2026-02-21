"""
Comparison of recommended portfolio against current open positions.

Fetches open positions from the database, computes overlap with recommendations,
estimates exit costs, and flags actionable rotation suggestions.
"""
import logging
from typing import Dict, List, Optional

from db.connection import get_connection
import config

logger = logging.getLogger(__name__)

_CFG = config.PORTFOLIO_CONFIG


def load_current_positions(user_id: int = 1) -> List[Dict]:
    """
    Load open positions from the positions table.

    Returns:
        List of dicts: {ticker/symbol, quantity, avg_price, current_value_est, ...}
    """
    conn = get_connection()
    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                SELECT symbol AS ticker,
                       quantity,
                       avg_price,
                       COALESCE(avg_price * quantity, 0) AS position_value_est
                FROM "stock-trading".positions
                WHERE user_id = %(user_id)s
                  AND is_closed = FALSE
                ORDER BY symbol
                """,
                {"user_id": user_id},
            )
            positions = cur.fetchall()
        logger.info(f"Loaded {len(positions)} open positions for user {user_id}")
        return positions
    except Exception as e:
        logger.error(f"Failed to load positions: {e}")
        return []
    finally:
        conn.close()


def compare(
    selected: List[Dict],
    current_positions: List[Dict],
    scored_candidates: Optional[Dict[str, float]] = None,
) -> Dict:
    """
    Compare the recommended portfolio against current holdings.

    Args:
        selected:          The 5 recommended stocks (output of optimizer).
        current_positions: Open positions from load_current_positions().
        scored_candidates: {ticker: composite_score} for ALL scored candidates (for rotation calc).

    Returns:
        Dict with keys:
            overlap_count   - number of current holdings in the recommended basket
            overlap_tickers - list of tickers that overlap
            exits           - list of {ticker, position_value_est, exit_cost_vnd, rotation_suggested, reason}
            new_entries     - list of recommended tickers not currently held
            no_positions    - True if no open positions exist
    """
    fee_rate  = _CFG["round_trip_fee_rate"]
    threshold = _CFG["rotation_score_improvement"]

    selected_tickers  = {s["ticker"] for s in selected}
    selected_score_map = {s["ticker"]: s["composite_score"] for s in selected}
    held_tickers      = {p["ticker"] for p in current_positions}

    overlap   = selected_tickers & held_tickers
    to_exit   = held_tickers - selected_tickers
    new_entry = selected_tickers - held_tickers

    exits = []
    for pos in current_positions:
        t = pos["ticker"]
        if t not in to_exit:
            continue

        pos_val = float(pos.get("position_value_est", 0) or 0)
        exit_cost_vnd = pos_val * fee_rate

        # Is there a recommended replacement that scores >= threshold% better?
        current_score = (scored_candidates or {}).get(t, 0.0)
        replacement_ticker = None
        rotation_suggested = False

        for sel in selected:
            if sel["ticker"] in new_entry:
                sel_score = sel["composite_score"]
                if current_score > 0 and sel_score >= current_score * (1 + threshold):
                    replacement_ticker = sel["ticker"]
                    rotation_suggested = True
                    break

        reason = (
            f"Replacement {replacement_ticker} scores {threshold:.0%} higher than current holding"
            if rotation_suggested
            else "Score improvement below rotation threshold — consider holding"
        )

        exits.append({
            "ticker":             t,
            "position_value_est": pos_val,
            "exit_cost_vnd":      round(exit_cost_vnd, 0),
            "rotation_suggested": rotation_suggested,
            "replacement":        replacement_ticker,
            "reason":             reason,
        })

    result = {
        "overlap_count":   len(overlap),
        "overlap_tickers": sorted(overlap),
        "exits":           exits,
        "new_entries":     sorted(new_entry),
        "no_positions":    len(current_positions) == 0,
    }
    logger.info(
        f"Comparison: overlap={len(overlap)}, exits_suggested={sum(1 for e in exits if e['rotation_suggested'])}, "
        f"new_entries={len(new_entry)}"
    )
    return result
