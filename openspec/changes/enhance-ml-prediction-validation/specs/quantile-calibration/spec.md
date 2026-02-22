# Spec: Quantile Calibration

Quantile calibration ensures that prediction intervals (p10 to p90) accurately reflect uncertainty by verifying empirical coverage matches theoretical coverage.

## ADDED Requirements

### Requirement: Empirical Coverage Calculation

The system MUST calculate the proportion of actual returns falling within predicted intervals using the indicator function method.

**Formula:**
$$\text{Empirical Coverage}_q = \frac{1}{N} \sum_{i=1}^{N} \mathbb{I}(y_i \in [Q_{0.1}(x_i), Q_{0.9}(x_i)])$$

#### Scenario: Check 80% interval calibration

**Given** 90 days of VCI 5-day predictions with actuals  
**And** each prediction has p10 and p90 quantiles  
**When** calibration is checked  
**Then** system should:

- Count predictions where actual ∈ [p10, p90] → within_range
- Count predictions where actual < p10 → below_p10
- Count predictions where actual > p90 → above_p90
- Calculate coverage = within_range / total_predictions

**Example:**

- Total: 90 predictions
- Within [p10, p90]: 72 predictions
- Below p10: 10 predictions
- Above p90: 8 predictions
- Empirical coverage = 72/90 = 0.80 (80%) ✓

---

### Requirement: Per-Quantile Coverage Verification

The system MUST verify each quantile level independently.

#### Scenario: Verify p10 quantile

**Given** 90 predictions with p10 values  
**When** p10 coverage is checked  
**Then** system should count actuals < p10  
**And** expected coverage = 0.10 (10%)  
**And** actual coverage should be in [0.08, 0.12] (acceptable range)  
**And** if actual coverage = 0.15, flag as WARNING (too pessimistic)  
**And** if actual coverage = 0.05, flag as ERROR (too optimistic, dangerous)

---

### Requirement: Calibration Status Classification

The system MUST classify calibration status based on the magnitude of coverage error.

#### Scenario: Classify calibration status

**Given** quantile check results  
**When** status is determined  
**Then** systemshould classify:

- OK: error within ±2% of expected (e.g., p10 actual = 0.09-0.11)
- WARNING: error within ±3% but outside ±2%
- ERROR: error > ±3% (requires immediate recalibration)

---

### Requirement: Recalibration Recommendations

The system MUST generate recalibration recommendations when quantiles are miscalibrated.

#### Scenario: Recommend quantile parameter adjustment

**Given** p10 actual coverage = 0.05 (too optimistic)  
**And** expected coverage = 0.10  
**When** recalibration recommendation generated  
**Then** system should suggest:

- "Increase alpha parameter from 0.1 to 0.15 for p10 quantile"
- "Current p10 predictions catching only 5% of downside, should catch 10%"
- "Risk: Underestimating worst-case scenarios"
