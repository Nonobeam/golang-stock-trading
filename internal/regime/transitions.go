package regime

import "fmt"

// TransitionDetector detects regime transitions and instability.
type TransitionDetector struct{}

// NewTransitionDetector creates a new transition detector.
func NewTransitionDetector() *TransitionDetector {
	return &TransitionDetector{}
}

// DetectTransition checks if market is transitioning between regimes.
func (d *TransitionDetector) DetectTransition(
	currentRegime *RegimeResult,
	regimeHistory []RegimeHistory,
	volatilityRecent float64,
	volatilityAverage float64,
) *TransitionState {

	transitionSignals := []string{}
	transitionScore := 0

	// Signal 1: Frequent regime changes
	if len(regimeHistory) >= 10 {
		recent10 := regimeHistory[len(regimeHistory)-10:]
		uniqueRegimes := make(map[RegimeType]bool)

		for _, r := range recent10 {
			uniqueRegimes[r.Regime] = true
		}

		uniqueCount := len(uniqueRegimes)

		if uniqueCount >= 4 {
			transitionSignals = append(transitionSignals, "High regime instability (4+ different regimes in 10 days)")
			transitionScore += 3
		} else if uniqueCount >= 3 {
			transitionSignals = append(transitionSignals, "Moderate regime instability (3 regimes in 10 days)")
			transitionScore += 2
		}
	}

	// Signal 2: Volatility spike
	if volatilityAverage > 0 && volatilityRecent > volatilityAverage*1.5 {
		percentAbove := ((volatilityRecent / volatilityAverage) - 1) * 100
		transitionSignals = append(transitionSignals, fmt.Sprintf("Volatility spike (%.0f%% above average)", percentAbove))
		transitionScore += 2
	}

	// Signal 3: Low confidence in current regime
	if currentRegime.Confidence == ConfidenceLow {
		transitionSignals = append(transitionSignals, "Low confidence in current regime classification")
		transitionScore += 2
	}

	// Signal 4: Conflicting factors (check regime factors)
	if factors, ok := currentRegime.Factors["basic_regime"].(map[string]interface{}); ok {
		basicScore := 0
		if score, ok := factors["score"].(int); ok {
			basicScore = score
		}

		adxScore := 0
		if adxFactors, ok := currentRegime.Factors["adx"].(map[string]interface{}); ok {
			if score, ok := adxFactors["score"].(int); ok {
				adxScore = score
			}
		}

		if (basicScore > 0 && adxScore < 0) || (basicScore < 0 && adxScore > 0) {
			transitionSignals = append(transitionSignals, "Conflicting signals between price action and trend strength")
			transitionScore += 1
		}
	}

	// Classify transition state
	var inTransition bool
	var severity, recommendation string
	var actionItems []string

	if transitionScore >= 5 {
		inTransition = true
		severity = "High"
		recommendation = "STOP NEW POSITIONS - Wait for regime clarity"
		actionItems = []string{
			"Close all positions that were opened based on previous regime",
			"Move to 70-80% cash",
			"Do not enter new positions",
			"Monitor daily for regime stabilization",
			"Consider only day trades if trading at all",
		}
	} else if transitionScore >= 3 {
		inTransition = true
		severity = "Moderate"
		recommendation = "REDUCE POSITION SIZES BY 50% - Exercise caution"
		actionItems = []string{
			"Reduce all position sizes by 50%",
			"Tighten stops on existing positions",
			"Only take highest quality setups (score ≥9)",
			"Increase cash to 40-50%",
			"Monitor closely for regime confirmation",
		}
	} else {
		inTransition = false
		severity = "Low"
		recommendation = "Normal trading - Regime stable"
		actionItems = []string{
			"Continue normal trading according to regime rules",
		}
	}

	return &TransitionState{
		InTransition:       inTransition,
		TransitionSeverity: severity,
		TransitionScore:    transitionScore,
		Signals:            transitionSignals,
		Recommendation:     recommendation,
		ActionItems:        actionItems,
	}
}
