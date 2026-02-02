"""
Database connection and query tests
"""
import pytest
from db import get_connection, test_connection
import config

def test_database_connection():
    """Test database connection succeeds"""
    assert test_connection() == True

def test_database_schema_exists(db_connection):
    """Test required tables exist"""
    required_tables = ['daily_bars', 'features', 'predictions', 'model_metadata']
    
    with db_connection.cursor() as cursor:
        for table in required_tables:
            cursor.execute("""
                SELECT EXISTS (
                    SELECT FROM information_schema.tables 
                    WHERE table_schema = %(schema)s 
                    AND table_name = %(table)s
                )
            """, {'table': table, 'schema': config.DB_SCHEMA})
            
            result = cursor.fetchone()
            assert result['exists'], f"Table {table} does not exist"

def test_daily_bars_columns(db_connection):
    """Test daily_bars table has required columns"""
    with db_connection.cursor() as cursor:
        cursor.execute("""
            SELECT column_name 
            FROM information_schema.columns 
            WHERE table_name = 'daily_bars'
        """)
        
        columns = {row['column_name'] for row in cursor.fetchall()}
        required = {'symbol', 'date', 'open', 'high', 'low', 'close', 'volume', 'turnover'}
        
        assert required.issubset(columns), f"Missing columns: {required - columns}"

def test_features_columns(db_connection):
    """Test features table has required columns"""
    with db_connection.cursor() as cursor:
        cursor.execute("""
            SELECT column_name 
            FROM information_schema.columns 
            WHERE table_name = 'features'
        """)
        
        columns = {row['column_name'] for row in cursor.fetchall()}
        required = {'ticker', 'date', 'return_1d', 'sma_20', 'rsi_14', 'features_complete'}
        
        assert required.issubset(columns), f"Missing columns: {required - columns}"