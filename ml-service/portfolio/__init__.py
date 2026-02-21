"""
Portfolio selection package for weekly stock universe scanning.

Modules:
    universe   - Load stock_universe from database
    correlation - Build pairwise Pearson correlation matrix
    filter     - Hard-rule candidate filtering
    scorer     - Composite score calculation
    optimizer  - Brute-force C(n,5) portfolio optimiser
    comparator - Compare recommendation against current holdings
    report     - Telegram-formatted report builder
    selector   - Top-level orchestrator (run weekly selection end-to-end)
"""
