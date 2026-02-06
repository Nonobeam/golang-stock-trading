#!/usr/bin/env python3
"""
Backfill Position Entries Migration Script

This script creates synthetic entries in the position_entries table for all
existing positions, preserving their current state as single-entry positions.

This ensures:
1. All positions have at least one entry in position_entries
2. Average cost calculations match existing entry_price
3. Historical P&L remains unchanged
4. Data model consistency for future multi-entry positions

Usage:
    python scripts/backfill_position_entries.py [--dry-run] [--verbose]
"""

import sys
import logging
from datetime import datetime
from decimal import Decimal
from typing import Dict, List, Tuple

# Add parent directory and ml-service to path for imports
import os
sys.path.insert(0, '.')
sys.path.insert(0, os.path.join(os.getcwd(), 'ml-service'))

from db.connection import get_connection

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def fetch_existing_positions(conn) -> List[Dict]:
    """
    Fetch all positions that need backfilling.
    
    Returns positions that don't have corresponding entries in position_entries.
    """
    query = """
        SELECT 
            p.id,
            p.user_id,
            p.symbol as ticker,
            p.entry_date,
            p.entry_price,
            p.quantity,
            p.is_closed
        FROM positions p
        LEFT JOIN position_entries pe 
            ON p.user_id = pe.user_id 
            AND p.symbol = pe.ticker
        WHERE pe.entry_id IS NULL
        ORDER BY p.user_id, p.symbol, p.entry_date
    """
    
    with conn.cursor() as cur:
        cur.execute(query)
        columns = [desc[0] for desc in cur.description]
        positions = []
        for row in cur.fetchall():
            positions.append(dict(zip(columns, row)))
    
    return positions


def calculate_entry_fee(shares: int, price: Decimal) -> Decimal:
    """Calculate entry fee (0.15% of purchase value)."""
    purchase_value = Decimal(shares) * price
    entry_fee = purchase_value * Decimal('0.0015')
    return entry_fee.quantize(Decimal('0.01'))


def create_position_entry(conn, position: Dict, dry_run: bool = False) -> Tuple[bool, str]:
    """
    Create a synthetic entry for an existing position.
    
    Args:
        conn: Database connection
        position: Position data dictionary
        dry_run: If True, don't actually insert, just validate
        
    Returns:
        Tuple of (success, message)
    """
    user_id = position['user_id']
    ticker = position['ticker']
    entry_date = position['entry_date']
    entry_price = position['entry_price']
    shares = position['quantity']
    
    # Calculate entry fee
    entry_fee = calculate_entry_fee(shares, entry_price)
    
    # Determine transaction type (all existing positions are treated as BUY_NEW)
    transaction_type = 'BUY_NEW'
    
    if dry_run:
        logger.info(
            f"[DRY RUN] Would create entry: user={user_id}, ticker={ticker}, "
            f"date={entry_date}, price={entry_price}, shares={shares}, fee={entry_fee}"
        )
        return True, "Dry run - entry not created"
    
    try:
        with conn.cursor() as cur:
            insert_query = """
                INSERT INTO position_entries (
                    user_id, ticker, entry_date, entry_price,
                    shares_purchased, entry_fee_paid, transaction_type
                ) VALUES (%s, %s, %s, %s, %s, %s, %s)
                RETURNING entry_id
            """
            cur.execute(insert_query, (
                user_id, ticker, entry_date, entry_price,
                shares, entry_fee, transaction_type
            ))
            entry_id = cur.fetchone()[0]
            
        logger.info(
            f"Created entry {entry_id}: user={user_id}, ticker={ticker}, "
            f"shares={shares}, price={entry_price}"
        )
        return True, f"Entry created: {entry_id}"
        
    except Exception as e:
        logger.error(f"Failed to create entry for {ticker} (user {user_id}): {e}")
        return False, str(e)


def update_position_aggregates(conn, position: Dict, dry_run: bool = False) -> bool:
    """
    Update aggregate fields in positions table after entry creation.
    
    This should match what the trigger does, but we do it explicitly
    to ensure consistency during migration.
    """
    if dry_run:
        return True
    
    user_id = position['user_id']
    ticker = position['ticker']
    entry_fee = calculate_entry_fee(position['quantity'], position['entry_price'])
    
    try:
        with conn.cursor() as cur:
            update_query = """
                UPDATE positions
                SET total_entries = 1,
                    total_fees_paid = %s,
                    first_entry_date = entry_date,
                    last_entry_date = entry_date,
                    updated_at = CURRENT_TIMESTAMP
                WHERE user_id = %s AND symbol = %s AND is_closed = FALSE
            """
            cur.execute(update_query, (entry_fee, user_id, ticker))
            
        logger.debug(f"Updated aggregates for {ticker} (user {user_id})")
        return True
        
    except Exception as e:
        logger.error(f"Failed to update aggregates for {ticker}: {e}")
        return False


def verify_average_cost(conn) -> List[Tuple[str, Decimal, Decimal]]:
    """
    Verify that calculated average cost matches stored entry_price.
    
    Returns list of (ticker, stored_price, calculated_avg) for any mismatches.
    """
    query = """
        SELECT 
            p.symbol,
            p.entry_price as stored_price,
            CASE 
                WHEN SUM(pe.shares_purchased) > 0 
                THEN SUM(pe.shares_purchased * pe.entry_price) / SUM(pe.shares_purchased)
                ELSE 0
            END as calculated_avg
        FROM positions p
        INNER JOIN position_entries pe 
            ON p.user_id = pe.user_id 
            AND p.symbol = pe.ticker
        WHERE p.is_closed = FALSE
        GROUP BY p.symbol, p.entry_price
        HAVING ABS(p.entry_price - 
            (SUM(pe.shares_purchased * pe.entry_price) / SUM(pe.shares_purchased))
        ) > 0.01
    """
    
    with conn.cursor() as cur:
        cur.execute(query)
        mismatches = cur.fetchall()
    
    return mismatches


def main():
    """Main script execution."""
    import argparse
    
    parser = argparse.ArgumentParser(description='Backfill position_entries table from existing positions')
    parser.add_argument('--dry-run', action='store_true', help='Show what would be done without making changes')
    parser.add_argument('--verbose', action='store_true', help='Enable verbose logging')
    args = parser.parse_args()
    
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)
    
    logger.info("=" * 60)
    logger.info("Position Entries Backfill Script")
    logger.info("=" * 60)
    
    if args.dry_run:
        logger.info("Running in DRY RUN mode - no changes will be made")
    
    # Connect to database
    try:
        conn = get_connection()
        conn.autocommit = False  # Use transaction
        logger.info("Connected to database")
    except Exception as e:
        logger.error(f"Failed to connect to database: {e}")
        return 1
    
    try:
        # Fetch positions needing backfill
        logger.info("Fetching positions to backfill...")
        positions = fetch_existing_positions(conn)
        logger.info(f"Found {len(positions)} positions to backfill")
        
        if len(positions) == 0:
            logger.info("No positions need backfilling. Migration may have already run.")
            return 0
        
        # Process each position
        success_count = 0
        error_count = 0
        
        for i, position in enumerate(positions, 1):
            logger.info(f"\nProcessing position {i}/{len(positions)}: {position['ticker']}")
            
            # Create entry
            success, message = create_position_entry(conn, position, args.dry_run)
            if success:
                # Update aggregates
                if update_position_aggregates(conn, position, args.dry_run):
                    success_count += 1
                else:
                    error_count += 1
            else:
                error_count += 1
        
        # Verification step
        if not args.dry_run:
            logger.info("\nVerifying average cost calculations...")
            mismatches = verify_average_cost(conn)
            
            if mismatches:
                logger.warning(f"Found {len(mismatches)} positions with average cost mismatches:")
                for ticker, stored, calculated in mismatches:
                    logger.warning(f"  {ticker}: stored={stored}, calculated={calculated}")
                logger.error("Verification failed! Rolling back transaction.")
                conn.rollback()
                return 1
            else:
                logger.info("✓ All average cost calculations verified")
        
        # Summary
        logger.info("\n" + "=" * 60)
        logger.info("Backfill Summary")
        logger.info("=" * 60)
        logger.info(f"Total positions: {len(positions)}")
        logger.info(f"Successfully processed: {success_count}")
        logger.info(f"Errors: {error_count}")
        
        if args.dry_run:
            logger.info("\nDRY RUN complete - no changes were made")
            conn.rollback()
        else:
            if error_count == 0:
                conn.commit()
                logger.info("\n✓ Transaction committed successfully")
            else:
                conn.rollback()
                logger.error("\n✗ Errors encountered - transaction rolled back")
                return 1
        
        return 0
        
    except Exception as e:
        logger.error(f"Unexpected error during backfill: {e}", exc_info=True)
        conn.rollback()
        return 1
        
    finally:
        conn.close()
        logger.info("Database connection closed")


if __name__ == '__main__':
    sys.exit(main())
