-- Create news_articles table for storing RSS feed data
CREATE TABLE IF NOT EXISTS news_articles (
    article_id SERIAL PRIMARY KEY,
    source VARCHAR(50) NOT NULL,
    title TEXT NOT NULL,
    link TEXT NOT NULL UNIQUE,
    published_date TIMESTAMP,
    scraped_date TIMESTAMP NOT NULL DEFAULT NOW(),
    summary TEXT,
    raw_content JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for fast queries
CREATE INDEX idx_news_articles_published_date ON news_articles(published_date DESC);
CREATE INDEX idx_news_articles_source ON news_articles(source);
CREATE INDEX idx_news_articles_scraped_date ON news_articles(scraped_date DESC);

-- Create article_stock_mentions table for ticker associations
CREATE TABLE IF NOT EXISTS article_stock_mentions (
    mention_id SERIAL PRIMARY KEY,
    article_id INTEGER NOT NULL REFERENCES news_articles(article_id) ON DELETE CASCADE,
    ticker VARCHAR(10) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(article_id, ticker)
);

CREATE INDEX idx_article_stock_mentions_ticker ON article_stock_mentions(ticker);
CREATE INDEX idx_article_stock_mentions_article_id ON article_stock_mentions(article_id);

-- Create view for recent articles
CREATE OR REPLACE VIEW recent_news AS
SELECT
    article_id,
    source,
    title,
    link,
    published_date,
    scraped_date,
    summary
FROM news_articles
WHERE published_date >= NOW() - INTERVAL '24 hours'
ORDER BY published_date DESC;

-- Create view for article counts by source
CREATE OR REPLACE VIEW news_article_counts AS
SELECT
    source,
    DATE(published_date) as date,
    COUNT(*) as article_count
FROM news_articles
WHERE published_date IS NOT NULL
GROUP BY source, DATE(published_date)
ORDER BY date DESC, source;

COMMENT ON TABLE news_articles IS 'Stores news articles collected from RSS feeds';
COMMENT ON TABLE article_stock_mentions IS 'Links articles to stock tickers mentioned in content';
COMMENT ON COLUMN news_articles.raw_content IS 'Full RSS entry as JSON for future analysis';
