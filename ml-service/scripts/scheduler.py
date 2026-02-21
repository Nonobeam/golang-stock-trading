"""
Weekly ML pipeline scheduler.

Sequence:
  Saturday/Sunday 22:00 ICT → retrain all models
  Monday 07:30 ICT → run portfolio selection

Run this as a long-lived process (e.g. via Windows Task Scheduler or a
service wrapper). It uses the `schedule` library to fire jobs at their
configured times.

Usage:
    python ml-service/scripts/scheduler.py

Requirements:
    pip install schedule
"""

import logging
import os
import subprocess
import sys
import time
from datetime import datetime

import schedule

# ---------------------------------------------------------------------------
# Paths – adjust if your layout differs
# ---------------------------------------------------------------------------
BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))  # ml-service/
PYTHON  = os.path.join(BASE_DIR, "venv", "Scripts", "python.exe")       # Windows venv
LOG_DIR = os.path.join(BASE_DIR, "logs")
os.makedirs(LOG_DIR, exist_ok=True)

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    handlers=[
        logging.StreamHandler(sys.stdout),
        logging.FileHandler(os.path.join(LOG_DIR, "scheduler.log"), encoding="utf-8"),
    ],
)
logger = logging.getLogger("scheduler")


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _run(label: str, *args: str) -> bool:
    """Run a Python module or script, stream output to log, return success."""
    logger.info(f"[{label}] Starting: {' '.join(args)}")
    try:
        result = subprocess.run(
            [PYTHON, *args],
            cwd=BASE_DIR,
            capture_output=False,  # inherit stdout/stderr so logs appear in console
            timeout=3600,          # 1 hour hard cap
        )
        if result.returncode == 0:
            logger.info(f"[{label}] Completed successfully.")
            return True
        else:
            logger.error(f"[{label}] Exited with code {result.returncode}")
            return False
    except subprocess.TimeoutExpired:
        logger.error(f"[{label}] Timed out after 1 hour.")
        return False
    except Exception as e:
        logger.error(f"[{label}] Unexpected error: {e}")
        return False


# ---------------------------------------------------------------------------
# Pipeline jobs
# ---------------------------------------------------------------------------

def job_retrain():
    """
    Weekend retraining job.
    Runs: feature backfill → prediction generation for ALL tickers.
    Fired Saturday and Sunday at 22:00 ICT.
    """
    logger.info("=" * 60)
    logger.info("WEEKLY RETRAIN JOB starting")
    logger.info("=" * 60)

    ok = _run("features", "-m", "daily.run_daily_features")
    if not ok:
        logger.error("Feature generation failed — skipping prediction step.")
        return

    ok = _run("predictions", "-m", "daily.run_daily_predictions")
    if not ok:
        logger.error("Prediction generation failed.")
        return

    logger.info("WEEKLY RETRAIN JOB complete.")


def job_portfolio_selection():
    """
    Monday portfolio selection job.
    Uses the most recent prediction date available in the DB.
    Fired Monday at 07:30 ICT.
    """
    logger.info("=" * 60)
    logger.info("WEEKLY PORTFOLIO SELECTION JOB starting")
    logger.info("=" * 60)

    _run("portfolio", "-m", "daily.run_weekly_portfolio")

    logger.info("WEEKLY PORTFOLIO SELECTION JOB complete.")


def job_retrain_and_portfolio():
    """
    Manual combined job: retrain then immediately run portfolio.
    Triggered via CLI:  python scheduler.py run-now
    """
    job_retrain()
    job_portfolio_selection()


# ---------------------------------------------------------------------------
# Schedule
# ---------------------------------------------------------------------------

def setup_schedule():
    # Saturday & Sunday at 22:00 local time → retrain
    schedule.every().saturday.at("22:00").do(job_retrain)
    schedule.every().sunday.at("22:00").do(job_retrain)

    # Monday at 07:30 local time → portfolio selection
    schedule.every().monday.at("07:30").do(job_portfolio_selection)

    logger.info("Scheduled jobs:")
    for job in schedule.jobs:
        logger.info(f"  {job}")


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "run-now":
        # Useful for testing the full chain immediately
        logger.info("Manual trigger: running full retrain + portfolio now.")
        job_retrain_and_portfolio()
        sys.exit(0)

    logger.info("ML Pipeline Scheduler starting. Press Ctrl+C to stop.")
    setup_schedule()

    while True:
        schedule.run_pending()
        time.sleep(30)   # check every 30 seconds
