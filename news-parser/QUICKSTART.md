# News Parser Quick Start Guide

## Installation

1. **Navigate to news-parser directory:**
```bash
cd news-parser
```

2. **Create virtual environment:**
```bash
python -m venv venv
```

3. **Activate virtual environment:**
```bash
# Windows
venv\Scripts\activate

# Linux/Mac
source venv/bin/activate
```

4. **Install dependencies:**
```bash
pip install -r requirements.txt
```

5. **Configure environment:**
```bash
# Copy example file
cp .env.example .env

# Edit .env with your database credentials
# Use the same credentials as your main stock-trading project
```

## Database Setup

Run the migration to create the `news_articles` table:

```bash
# From the golang-stock-trading root directory
migrate -path db/migrations -database "postgresql://user:password@localhost:5432/stock_trading?sslmode=disable" up
```

Or if you have a migration script:
```bash
# Windows
migrate.bat up

# Linux
./migrate.sh up
```

## Testing

### Test RSS Feeds (No Database Required)

Before setting up the database, test if RSS feeds are accessible:

```bash
python test_feeds.py
```

This will:
- Fetch all 5 RSS feeds
- Display article counts per source
- Show sample articles
- List available fields from each feed

Expected output:
```
Testing RSS feeds...
Source: cafef
Articles fetched: 20
Sample article:
  Title: VN-Index tăng điểm, thanh khoản cải thiện...
  Link: https://cafef.vn/...
  Published: 2026-02-07 14:30:00
  Available fields: published_date, summary
```

### Full Database Test

Run the scraper once to test full pipeline:

```bash
python main.py
```

Check logs in `logs/scraper_YYYYMMDD.log` for results.

## Verify Database

Check if articles were inserted:

```sql
-- Total article count by source
SELECT source, COUNT(*) as count
FROM news_articles
GROUP BY source;

-- Recent articles (last 24 hours)
SELECT source, title, published_date
FROM recent_news
LIMIT 10;

-- Check scraping activity
SELECT source, MAX(scraped_date) as last_scraped
FROM news_articles
GROUP BY source;
```

## Scheduling

### Windows Task Scheduler

Create a scheduled task to run every 30 minutes:

1. Open Task Scheduler
2. Create Basic Task
3. Set trigger: Daily, repeat every 30 minutes
4. Action: Start a program
5. Program: `C:\path\to\news-parser\venv\Scripts\python.exe`
6. Arguments: `C:\path\to\news-parser\main.py`
7. Start in: `C:\path\to\news-parser`

### Linux Cron

Add to crontab:

```bash
# Edit crontab
crontab -e

# Add line (every 30 minutes)
*/30 * * * * cd /path/to/news-parser && /path/to/venv/bin/python main.py

# Or only during market hours (9:00-14:30 weekdays)
*/30 9-14 * * 1-5 cd /path/to/news-parser && /path/to/venv/bin/python main.py
```

## Monitoring

### Check Logs

```bash
# View today's log
tail -f logs/scraper_$(date +%Y%m%d).log

# Search for errors
grep ERROR logs/scraper_*.log
```

### Database Queries

```sql
-- Articles per day
SELECT * FROM news_article_counts
WHERE date >= CURRENT_DATE - 7
ORDER BY date DESC, source;

-- Check for gaps in scraping
SELECT source,
       DATE_TRUNC('hour', scraped_date) as hour,
       COUNT(*) as articles
FROM news_articles
WHERE scraped_date >= NOW() - INTERVAL '24 hours'
GROUP BY source, hour
ORDER BY hour DESC;
```

## Troubleshooting

### No articles fetched

1. Check internet connectivity
2. Verify RSS URLs are still valid
3. Check firewall/proxy settings
4. Review logs for specific errors

### Database errors

1. Verify PostgreSQL is running
2. Check credentials in .env
3. Ensure migration has been run
4. Check database permissions

### Encoding issues

If Vietnamese characters are garbled:
1. Ensure database uses UTF-8 encoding
2. Check terminal encoding
3. psycopg3 should handle this automatically

## Next Steps

Once you have data flowing:

1. **Analyze article patterns** - Which sources update most frequently?
2. **Ticker extraction** - Identify stock mentions in titles/summaries
3. **Sentiment analysis** - Classify articles as positive/negative/neutral
4. **Integration** - Connect with signal generation pipeline
5. **Alerts** - Real-time notifications for breaking news

See `README.md` for detailed documentation.
