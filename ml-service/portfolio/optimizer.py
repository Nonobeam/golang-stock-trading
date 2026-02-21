"""
Brute-force C(n, 5) portfolio optimiser.

Enumerates all combinations of 5 from the scored candidate pool and selects the
basket with the highest total composite score that satisfies:
  - Sector cap: max 2 stocks from the same sector.
  - Correlation cap: max pairwise Pearson correlation of 0.70.
"""
import logging
from itertools import combinations
from typing import Dict, List, Optional, Tuple

import pandas as pd

from portfolio.correlation import get_pair_corr
import config

logger = logging.getLogger(__name__)

_CFG = config.PORTFOLIO_CONFIG


def _sector_cap_ok(combo: List[Dict]) -> bool:
    """Return True if no sector appears more than max_sector_count times."""
    max_s = _CFG["max_sector_count"]
    counts: Dict[str, int] = {}
    for stock in combo:
        s = stock.get("sector", "Unknown")
        counts[s] = counts.get(s, 0) + 1
        if counts[s] > max_s:
            return False
    return True


def _correlation_cap_ok(combo: List[Dict], corr_df: pd.DataFrame) -> bool:
    """Return True if no pairwise correlation in the combo exceeds the cap."""
    max_corr = _CFG["max_pairwise_corr"]
    tickers = [s["ticker"] for s in combo]
    for i in range(len(tickers)):
        for j in range(i + 1, len(tickers)):
            c = get_pair_corr(corr_df, tickers[i], tickers[j])
            if c > max_corr:
                return False
    return True


def optimize(
    scored_candidates: List[Dict],
    corr_df: pd.DataFrame,
) -> Tuple[List[Dict], List[Dict], Dict]:
    """
    Find the optimal portfolio basket and collect near-misses.

    Args:
        scored_candidates: Sorted (desc score) list of candidate dicts (output of scorer).
        corr_df:           Correlation matrix DataFrame (from correlation.py).

    Returns:
        (selected, near_misses, meta)
        selected    - Best 5-stock combination (sorted desc by score). Fewer if <5 candidates.
        near_misses - Candidates that passed filtering but were not selected (due to constraints);
                      includes rejected combo members with exclusion reason.
        meta        - Dict with run stats: n_candidates, n_combos_evaluated, n_combos_valid.
    """
    n = len(scored_candidates)
    k = _CFG["portfolio_size"]

    if n == 0:
        logger.error("No candidates available for optimisation")
        return [], [], {"n_candidates": 0, "n_combos_evaluated": 0, "n_combos_valid": 0}

    if n < k:
        logger.warning(f"Fewer candidates ({n}) than portfolio size ({k}); returning all available")
        selected = scored_candidates
        near_misses = []
        return selected, near_misses, {"n_candidates": n, "n_combos_evaluated": 1, "n_combos_valid": 1}

    best_combo: Optional[List[Dict]] = None
    best_score = -1.0
    n_valid = 0
    n_evaluated = 0

    for combo in combinations(scored_candidates, k):
        n_evaluated += 1
        combo_list = list(combo)

        if not _sector_cap_ok(combo_list):
            continue
        if not _correlation_cap_ok(combo_list, corr_df):
            continue

        n_valid += 1
        total_score = sum(s["composite_score"] for s in combo_list)
        if total_score > best_score:
            best_score = total_score
            best_combo = combo_list

    meta = {
        "n_candidates":      n,
        "n_combos_evaluated": n_evaluated,
        "n_combos_valid":    n_valid,
    }

    if best_combo is None:
        logger.error(
            f"No valid combination found after {n_evaluated} evaluations "
            f"(sector cap or correlation cap too strict for this candidate pool)"
        )
        # Fall back: return top-k ignoring constraints
        best_combo = scored_candidates[:k]
        logger.warning("Returning top-k candidates ignoring constraints as fallback")

    logger.info(
        f"Optimiser: evaluated {n_evaluated} combos, {n_valid} valid; "
        f"best score sum = {best_score:.4f}"
    )

    # Tag selected tickers
    selected_tickers = {s["ticker"] for s in best_combo}
    for rank, stock in enumerate(sorted(best_combo, key=lambda x: x["composite_score"], reverse=True), start=1):
        stock["rank"] = rank
        stock["is_selected"] = True
        stock["selection_reason"] = (
            f"Rank {rank}: composite_score={stock['composite_score']:.4f}, "
            f"sector={stock.get('sector', '?')}"
        )

    # Near-misses: scored candidates NOT in selected set
    near_misses = []
    for rank, stock in enumerate(scored_candidates, start=1):
        if stock["ticker"] not in selected_tickers:
            # Determine why this ticker was not selected
            reason = _infer_exclusion_reason(stock, best_combo, corr_df)
            near_candidate = dict(stock)
            near_candidate["rank"] = k + rank
            near_candidate["is_selected"] = False
            near_candidate["selection_reason"] = reason
            near_misses.append(near_candidate)

    return best_combo, near_misses, meta


def _infer_exclusion_reason(
    candidate: Dict,
    selected: List[Dict],
    corr_df: pd.DataFrame,
) -> str:
    """
    Provide a human-readable reason why this candidate was not selected.
    Checks sector cap and correlation violations against the selected basket.
    """
    sector = candidate.get("sector", "?")
    selected_sectors = [s.get("sector", "") for s in selected]
    sector_count = selected_sectors.count(sector)
    max_s = _CFG["max_sector_count"]

    if sector_count >= max_s:
        return f"Excluded: sector cap — {sector} already has {sector_count} stocks in basket"

    max_corr = _CFG["max_pairwise_corr"]
    for sel in selected:
        c = get_pair_corr(corr_df, candidate["ticker"], sel["ticker"])
        if c > max_corr:
            return (
                f"Excluded: correlation {c:.2f} with {sel['ticker']} exceeds cap {max_corr}"
            )

    return f"Excluded: lower score ({candidate['composite_score']:.4f}) than selected basket"
