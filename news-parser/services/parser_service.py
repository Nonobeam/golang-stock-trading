from bs4 import BeautifulSoup
from typing import List, Optional


class ParserService:
    def __init__(self, html_content: str, parser: str = 'lxml'):
        self.soup = BeautifulSoup(html_content, parser)

    def find_by_selector(self, selector: str) -> Optional[BeautifulSoup]:
        return self.soup.select_one(selector)

    def find_all_by_selector(self, selector: str) -> List[BeautifulSoup]:
        return self.soup.select(selector)

    def find_by_tag(self, tag: str, attrs: Optional[dict] = None) -> Optional[BeautifulSoup]:
        return self.soup.find(tag, attrs=attrs)

    def find_all_by_tag(self, tag: str, attrs: Optional[dict] = None) -> List[BeautifulSoup]:
        return self.soup.find_all(tag, attrs=attrs)

    def get_text(self, element: Optional[BeautifulSoup] = None, strip: bool = True) -> str:
        target = element if element else self.soup
        return target.get_text(strip=strip)

    def get_attribute(self, element: BeautifulSoup, attr: str) -> Optional[str]:
        return element.get(attr)
