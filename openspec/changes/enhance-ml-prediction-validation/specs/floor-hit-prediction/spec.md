# Spec: Floor-Hit Prediction

Floor-hit prediction uses binary classification to predict circuit breaker trap risk (±7% HOSE, ±10% HNX).

## ADDED Requirements

### Requirement: Binary Classifier Training

The system MUST train XGBoost classifier to predict floor/ceiling hits.

#### Scenario: Label training data

**Given** VCI historical daily bars  
**When** labeling for HOSE ±7% limits  
**Then** for each day:

- hit_floor = 1 if next_day_return ≤ -0.07, else 0
- hit_ceiling = 1 if next_day_return ≥ 0.07, else 0

#### Scenario: Define predictive features

**When** classifier features defined  
**Then** features include:

- momentum_5d: 5-day price change %
- volume_surge: current_volume / avg_volume_20d
- consecutive_down_days: streak of negative returns
- distance_from_support: (price - support_level) / support_level
- vn_index_momentum: VN-Index 5-day change
- relative_strength: stock_momentum - market_momentum

---

### Requirement: Probability Prediction

The system MUST predict the probability of hitting floor or ceiling price limits for the next trading day.

#### Scenario: Predict floor risk for VCI

**Given** trained floor-hit classifier  
**And** VCI current features: momentum=-4.2%, volume_surge=1.8, consecutive_down=2  
**When** classifier predicts  
**Then** floor_probability = model.predict_proba(features)[1]  
**Example:** floor_probability = 0.23 (23%)

---

### Requirement: Risk-Based Signal Override

The system MUST override ML prediction signals when floor-hit probability exceeds risk thresholds.

#### Scenario: Override signal for high floor risk

**Given** ML model predicts p50 return = +2.5% (BUY signal)  
**And** floor-hit probability = 0.42 (42%)  
**When** signal generated  
**Then** floor risk > 0.40 threshold  
**And** signal overridden to SELL  
**And** rationale = "Floor risk 42% overrides ML prediction, exit position"

#### Scenario: Reduce position for medium floor risk

**Given** ML model suggests BUY_MORE  
**And** floor-hit probability = 0.25 (25%)  
**When** signal generated  
**Then** floor risk in [0.20, 0.40] range  
**And** recommended position reduced by 50%  
**And** warning = "Floor risk 25%, halving position size"
