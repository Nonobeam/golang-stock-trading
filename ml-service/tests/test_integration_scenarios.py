import pytest
from unittest.mock import MagicMock, patch, call
from position_manager.manager import PositionManager

class TestIntegrationScenarios:
    """
    Simulated Integration Tests using Mocks to verify business logic scenarios 
    defined in Section 16 of tasks.md.
    """
    
    @pytest.fixture
    def mock_db_connection(self):
        conn = MagicMock()
        cursor = MagicMock()
        conn.cursor.return_value.__enter__.return_value = cursor
        return conn

    @pytest.fixture
    def manager(self, mock_db_connection):
        return PositionManager(db_connection=mock_db_connection)

    def test_scenario_single_entry_backward_compat(self, manager, mock_db_connection):
        """16.1 Scenario: Single entry position (verify backward compatibility)"""
        # System should handle single entry positions correctly
        cursor = mock_db_connection.cursor.return_value.__enter__.return_value
        
        # Setup: One entry
        cursor.fetchone.return_value = (10000.0, 100) # total_cost, total_shares
        
        avg_cost = manager.calculate_average_cost("TEST")
        
        # Verify: Average Cost = Entry Price
        assert avg_cost == 100.0

    def test_scenario_two_entries_avg_cost(self, manager, mock_db_connection):
        """16.2 Scenario: Two entries at different prices (verify average cost)"""
        cursor = mock_db_connection.cursor.return_value.__enter__.return_value
        
        # Entry 1: 100 @ 100 = 10,000
        # Entry 2: 50 @ 120 = 6,000
        # Total: 16,000 / 150 = 106.666...
        cursor.fetchone.return_value = (16000.0, 150)
        
        avg_cost = manager.calculate_average_cost("TEST")
        
        assert avg_cost == pytest.approx(106.66666666666667)

    def test_scenario_capacity_limit(self, manager, mock_db_connection):
        """16.3 Scenario: Position at 20% limit"""
        cursor = mock_db_connection.cursor.return_value.__enter__.return_value
        
        # Mocking check_buying_capacity flow
        # 1. get_position (18 cols)
        pos_row = (
            'id', 1, 'TEST', '2023-01-01', 
            100.0, 200, 95.0, None, None, None, 
            'MANUAL', 0, 'notes', None, None, None, None, None
        )
        
        # 2. first entry price
        entry_row = (100.0,)
        
        # 3. avg volume
        val_row = (1000000.0,)
        
        cursor.fetchone.side_effect = [pos_row, entry_row, val_row]
        
        # Account Value = 100,000
        # Position Value = 200 * 100 = 20,000 (20%)
        # Should be AT LIMIT
        
        cap = manager.check_buying_capacity("TEST", 100.0, 100000.0)
        
        assert cap['at_limit'] is True
        assert cap['limit_reason'] == "portfolio_allocation_20pct"
        assert cap['max_buyable_shares'] == 0

    def test_scenario_partial_exit(self, manager, mock_db_connection):
        """16.4 Scenario: Partial exit of multi-entry position"""
        cursor = mock_db_connection.cursor.return_value.__enter__.return_value
        
        # Existing Position: 200 shares, Avg 100, Total Fees 300
        cursor.fetchone.return_value = ('TEST', 1, 100.0, 200, 300.0)
        
        # Sell 50% (100 shares)
        result = manager.partial_exit_position("uuid", 100, 150.0, "2023-01-01")
        
        # Verify Fees Reduced Proportionally
        assert result['proportional_entry_fees'] == 150.0  # 50% of 300
        assert result['shares_sold'] == 100
        assert result['remaining_shares'] == 100
        
        # Verify DB Update for remaining_fees
        # Last execute call should look like: UPDATE positions SET ...
        args, kwargs = cursor.execute.call_args
        params = args[1]
        assert params['remaining_fees'] == 150.0

    def test_scenario_stop_loss_risk_calc(self, manager, mock_db_connection):
        """16.5 Scenario: Stop loss trigger using first entry price"""
        # This tests checking capacity based on risk, using first entry price
        cursor = mock_db_connection.cursor.return_value.__enter__.return_value
        
        # Position: 100 shares @ Avg 110 (Entry 1 @ 100, Entry 2 @ 120)
        # Stop Loss: 95
        # Risk per share (First Entry): 100 - 95 = 5 (Per validation rule)
        # Risk per share (Avg Cost): 110 - 95 = 15 (Incorrect rule)
        
        # Mock get_position
        pos_row = (
            'id', 1, 'TEST', '2023-01-01', 
            110.0, 100, 95.0, None, None, None, 
            'MANUAL', 0, 'notes', None, None, None, None, None
        )
        
        # Mock first entry query -> Returns 100.0
        entry_row = (100.0,)
        
        # Mock volume -> unlimited
        val_row = (1000000.0,)
        
        cursor.fetchone.side_effect = [pos_row, entry_row, val_row]
        
        # Account Value: 100,000 -> Max Risk (2%) = 2,000
        # Current Risk (using first entry): 100 * (100 - 95) = 500
        # Remaining Risk Capacity: 1500
        
        # New Purchase Price: 100
        # Risk per new share: 100 - 95 = 5
        # Max shares by risk: 1500 / 5 = 300
        
        cap = manager.check_buying_capacity("TEST", 100.0, 100000.0)
        
        # Verify total risk calculation uses first entry price logic implicitly
        assert cap['total_risk'] == 500.0 # 100 shares * 5 risk
        assert cap['max_risk_allowed'] == 2000.0
