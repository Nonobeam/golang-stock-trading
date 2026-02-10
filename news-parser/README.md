# News Parser - RSS News Scraper

Python service for fetching financial news from Vietnamese RSS feeds and storing them in PostgreSQL.

## Overview

This service scrapes news articles from major Vietnamese financial news sources:
- CafeF - Stock market news
- VnExpress - Business news
- Vietstock - Forum and analysis
- NDH - Stock market news

Articles are stored in the `news_articles` table in the stock-trading database for future analysis and integration with trading signals.

## Features

- RSS feed parsing using feedparser
- Automatic duplicate detection
- Retry logic for failed fetches
- Structured logging with daily log files
- Database views for common queries
- Support for article-to-ticker associations

## Installation

1. Create virtual environment:
```bash
cd news-parser
python -m venv venv
```

2. Activate virtual environment:
```bash
# Windows
venv\Scripts\activate

# Linux/Mac
source venv/bin/activate
```

3. Install dependencies:
```bash
pip install -r requirements.txt
```

4. Configure environment:
```bash
cp .env.example .env
# Edit .env with your database credentials
```

5. Run database migration:
```bash
cd ..
migrate -path db/migrations -database "postgresql://user:pass@localhost:5432/stock_trading?sslmode=disable" up
```

## Usage

### Manual Run

```bash
python main.py
```

### Scheduled Execution (Cron)

Add to crontab for running every 30 minutes during market hours (9:00-14:30 Vietnam time):

```cron
# Run every 30 minutes from 9:00 to 14:30 on weekdays
*/30 9-14 * * 1-5 cd /path/to/news-parser && /path/to/venv/bin/python main.py

# Alternative: Run every 30 minutes all day
*/30 * * * * cd /path/to/news-parser && /path/to/venv/bin/python main.py
```

## Project Structure

```
news-parser/
├── config.py              # Configuration settings
├── main.py                # Main entry point
├── requirements.txt       # Python dependencies
├── .env                   # Environment variables (not in git)
├── .env.example          # Example environment file
│
├── services/             # Service layer
│   ├── rss_scraper.py   # RSS feed fetching and parsing
│   ├── scraper_service.py  # HTTP scraper (generic)
│   └── parser_service.py   # HTML parser (generic)
│
├── database/             # Database layer
│   └── db_manager.py    # PostgreSQL operations
│
├── utils/                # Utilities
│   └── logger_config.py # Logging setup
│
└── logs/                 # Log files (auto-created)
    └── scraper_YYYYMMDD.log
```

## Database Schema

### news_articles Table

```sql
- article_id (PK)
- source (varchar) - RSS feed source identifier
- title (text) - Article headline
- link (text, unique) - Article URL
- published_date (timestamp) - When article was published
- scraped_date (timestamp) - When we fetched it
- summary (text) - Article summary/description
- raw_content (jsonb) - Full RSS entry for future analysis
```

### Indexes

- `idx_news_articles_published_date` - Fast date queries
- `idx_news_articles_source` - Filter by source
- `idx_news_articles_scraped_date` - Track scraping history

### Views

- `recent_news` - Articles from last 24 hours
- `news_article_counts` - Daily article counts by source

## Configuration

Edit `config.py` or set environment variables:

```python
RSS_FEEDS = {
    'cafef': 'https://cafef.vn/thi-truong-chung-khoan.rss',
    'vnexpress': 'https://vnexpress.net/rss/kinh-doanh.rss',
    'vietstock_news': 'https://vietstock.vn/forum.rss',
    'vietstock_analysis': 'https://vietstock.vn/phan-tich.rss',
    'ndh': 'https://ndh.vn/rss/chung-khoan.rss'
}
```

## Logging

Logs are written to:
- Console (stdout) - Timestamped messages
- File - `logs/scraper_YYYYMMDD.log` (daily rotation)

Log levels:
- INFO - Normal operation
- WARNING - Feed parsing issues, missing fields
- ERROR - Database errors, fetch failures
- DEBUG - Detailed article processing

## Error Handling

- Automatic retry (3 attempts) for failed RSS fetches
- Graceful handling of missing article fields
- Duplicate detection prevents re-inserting same articles
- Database transaction rollback on errors

## Future Enhancements (Planned)

Phase 2-3:
- Ticker extraction from article titles/summaries
- Article-to-stock associations in `article_stock_mentions` table
- Query functions for ML service integration

Phase 4+:
- Sentiment analysis on Vietnamese text
- Category/tag classification
- Real-time alerts for breaking news
- Integration with trading signal generation

## Maintenance

### Check Scraping Status

```sql
-- Recent scraping activity
SELECT source, COUNT(*), MAX(scraped_date)
FROM news_articles
WHERE scraped_date >= NOW() - INTERVAL '24 hours'
GROUP BY source;

-- Articles per day
SELECT * FROM news_article_counts
WHERE date >= CURRENT_DATE - 7
ORDER BY date DESC;
```

### Database Cleanup

```sql
-- Remove old articles (optional, after 6+ months)
DELETE FROM news_articles
WHERE published_date < NOW() - INTERVAL '180 days';
```

## Troubleshooting

**No articles fetched:**
- Check RSS feed URLs are accessible
- Verify network connectivity
- Check logs for parsing errors

**Database connection errors:**
- Verify database credentials in .env
- Check PostgreSQL is running
- Ensure migration has been run

**Duplicate articles:**
- Normal behavior - duplicate detection prevents re-insertion
- Check `duplicates` count in logs

**Vietnamese character encoding:**
- Database should use UTF-8 encoding
- psycopg3 handles encoding automatically

## License

Part of the golang-stock-trading project.
