import pytest
from unittest.mock import MagicMock
from position_manager.manager import PositionManager

class TestPositionManagerAverageCost:
    @pytest.fixture
    def mock_db_connection(self):
        conn = MagicMock()
        cursor = MagicMock()
        # Ensure context manager returns the cursor mock
        conn.cursor.return_value.__enter__.return_value = cursor
        return conn

    @pytest.fixture
    def manager(self, mock_db_connection):
        return PositionManager(db_connection=mock_db_connection)

    def test_calculate_average_cost_single_entry(self, manager, mock_db_connection):
        cursor = mock_db_connection.cursor.return_value.__enter__.return_value
        cursor.fetchone.return_value = (10000.0, 100)
        avg_cost = manager.calculate_average_cost("TEST")
        assert avg_cost == 100.0

    def test_calculate_average_cost_multiple_entries(self, manager, mock_db_connection):
        cursor = mock_db_connection.cursor.return_value.__enter__.return_value
        cursor.fetchone.return_value = (15500.0, 150)
        avg_cost = manager.calculate_average_cost("TEST")
        assert avg_cost == pytest.approx(103.33333333333333)

    def test_calculate_average_cost_no_entries(self, manager, mock_db_connection):
        cursor = mock_db_connection.cursor.return_value.__enter__.return_value
        cursor.fetchone.return_value = None
        avg_cost = manager.calculate_average_cost("TEST")
        assert avg_cost is None


class TestPositionManagerCapacity:
    @pytest.fixture
    def mock_db_connection(self):
        conn = MagicMock()
        cursor = MagicMock()
        conn.cursor.return_value.__enter__.return_value = cursor
        return conn

    @pytest.fixture
    def manager(self, mock_db_connection):
        return PositionManager(db_connection=mock_db_connection)

    def test_check_capacity_allocation_limit(self, manager, mock_db_connection):
        # account_value = 100,000 -> Max Alloc 20,000
        cursor = mock_db_connection.cursor.return_value.__enter__.return_value
        
        # 1. get_position -> None
        # 2. avg_volume -> 1,000,000
        cursor.fetchone.side_effect = [None, (1000000.0,)]
        
        cap = manager.check_buying_capacity("TEST", 100.0, 100000.0)
        
        # 20,000 / 100 = 200 shares
        assert cap['max_buyable_shares'] == 200
        assert cap['at_limit'] is False

    def test_check_capacity_liquidity_limit(self, manager, mock_db_connection):
        cursor = mock_db_connection.cursor.return_value.__enter__.return_value
        
        # 1. get_position -> None
        # 2. avg_volume -> 10,000
        cursor.fetchone.side_effect = [None, (10000.0,)]
        
        cap = manager.check_buying_capacity("TEST", 100.0, 1000000.0)
        
        # 1% of 10,000 = 100 shares
        assert cap['max_buyable_shares'] == 100

    def test_check_capacity_at_limit(self, manager, mock_db_connection):
        cursor = mock_db_connection.cursor.return_value.__enter__.return_value
        
        # 1. get_position -> Need 18 elements to match query columns
        # Indices: 4=entry_price, 5=qty; 6=stop_loss; 10=signal_type; ...
        # (id, user_id, symbol, date, price, qty, sl, t1, t2, t3, sig, score, notes, pnl, pnl%, rmult, created, updated)
        pos_row = (
            'id', 1, 'TEST', '2023-01-01', 
            100.0, 200, 95.0, 
            110.0, 120.0, 130.0, 
            'MANUAL', 0, 'notes', 
            None, None, None, 
            None, None
        )
        
        # 2. first entry price
        entry_row = (100.0,)
        
        # 3. avg volume
        val_row = (100000.0, 0, 0, 0)
        
        cursor.fetchone.side_effect = [pos_row, entry_row, val_row]
        
        cap = manager.check_buying_capacity("TEST", 100.0, 100000.0)
        
class TestPositionManagerPartialExit:
    @pytest.fixture
    def mock_db_connection(self):
        conn = MagicMock()
        cursor = MagicMock()
        conn.cursor.return_value.__enter__.return_value = cursor
        return conn

    @pytest.fixture
    def manager(self, mock_db_connection):
        return PositionManager(db_connection=mock_db_connection)

    def test_partial_exit_logic(self, manager, mock_db_connection):
        cursor = mock_db_connection.cursor.return_value.__enter__.return_value
        
        # Mock Position:
        # Ticker: TEST
        # UserID: 1
        # Avg Cost: 100.0
        # Qty: 200
        # Total Fees Paid: 300.0
        # (symbol, user_id, entry_price, quantity, total_fees_paid)
        cursor.fetchone.return_value = ('TEST', 1, 100.0, 200, 300.0)
        
        # Action: Sell 100 shares @ 120.0
        result = manager.partial_exit_position("uuid-123", 100, 120.0, "2023-01-02")
        
        # Validations
        # Shares: 200 -> 100
        assert result['remaining_shares'] == 100
        assert result['shares_sold'] == 100
        
        # Fees
        # Proportional Entry Fee: 300 * (100/200) = 150.0
        assert result['proportional_entry_fees'] == pytest.approx(150.0)
        # Exit Fee: 100 * 120 * 0.0025 = 12000 * 0.0025 = 30.0
        assert result['exit_fee'] == pytest.approx(30.0)
        
        # P&L
        # Proceeds: 12000
        # Cost Basis: 100 * 100 = 10000
        # Total Fees: 150 + 30 = 180
        # Realized PnL: 12000 - 10000 - 180 = 1820.0
        assert result['realized_pnl'] == pytest.approx(1820.0)
        
        # DB Update Verification
        cursor.execute.assert_called()
        # Verify update call args?
        # The last execute call should be the update
        args, kwargs = cursor.execute.call_args
        sql = args[0]
        params = args[1]
        
        assert "UPDATE positions" in sql
        assert params['remaining_shares'] == 100
        assert params['remaining_fees'] == pytest.approx(150.0)
