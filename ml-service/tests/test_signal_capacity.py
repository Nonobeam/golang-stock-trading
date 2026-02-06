import pytest
from unittest.mock import MagicMock, patch
from signals.generator import SignalGenerator

class TestSignalGeneratorCapacity:
    
    @pytest.fixture
    def mock_db_connection(self):
        return MagicMock()
        
    @pytest.fixture
    def generator(self):
        return SignalGenerator()
        
    @pytest.fixture
    def strong_buy_predictions(self):
        return {
            10: {'p50': 0.15, 'p10': -0.02, 'p90': 0.25, 'confidence': 0.8}, # Good return, acceptable risk
            5:  {'p50': 0.10, 'p10': -0.02, 'p90': 0.20, 'confidence': 0.8},
            1:  {'p50': 0.01, 'p10': -0.01, 'p90': 0.03, 'confidence': 0.8}
        }

    @patch('signals.generator.PositionManager')
    def test_generate_signal_buy_new_at_capacity(self, MockPM, generator, mock_db_connection, strong_buy_predictions):
        # Setup mock PM
        pm_instance = MockPM.return_value
        pm_instance.get_position_for_signal.return_value = None # No existing position
        
        # Setup Capacity Check: At Limit
        pm_instance.check_buying_capacity.return_value = {
            'at_limit': True,
            'limit_reason': 'portfolio_allocation_20pct'
        }
        
        # Execute
        signal, strength, reason = generator.generate_signal(
            ticker="TEST",
            predictions=strong_buy_predictions,
            current_price=100.0,
            db_connection=mock_db_connection,
            account_value=100000.0
        )
        
        # Verify
        assert signal == "HOLD_NONE"
        assert "Capacity limit reached" in reason
        assert "portfolio_allocation_20pct" in reason
        assert strength == 0.0

    @patch('signals.generator.PositionManager')
    def test_generate_signal_buy_more_at_capacity(self, MockPM, generator, mock_db_connection, strong_buy_predictions):
        # Setup mock PM
        pm_instance = MockPM.return_value
        # Existing position
        pm_instance.get_position_for_signal.return_value = {
            'quantity': 100, 
            'avg_price': 90.0
        }
        
        # Setup Capacity Check: At Limit
        pm_instance.check_buying_capacity.return_value = {
            'at_limit': True,
            'limit_reason': 'liquidity_1pct_volume'
        }
        
        # Execute
        signal, strength, reason = generator.generate_signal(
            ticker="TEST",
            predictions=strong_buy_predictions, 
            current_price=100.0,
            db_connection=mock_db_connection,
            account_value=100000.0
        )
        
        # Verify
        assert signal == "HOLD"
        assert "Capacity limit reached" in reason
        assert "liquidity" in reason

    @patch('signals.generator.PositionManager')
    def test_generate_signal_buy_new_with_capacity(self, MockPM, generator, mock_db_connection, strong_buy_predictions):
        # Setup mock PM
        pm_instance = MockPM.return_value
        pm_instance.get_position_for_signal.return_value = None
        
        # Setup Capacity: OK
        pm_instance.check_buying_capacity.return_value = {
            'at_limit': False,
            'limit_reason': None
        }
        
        # Execute
        signal, strength, reason = generator.generate_signal(
            ticker="TEST",
            predictions=strong_buy_predictions,
            current_price=100.0,
            db_connection=mock_db_connection,
            account_value=100000.0
        )
        
        # Verify
        assert signal == "BUY_NEW"
        assert strength > 0.0

    @patch('signals.generator.PositionManager')
    def test_generate_signal_no_account_value_ignored(self, MockPM, generator, mock_db_connection, strong_buy_predictions):
        # If account_value is NOT passed, capacity check should be skipped
        pm_instance = MockPM.return_value
        pm_instance.get_position_for_signal.return_value = None
        
        # Execute
        signal, strength, reason = generator.generate_signal(
            ticker="TEST",
            predictions=strong_buy_predictions,
            current_price=100.0,
            db_connection=mock_db_connection,
            account_value=None  # Explicitly None
        )
        
        # Verify
        assert signal == "BUY_NEW"
        # Mock capacity check should NOT have been called
        pm_instance.check_buying_capacity.assert_not_called()
