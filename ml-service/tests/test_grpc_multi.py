
import pytest
from unittest.mock import MagicMock, patch
import sys
import os

# Add parent directory to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
sys.path.append(os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "generated"))

from server.grpc_server import MLPredictionServicer
from generated import ml_service_pb2

def test_grpc_predict_multi_horizon():
    # Mock predictor during initialization
    with patch('server.grpc_server.Predictor') as MockPredictor:
        servicer = MLPredictionServicer()
        
        # Setup mock predictor instance
        mock_pred_instance = MockPredictor.return_value
        servicer.predictor = mock_pred_instance
        
        # Mock predict_for_date return value
        mock_pred_instance.predict_for_date.return_value = {
            1: {'p10': -0.01, 'p50': 0.01, 'p90': 0.03, 'confidence': 0.8},
            5: {'p10': -0.02, 'p50': 0.05, 'p90': 0.12, 'confidence': 0.7}
        }
        mock_pred_instance.get_model_version.return_value = "v_test"
        
        # Create request
        request = ml_service_pb2.PredictRequest(ticker="ABC", date="2024-01-01")
        context = MagicMock()
        
        # Call Predict
        response = servicer.Predict(request, context)
        
        # Assertions
        assert response.success
        assert response.model_version == "v_test"
        
        # Check legacy fields (should match 1d)
        assert abs(response.p50 - 0.01) < 1e-6
        assert abs(response.confidence - 0.8) < 1e-6
        
        # Check new fields
        assert len(response.predictions) == 2
        
        # Verify contents
        lookup = {p.horizon: p for p in response.predictions}
        assert 1 in lookup
        assert 5 in lookup
        assert abs(lookup[5].p50 - 0.05) < 1e-6

if __name__ == "__main__":
    test_grpc_predict_multi_horizon()
    print("gRPC Multi-Horizon Test Passed!")
