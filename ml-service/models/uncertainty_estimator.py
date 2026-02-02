"""
Uncertainty Quantification using Bootstrap Ensembles

Separates uncertainty into two components:
- Aleatoric: Irreducible market randomness (average prediction range)
- Epistemic: Model parameter uncertainty (variance across ensemble)

Formula:
    σ_epistemic = std([pred_1(x), pred_2(x), ..., pred_10(x)])
    σ_aleatoric = mean([p90_i - p10_i for i=1..10])
    σ_total = sqrt(σ_epistemic² + σ_aleatoric²)
    Confidence = 1 / σ_total

Author: ML Trading System
Created: 2026-02-02
"""

import numpy as np
import pandas as pd
from typing import Dict, List, Optional
import xgboost as xgb
from sklearn.utils import resample

from config import QUANTILES, HYPERPARAMETERS, BASE_DIR


class UncertaintyEstimator:
    """
    Quantify prediction uncertainty using bootstrap ensembles.
    
    Strategy:
    - Train 10 XGBoost models on different bootstrap samples
    - Aggregate predictions to measure model disagreement (epistemic)
    - Average prediction intervals to measure market noise (aleatoric)
    - Combine for total uncertainty and confidence score
    """
    
    def __init__(self, num_models: int = 10):
        """
        Initialize uncertainty estimator.
        
        Args:
            num_models: Number of bootstrap ensemble models (default 10)
        """
        self.num_models = num_models
        self.ensemble_models = []
        self.models_dir = BASE_DIR / 'models' / 'saved' / 'ensemble'
        self.models_dir.mkdir(parents=True, exist_ok=True)
    
    def train_ensemble(self, 
                       X_train: pd.DataFrame,
                       y_train: pd.Series,
                       ticker: str,
                       horizon: int):
        """
        Train ensemble of models on bootstrap samples.
        
        Args:
            X_train: Training features
            y_train: Training targets
            ticker: Stock symbol
            horizon: Prediction horizon
        """
        self.ensemble_models = []
        
        for i in range(self.num_models):
            # Create bootstrap sample (resample with replacement)
            X_boot, y_boot = resample(
                X_train, y_train,
                n_samples=len(X_train),
                random_state=42 + i  # Different seed for each model
            )
            
            # Train separate model for each quantile
            models_dict = {}
            
            for quantile in QUANTILES:
                model = xgb.XGBRegressor(
                    objective='reg:quantileerror',
                    quantile_alpha=quantile,
                    **HYPERPARAMETERS
                )
                
                model.fit(X_boot, y_boot, verbose=False)
                models_dict[quantile] = model
            
            self.ensemble_models.append(models_dict)
            
            # Save model
            for quantile, model in models_dict.items():
                model_name = f'{ticker}_h{horizon}_q{int(quantile*100)}_ensemble{i}.json'
                model.save_model(str(self.models_dir / model_name))
        
        print(f"Trained {self.num_models} ensemble models for {ticker} (horizon={horizon})")
    
    def predict_with_uncertainty(self, 
                                  X: pd.DataFrame,
                                  ticker: str = None,
                                  horizon: int = None) -> Dict:
        """
        Generate predictions with uncertainty quantification.
        
        Args:
            X: Feature matrix
            ticker: Stock symbol (for loading saved models)
            horizon: Prediction horizon
            
        Returns:
            Dictionary with:
            - mean_p10, mean_p50, mean_p90: Aggregated predictions
            - epistemic_uncertainty: Model disagreement (std dev across ensemble)
            - aleatoric_uncertainty: Average prediction range (p90 - p10)
            - total_uncertainty: Combined uncertainty
            - confidence: 1 / total_uncertainty
        """
        if not self.ensemble_models:
            # Try to load saved models
            if ticker and horizon:
                self._load_ensemble(ticker, horizon)
            else:
                raise ValueError("Ensemble not trained and no ticker/horizon provided for loading")
        
        # Collect predictions from all ensemble models
        predictions_p10 = []
        predictions_p50 = []
        predictions_p90 = []
        
        for models_dict in self.ensemble_models:
            p10_pred = models_dict[0.10].predict(X)
            p50_pred = models_dict[0.50].predict(X)
            p90_pred = models_dict[0.90].predict(X)
            
            predictions_p10.append(p10_pred)
            predictions_p50.append(p50_pred)
            predictions_p90.append(p90_pred)
        
        # Convert to numpy arrays for easier manipulation
        predictions_p10 = np.array(predictions_p10)  # shape: (num_models, num_samples)
        predictions_p50 = np.array(predictions_p50)
        predictions_p90 = np.array(predictions_p90)
        
        # Aggregate predictions (mean across ensemble)
        mean_p10 = np.mean(predictions_p10, axis=0)
        mean_p50 = np.mean(predictions_p50, axis=0)
        mean_p90 = np.mean(predictions_p90, axis=0)
        
        # Epistemic uncertainty: Standard deviation across models
        # Measures model disagreement - how much models disagree on prediction
        epistemic_p50 = np.std(predictions_p50, axis=0)
        
        # Aleatoric uncertainty: Average prediction range
        # Measures inherent market randomness - how wide the prediction interval is
        prediction_ranges = predictions_p90 - predictions_p10  # shape: (num_models, num_samples)
        aleatoric = np.mean(prediction_ranges, axis=0)
        
        # Total uncertainty: Combine epistemic and aleatoric
        # Formula: sqrt(σ_epistemic² + σ_aleatoric²)
        total_uncertainty = np.sqrt(epistemic_p50**2 + aleatoric**2)
        
        # Confidence score: Inverse of total uncertainty
        # Higher confidence when uncertainty is low
        confidence = 1.0 / (total_uncertainty + 1e-6)  # Add small epsilon to avoid division by zero
        
        # For single sample, return scalars instead of arrays
        if len(X) == 1:
            return {
                'mean_p10': float(mean_p10[0]),
                'mean_p50': float(mean_p50[0]),
                'mean_p90': float(mean_p90[0]),
                'epistemic_uncertainty': float(epistemic_p50[0]),
                'aleatoric_uncertainty': float(aleatoric[0]),
                'total_uncertainty': float(total_uncertainty[0]),
                'confidence': float(confidence[0])
            }
        else:
            return {
                'mean_p10': mean_p10,
                'mean_p50': mean_p50,
                'mean_p90': mean_p90,
                'epistemic_uncertainty': epistemic_p50,
                'aleatoric_uncertainty': aleatoric,
                'total_uncertainty': total_uncertainty,
                'confidence': confidence
            }
    
    def _load_ensemble(self, ticker: str, horizon: int):
        """Load saved ensemble models from disk."""
        self.ensemble_models = []
        
        for i in range(self.num_models):
            models_dict = {}
            
            for quantile in QUANTILES:
                model_name = f'{ticker}_h{horizon}_q{int(quantile*100)}_ensemble{i}.json'
                model_path = self.models_dir / model_name
                
                if not model_path.exists():
                    raise FileNotFoundError(f"Ensemble model not found: {model_path}")
                
                model = xgb.XGBRegressor()
                model.load_model(str(model_path))
                models_dict[quantile] = model
            
            self.ensemble_models.append(models_dict)
    
    def interpret_uncertainty(self, uncertainty_dict: Dict) -> str:
        """
        Provide interpretation of uncertainty components for decision making.
        
        Args:
            uncertainty_dict: Output from predict_with_uncertainty()
            
        Returns:
            Human-readable interpretation with recommended action
        """
        epistemic = uncertainty_dict['epistemic_uncertainty']
        aleatoric = uncertainty_dict['aleatoric_uncertainty']
        total = uncertainty_dict['total_uncertainty']
        
        # Thresholds
        HIGH_EPISTEMIC = 0.01  # 1% disagreement among models
        HIGH_ALEATORIC = 0.10  # 10% average prediction range
        
        interpretation = []
        
        if epistemic > HIGH_EPISTEMIC:
            interpretation.append(
                f"⚠ High epistemic uncertainty ({epistemic:.2%}) - "
                f"Models disagree significantly. Recommendation: Collect more data or retrain."
            )
        else:
            interpretation.append(
                f"✓ Low epistemic uncertainty ({epistemic:.2%}) - "
                f"Models have consensus on prediction."
            )
        
        if aleatoric > HIGH_ALEATORIC:
            interpretation.append(
                f"⚠ High aleatoric uncertainty ({aleatoric:.2%}) - "
                f"Market highly unpredictable. Recommendation: Reduce position size."
            )
        else:
            interpretation.append(
                f"✓ Moderate aleatoric uncertainty ({aleatoric:.2%}) - "
                f"Market volatility within normal range."
            )
        
        interpretation.append(
            f"Total uncertainty: {total:.2%}, Confidence: {uncertainty_dict['confidence']:.2f}"
        )
        
        return "\n".join(interpretation)


if __name__ == '__main__':
    # Example usage
    from sklearn.datasets import make_regression
    
    # Create synthetic data
    X, y = make_regression(n_samples=1000, n_features=10, noise=10, random_state=42)
    X_train = pd.DataFrame(X[:800])
    y_train = pd.Series(y[:800])
    X_test = pd.DataFrame(X[800:805])  # 5 test samples
    
    # Train ensemble
    estimator = UncertaintyEstimator(num_models=10)
    estimator.train_ensemble(X_train, y_train, ticker='TEST', horizon=5)
    
    # Predict with uncertainty
    result = estimator.predict_with_uncertainty(X_test)
    
    print("Uncertainty Quantification Results")
    print("=" * 60)
    print(f"Mean Predictions (p50): {result['mean_p50'][:3]}")
    print(f"\nEpistemic Uncertainty: {result['epistemic_uncertainty'][:3]}")
    print(f"Aleatoric Uncertainty: {result['aleatoric_uncertainty'][:3]}")
    print(f"Total Uncertainty: {result['total_uncertainty'][:3]}")
    print(f"Confidence: {result['confidence'][:3]}")
    
    # Interpret first sample
    single_result = estimator.predict_with_uncertainty(X_test.iloc[[0]])
    print("\nInterpretation for Sample 1:")
    print(estimator.interpret_uncertainty(single_result))
