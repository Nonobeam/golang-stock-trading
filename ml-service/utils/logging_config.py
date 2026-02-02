"""
Centralized logging configuration for ML Service.
"""
import os
import sys
import logging
import logging.handlers
from datetime import datetime

# Ensure logs directory exists
LOG_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "logs")
os.makedirs(LOG_DIR, exist_ok=True)

def setup_logging(name: str = "ml_service", log_level: str = "INFO") -> logging.Logger:
    """
    Setup structured logging with file rotation and console output.
    
    Args:
        name: Logger name
        log_level: Logging level (INFO, DEBUG, ERROR)
        
    Returns:
        Configured logger instance
    """
    logger = logging.getLogger(name)
    logger.setLevel(getattr(logging, log_level.upper()))
    
    # Prevent duplicate handlers
    if logger.handlers:
        return logger
        
    formatter = logging.Formatter(
        '%(asctime)s - %(name)s - %(levelname)s - %(filename)s:%(lineno)d - %(message)s'
    )
    
    # 1. Console Handler
    console_handler = logging.StreamHandler(sys.stdout)
    console_handler.setFormatter(formatter)
    logger.addHandler(console_handler)
    
    # 2. File Handler (Rotating)
    # Rotate every day, keep 30 days of logs
    log_file = os.path.join(LOG_DIR, "ml_service.log")
    file_handler = logging.handlers.TimedRotatingFileHandler(
        log_file, when="midnight", interval=1, backupCount=30
    )
    file_handler.setFormatter(formatter)
    logger.addHandler(file_handler)
    
    # 3. Error File Handler (Separate file for errors)
    error_log_file = os.path.join(LOG_DIR, "error.log")
    error_handler = logging.handlers.RotatingFileHandler(
        error_log_file, maxBytes=10*1024*1024, backupCount=10 # 10MB x 10
    )
    error_handler.setLevel(logging.ERROR)
    error_handler.setFormatter(formatter)
    logger.addHandler(error_handler)
    
    return logger
