"""
Alerting module for sending notifications via Telegram.
"""
import os
import requests
import json
from datetime import datetime

from utils.logging_config import setup_logging

logger = setup_logging("monitoring.alerter")

class Alerter:
    """Handles sending alerts to external channels."""
    
    def __init__(self):
        """Initialize Alerter with credentials from environment."""
        self.bot_token = os.getenv('TELEGRAM_BOT_TOKEN')
        self.chat_id = os.getenv('TELEGRAM_CHAT_ID')
        
        if not self.bot_token or not self.chat_id:
            logger.warning("Telegram credentials not found. Alerting disabled.")
            
    def send_alert(self, message: str, level: str = "WARNING") -> bool:
        """
        Send an alert message via Telegram.
        
        Args:
            message: Content of the alert
            level: Severity level (INFO, WARNING, CRITICAL)
            
        Returns:
            True if sent successfully
        """
        if not self.bot_token or not self.chat_id:
            logger.info(f"Alert ({level}) skipped (no credentials): {message}")
            return False
            
        try:
            # Format message
            timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            formatted_msg = f"*{level}* - {timestamp}\n{message}"
            
            url = f"https://api.telegram.org/bot{self.bot_token}/sendMessage"
            payload = {
                'chat_id': self.chat_id,
                'text': formatted_msg,
                'parse_mode': 'Markdown'
            }
            
            response = requests.post(url, json=payload, timeout=5)
            
            if response.status_code == 200:
                logger.info(f"Alert sent: {message}")
                return True
            else:
                logger.error(f"Failed to send alert: {response.text}")
                return False
                
        except Exception as e:
            logger.error(f"Error sending alert: {e}")
            return False

# Global instance
alerter = Alerter()
