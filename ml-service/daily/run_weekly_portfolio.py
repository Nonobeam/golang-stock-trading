"""
Weekly portfolio selection runner script.

Invoked by cron/Task Scheduler every Monday at 07:30 ICT (00:30 UTC).
Also usable as ad-hoc manual trigger:
    python -m daily.run_weekly_portfolio [YYYY-MM-DD]
"""
import sys
import os
import logging
from datetime import date

# Ensure ml-service root is on path when run directly
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from utils.logging_config import setup_logging
import portfolio.selector as selector
from monitoring.alerter import alerter

logger = setup_logging("weekly_portfolio")


def main() -> None:
    pred_date: str | None = None
    if len(sys.argv) > 1:
        pred_date = sys.argv[1]
        logger.info(f"Ad-hoc historical run for date: {pred_date}")
    else:
        # Instead of defaulting to today, find the most recent date with ML predictions
        from data.loader import DataLoader
        latest_date = DataLoader.get_latest_prediction_date()
        if latest_date:
            pred_date = latest_date
            logger.info(f"No date provided. Found latest prediction date in DB: {pred_date}")
        else:
            # Fallback if DB is completely empty
            pred_date = date.today().isoformat()
            logger.warning(f"No date provided and no predictions found in DB! Defaulting to today: {pred_date}")

    try:
        messages = selector.run(pred_date=pred_date)
        logger.info(f"Run complete. {len(messages)} Telegram message(s) sent.")
        # Note: selector.run() already sends its own very detailed telegram messages.
        # We will just send a short confirmation/failure status here.
        alerter.send_alert(f"✅ Weekly Portfolio Strategy generated for {pred_date}.\nSent {len(messages)} detailed notification(s).", level="INFO")
    except Exception as e:
        logger.critical(f"Weekly portfolio selection failed: {e}", exc_info=True)
        alerter.send_alert(f"❌ Weekly Portfolio Selection failed.\nError: {e}", level="CRITICAL")
        sys.exit(1)


if __name__ == "__main__":
    main()
