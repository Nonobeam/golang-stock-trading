import os
import psycopg
from dotenv import load_dotenv

# Load env from root
load_dotenv(os.path.join(os.path.dirname(os.path.dirname(__file__)), '.env'))

DB_HOST = os.getenv('DB_HOST', 'localhost')
DB_PORT = os.getenv('DB_PORT', '5432')
DB_USER = os.getenv('DB_USER', 'postgres')
DB_PASSWORD = os.getenv('DB_PASSWORD', '')
DB_NAME = os.getenv('DB_NAME', 'stock_trading')

MIGRATION_FILE = os.path.join(os.path.dirname(os.path.dirname(__file__)), 'db', 'migrations', '000008_add_multi_horizon_tables.up.sql')

def apply_migration():
    print(f"Connecting to {DB_NAME} at {DB_HOST}:{DB_PORT}...")
    try:
        conn = psycopg.connect(
            host=DB_HOST,
            port=DB_PORT,
            user=DB_USER,
            password=DB_PASSWORD,
            dbname=DB_NAME,
            autocommit=True
        )
        
        with open(MIGRATION_FILE, 'r') as f:
            sql = f.read()
            
        print(f"Executing migration from {MIGRATION_FILE}...")
        with conn.cursor() as cur:
            cur.execute(sql)
            
        print("Migration applied successfully!")
        conn.close()
        
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    apply_migration()
