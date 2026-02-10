#!/usr/bin/env python3
"""
Test script to verify RSS feeds are accessible and parseable.
Run this before setting up database to validate feed sources.
"""

from services import RSSScraperService
from utils import setup_logger


def test_feeds():
    """Test each RSS feed independently."""
    logger = setup_logger('test-rss')
    scraper = RSSScraperService()

    logger.info("Testing RSS feeds...")
    logger.info("=" * 80)

    all_results = scraper.fetch_all_feeds()

    for source, articles in all_results.items():
        logger.info(f"\nSource: {source}")
        logger.info(f"Articles fetched: {len(articles)}")

        if articles:
            # Show first article as sample
            first = articles[0]
            logger.info(f"Sample article:")
            logger.info(f"  Title: {first['title'][:80]}...")
            logger.info(f"  Link: {first['link']}")
            logger.info(f"  Published: {first.get('published_date', 'N/A')}")
            logger.info(f"  Summary: {first.get('summary', 'N/A')[:100]}...")

            # Check what fields are available
            available_fields = []
            if first.get('published_date'):
                available_fields.append('published_date')
            if first.get('summary'):
                available_fields.append('summary')
            if first.get('raw_content', {}).get('author'):
                available_fields.append('author')
            if first.get('raw_content', {}).get('tags'):
                available_fields.append('tags')

            logger.info(f"  Available fields: {', '.join(available_fields)}")
        else:
            logger.warning(f"No articles fetched from {source}")

    logger.info("=" * 80)
    logger.info("Test complete!")


if __name__ == '__main__':
    test_feeds()
