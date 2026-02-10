#!/usr/bin/env python3
"""
RSS News Scraper - Main Entry Point

Fetches news articles from configured RSS feeds and stores them in the database.
Designed for scheduled execution (e.g., every 30 minutes via cron).
"""

import sys
from datetime import datetime
from services import RSSScraperService
from database import DatabaseManager
from utils import setup_logger
from config import Config


def main():
    """Main scraping workflow."""
    # Setup logging
    logger = setup_logger()
    logger.info("=" * 80)
    logger.info(f"News scraping started at {datetime.now()}")
    logger.info("=" * 80)

    db = None
    try:
        # Initialize services
        logger.info("Initializing RSS scraper...")
        scraper = RSSScraperService()

        logger.info("Connecting to database...")
        db = DatabaseManager()
        db.connect()

        # Fetch all RSS feeds
        logger.info(f"Fetching {len(Config.RSS_FEEDS)} RSS feeds...")
        all_articles = scraper.fetch_all_feeds()

        # Insert articles into database
        total_stats = {'inserted': 0, 'duplicates': 0, 'errors': 0}

        for source, articles in all_articles.items():
            if not articles:
                logger.info(f"No articles from {source}")
                continue

            logger.info(f"Processing {len(articles)} articles from {source}...")
            stats = db.insert_articles_batch(articles)

            # Aggregate stats
            total_stats['inserted'] += stats['inserted']
            total_stats['duplicates'] += stats['duplicates']
            total_stats['errors'] += stats['errors']

        # Log final results
        logger.info("=" * 80)
        logger.info("Scraping Summary:")
        logger.info(f"  New articles inserted: {total_stats['inserted']}")
        logger.info(f"  Duplicates skipped: {total_stats['duplicates']}")
        logger.info(f"  Errors: {total_stats['errors']}")

        # Show article counts by source
        counts = db.get_article_count_by_source()
        logger.info("\nTotal articles by source:")
        for source, count in counts.items():
            logger.info(f"  {source}: {count}")

        logger.info("=" * 80)
        logger.info(f"News scraping completed at {datetime.now()}")
        logger.info("=" * 80)

        # Return exit code based on results
        if total_stats['errors'] > 0:
            return 1
        return 0

    except KeyboardInterrupt:
        logger.info("Scraping interrupted by user")
        return 130

    except Exception as e:
        logger.error(f"Fatal error during scraping: {e}", exc_info=True)
        return 1

    finally:
        if db:
            db.close()


if __name__ == '__main__':
    sys.exit(main())

