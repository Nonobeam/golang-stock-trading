# Spec: Uncertainty Quantification

Uncertainty quantification separates aleatoric uncertainty (market randomness) from epistemic uncertainty (model limitations) using bootstrap ensembles.

## ADDED Requirements

### Requirement: Bootstrap Ensemble Training

The system MUST train 10 XGBoost models on different bootstrap samples.

#### Scenario: Create bootstrap ensemble

**Given** training data with 5,000 samples  
**When** ensemble created  
**Then** for each of 10 models:

- Resample 5,000 samples with replacement (bootstrap)
- Train XGBoost on bootstrap sample
- Save model as `ensemble_model_{i}.json`

---

### Requirement: Epistemic Uncertainty Calculation

The system MUST measure model disagreement as epistemic uncertainty.

**Formula:**
$$\sigma_{epistemic} = \text{std}([pred_1(x), pred_2(x), ..., pred_{10}(x)])$$

#### Scenario: Calculate epistemic uncertainty

**Given** 10 ensemble models predict VCI 5-day return  
**And** p50 predictions: [2.8%, 3.1%, 2.5%, 3.4%, 2.9%, 3.2%, 2.7%, 3.0%, 3.3%, 2.6%]  
**When** epistemic uncertainty calculated  
**Then** mean prediction = 2.95%  
**And** std dev = 0.28%  
**And** epistemic_uncertainty = 0.28%

**Interpretation:** Models agree closely (low uncertainty)

---

### Requirement: Aleatoric Uncertainty Calculation

The system MUST measure average prediction width as aleatoric uncertainty.

**Formula:**
$$\sigma_{aleatoric} = \text{mean}([p90_i - p10_i \text{ for } i=1..10])$$

#### Scenario: Calculate aleatoric uncertainty

**Given** 10 ensemble models each predict p10 and p90  
**And** prediction ranges: [8.5%, 9.2%, 8.8%, 9.0%, ...]  
**When** aleatoric uncertainty calculated  
**Then** average_range = 8.9%  
**And** aleatoric_uncertainty = 8.9%

**Interpretation:** Market inherently volatile (high randomness)

---

### Requirement: Total Uncertainty and Confidence

The system MUST calculate total uncertainty by combining epistemic and aleatoric components.

**Formula:**
$$\sigma_{total} = \sqrt{\sigma_{epistemic}^2 + \sigma_{aleatoric}^2}$$
$$\text{Confidence} = \frac{1}{\sigma_{total}}$$

#### Scenario: Calculate confidence score

**Given** epistemic_uncertainty = 0.28%  
**And** aleatoric_uncertainty = 8.9%  
**When** total uncertainty calculated  
**Then** total = sqrt(0.28² + 8.9²) = sqrt(0.08 + 79.21) = 8.91%  
**And** confidence = 1 / 8.91 = 0.11 (11%)

**Interpretation:** Low confidence due to high market randomness

---

### Requirement: Uncertainty-Based Position Sizing

The system MUST adjust position sizes based on uncertainty levels.

#### Scenario: Reduce position for high epistemic uncertainty

**Given** prediction with epistemic_uncertainty = 1.2%  
**And** position sizer recommends 200 shares  
**When** epistemic > 1.0% threshold  
**Then** skip trade entirely  
**And** rationale = "Models disagree too much (epistemic=1.2%), insufficient consensus"

#### Scenario: Reduce position for high aleatoric uncertainty

**Given** prediction with aleatoric_uncertainty = 12.5%  
**And** base position = 200 shares  
**When** aleatoric > 10% threshold  
**Then** reduce position by 50%  
**And** final_position = 100 shares  
**And** rationale = "High market volatility (aleatoric=12.5%), reducing risk"
