import psycopg
from psycopg.rows import dict_row
import config

def get_connection():
    """Create PostgreSQL database connection"""
    conn = psycopg.connect(
        host=config.DB_HOST,
        port=config.DB_PORT,
        dbname=config.DB_NAME,
        user=config.DB_USER,
        password=config.DB_PASSWORD,
        row_factory=dict_row,
        options=f"-c search_path=\"{config.DB_SCHEMA}\""
    )
    return conn

def test_connection():
    """Test database connection"""
    try:
        conn = get_connection()
        cursor = conn.cursor()
        cursor.execute("SELECT 1")
        cursor.close()
        conn.close()
        print("Database connection successful")
        return True
    except Exception as e:
        print(f"Database connection failed: {e}")
        return False

class DatabaseConnection:
    """Connection factory for backward compatibility"""
    
    @staticmethod
    def get_connection():
        return get_connection()
        
    @staticmethod
    def return_connection(conn):
        try:
            conn.close()
        except:
            pass