# Proposal: Enhance ML Prediction Validation

## Summary

Transform the ML trading system from a theoretical backtesting tool into a production-ready prediction engine with comprehensive validation, Vietnamese market-specific safeguards, and robust uncertainty quantification.

## Problem Statement

The current ML service generates predictions and signals without:

1. **Temporal validation** - Single backtest on historical data without walk-forward testing across multiple time periods (overfitting risk)
2. **Calibration verification** - Quantile predictions (p10, p90) never validated for accuracy
3. **Feature quality assessment** - No tracking of which features are stable vs noisy predictors
4. **Transaction cost awareness** - Vietnamese market fees (0.4% round-trip) completely ignored in validation
5. **Price limit risk** - Circuit breakers (±7% HOSE, ±10% HNX) can trap positions without escape
6. **Liquidity constraints** - Position sizing ignores daily volume, risking execution slippage
7. **Uncertainty quantification** - No separation of market randomness vs model uncertainty
8. **Regime adaptation** - Single model averages across bull/bear/volatile regimes

## Proposed Solution

Implement **8 core prediction enhancements** with **3 Vietnamese market-specific validations**:

### Core Enhancements

1. **Walk-Forward Validation** - Rolling time-series cross-validation simulating real trading
2. **Quantile Calibration** - Automated verification that prediction intervals match realized distributions
3. **Feature Stability Analysis** - Track feature importance over time to identify reliable predictors
4. **Uncertainty Quantification** - Bootstrap ensembles to separate aleatoric vs epistemic uncertainty
5. **Sharpe Ratio Forecasting** - Predict both returns and volatility for risk-adjusted allocation
6. **Regime-Conditional Models** - Separate models for bull/bear/high-volatility market conditions
7. **Prediction Interval Coverage** - Daily monitoring of whether actuals fall within predicted ranges
8. **Cross-Stock Validation** - Test model architecture across 10 Vietnamese stocks

### Vietnamese Market Validations

1. **Transaction Cost Integration** - Fee-adjusted metrics (0.15% brokerage + 0.1% tax)
2. **Floor-Hit Probability** - Binary classifier to predict circuit breaker trap risk
3. **Liquidity Constraints** - Position caps based on daily volume (1% maximum)

## Success Metrics

- Walk-forward MAE < 3%
- Information Coefficient > 0.15
- Directional accuracy > 55%
- Fee-adjusted Sharpe > 1.0
- Calibration error < 2% for all quantiles
- Prediction coverage 75-85%
- Floor-hit prediction accuracy > 70%

## Impact

- **Quality over Quantity**: 40-60% reduction in signal frequency, but higher win rate
- **Risk Management**: Prevent circuit breaker traps and execution slippage disasters
- **Model Trust**: Mathematical proof that models work out-of-sample, not just in-sample
- **Vietnamese Market Fit**: Address local microstructure (fees, price limits, thin liquidity)

## Timeline

- **Weeks 1-2**: Transaction costs + Walk-forward validation + Calibration (critical foundation)
- **Weeks 3-4**: Liquidity constraints + Coverage tracking + Feature stability (high priority)
- **Weeks 5-6**: Floor-hit prediction + Uncertainty quantification + Sharpe forecasting (advanced)
- **Week 7+**: Regime models + Cross-stock validation + Integration (sophistication)

## Dependencies

- PostgreSQL database (existing)
- Python ML service (existing)
- XGBoost models (existing)
- 2+ years historical data for VCI, HPG, FPT, VNM, VHM (existing)

## Risks

1. **Computational Cost**: Walk-forward validation takes 4-6 hours per stock - mitigate with weekly runs
2. **Signal Reduction**: Stricter filters will reduce trade frequency - acceptable if quality improves
3. **Storage Growth**: ~50MB/year for validation tables - minimal impact
4. **Complexity**: 27 models for regime-conditional approach - defer to Phase 2 if needed
