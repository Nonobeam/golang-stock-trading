import os
from dotenv import load_dotenv

load_dotenv()


class Config:
    USER_AGENT = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'
    REQUEST_TIMEOUT = 30
    RETRY_ATTEMPTS = 3
    RETRY_DELAY = 2

    # RSS Feed URLs
    RSS_FEEDS = {
        'cafef': 'https://cafef.vn/thi-truong-chung-khoan.rss',
        'vnexpress': 'https://vnexpress.net/rss/kinh-doanh.rss',
        'vietstock_news': 'https://vietstock.vn/forum.rss',
        'vietstock_analysis': 'https://vietstock.vn/phan-tich.rss',
        'ndh': 'https://ndh.vn/rss/chung-khoan.rss'
    }

    # Database Configuration
    DB_HOST = os.getenv('DB_HOST', 'localhost')
    DB_PORT = os.getenv('DB_PORT', '5432')
    DB_NAME = os.getenv('DB_NAME', 'stock_trading')
    DB_USER = os.getenv('DB_USER', 'postgres')
    DB_PASSWORD = os.getenv('DB_PASSWORD', '')

    # Scraping Schedule
    SCRAPE_INTERVAL_MINUTES = 30
