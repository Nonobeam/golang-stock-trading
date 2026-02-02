package risk

import (
	"math"
	"sort"
)

// CalculateComprehensiveTargets runs all applicable target methods and finds consensus.
//
// Example:
//
//	params := ComprehensiveTargetParams{
//	    EntryPrice: 52000,
//	    StopLoss: 49000,
//	    IsLong: true,
//	    ATR: 2500,
//	    ResistanceLevels: []float64{55000, 58000, 61000, 65000},
//	}
//	result := CalculateComprehensiveTargets(params)
func CalculateComprehensiveTargets(params ComprehensiveTargetParams) ComprehensiveResult {
	allMethods := make(map[string]interface{})

	rResult := CalculateRMultipleTargets(params.EntryPrice, params.StopLoss, nil, params.IsLong)
	allMethods["r_multiple"] = rResult

	if params.ATR > 0 {
		atrResult := CalculateATRTargets(params.EntryPrice, params.ATR, nil, params.IsLong)
		allMethods["atr"] = atrResult
	}

	if len(params.ResistanceLevels) > 0 {
		techResult := CalculateTechnicalTargets(params.EntryPrice, params.StopLoss, params.ResistanceLevels, params.IsLong)
		if len(techResult.Targets) > 0 {
			allMethods["technical"] = techResult
		}
	}

	if params.FibParams != nil {
		fibResult := CalculateFibonacciExtensions(*params.FibParams)
		if len(fibResult.Targets) > 0 {
			allMethods["fibonacci"] = fibResult
		}
	}

	if params.MeasuredParams != nil {
		measuredResult := CalculateMeasuredMove(*params.MeasuredParams)
		if len(measuredResult.Targets) > 0 {
			allMethods["measured_move"] = measuredResult
		}
	}

	consensus := findConsensusTargets(allMethods)

	strategy := generateTargetStrategy(params.EntryPrice, params.StopLoss, consensus)

	return ComprehensiveResult{
		EntryPrice:          params.EntryPrice,
		StopLoss:            params.StopLoss,
		AllMethods:          allMethods,
		ConsensusTargets:    consensus,
		RecommendedStrategy: strategy,
	}
}

// findConsensusTargets groups targets within 3% of each other.
func findConsensusTargets(allMethods map[string]interface{}) []ConsensusTarget {
	type targetWithMethod struct {
		price  float64
		method string
	}

	var allTargets []targetWithMethod

	for methodName, methodData := range allMethods {
		switch result := methodData.(type) {
		case RMultipleResult:
			for _, t := range result.Targets {
				allTargets = append(allTargets, targetWithMethod{t.TargetPrice, methodName})
			}
		case ATRResult:
			for _, t := range result.Targets {
				allTargets = append(allTargets, targetWithMethod{t.TargetPrice, methodName})
			}
		case TechnicalResult:
			for _, t := range result.Targets {
				allTargets = append(allTargets, targetWithMethod{t.TargetPrice, methodName})
			}
		case FibonacciResult:
			for _, t := range result.Targets {
				allTargets = append(allTargets, targetWithMethod{t.TargetPrice, methodName})
			}
		case MeasuredMoveResult:
			for _, t := range result.Targets {
				allTargets = append(allTargets, targetWithMethod{t.TargetPrice, methodName})
			}
		}
	}

	if len(allTargets) == 0 {
		return []ConsensusTarget{}
	}

	sort.Slice(allTargets, func(i, j int) bool {
		return allTargets[i].price < allTargets[j].price
	})

	var consensusGroups [][]targetWithMethod
	currentGroup := []targetWithMethod{allTargets[0]}

	for i := 1; i < len(allTargets); i++ {
		groupAvg := 0.0
		for _, t := range currentGroup {
			groupAvg += t.price
		}
		groupAvg /= float64(len(currentGroup))

		percentDiff := math.Abs(allTargets[i].price-groupAvg) / groupAvg

		if percentDiff <= 0.03 {
			currentGroup = append(currentGroup, allTargets[i])
		} else {
			if len(currentGroup) >= 2 {
				consensusGroups = append(consensusGroups, currentGroup)
			}
			currentGroup = []targetWithMethod{allTargets[i]}
		}
	}

	if len(currentGroup) >= 2 {
		consensusGroups = append(consensusGroups, currentGroup)
	}

	var consensusTargets []ConsensusTarget
	for i, group := range consensusGroups {
		avgPrice := 0.0
		methodMap := make(map[string]bool)

		for _, t := range group {
			avgPrice += t.price
			methodMap[t.method] = true
		}
		avgPrice /= float64(len(group))

		var methods []string
		for method := range methodMap {
			methods = append(methods, method)
		}

		confidence := "Moderate"
		if len(methods) >= 3 {
			confidence = "High"
		}

		consensusTargets = append(consensusTargets, ConsensusTarget{
			ConsensusNumber: i + 1,
			TargetPrice:     roundToNearest100(avgPrice),
			NumMethodsAgree: len(methods),
			Methods:         methods,
			Confidence:      confidence,
		})
	}

	return consensusTargets
}

// generateTargetStrategy creates recommended scaling approach.
func generateTargetStrategy(entry, stop float64, consensus []ConsensusTarget) TargetStrategy {
	risk := math.Abs(entry - stop)

	if len(consensus) == 0 {
		return TargetStrategy{
			Targets: []StrategyTarget{
				{
					Name:        "target_1",
					Price:       roundToNearest100(entry + risk*2),
					SellPercent: 25,
					RMultiple:   2.0,
					Rationale:   "2R default target",
				},
				{
					Name:        "target_2",
					Price:       roundToNearest100(entry + risk*3),
					SellPercent: 25,
					RMultiple:   3.0,
					Rationale:   "3R default target",
				},
				{
					Name:        "trailing",
					Price:       0,
					SellPercent: 50,
					Rationale:   "Trail remaining 50%",
				},
			},
		}
	}

	var targets []StrategyTarget

	if len(consensus) >= 1 {
		targets = append(targets, StrategyTarget{
			Name:         "target_1",
			Price:        consensus[0].TargetPrice,
			SellPercent:  25,
			RMultiple:    (consensus[0].TargetPrice - entry) / risk,
			MethodsAgree: consensus[0].Methods,
			Confidence:   consensus[0].Confidence,
		})
	}

	if len(consensus) >= 2 {
		targets = append(targets, StrategyTarget{
			Name:         "target_2",
			Price:        consensus[1].TargetPrice,
			SellPercent:  25,
			RMultiple:    (consensus[1].TargetPrice - entry) / risk,
			MethodsAgree: consensus[1].Methods,
			Confidence:   consensus[1].Confidence,
		})
	}

	if len(consensus) >= 3 {
		targets = append(targets, StrategyTarget{
			Name:         "target_3",
			Price:        consensus[2].TargetPrice,
			SellPercent:  25,
			RMultiple:    (consensus[2].TargetPrice - entry) / risk,
			MethodsAgree: consensus[2].Methods,
			Confidence:   consensus[2].Confidence,
		})

		targets = append(targets, StrategyTarget{
			Name:        "target_4",
			Price:       0,
			SellPercent: 25,
			Rationale:   "Trail final 25%",
		})
	} else {
		remainingPercent := 100
		for _, t := range targets {
			remainingPercent -= t.SellPercent
		}

		targets = append(targets, StrategyTarget{
			Name:        "trailing",
			Price:       0,
			SellPercent: remainingPercent,
			Rationale:   "Trail remaining position",
		})
	}

	return TargetStrategy{Targets: targets}
}
