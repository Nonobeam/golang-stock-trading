import os
from dotenv import load_dotenv

load_dotenv()

# Database configuration
DB_HOST = os.getenv("DB_HOST", "localhost")
DB_PORT = os.getenv("DB_PORT", "5432")
DB_NAME = os.getenv("DB_NAME", "stock-trading")
DB_USER = os.getenv("DB_USER", "postgres")
DB_PASSWORD = os.getenv("DB_PASSWORD", "postgres")
DB_SCHEMA = os.getenv("DB_SCHEMA", "public")

# Training configuration
TRAINING_WINDOW = 252  # days
VALIDATION_WINDOW = 80  # days
TEST_WINDOW = 20  # days
MIN_HISTORY_REQUIRED = 252  # minimum days needed

# Model configuration
QUANTILES = [0.10, 0.50, 0.90]
HYPERPARAMETERS = {
    "max_depth": 5,
    "learning_rate": 0.05,
    "n_estimators": 200,
    "subsample": 0.8,
    "colsample_bytree": 0.8,
    "reg_lambda": 5,
    "reg_alpha": 0,
    "random_state": 42
}

from pathlib import Path

# Paths
BASE_DIR = Path(__file__).resolve().parent
MODELS_DIR = BASE_DIR / "models" / "saved"

# ------------------------------------------------------------------ #
# Portfolio selection configuration
# Tune these weights after reviewing real weekly outputs.
# ------------------------------------------------------------------ #
PORTFOLIO_CONFIG = {
    # --- Scoring weights (must sum to 1.0) ---
    "weight_return":    0.30,   # weighted p50 return across horizons
    "weight_risk_adj":  0.25,   # penalise wide p10-p90 spread (uncertainty)
    "weight_liquidity": 0.20,   # tier-based liquidity score
    "weight_floor":     0.15,   # floor-hit penalty score
    "weight_momentum":  0.10,   # p10(10d) > 0 bonus

    # --- Horizon weights for return score (must sum to 1.0) ---
    "horizon_weight_1d":  0.20,
    "horizon_weight_5d":  0.35,
    "horizon_weight_10d": 0.45,

    # --- Hard filter thresholds ---
    "max_floor_prob":       0.20,    # stocks with floor_hit_prob > this are eliminated
    "min_daily_vol_k":      100,     # min avg daily volume in thousands of shares
    "min_expected_return":  0.004,   # minimum net-of-fee expected return (0.4%)
    "min_confidence":       0.60,    # minimum model prediction confidence

    # --- Optimiser constraints ---
    "portfolio_size":       5,       # number of stocks to select
    "max_sector_count":     2,       # max stocks from same sector in portfolio
    "max_pairwise_corr":    0.70,    # max allowed pairwise correlation
    "default_unknown_corr": 0.50,    # assumed correlation when history < 30 days overlap
    "min_history_days":     60,      # min trading days needed for correlation

    # --- Rotation threshold ---
    "rotation_score_improvement": 0.15,  # replacement must score >= 15% higher to recommend swap

    # --- Trading fee estimate for exit cost ---
    "round_trip_fee_rate": 0.004,   # ~0.4% round trip (sell fee + taxes)

    # --- Scheduling ---
    "run_time_hour_ict":   7,    # 07:30 ICT Monday
    "run_time_minute_ict": 30,
}
