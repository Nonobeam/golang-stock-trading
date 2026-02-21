package ftd

import (
	"context"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

// FTDScoreResult holds the scoring details.
type FTDScoreResult struct {
	PriceScore   int
	VolumeScore  int
	BreadthScore int
	LeaderScore  int
	Total        int
	Strength     string
}

// FTDScorer calculates the strength of a Follow-Through Day.
type FTDScorer struct {
	patternRec *PatternRecognizer
}

// NewFTDScorer creates a new FTD scorer.
func NewFTDScorer() *FTDScorer {
	return &FTDScorer{
		patternRec: NewPatternRecognizer(),
	}
}

// CalculateScore evaluates if today is an FTD and calculates its strength.
func (s *FTDScorer) CalculateScore(current, prev *MarketRegime, repo *Repository) (bool, FTDScoreResult) {
	result := FTDScoreResult{}

	// 1. Basic Criteria Check
	// Price Gain > 1.2% (or 1.5% for safer confirmation)
	priceChange := (current.IndexValue - prev.IndexValue) / prev.IndexValue
	if priceChange < 0.012 {
		return false, result // Not an FTD
	}

	// Volume Higher than Previous Day
	if current.Volume <= prev.Volume {
		// Strict O'Neil rule: Volume must be higher.
		// Some interpretations allow if volume is huge relative to average.
		// For now, enforce strict rule.
		return false, result
	}

	// 2. Scoring

	// Price Score (30 pts)
	if priceChange >= 0.02 {
		result.PriceScore = 30
	} else if priceChange >= 0.015 {
		result.PriceScore = 25
	} else {
		result.PriceScore = 20
	}

	// Volume Score (30 pts)
	// Compare to 20-day average
	volRatio := 1.0
	if current.VolumeVsAvg20d != nil {
		volRatio = *current.VolumeVsAvg20d
	}
	
	if volRatio >= 2.0 {
		result.VolumeScore = 30
	} else if volRatio >= 1.5 {
		result.VolumeScore = 25
	} else if volRatio >= 1.2 {
		result.VolumeScore = 20
	} else if volRatio > 1.0 {
		result.VolumeScore = 10
	}

	// Breadth Score (20 pts)
	// Requires Breadth Data. Assuming it's populated in MarketRegime or fetched.
	if current.BreadthRatio != nil {
		ratio := *current.BreadthRatio
		if ratio >= 3.0 {
			result.BreadthScore = 20
		} else if ratio >= 2.0 {
			result.BreadthScore = 15
		} else if ratio >= 1.5 {
			result.BreadthScore = 10
		}
	} else {
		// Use proxy if breadth missing? Or 0.
		// Attempt to fetch if not present?
		// For now 0.
	}

	// Leader Score (20 pts)
	// Needs external input for sector performance.
	// We'll use the pre-calculated LeaderParticipationScore from MarketRegime if available.
	result.LeaderScore = current.LeaderParticipationScore

	// 3. Pattern Recognition Bonus
	// Check for Double Bottom or Inverse Head & Shoulders
	// Need recent history. Accessing via repo here would be slow inside scorer?
	// Scorer signature has repo.
	
	// Fetch history (last 60 days) for pattern recognition
	// Optimally this should be passed in or cached, but strictly following signature:
	hist, err := repo.GetMarketRegimes(context.Background(), current.Date.AddDate(0, 0, -60), current.Date)
	if err == nil {
		prices := make([]float64, len(hist))
		for i, h := range hist {
			if h.Low != nil {
				prices[i] = *h.Low 
			} else {
				prices[i] = h.IndexValue
			}
		}
		
		pattern, quality := s.patternRec.RecognizePattern(prices)
		if pattern != PatternNone {
			logger.Info().Str("pattern", string(pattern)).Int("quality", quality).Msg("FTD Pattern Recognized")
			// Add bonus points
			bonus := quality / 5 // Max 20 points
			result.Total += bonus 
		}
	}



	// Filter: Close Range (Stalling Action)
	// If Close is in the lower 40% of the daily range (High - Low), it's a stalling day, likely faltering.
	// We penalize this heavily or disqualify.
	if current.High != nil && current.Low != nil {
		rangeSize := *current.High - *current.Low
		if rangeSize > 0 {
			closePos := (current.IndexValue - *current.Low) / rangeSize
			if closePos < 0.4 {
				// Close in lower 40% -> Weak finish
				// Deduct points or disqualify?
				// O'Neil says consistent FTDs close near high.
				// We'll cap the max score or reduce it.
				// Let's reduce total by 20 pts (making it hard to reach 80 Strong)
				result.Total -= 20
				logger.Info().Float64("close_pos", closePos).Msg("FTD candidate closed in lower range (weak finish)")
			} else if closePos > 0.7 {
				// Strong finish
				result.Total += 5
			}
		}
	}

	// Total
	result.Total = result.PriceScore + result.VolumeScore + result.BreadthScore + result.LeaderScore

	// Strength
	if result.Total >= 80 {
		result.Strength = "strong"
	} else if result.Total >= 60 {
		result.Strength = "moderate"
	} else {
		result.Strength = "weak"
	}

	return true, result
}

// CalculateAvgVolume helper (duplicates ingestion logic, should consolidate)
func (s *FTDScorer) getAvgVolume(ctx context.Context, repo *Repository, date time.Time) (int64, error) {
	// ...
	return 0, nil
}
