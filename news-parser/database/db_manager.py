import psycopg
from psycopg import sql
from psycopg.rows import dict_row
import logging
from typing import List, Dict, Optional
from datetime import datetime
import json
from config import Config

logger = logging.getLogger(__name__)


class DatabaseManager:
    """Manages database connections and operations for news articles."""

    def __init__(self):
        self.connection_string = (
            f"host={Config.DB_HOST} "
            f"port={Config.DB_PORT} "
            f"dbname={Config.DB_NAME} "
            f"user={Config.DB_USER} "
            f"password={Config.DB_PASSWORD}"
        )
        self.conn = None

    def connect(self):
        """Establish database connection."""
        try:
            self.conn = psycopg.connect(self.connection_string, row_factory=dict_row)
            logger.info("Database connection established")
        except psycopg.Error as e:
            logger.error(f"Database connection failed: {e}")
            raise

    def close(self):
        """Close database connection."""
        if self.conn:
            self.conn.close()
            logger.info("Database connection closed")

    def insert_article(self, article: Dict) -> Optional[int]:
        """
        Insert a single article into the database.

        Args:
            article: Article dictionary with keys: source, title, link,
                    published_date, summary, raw_content

        Returns:
            Article ID if successful, None if duplicate or error
        """
        if not self.conn:
            logger.error("No database connection")
            return None

        try:
            with self.conn.cursor() as cur:
                cur.execute(
                    """
                    INSERT INTO news_articles
                    (source, title, link, published_date, summary, raw_content)
                    VALUES (%s, %s, %s, %s, %s, %s)
                    ON CONFLICT (link) DO NOTHING
                    RETURNING article_id
                    """,
                    (
                        article['source'],
                        article['title'],
                        article['link'],
                        article.get('published_date'),
                        article.get('summary'),
                        json.dumps(article.get('raw_content', {}))
                    )
                )

                result = cur.fetchone()
                self.conn.commit()

                if result:
                    article_id = result['article_id']
                    logger.debug(f"Inserted article {article_id}: {article['title'][:50]}...")
                    return article_id
                else:
                    logger.debug(f"Article already exists: {article['link']}")
                    return None

        except psycopg.Error as e:
            logger.error(f"Error inserting article: {e}")
            self.conn.rollback()
            return None

    def insert_articles_batch(self, articles: List[Dict]) -> Dict[str, int]:
        """
        Insert multiple articles in batch.

        Args:
            articles: List of article dictionaries

        Returns:
            Dictionary with counts: {'inserted': N, 'duplicates': M, 'errors': K}
        """
        stats = {'inserted': 0, 'duplicates': 0, 'errors': 0}

        for article in articles:
            result = self.insert_article(article)
            if result is not None:
                stats['inserted'] += 1
            elif result is None and 'link' in article:
                # Check if it was a duplicate or error
                if self.article_exists(article['link']):
                    stats['duplicates'] += 1
                else:
                    stats['errors'] += 1

        logger.info(f"Batch insert complete: {stats}")
        return stats

    def article_exists(self, link: str) -> bool:
        """
        Check if article with given link already exists.

        Args:
            link: Article URL

        Returns:
            True if exists, False otherwise
        """
        if not self.conn:
            return False

        try:
            with self.conn.cursor() as cur:
                cur.execute(
                    "SELECT 1 FROM news_articles WHERE link = %s LIMIT 1",
                    (link,)
                )
                return cur.fetchone() is not None
        except psycopg.Error as e:
            logger.error(f"Error checking article existence: {e}")
            return False

    def get_recent_articles(self, hours: int = 24, source: Optional[str] = None) -> List[Dict]:
        """
        Get articles from the last N hours.

        Args:
            hours: Number of hours to look back
            source: Optional source filter

        Returns:
            List of article dictionaries
        """
        if not self.conn:
            return []

        try:
            with self.conn.cursor() as cur:
                if source:
                    cur.execute(
                        """
                        SELECT article_id, source, title, link, published_date,
                               scraped_date, summary
                        FROM news_articles
                        WHERE published_date >= NOW() - INTERVAL '%s hours'
                          AND source = %s
                        ORDER BY published_date DESC
                        """,
                        (hours, source)
                    )
                else:
                    cur.execute(
                        """
                        SELECT article_id, source, title, link, published_date,
                               scraped_date, summary
                        FROM news_articles
                        WHERE published_date >= NOW() - INTERVAL '%s hours'
                        ORDER BY published_date DESC
                        """,
                        (hours,)
                    )

                return cur.fetchall()

        except psycopg.Error as e:
            logger.error(f"Error fetching recent articles: {e}")
            return []

    def get_article_count_by_source(self) -> Dict[str, int]:
        """
        Get article counts grouped by source.

        Returns:
            Dictionary mapping source to count
        """
        if not self.conn:
            return {}

        try:
            with self.conn.cursor() as cur:
                cur.execute(
                    """
                    SELECT source, COUNT(*) as count
                    FROM news_articles
                    GROUP BY source
                    ORDER BY count DESC
                    """
                )

                results = cur.fetchall()
                return {row['source']: row['count'] for row in results}

        except psycopg.Error as e:
            logger.error(f"Error getting article counts: {e}")
            return {}

    def log_scrape_run(self, stats: Dict) -> None:
        """
        Log scraping run statistics.

        Args:
            stats: Dictionary with scraping statistics
        """
        logger.info(
            f"Scrape run completed: "
            f"{stats.get('inserted', 0)} new articles, "
            f"{stats.get('duplicates', 0)} duplicates, "
            f"{stats.get('errors', 0)} errors"
        )
