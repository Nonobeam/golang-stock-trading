#!/usr/bin/env python3
"""
Simple Migration Runner for 000013_add_position_entries_tracking
"""
import sys
import logging
import os

# Add ml-service to path so we can import db.connection
sys.path.insert(0, os.path.join(os.getcwd(), 'ml-service'))

from db.connection import get_connection

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

MIGRATION_FILE = 'db/migrations/000013_add_position_entries_tracking.up.sql'

def run_migration():
    if not os.path.exists(MIGRATION_FILE):
        logger.error(f"Migration file not found: {MIGRATION_FILE}")
        return 1
        
    try:
        with open(MIGRATION_FILE, 'r') as f:
            sql_content = f.read()
            
        conn = get_connection()
        conn.autocommit = False
        
        try:
            with conn.cursor() as cur:
                logger.info(f"Executing migration: {MIGRATION_FILE}")
                cur.execute(sql_content)
                
            conn.commit()
            logger.info("Migration executed successfully.")
            return 0
            
        except Exception as e:
            conn.rollback()
            logger.error(f"Migration failed: {e}")
            # Check if error is "already exists" to handle re-runs gracefully
            if "already exists" in str(e):
                logger.warning("It seems some objects already exist. Please check manual intervention.")
            return 1
            
        finally:
            conn.close()
            
    except Exception as e:
        logger.error(f"Failed to connect or read file: {e}")
        return 1

if __name__ == '__main__':
    sys.exit(run_migration())
