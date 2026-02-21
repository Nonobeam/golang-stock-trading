"""
Telegram-formatted report builder for the weekly portfolio selection.

Assembles a structured plain-text report from all pipeline stage outputs.
Splits messages automatically if Telegram's 4096-character limit would be exceeded.
"""
import logging
from datetime import datetime
from typing import Dict, List, Tuple

logger = logging.getLogger(__name__)

TELEGRAM_MAX_CHARS = 4096


def build_report(
    week_start: str,
    selected: List[Dict],
    near_misses: List[Dict],
    audit_trail: List[Dict],
    comparison: Dict,
    corr_df,
    optimizer_meta: Dict,
    run_date: str = None,
) -> List[str]:
    """
    Build the full portfolio report as a list of Telegram message strings.

    Args:
        week_start:     ISO date string for the Monday of the trading week.
        selected:       The final 5 recommended stocks (from optimizer).
        near_misses:    Stocks that passed filter but were excluded by constraints.
        audit_trail:    Per-stock filter audit records.
        comparison:     Output of comparator.compare().
        corr_df:        Correlation matrix DataFrame (to display avg pairwise corr).
        optimizer_meta: Dict with n_candidates, n_combos_evaluated, n_combos_valid.
        run_date:       Date the scan was run (defaults to today).

    Returns:
        List of message strings, each <= TELEGRAM_MAX_CHARS characters.
    """
    run_date = run_date or datetime.now().strftime("%Y-%m-%d")
    n_universe = optimizer_meta.get("n_candidates", "?")
    n_eliminated = sum(1 for a in audit_trail if a.get("eliminated"))
    n_candidates = n_universe  # after filter

    sections: List[str] = []

    # ── Section 1: Header ───────────────────────────────────────────
    header = (
        f"📊 *Weekly Portfolio Scan*\n"
        f"Week of: {week_start}  |  Run: {run_date}\n"
        f"Universe: 50 stocks  |  After filter: {50 - n_eliminated}/50 candidates\n"
        f"Combos evaluated: {optimizer_meta.get('n_combos_evaluated', '?'):,}  "
        f"Valid: {optimizer_meta.get('n_combos_valid', '?'):,}\n"
        f"{'─' * 40}"
    )
    sections.append(header)

    # ── Section 2: Recommended Portfolio ────────────────────────────
    rec_lines = ["🏆 *RECOMMENDED PORTFOLIO (Top 5)*\n"]
    for stock in sorted(selected, key=lambda x: x.get("rank", 99)):
        ticker  = stock["ticker"]
        sector  = stock.get("sector", "?")
        score   = stock.get("composite_score", 0)
        bd      = stock.get("score_breakdown", {})
        preds   = bd.get("predictions", {})
        p10d    = preds.get(10, {})

        p10 = p10d.get("p10", 0.0) if p10d else 0.0
        p50 = p10d.get("p50", 0.0) if p10d else 0.0
        p90 = p10d.get("p90", 0.0) if p10d else 0.0

        rec_lines.append(
            f"#{stock.get('rank', '?')} {ticker} [{sector}]\n"
            f"   Score: {score:.4f}  |  Vol: {stock.get('avg_daily_vol_k', 0)}k/day\n"
            f"   10d p10/p50/p90: {p10:.1%} / {p50:.1%} / {p90:.1%}\n"
            f"   ↩ {stock.get('selection_reason', '')[:80]}\n"
        )
    sections.append("\n".join(rec_lines))

    # ── Section 3: Holdings Comparison ──────────────────────────────
    cmp = comparison
    if cmp.get("no_positions"):
        cmp_text = (
            "📂 *CURRENT HOLDINGS*\n"
            "No open positions — entering fresh this week.\n"
            f"New entries: {', '.join(cmp.get('new_entries', []))}"
        )
    else:
        cmp_lines = [
            f"📂 *CURRENT HOLDINGS*\n"
            f"Overlap with recommendation: {cmp['overlap_count']}/5  "
            f"({', '.join(cmp.get('overlap_tickers', [])) or 'none'})\n"
        ]
        if cmp.get("exits"):
            cmp_lines.append("Rotation analysis:")
            for ex in cmp["exits"]:
                cost_m = ex["exit_cost_vnd"] / 1_000_000
                flag = "✅ ROTATE" if ex["rotation_suggested"] else "⏸ HOLD"
                repl = f" → {ex['replacement']}" if ex.get("replacement") else ""
                cmp_lines.append(
                    f"  {flag} {ex['ticker']}{repl}  "
                    f"Exit cost ≈ {cost_m:.1f}M VND"
                )
                cmp_lines.append(f"    {ex['reason']}")
        if cmp.get("new_entries"):
            cmp_lines.append(f"New entries: {', '.join(cmp['new_entries'])}")
        cmp_text = "\n".join(cmp_lines)
    sections.append(cmp_text)

    # ── Section 4: Diversification Summary ─────────────────────────
    sector_counts: Dict[str, int] = {}
    selected_tickers = [s["ticker"] for s in selected]
    for s in selected:
        sec = s.get("sector", "Unknown")
        sector_counts[sec] = sector_counts.get(sec, 0) + 1

    avg_corr = _avg_pairwise_corr(corr_df, selected_tickers)
    div_lines = ["🌐 *DIVERSIFICATION*\n"]
    for sec, cnt in sorted(sector_counts.items()):
        div_lines.append(f"  {sec}: {cnt}")
    div_lines.append(f"Avg pairwise correlation: {avg_corr:.2f}")
    sections.append("\n".join(div_lines))

    # ── Section 5: Near-Misses & Warnings ──────────────────────────
    warn_lines = ["⚠️ *NEAR-MISSES & WARNINGS*\n"]
    near_top3 = near_misses[:3]
    if near_top3:
        warn_lines.append("Near-misses (passed filter, excluded by constraints):")
        for nm in near_top3:
            warn_lines.append(
                f"  {nm['ticker']} [{nm.get('sector', '?')}] "
                f"score={nm.get('composite_score', 0):.4f} — "
                f"{nm.get('selection_reason', '')[:80]}"
            )

    # Filter stage counts
    elim_reasons: Dict[str, int] = {}
    for a in audit_trail:
        if a.get("eliminated"):
            reason_short = a["reason"].split(":")[0]
            elim_reasons[reason_short] = elim_reasons.get(reason_short, 0) + 1

    if elim_reasons:
        warn_lines.append(f"\nFilter eliminated {n_eliminated} stocks:")
        for reason, count in sorted(elim_reasons.items(), key=lambda x: -x[1]):
            warn_lines.append(f"  {reason}: {count}")

    sections.append("\n".join(warn_lines))

    # ── Assemble and split messages ─────────────────────────────────
    full_text = "\n\n".join(sections)
    messages = _split_messages(full_text, week_start)
    logger.info(f"Portfolio report generated: {len(messages)} Telegram message(s)")
    return messages


def _avg_pairwise_corr(corr_df, tickers: List[str]) -> float:
    """Compute average of all j>i pairs in the selected basket."""
    if corr_df is None or len(tickers) < 2:
        return 0.0
    total, count = 0.0, 0
    for i in range(len(tickers)):
        for j in range(i + 1, len(tickers)):
            try:
                total += float(corr_df.loc[tickers[i], tickers[j]])
                count += 1
            except KeyError:
                pass
    return total / count if count else 0.0


def _split_messages(text: str, week_start: str) -> List[str]:
    """Split a long message into chunks under TELEGRAM_MAX_CHARS."""
    if len(text) <= TELEGRAM_MAX_CHARS:
        return [text]

    parts = []
    remaining = text
    while remaining:
        if len(remaining) <= TELEGRAM_MAX_CHARS:
            parts.append(remaining)
            break
        split_at = remaining.rfind("\n\n", 0, TELEGRAM_MAX_CHARS)
        if split_at == -1:
            split_at = TELEGRAM_MAX_CHARS
        parts.append(remaining[:split_at].strip())
        remaining = remaining[split_at:].strip()

    total = len(parts)
    return [f"Portfolio Report ({i + 1}/{total}) — {week_start}\n\n{p}" for i, p in enumerate(parts)]
