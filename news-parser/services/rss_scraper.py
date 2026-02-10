import feedparser
import logging
from typing import List, Dict, Optional
from datetime import datetime
from config import Config
import time

logger = logging.getLogger(__name__)


class RSSScraperService:
    """Service for scraping RSS feeds and parsing articles."""

    def __init__(self):
        self.feeds = Config.RSS_FEEDS
        self.retry_attempts = Config.RETRY_ATTEMPTS
        self.retry_delay = Config.RETRY_DELAY

    def fetch_feed(self, url: str, source: str) -> Optional[List[Dict]]:
        """
        Fetch and parse a single RSS feed.

        Args:
            url: RSS feed URL
            source: Source identifier (e.g., 'cafef', 'vnexpress')

        Returns:
            List of parsed articles or None if error
        """
        for attempt in range(self.retry_attempts):
            try:
                logger.info(f"Fetching RSS feed from {source}: {url} (attempt {attempt + 1})")
                feed = feedparser.parse(url)

                if feed.bozo:
                    logger.warning(f"Feed parsing warning for {source}: {feed.bozo_exception}")

                if not feed.entries:
                    logger.warning(f"No entries found in feed: {source}")
                    return []

                articles = []
                for entry in feed.entries:
                    article = self._parse_entry(entry, source)
                    if article:
                        articles.append(article)

                logger.info(f"Successfully fetched {len(articles)} articles from {source}")
                return articles

            except Exception as e:
                logger.error(f"Error fetching feed {source} (attempt {attempt + 1}): {e}")
                if attempt < self.retry_attempts - 1:
                    time.sleep(self.retry_delay)

        logger.error(f"Failed to fetch feed {source} after {self.retry_attempts} attempts")
        return None

    def _parse_entry(self, entry, source: str) -> Optional[Dict]:
        """
        Parse a single RSS entry into standardized format.

        Args:
            entry: feedparser entry object
            source: Source identifier

        Returns:
            Parsed article dict or None if parsing fails
        """
        try:
            # Extract title (required)
            title = entry.get('title', '').strip()
            if not title:
                logger.warning(f"Entry from {source} missing title, skipping")
                return None

            # Extract link (required)
            link = entry.get('link', '').strip()
            if not link:
                logger.warning(f"Entry from {source} missing link, skipping")
                return None

            # Extract published date (optional, parse if available)
            published_date = None
            if hasattr(entry, 'published_parsed') and entry.published_parsed:
                try:
                    published_date = datetime(*entry.published_parsed[:6])
                except (ValueError, TypeError) as e:
                    logger.warning(f"Could not parse published date for {link}: {e}")

            # Extract summary (optional)
            summary = entry.get('summary', '').strip() or entry.get('description', '').strip()

            # Store raw entry as JSON for future analysis
            raw_content = {
                'title': entry.get('title', ''),
                'link': entry.get('link', ''),
                'published': entry.get('published', ''),
                'summary': entry.get('summary', ''),
                'description': entry.get('description', ''),
                'author': entry.get('author', ''),
                'tags': [tag.get('term', '') for tag in entry.get('tags', [])],
                'id': entry.get('id', ''),
            }

            return {
                'source': source,
                'title': title,
                'link': link,
                'published_date': published_date,
                'summary': summary,
                'raw_content': raw_content
            }

        except Exception as e:
            logger.error(f"Error parsing entry from {source}: {e}")
            return None

    def fetch_all_feeds(self) -> Dict[str, List[Dict]]:
        """
        Fetch all configured RSS feeds.

        Returns:
            Dictionary mapping source to list of articles
        """
        results = {}
        total_articles = 0

        for source, url in self.feeds.items():
            articles = self.fetch_feed(url, source)
            if articles is not None:
                results[source] = articles
                total_articles += len(articles)
            else:
                results[source] = []

        logger.info(f"Total articles fetched: {total_articles} from {len(results)} sources")
        return results
