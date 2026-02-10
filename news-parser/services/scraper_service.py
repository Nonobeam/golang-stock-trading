import requests
from typing import Optional, Dict
from config import Config


class ScraperService:
    def __init__(self, timeout: int = Config.REQUEST_TIMEOUT):
        self.timeout = timeout
        self.session = requests.Session()
        self.session.headers.update({'User-Agent': Config.USER_AGENT})

    def fetch_page(self, url: str, headers: Optional[Dict] = None) -> Optional[str]:
        try:
            response = self.session.get(
                url,
                headers=headers,
                timeout=self.timeout
            )
            response.raise_for_status()
            return response.text
        except requests.RequestException as e:
            print(f"Error fetching {url}: {e}")
            return None

    def fetch_page_with_retry(self, url: str, retries: int = Config.RETRY_ATTEMPTS) -> Optional[str]:
        import time

        for attempt in range(retries):
            result = self.fetch_page(url)
            if result:
                return result
            if attempt < retries - 1:
                time.sleep(Config.RETRY_DELAY)
        return None

    def close(self):
        self.session.close()
