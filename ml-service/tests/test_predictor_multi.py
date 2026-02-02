
import pytest
from unittest.mock import MagicMock, patch
import pandas as pd
import numpy as np
import sys
import os

# Add parent directory to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from inference.predictor import Predictor

class MockBooster:
    def __init__(self, val=0.1):
        self.val = val
    def predict(self, matrix):
        return [self.val]

def test_load_models_multi_horizon():
    """Test loading models groups them by horizon"""
    with patch('inference.predictor.DatabaseConnection') as MockDB:
        mock_conn = MagicMock()
        mock_cursor = MagicMock()
        MockDB.get_connection.return_value = mock_conn
        mock_conn.cursor.return_value = mock_cursor
        
        # Mock fetchall returns rows for 1d and 5d
        mock_cursor.fetchall.return_value = [
            {'model_id': 'm1', 'ticker': 'ABC', 'quantile': 0.1, 'horizon': 1, 'file_path': 'f1', 'hyperparameters': {}, 'metrics': {}},
            {'model_id': 'm2', 'ticker': 'ABC', 'quantile': 0.5, 'horizon': 1, 'file_path': 'f2', 'hyperparameters': {}, 'metrics': {}},
            {'model_id': 'm3', 'ticker': 'ABC', 'quantile': 0.9, 'horizon': 1, 'file_path': 'f3', 'hyperparameters': {}, 'metrics': {}},
            {'model_id': 'm4', 'ticker': 'ABC', 'quantile': 0.1, 'horizon': 5, 'file_path': 'f4', 'hyperparameters': {}, 'metrics': {}},
            {'model_id': 'm5', 'ticker': 'ABC', 'quantile': 0.5, 'horizon': 5, 'file_path': 'f5', 'hyperparameters': {}, 'metrics': {}},
            {'model_id': 'm6', 'ticker': 'ABC', 'quantile': 0.9, 'horizon': 5, 'file_path': 'f6', 'hyperparameters': {}, 'metrics': {}},
        ]
        
        # Mock os.path and ModelTrainer
        with patch('inference.predictor.os.path.exists', return_value=True), \
             patch('inference.predictor.ModelTrainer') as MockTrainer:
            
            mock_trainer_instance = MockTrainer.return_value
            mock_trainer_instance.load_model.return_value = MockBooster()
            
            predictor = Predictor()
            success = predictor.load_production_models('ABC')
            
            assert success
            assert 'ABC' in predictor.models
            assert 1 in predictor.models['ABC']
            assert 5 in predictor.models['ABC']
            assert len(predictor.models['ABC'][1]) == 3
            assert len(predictor.models['ABC'][5]) == 3

def test_predict_multi_horizon():
    """Test prediction returns dictionary for all horizons"""
    predictor = Predictor()
    predictor.models['ABC'] = {
        1: {0.1: MockBooster(0.01), 0.5: MockBooster(0.02), 0.9: MockBooster(0.03)},
        5: {0.1: MockBooster(0.05), 0.5: MockBooster(0.10), 0.9: MockBooster(0.15)}
    }
    
    features = pd.DataFrame({'f1': [1]}, index=[0])
    
    # We need to mock xgb.DMatrix because it requires data compatible with xgb
    with patch('inference.predictor.xgb.DMatrix'):
        results = predictor.predict('ABC', features)
    
    assert 1 in results
    assert 5 in results
    
    assert results[1]['p50'] == 0.02
    assert results[5]['p50'] == 0.10
    
    # Check ordering enforcement
    # If p10 > p50, it should be corrected
    predictor.models['Broken'] = {
        10: {0.1: MockBooster(0.05), 0.5: MockBooster(0.02), 0.9: MockBooster(0.03)}
    }
    with patch('inference.predictor.xgb.DMatrix'):
        res = predictor.predict('Broken', features)
    
    # p10 should be min(0.05, 0.02, 0.03) -> 0.02
    assert res[10]['p10'] == 0.02

if __name__ == "__main__":
    test_load_models_multi_horizon()
    test_predict_multi_horizon()
    print("All tests passed!")
