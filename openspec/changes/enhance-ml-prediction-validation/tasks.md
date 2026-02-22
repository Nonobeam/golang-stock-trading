# Tasks: Enhance ML Prediction Validation

## Phase 1: Critical Foundation (Weeks 1-2)

### Database Schema

- [x] Create migration `000011_validation_infrastructure.up.sql`
- [x] Add `walk_forward_results` table
- [x] Add `calibration_reports` table
- [x] Add `prediction_coverage` table
- [x] Add `feature_stability` table
- [x] Add `floor_hit_probabilities` table
- [x] Create necessary indexes
- [ ] Test migration on development database

### Transaction Cost Module

- [x] Create `ml-service/validation/transaction_costs.py`
- [x] Implement `calculate_fees()` function with Vietnamese rates
- [x] Implement `get_minimum_profit_threshold()` by horizon
- [x] Implement `calculate_fee_adjusted_return()` function
- [ ] Write unit tests for fee calculations
- [ ] Validate against real trade history

### Walk-Forward Validator

- [x] Create `ml-service/validation/walk_forward_validator.py`
- [x] Implement `WalkForwardValidator` class with expanding/rolling windows
- [x] Implement `validate()` method for time-series splitting
- [x] Implement `_calculate_period_metrics()` with MAE, IC, directional accuracy
- [x] Integrate fee-adjusted Sharpe calculation
- [x] Add database storage for validation results
- [ ] Write unit tests with synthetic data
- [ ] Run validation on VCI historical data (2024-2025)
- [ ] Generate validation report

### Calibration Checker

- [x] Create `ml-service/validation/calibration_checker.py`
- [x] Implement `CalibrationChecker` class
- [x] Implement `check_calibration()` method with empirical coverage formula
- [x] Implement `is_calibrated()` with acceptable threshold logic
- [x] Implement `recommend_recalibration()` for miscalibrated quantiles
- [x] Add database storage for calibration reports
- [ ] Write unit tests
- [ ] Run calibration check on last 90 days of VCI predictions
- [ ] Generate calibration report

## Phase 2: High Priority (Weeks 3-4)

### Liquidity Manager

- [x] Create `ml-service/validation/liquidity_manager.py`
- [x] Implement `LiquidityManager` class
- [x] Implement `get_average_volume()` query from `daily_bars`
- [x] Implement `calculate_position_cap()` with 1% volume rule
- [x] Implement `get_liquidity_score()` (1-10 scale)
- [x] Implement `filter_tradeable_universe()` (exclude <100K volume)
- [ ] Implement `recommend_execution_strategy()` for order splitting
- [ ] Write unit tests
- [ ] Test with VCI, HPG, low-volume stocks

### Coverage Tracker

- [ ] Create `ml-service/monitoring/coverage_tracker.py`
- [ ] Implement `CoverageTracker` class
- [ ] Implement `check_daily_coverage()` comparing predictions to actuals
- [ ] Implement `get_coverage_by_horizon()` for 1/5/10-day breakdowns
- [ ] Implement `detect_systematic_bias()` for directional misses
- [ ] Add database storage for coverage results
- [ ] Create daily cron job for coverage monitoring
- [ ] Write unit tests
- [ ] Set up alerts for coverage < 75%

### Feature Stability Analyzer

- [ ] Create `ml-service/features/stability_analyzer.py`
- [ ] Implement `FeatureStabilityAnalyzer` class
- [ ] Implement `record_feature_importance()` called after training
- [ ] Implement `calculate_stability_metrics()` with std dev, rank changes
- [ ] Implement `get_core_feature_set()` returning stable features only
- [ ] Implement `recommend_feature_selection()` based on regime
- [ ] Add database storage for feature stability
- [ ] Modify `models/trainer.py` to call stability analyzer
- [ ] Write unit tests
- [ ] Generate feature stability report for last 6 months

## Phase 3: Advanced Risk Management (Weeks 5-6)

### Floor-Hit Classifier

- [x] Create `ml-service/models/floor_hit_classifier.py`
- [x] Implement `FloorHitClassifier` class
- [x] Define features: momentum, volume surge, consecutive down days, distance from support
- [x] Implement `prepare_training_data()` with binary labels for ±7%/±10% hits
- [x] Train XGBoost binary classifiers for floor and ceiling
- [x] Implement `predict_floor_probability()` method
- [x] Implement `predict_ceiling_probability()` method
- [x] Add database storage for probabilities
- [ ] Write unit tests
- [ ] Validate classifier accuracy on 2024-2025 data
- [ ] Generate classification report

### Uncertainty Estimator

- [x] Create `ml-service/models/uncertainty_estimator.py`
- [x] Implement `UncertaintyEstimator` class
- [x] Implement `train_ensemble()` with bootstrap sampling (10 models)
- [x] Implement `predict_with_uncertainty()` aggregating ensemble predictions
- [x] Calculate epistemic uncertainty (variance across models)
- [x] Calculate aleatoric uncertainty (average prediction range)
- [x] Calculate total uncertainty and confidence score
- [ ] Add uncertainty fields to `predictions` table
- [ ] Write unit tests
- [ ] Compare single-model vs ensemble predictions

### Volatility Predictor

- [ ] Create `ml-service/models/volatility_predictor.py`
- [ ] Implement `VolatilityPredictor` class
- [ ] Define volatility features: realized vol, Bollinger width, ATR, volume vol
- [ ] Train XGBoost quantile regression for volatility prediction
- [ ] Implement `predict_volatility()` method
- [ ] Implement `predict_volatility_quantiles()` for uncertainty
- [ ] Calculate predicted Sharpe ratio (return / volatility)
- [ ] Add volatility and Sharpe fields to `predictions` table
- [ ] Write unit tests
- [ ] Validate volatility predictions against realized volatility

## Phase 4: Model Sophistication (Week 7+)

### Regime Detector

- [ ] Create `ml-service/models/regime_detector.py`
- [ ] Implement `RegimeDetector` class
- [ ] Implement `detect_regime()` with VN-Index 60-day return and volatility rules
- [ ] Label historical data with BULL/BEAR/HIGH_VOL regimes
- [ ] Implement `train_regime_models()` for 3 regimes × 9 horizons = 27 models
- [ ] Implement `predict_with_regime()` loading appropriate model
- [ ] Add `regime_performance` table
- [ ] Write unit tests
- [ ] Compare regime-conditional vs single-model performance

### Cross-Stock Validator

- [ ] Create `ml-service/validation/cross_stock_validator.py`
- [ ] Implement `CrossStockValidator` class
- [ ] Define validation universe: VCI, HPG, FPT, VNM, VHM, VIC, MBB, VPB, GAS, PLX
- [ ] Implement `validate_all_stocks()` running walk-forward on all 10
- [ ] Calculate cross-stock aggregate metrics
- [ ] Implement `identify_universal_features()` comparing importance across stocks
- [ ] Implement `test_hyperparameter_robustness()` across stocks
- [ ] Add `cross_stock_validation` table
- [ ] Write unit tests
- [ ] Generate cross-stock validation report

## Phase 5: Integration (Weeks 8-9)

### Enhanced Signal Generation

- [ ] Modify `ml-service/signals/generator.py`
- [ ] Add floor-hit probability check (override if > 20%)
- [ ] Add fee threshold validation
- [ ] Add liquidity cap enforcement
- [ ] Add uncertainty filtering (reject high epistemic uncertainty)
- [ ] Add new signal metadata fields: `floor_risk`, `fee_adjusted_return`, `liquidity_score`
- [ ] Update `signals` table schema
- [ ] Write integration tests
- [ ] Test with real VCI position scenario

### Enhanced Position Sizing

- [ ] Modify `ml-service/position_sizing/kelly.py`
- [ ] Add Sharpe ratio adjustment multiplier
- [ ] Add confidence/uncertainty adjustment
- [ ] Add liquidity cap as hard constraint
- [ ] Add regime-based position multiplier
- [ ] Implement execution strategy recommendation (order splitting)
- [ ] Write unit tests
- [ ] Test with various signal scenarios

### Daily Workflow Enhancement

- [ ] Modify `ml-service/daily/daily_signals.py`
- [ ] Add validation health check section
- [ ] Add calibration status check
- [ ] Add coverage trend reporting
- [ ] Add feature stability alerts
- [ ] Add floor-hit warnings for positions
- [ ] Enhance position monitoring with liquidity constraints
- [ ] Format comprehensive daily report
- [ ] Write integration tests
- [ ] Schedule cron job for daily 8:00 AM execution

### Monitoring Dashboard

- [ ] Create `ml-service/monitoring/validation_monitor.py`
- [ ] Implement daily validation health checks
- [ ] Add calibration drift alerts
- [ ] Add coverage degradation alerts
- [ ] Add feature instability warnings
- [ ] Generate weekly validation summary
- [ ] Integrate with existing notification system (email/Telegram)
- [ ] Write unit tests
- [ ] Test alert triggering

## Phase 6: Documentation and Testing (Week 10)

### Documentation

- [ ] Update `ml-service/README.md` with validation features
- [ ] Create `VALIDATION_GUIDE.md` explaining all metrics and formulas
- [ ] Document Vietnamese market-specific considerations
- [ ] Create troubleshooting guide for validation failures
- [ ] Update API documentation for enhanced signals
- [ ] Create example usage scripts

### Comprehensive Testing

- [ ] Achieve 80% unit test coverage for all new modules
- [ ] Write end-to-end integration tests
- [ ] Run backtest comparison (old vs enhanced system)
- [ ] Performance benchmarking for validation runtime
- [ ] Load testing for multiple stocks
- [ ] User acceptance testing with real trading scenarios

### Deployment

- [ ] Deploy to development environment
- [ ] Run parallel validation (old vs new) for 2 weeks
- [ ] Monitor validation metrics daily
- [ ] Fix any issues identified during parallel run
- [ ] Gradual rollout to production
- [ ] Post-deployment monitoring (30 days)
- [ ] Archive implementation documentation

## Validation Checkpoints

After each phase:

- [ ] All tests passing
- [ ] Code review completed
- [ ] Performance benchmarks met
- [ ] Documentation updated
- [ ] Demo to stakeholders
