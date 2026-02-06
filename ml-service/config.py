import os
from dotenv import load_dotenv

load_dotenv()

# Database configuration
DB_HOST = os.getenv("DB_HOST", "localhost")
DB_PORT = os.getenv("DB_PORT", "5432")
DB_NAME = os.getenv("DB_NAME", "stock-trading")
DB_USER = os.getenv("DB_USER", "postgres")
DB_PASSWORD = os.getenv("DB_PASSWORD", "")
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
