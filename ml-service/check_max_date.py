import os
import psycopg2
from dotenv import load_dotenv

load_dotenv()

conn = psycopg2.connect(
    host=os.getenv('DB_HOST', 'localhost'),
    port=os.getenv('DB_PORT', '5432'),
    user=os.getenv('DB_USER', 'postgres'),
    password=os.getenv('DB_PASSWORD', 'postgres'),
    dbname=os.getenv('DB_NAME', 'postgres')
)
cursor = conn.cursor()

query = """
SELECT MAX(date) FROM "stock-trading".daily_bars;
"""
cursor.execute(query)
max_date = cursor.fetchone()[0]
print(f"MAX DATE IN DB: {max_date}")
cursor.close()
conn.close()
