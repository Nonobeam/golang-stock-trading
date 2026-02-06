"""
Data Migration Script for Settlement Tracking

Backfills settlement data for existing positions in the database.
Sets all existing positions to LIQUID status (assume already settled).
"""

import psycopg3
from datetime import datetime, timedelta
import logging
import sys

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("settlement_migration")


class SettlementMigration:
    """Handles backfilling settlement data for existing positions."""

    def __init__(self, db_conn: psycopg3.Connection):
        """
        Initialize migration.

        Args:
            db_conn: Database connection
        """
        self.db = db_conn

    def infer_exchange_from_ticker(self, ticker: str) -> str:
        """
        Infer exchange from ticker symbol.

        This is a simplified heuristic. In production, use actual exchange lookup.

        Args:
            ticker: Stock ticker

        Returns:
            Exchange code (HOSE, HNX, or UPCOM)
        """
        # Common patterns:
        # HOSE: VNM, HPG, VIC, VHM, etc. (blue chips)
        # HNX: ACB, SHB, PVS, etc.
        # UPCOM: smaller caps

        # For now, default to HOSE (most positions are likely HOSE)
        # TODO: Replace with actual exchange lookup
        return 'HOSE'

    def backfill_existing_positions(self, dry_run: bool = True) -> dict:
        """
        Backfill settlement data for all existing open positions.

        Sets all existing positions to LIQUID status and calculates settlement dates.

        Args:
            dry_run: If True, only shows what would be updated without making changes

        Returns:
            Dict with migration statistics
        """
        stats = {
            'total_positions': 0,
            'updated': 0,
            'skipped': 0,
            'errors': []
        }

        with self.db.cursor() as cur:
            # Find all open positions without settlement_status
            cur.execute("""
                SELECT id, user_id, symbol, entry_date, quantity, entry_price
                FROM positions
                WHERE is_closed = FALSE
                  AND (settlement_status IS NULL OR settlement_status = '')
                ORDER BY entry_date DESC
            """)

            positions = cur.fetchall()
            stats['total_positions'] = len(positions)

            logger.info(f"Found {stats['total_positions']} positions to migrate")

            if dry_run:
                logger.info("DRY RUN MODE - No changes will be made")

            for pos in positions:
                pos_id, user_id, symbol, entry_date, quantity, entry_price = pos

                try:
                    # Infer exchange from ticker
                    exchange = self.infer_exchange_from_ticker(symbol)

                    # Set purchase_date to entry_date
                    purchase_date = entry_date

                    # Calculate settlement_date (T+2) - conservatively assume 3 calendar days
                    settlement_date = entry_date + timedelta(days=3)

                    # Calculate can_sell_date (T+3) - conservatively assume 4 calendar days
                    can_sell_date = entry_date + timedelta(days=4)

                    # All existing positions are assumed LIQUID (already settled)
                    settlement_status = 'LIQUID'

                    # Locked capital = 0 (all liquid)
                    locked_capital = 0.0

                    # Liquid capital = total position value
                    liquid_capital = quantity * entry_price

                    if dry_run:
                        logger.info(
                            f"  Would update {symbol} (ID: {pos_id}): "
                            f"status={settlement_status}, exchange={exchange}, "
                            f"liquid_capital={liquid_capital:,.0f}"
                        )
                    else:
                        # Update position with settlement data
                        cur.execute("""
                            UPDATE positions
                            SET settlement_status = %s,
                                purchase_date = %s,
                                settlement_date = %s,
                                can_sell_date = %s,
                                locked_capital = %s,
                                liquid_capital = %s,
                                exchange = %s,
                                updated_at = NOW()
                            WHERE id = %s
                        """, (
                            settlement_status,
                            purchase_date,
                            settlement_date,
                            can_sell_date,
                            locked_capital,
                            liquid_capital,
                            exchange,
                            pos_id
                        ))

                        stats['updated'] += 1

                        if stats['updated'] % 10 == 0:
                            logger.info(f"  Updated {stats['updated']} positions...")

                except Exception as e:
                    error_msg = f"Error updating position {pos_id} ({symbol}): {str(e)}"
                    logger.error(error_msg)
                    stats['errors'].append(error_msg)
                    stats['skipped'] += 1

            if not dry_run:
                self.db.commit()
                logger.info("Changes committed to database")

        return stats

    def validate_migration(self) -> dict:
        """
        Validate that migration was successful.

        Returns:
            Dict with validation results
        """
        issues = []

        with self.db.cursor() as cur:
            # Check for open positions still missing settlement_status
            cur.execute("""
                SELECT COUNT(*)
                FROM positions
                WHERE is_closed = FALSE
                  AND (settlement_status IS NULL OR settlement_status = '')
            """)
            missing_status = cur.fetchone()[0]
            if missing_status > 0:
                issues.append(f"{missing_status} positions still have NULL settlement_status")

            # Check for positions missing exchange
            cur.execute("""
                SELECT COUNT(*)
                FROM positions
                WHERE is_closed = FALSE
                  AND settlement_status IS NOT NULL
                  AND (exchange IS NULL OR exchange = '')
            """)
            missing_exchange = cur.fetchone()[0]
            if missing_exchange > 0:
                issues.append(f"{missing_exchange} positions missing exchange")

            # Check for LIQUID positions with locked_capital > 0
            cur.execute("""
                SELECT COUNT(*)
                FROM positions
                WHERE is_closed = FALSE
                  AND settlement_status = 'LIQUID'
                  AND locked_capital > 0
            """)
            invalid_locked = cur.fetchone()[0]
            if invalid_locked > 0:
                issues.append(f"{invalid_locked} LIQUID positions have locked_capital > 0")

            # Get total positions with settlement data
            cur.execute("""
                SELECT COUNT(*)
                FROM positions
                WHERE is_closed = FALSE
                  AND settlement_status IS NOT NULL
            """)
            total_with_settlement = cur.fetchone()[0]

        return {
            'valid': len(issues) == 0,
            'issues': issues,
            'total_positions_with_settlement': total_with_settlement
        }

    def print_summary(self, stats: dict, validation: dict):
        """
        Print migration summary.

        Args:
            stats: Migration statistics
            validation: Validation results
        """
        print("\n" + "=" * 60)
        print("SETTLEMENT DATA MIGRATION SUMMARY")
        print("=" * 60)
        print(f"Total positions found: {stats['total_positions']}")
        print(f"Successfully updated: {stats['updated']}")
        print(f"Skipped (errors): {stats['skipped']}")

        if stats['errors']:
            print(f"\nErrors encountered: {len(stats['errors'])}")
            for error in stats['errors'][:5]:  # Show first 5 errors
                print(f"  - {error}")
            if len(stats['errors']) > 5:
                print(f"  ... and {len(stats['errors']) - 5} more")

        print("\n" + "-" * 60)
        print("VALIDATION RESULTS")
        print("-" * 60)
        print(f"Status: {'✅ VALID' if validation['valid'] else '❌ ISSUES FOUND'}")
        print(f"Positions with settlement data: {validation['total_positions_with_settlement']}")

        if validation['issues']:
            print("\nIssues:")
            for issue in validation['issues']:
                print(f"  - {issue}")
        else:
            print("\n✅ No issues detected")

        print("=" * 60)


def main():
    """Main migration script."""
    import argparse
    from db.connection import get_connection

    parser = argparse.ArgumentParser(description='Migrate existing positions to settlement tracking')
    parser.add_argument('--dry-run', action='store_true', help='Show what would be done without making changes')
    parser.add_argument('--validate-only', action='store_true', help='Only run validation, no migration')
    args = parser.parse_args()

    logger.info("Starting settlement tracking migration")

    conn = get_connection()
    migration = SettlementMigration(conn)

    if args.validate_only:
        logger.info("Running validation only...")
        validation = migration.validate_migration()
        migration.print_summary({'total_positions': 0, 'updated': 0, 'skipped': 0, 'errors': []}, validation)
        conn.close()
        return

    # Run migration
    stats = migration.backfill_existing_positions(dry_run=args.dry_run)

    # Validate results
    if not args.dry_run:
        validation = migration.validate_migration()
    else:
        validation = {'valid': True, 'issues': [], 'total_positions_with_settlement': 0}

    # Print summary
    migration.print_summary(stats, validation)

    conn.close()

    if not args.dry_run and validation['valid']:
        logger.info("✅ Migration completed successfully!")
        sys.exit(0)
    elif not args.dry_run:
        logger.error("❌ Migration completed with issues")
        sys.exit(1)
    else:
        logger.info("Dry run completed. Use --dry-run=false to apply changes.")
        sys.exit(0)


if __name__ == "__main__":
    main()
