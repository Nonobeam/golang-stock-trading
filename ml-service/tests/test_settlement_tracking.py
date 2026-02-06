"""
Tests for Settlement Tracking System

Comprehensive tests for T+2 settlement risk management.
"""

import pytest
from datetime import datetime, timedelta
from position_sizing.locked_risk import LockedRiskCalculator, get_user_locked_risk_threshold


class TestLockedRiskCalculator:
    """Test locked risk calculation logic."""

    def test_calculate_locked_risk_hose(self):
        """Test locked risk calculation for HOSE exchange."""
        calc = LockedRiskCalculator(None)  # Don't need DB for this

        shares = 100
        price = 50000
        exchange = 'HOSE'

        locked_risk = calc.calculate_locked_risk(shares, price, exchange)

        # Expected: 100 * 50000 * 0.20 = 1,000,000
        assert locked_risk == 1000000.0

    def test_calculate_locked_risk_hnx(self):
        """Test locked risk calculation for HNX exchange."""
        calc = LockedRiskCalculator(None)

        shares = 100
        price = 30000
        exchange = 'HNX'

        locked_risk = calc.calculate_locked_risk(shares, price, exchange)

        # Expected: 100 * 30000 * 0.30 = 900,000
        assert locked_risk == 900000.0

    def test_calculate_locked_risk_upcom(self):
        """Test locked risk calculation for UPCOM exchange."""
        calc = LockedRiskCalculator(None)

        shares = 100
        price = 20000
        exchange = 'UPCOM'

        locked_risk = calc.calculate_locked_risk(shares, price, exchange)

        # Expected: 100 * 20000 * 0.40 = 800,000
        assert locked_risk == 800000.0

    def test_exchange_risk_multipliers(self):
        """Test correct risk multipliers for each exchange."""
        calc = LockedRiskCalculator(None)

        assert calc.get_exchange_risk_multiplier('HOSE') == 0.20
        assert calc.get_exchange_risk_multiplier('HNX') == 0.30
        assert calc.get_exchange_risk_multiplier('UPCOM') == 0.40
        assert calc.get_exchange_risk_multiplier('UNKNOWN') == 0.20  # Default to HOSE


class TestSettlementDateCalculations:
    """Test settlement date and status calculations."""

    def test_settlement_status_t0(self):
        """Test settlement status on purchase day (T+0)."""
        from internal.vn import CalculateSettlementStatusFromDates

        purchase_date = datetime(2026, 2, 10)  # Monday
        current_date = datetime(2026, 2, 10)   # Same day

        status = CalculateSettlementStatusFromDates(purchase_date, current_date)
        assert status == "LOCKED_T0"

    def test_settlement_status_t1(self):
        """Test settlement status one trading day after purchase (T+1)."""
        from internal.vn import CalculateSettlementStatusFromDates

        purchase_date = datetime(2026, 2, 10)  # Monday
        current_date = datetime(2026, 2, 11)   # Tuesday (T+1)

        status = CalculateSettlementStatusFromDates(purchase_date, current_date)
        assert status == "LOCKED_T1"

    def test_settlement_status_t2(self):
        """Test settlement status two trading days after purchase (T+2)."""
        from internal.vn import CalculateSettlementStatusFromDates

        purchase_date = datetime(2026, 2, 10)  # Monday
        current_date = datetime(2026, 2, 12)   # Wednesday (T+2)

        status = CalculateSettlementStatusFromDates(purchase_date, current_date)
        assert status == "LOCKED_T2"

    def test_settlement_status_liquid(self):
        """Test settlement status three+ trading days after purchase (LIQUID)."""
        from internal.vn import CalculateSettlementStatusFromDates

        purchase_date = datetime(2026, 2, 10)  # Monday
        current_date = datetime(2026, 2, 13)   # Thursday (T+3)

        status = CalculateSettlementStatusFromDates(purchase_date, current_date)
        assert status == "LIQUID"

    def test_weekend_skip(self):
        """Test that weekends are skipped in trading day count."""
        from internal.vn import CountTradingDaysBetween

        # Friday to Monday
        friday = datetime(2026, 2, 13)
        monday = datetime(2026, 2, 16)

        # Should be 1 trading day (Monday only)
        days = CountTradingDaysBetween(friday, monday)
        assert days == 1

    def test_days_until_liquid(self):
        """Test calculation of days until position becomes liquid."""
        from internal.vn import GetDaysUntilLiquid

        purchase_date = datetime(2026, 2, 10)  # Monday

        # T+0: 3 days until liquid
        current_t0 = datetime(2026, 2, 10)
        assert GetDaysUntilLiquid(purchase_date, current_t0) == 3

        # T+1: 2 days until liquid
        current_t1 = datetime(2026, 2, 11)
        assert GetDaysUntilLiquid(purchase_date, current_t1) == 2

        # T+2: 1 day until liquid
        current_t2 = datetime(2026, 2, 12)
        assert GetDaysUntilLiquid(purchase_date, current_t2) == 1

        # T+3: 0 days (already liquid)
        current_liquid = datetime(2026, 2, 13)
        assert GetDaysUntilLiquid(purchase_date, current_liquid) == 0


class TestEntryDayRestrictions:
    """Test entry day restrictions (Thursday/Friday)."""

    def test_monday_entry_full_size(self):
        """Test full position size allowed on Monday."""
        from internal.vn import GetEntryDayMultiplier

        monday = datetime(2026, 2, 9)  # Monday
        multiplier = GetEntryDayMultiplier(monday)

        assert multiplier == 1.0

    def test_thursday_entry_half_size(self):
        """Test 50% position size on Thursday."""
        from internal.vn import GetEntryDayMultiplier

        thursday = datetime(2026, 2, 12)  # Thursday
        multiplier = GetEntryDayMultiplier(thursday)

        assert multiplier == 0.5

    def test_friday_entry_half_size(self):
        """Test 50% position size on Friday."""
        from internal.vn import GetEntryDayMultiplier

        friday = datetime(2026, 2, 13)  # Friday
        multiplier = GetEntryDayMultiplier(friday)

        assert multiplier == 0.5


class TestHolidayHandling:
    """Test Vietnamese holiday handling."""

    def test_tet_holiday_2026(self):
        """Test Tet holiday recognition for 2026."""
        from internal.vn import IsVietnameseHoliday

        # Tet 2026: Feb 16-21
        tet_day1 = datetime(2026, 2, 17)
        assert IsVietnameseHoliday(tet_day1) == True

        tet_day3 = datetime(2026, 2, 19)
        assert IsVietnameseHoliday(tet_day3) == True

    def test_hung_kings_day_2026(self):
        """Test Hung Kings' Day recognition for 2026."""
        from internal.vn import IsVietnameseHoliday

        hung_kings = datetime(2026, 3, 26)
        assert IsVietnameseHoliday(hung_kings) == True

    def test_national_day(self):
        """Test National Day recognition."""
        from internal.vn import IsVietnameseHoliday

        national_day = datetime(2026, 9, 2)
        assert IsVietnameseHoliday(national_day) == True

    def test_regular_day_not_holiday(self):
        """Test that regular days are not holidays."""
        from internal.vn import IsVietnameseHoliday

        regular_day = datetime(2026, 3, 15)
        assert IsVietnameseHoliday(regular_day) == False


class TestLockedRiskBudgetValidation:
    """Test locked risk budget validation."""

    @pytest.fixture
    def mock_db(self):
        """Mock database connection for testing."""
        # In real tests, use a test database or mock
        return None

    def test_budget_exceeded(self, mock_db):
        """Test that purchases are rejected when budget exceeded."""
        # This would require a test database with sample data
        # Placeholder for integration test
        pass

    def test_budget_warning_at_80_percent(self, mock_db):
        """Test warning when approaching 80% of threshold."""
        # Placeholder for integration test
        pass

    def test_max_shares_calculation(self, mock_db):
        """Test calculation of maximum affordable shares."""
        calc = LockedRiskCalculator(mock_db)

        # With available budget of 2,000,000 VND
        # HOSE exchange (20% multiplier)
        # Price 50,000 VND
        # Max capital = 2,000,000 / 0.20 = 10,000,000
        # Max shares = 10,000,000 / 50,000 = 200
        # Rounded to lot size 100: 200 shares

        # This would need actual DB testing
        pass


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
