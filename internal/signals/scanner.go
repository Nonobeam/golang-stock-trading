package signals

import "github.com/nonobeam/golang-stock-trading/internal/logger"

// SignalScanner orchestrates signal detection across multiple detectors and symbols.
type SignalScanner struct {
	pullbackDetector      *PullbackDetector
	breakoutDetector      *BreakoutDetector
	crossoverDetector     *CrossoverDetector
	meanReversionDetector *MeanReversionDetector
}

// NewSignalScanner creates a new signal scanner with configured detectors.
func NewSignalScanner(
	pullbackCfg PullbackConfig,
	breakoutCfg BreakoutConfig,
	crossoverCfg CrossoverConfig,
	meanReversionCfg MeanReversionConfig,
) *SignalScanner {
	return &SignalScanner{
		pullbackDetector:      NewPullbackDetector(pullbackCfg),
		breakoutDetector:      NewBreakoutDetector(breakoutCfg),
		crossoverDetector:     NewCrossoverDetector(crossoverCfg),
		meanReversionDetector: NewMeanReversionDetector(meanReversionCfg),
	}
}

// NewDefaultSignalScanner creates a scanner with default configurations.
func NewDefaultSignalScanner() *SignalScanner {
	return NewSignalScanner(
		DefaultPullbackConfig(),
		DefaultBreakoutConfig(),
		DefaultCrossoverConfig(),
		DefaultMeanReversionConfig(),
	)
}

// ScanForSignals scans multiple symbols for entry signals.
func (s *SignalScanner) ScanForSignals(symbols []string, dataProvider DataProvider) ([]EntrySignal, error) {
	signals := []EntrySignal{}

	for _, symbol := range symbols {
		// Fetch market data for symbol
		data, err := dataProvider.GetDailyData(symbol)
		if err != nil {
			logger.Warn().
				Str("symbol", symbol).
				Err(err).
				Msg("Failed to get market data, skipping symbol")
			continue
		}

		// Get volume statistics
		volumeMA, volumePercentile, err := dataProvider.GetVolumeStats(symbol)
		if err == nil {
			data.VolumeMA20 = volumeMA
			data.VolumePercentile = volumePercentile
		}

		// Run pullback detector
		pullbackResult, err := s.pullbackDetector.Scan(data)
		if err != nil {
			logger.Error().
				Str("symbol", symbol).
				Err(err).
				Msg("Pullback detector error")
		} else if pullbackResult.SetupDetected {
			signals = append(signals, *pullbackResult.Signal)
			logger.Info().
				Str("symbol", symbol).
				Str("type", "Pullback").
				Str("confidence", string(pullbackResult.Signal.Confidence)).
				Msg("Signal detected")
		}

		// Run breakout detector
		breakoutResult, err := s.breakoutDetector.Scan(data)
		if err != nil {
			logger.Error().
				Str("symbol", symbol).
				Err(err).
				Msg("Breakout detector error")
		} else if breakoutResult.SetupDetected {
			signals = append(signals, *breakoutResult.Signal)
			logger.Info().
				Str("symbol", symbol).
				Str("type", "Breakout").
				Str("confidence", string(breakoutResult.Signal.Confidence)).
				Msg("Signal detected")
		}

		// Run crossover detector
		crossoverResult, err := s.crossoverDetector.Scan(data)
		if err != nil {
			logger.Error().
				Str("symbol", symbol).
				Err(err).
				Msg("Crossover detector error")
		} else if crossoverResult.SetupDetected {
			signals = append(signals, *crossoverResult.Signal)
			logger.Info().
				Str("symbol", symbol).
				Str("type", "Crossover").
				Str("confidence", string(crossoverResult.Signal.Confidence)).
				Msg("Signal detected")
		}

		// Run mean reversion detector
		// First get support/resistance levels and market regime
		srLevels, err := dataProvider.GetSupportResistance(symbol)
		if err != nil {
			logger.Warn().
				Str("symbol", symbol).
				Err(err).
				Msg("Failed to get support/resistance levels")
			srLevels = nil
		}

		marketRegime, err := dataProvider.GetMarketRegime(symbol)
		if err != nil {
			logger.Warn().
				Str("symbol", symbol).
				Err(err).
				Msg("Failed to get market regime")
			marketRegime = map[string]interface{}{}
		}

		meanRevResult, err := s.meanReversionDetector.Scan(data, srLevels, marketRegime)
		if err != nil {
			logger.Error().
				Str("symbol", symbol).
				Err(err).
				Msg("Mean reversion detector error")
		} else if meanRevResult.SetupDetected {
			signals = append(signals, *meanRevResult.Signal)
			logger.Info().
				Str("symbol", symbol).
				Str("type", "MeanReversion").
				Str("confidence", string(meanRevResult.Signal.Confidence)).
				Msg("Signal detected")
		}
	}

	// Rank signals by confidence and type
	rankedSignals := RankSignals(signals)

	logger.Info().
		Int("total_scanned", len(symbols)).
		Int("signals_found", len(rankedSignals)).
		Msg("Signal scanning complete")

	return rankedSignals, nil
}

// RankSignals sorts signals by confidence level and type preference.
func RankSignals(signals []EntrySignal) []EntrySignal {
	if len(signals) == 0 {
		return signals
	}

	// Sort by confidence (Very High > High > Moderate > Low)
	// Then by type (Pullback > Breakout within same confidence)

	ranked := make([]EntrySignal, len(signals))
	copy(ranked, signals)

	// Simple bubble sort for now (can optimize later)
	for i := 0; i < len(ranked)-1; i++ {
		for j := i + 1; j < len(ranked); j++ {
			if shouldSwap(ranked[i], ranked[j]) {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}

	return ranked
}

// shouldSwap determines if signal2 should be ranked higher than signal1.
func shouldSwap(signal1, signal2 EntrySignal) bool {
	// Compare confidence levels first
	conf1Score := confidenceToScore(signal1.Confidence)
	conf2Score := confidenceToScore(signal2.Confidence)

	if conf2Score > conf1Score {
		return true // signal2 has higher confidence
	}

	if conf2Score == conf1Score {
		// Same confidence - use type preference
		// Priority: Pullback > Breakout > Crossover > MeanReversion
		type1Score := signalTypeToScore(signal1.Type)
		type2Score := signalTypeToScore(signal2.Type)

		if type2Score > type1Score {
			return true
		}
	}

	return false
}

// signalTypeToScore converts signal type to numeric score for ranking.
func signalTypeToScore(signalType SignalType) int {
	switch signalType {
	case SignalTypePullback:
		return 4
	case SignalTypeBreakout:
		return 3
	case SignalTypeCrossover:
		return 2
	case SignalTypeMeanReversion:
		return 1
	default:
		return 0
	}
}

// confidenceToScore converts confidence level to numeric score for sorting.
func confidenceToScore(confidence ConfidenceLevel) int {
	switch confidence {
	case ConfidenceVeryHigh:
		return 4
	case ConfidenceHigh:
		return 3
	case ConfidenceModerate:
		return 2
	case ConfidenceLow:
		return 1
	default:
		return 0
	}
}

// FilterByConfidence filters signals to only those meeting minimum confidence.
func FilterByConfidence(signals []EntrySignal, minConfidence ConfidenceLevel) []EntrySignal {
	minScore := confidenceToScore(minConfidence)
	filtered := []EntrySignal{}

	for _, signal := range signals {
		if confidenceToScore(signal.Confidence) >= minScore {
			filtered = append(filtered, signal)
		}
	}

	return filtered
}

// ScoredSignal combines an entry signal with its trade score.
type ScoredSignal struct {
	Signal      EntrySignal
	Score       int
	ShouldTrade bool
}

// ScoreSignals integrates signals with trade scoring system.
// This is a placeholder for integration with Phase 3.2 TradeScorer.
func (s *SignalScanner) ScoreSignals(signals []EntrySignal) []ScoredSignal {
	scoredSignals := []ScoredSignal{}

	for _, signal := range signals {
		// Placeholder: In Phase 3.2, this will call TradeScorer.Score()
		// For now, assign baseline scores based on confidence
		baseScore := 7

		switch signal.Confidence {
		case ConfidenceVeryHigh:
			baseScore = 10
		case ConfidenceHigh:
			baseScore = 9
		case ConfidenceModerate:
			baseScore = 7
		case ConfidenceLow:
			baseScore = 6
		}

		scoredSignal := ScoredSignal{
			Signal:      signal,
			Score:       baseScore,
			ShouldTrade: baseScore >= 7,
		}

		scoredSignals = append(scoredSignals, scoredSignal)
	}

	return scoredSignals
}
