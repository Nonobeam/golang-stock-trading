"""
Settlement Monitoring and Validation

Monitors daily settlement status updates and validates settlement tracking accuracy.
"""

import psycopg3
from typing import Dict, List
from datetime import datetime, timedelta
import logging

logger = logging.getLogger("settlement_monitor")


class SettlementMonitor:
    """Monitors settlement tracking and generates daily reports."""

    def __init__(self, db_conn: psycopg3.Connection):
        """
        Initialize settlement monitor.

        Args:
            db_conn: Database connection
        """
        self.db = db_conn

    def get_settlement_distribution(self) -> Dict[str, int]:
        """
        Get distribution of positions by settlement status.

        Returns:
            Dict with status as key and count as value
        """
        with self.db.cursor() as cur:
            cur.execute("""
                SELECT settlement_status, COUNT(*)
                FROM positions
                WHERE is_closed = FALSE
                  AND settlement_status IS NOT NULL
                GROUP BY settlement_status
                ORDER BY settlement_status
            """)

            results = cur.fetchall()
            return {status: count for status, count in results}

    def get_locked_risk_utilization(self) -> List[Dict]:
        """
        Get locked risk utilization by user.

        Returns:
            List of dicts with user_id, total_locked_risk, threshold, utilization
        """
        with self.db.cursor() as cur:
            cur.execute("""
                SELECT
                    p.user_id,
                    COALESCE(SUM(
                        CASE
                            WHEN p.exchange = 'HOSE' THEN p.locked_capital * 0.20
                            WHEN p.exchange = 'HNX' THEN p.locked_capital * 0.30
                            WHEN p.exchange = 'UPCOM' THEN p.locked_capital * 0.40
                            ELSE p.locked_capital * 0.20
                        END
                    ), 0) as total_locked_risk,
                    COALESCE(uc.locked_risk_threshold, 0.10) as threshold,
                    COALESCE(uc.initial_capital, 100000000) as account_value
                FROM positions p
                LEFT JOIN user_config uc ON p.user_id = uc.user_id
                WHERE p.is_closed = FALSE
                  AND p.settlement_status IN ('LOCKED_T0', 'LOCKED_T1', 'LOCKED_T2')
                GROUP BY p.user_id, uc.locked_risk_threshold, uc.initial_capital
            """)

            results = []
            for row in cur.fetchall():
                user_id, total_locked_risk, threshold, account_value = row
                max_allowed = account_value * threshold
                utilization = (total_locked_risk / max_allowed * 100) if max_allowed > 0 else 0

                results.append({
                    'user_id': user_id,
                    'total_locked_risk': total_locked_risk,
                    'threshold': threshold,
                    'account_value': account_value,
                    'max_allowed': max_allowed,
                    'utilization_percent': utilization
                })

            return results

    def get_theoretical_stop_breach_summary(self, days: int = 7) -> List[Dict]:
        """
        Get summary of theoretical stop breaches in the last N days.

        Args:
            days: Number of days to look back

        Returns:
            List of breach records
        """
        since_date = datetime.now() - timedelta(days=days)

        with self.db.cursor() as cur:
            cur.execute("""
                SELECT
                    tsb.position_id,
                    p.symbol,
                    tsb.breach_date,
                    tsb.stop_price,
                    tsb.actual_price,
                    tsb.settlement_status,
                    tsb.days_until_executable
                FROM theoretical_stop_breaches tsb
                JOIN positions p ON tsb.position_id = p.id
                WHERE tsb.breach_date >= %s
                ORDER BY tsb.breach_date DESC
            """, (since_date,))

            results = []
            for row in cur.fetchall():
                results.append({
                    'position_id': row[0],
                    'symbol': row[1],
                    'breach_date': row[2],
                    'stop_price': row[3],
                    'actual_price': row[4],
                    'settlement_status': row[5],
                    'days_until_executable': row[6]
                })

            return results

    def validate_settlement_transitions(self) -> Dict:
        """
        Validate that settlement status transitions are accurate.

        Returns:
            Dict with validation results
        """
        issues = []
        warnings = []

        with self.db.cursor() as cur:
            # Check for positions with null settlement_status but recent purchases
            cur.execute("""
                SELECT id, symbol, purchase_date
                FROM positions
                WHERE is_closed = FALSE
                  AND settlement_status IS NULL
                  AND purchase_date IS NOT NULL
                  AND purchase_date > NOW() - INTERVAL '30 days'
            """)

            null_status = cur.fetchall()
            if null_status:
                issues.append(f"Found {len(null_status)} positions with NULL settlement_status but recent purchase dates")

            # Check for LIQUID positions with locked_capital > 0
            cur.execute("""
                SELECT id, symbol, settlement_status, locked_capital
                FROM positions
                WHERE is_closed = FALSE
                  AND settlement_status = 'LIQUID'
                  AND locked_capital > 0
            """)

            liquid_with_locked = cur.fetchall()
            if liquid_with_locked:
                warnings.append(f"Found {len(liquid_with_locked)} LIQUID positions with locked_capital > 0")

            # Check for LOCKED positions with liquid_capital > 0
            cur.execute("""
                SELECT id, symbol, settlement_status, liquid_capital
                FROM positions
                WHERE is_closed = FALSE
                  AND settlement_status IN ('LOCKED_T0', 'LOCKED_T1', 'LOCKED_T2')
                  AND liquid_capital > 0
            """)

            locked_with_liquid = cur.fetchall()
            if locked_with_liquid:
                warnings.append(f"Found {len(locked_with_liquid)} LOCKED positions with liquid_capital > 0")

            # Check for positions stuck in LOCKED status beyond expected date
            cur.execute("""
                SELECT id, symbol, settlement_status, purchase_date, can_sell_date
                FROM positions
                WHERE is_closed = FALSE
                  AND settlement_status IN ('LOCKED_T0', 'LOCKED_T1', 'LOCKED_T2')
                  AND can_sell_date < NOW()
            """)

            stuck_locked = cur.fetchall()
            if stuck_locked:
                issues.append(f"Found {len(stuck_locked)} positions stuck in LOCKED status past can_sell_date")

        return {
            'valid': len(issues) == 0,
            'issues': issues,
            'warnings': warnings,
            'checked_at': datetime.now()
        }

    def generate_daily_report(self) -> str:
        """
        Generate daily settlement monitoring report.

        Returns:
            Formatted report string
        """
        report = []
        report.append("=" * 60)
        report.append("SETTLEMENT MONITORING DAILY REPORT")
        report.append(f"Generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
        report.append("=" * 60)
        report.append("")

        # Settlement distribution
        distribution = self.get_settlement_distribution()
        report.append("SETTLEMENT STATUS DISTRIBUTION")
        report.append("-" * 60)
        for status, count in distribution.items():
            report.append(f"  {status}: {count} position(s)")
        report.append("")

        # Locked risk utilization
        utilization = self.get_locked_risk_utilization()
        report.append("LOCKED RISK UTILIZATION BY USER")
        report.append("-" * 60)
        for user_data in utilization:
            report.append(f"  User ID: {user_data['user_id']}")
            report.append(f"    Total Locked Risk: {user_data['total_locked_risk']:,.0f} VND")
            report.append(f"    Max Allowed: {user_data['max_allowed']:,.0f} VND")
            report.append(f"    Utilization: {user_data['utilization_percent']:.1f}%")
            if user_data['utilization_percent'] > 80:
                report.append(f"    ⚠️  WARNING: Approaching threshold!")
            report.append("")

        # Theoretical stop breaches
        breaches = self.get_theoretical_stop_breach_summary(days=7)
        report.append(f"THEORETICAL STOP BREACHES (Last 7 Days)")
        report.append("-" * 60)
        if breaches:
            for breach in breaches:
                report.append(f"  {breach['symbol']} - {breach['breach_date'].strftime('%Y-%m-%d')}")
                report.append(f"    Stop: {breach['stop_price']:,.0f} VND, Actual: {breach['actual_price']:,.0f} VND")
                report.append(f"    Status: {breach['settlement_status']}, Days to executable: {breach['days_until_executable']}")
                report.append("")
        else:
            report.append("  No theoretical stop breaches in the last 7 days")
            report.append("")

        # Validation
        validation = self.validate_settlement_transitions()
        report.append("SETTLEMENT VALIDATION")
        report.append("-" * 60)
        report.append(f"  Status: {'VALID' if validation['valid'] else 'ISSUES FOUND'}")
        if validation['issues']:
            report.append("  Issues:")
            for issue in validation['issues']:
                report.append(f"    - {issue}")
        if validation['warnings']:
            report.append("  Warnings:")
            for warning in validation['warnings']:
                report.append(f"    - {warning}")
        if not validation['issues'] and not validation['warnings']:
            report.append("  No issues or warnings detected")
        report.append("")

        report.append("=" * 60)

        return "\n".join(report)


if __name__ == "__main__":
    # Example usage
    from db.connection import get_connection

    conn = get_connection()
    monitor = SettlementMonitor(conn)

    # Generate and print daily report
    report = monitor.generate_daily_report()
    print(report)

    conn.close()
