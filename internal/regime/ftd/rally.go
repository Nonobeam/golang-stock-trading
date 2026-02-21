package ftd

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

// RallyAttemptTracker tracks the state of market rally attempts (Day 1 to Day 7).
type RallyAttemptTracker struct {
	repo         *Repository
	downtrendDet *DowntrendDetector
	scorer       *FTDScorer
}

// NewRallyAttemptTracker creates a new rally attempt tracker.
func NewRallyAttemptTracker(repo *Repository, downtrendDet *DowntrendDetector) *RallyAttemptTracker {
	return &RallyAttemptTracker{
		repo:         repo,
		downtrendDet: downtrendDet,
		scorer:       NewFTDScorer(),
	}
}

// UpdateState analyzes the latest market data and updates the rally attempt state.
func (t *RallyAttemptTracker) UpdateState(ctx context.Context, date time.Time) error {
	// 1. Get current day's data
	current, err := t.repo.GetMarketRegime(ctx, date)
	if err != nil {
		return fmt.Errorf("failed to get current regime: %w", err)
	}
	if current == nil {
		return fmt.Errorf("no data for date %s", date)
	}

	// 2. Get previous day's data
	// prevDate := date.AddDate(0, 0, -1) // Simplified, ideally check trading calendar
	// We'll search back up to 5 days to find previous trading day
	var prev *MarketRegime
	for i := 1; i <= 5; i++ {
		p, err := t.repo.GetMarketRegime(ctx, date.AddDate(0, 0, -i))
		if err == nil && p != nil {
			prev = p
			// prevDate = p.Date // Unused
			break
		}
	}

	if prev == nil {
		// Can't track without history
		return nil
	}

	// 3. Determine current state
	// If previous day had a rally attempt count, we continue from there
	currentRallyDay := 0
	if prev.RallyAttemptDay != nil {
		currentRallyDay = *prev.RallyAttemptDay
	}

	baselineLow := 0.0
	if prev.RallyAttemptBaseline != nil {
		baselineLow = *prev.RallyAttemptBaseline
	} else if prev.Low != nil {
		baselineLow = *prev.Low
	} else {
		baselineLow = prev.IndexValue
	}

	// Logic Branching
	if currentRallyDay == 0 {
		// Case A: No active rally. Check for Day 1.
		// Requirement: Market in Downtrend/Correction AND Close > Prev Close
		// Note: We might want to be lenient on "Downtrend" status if we just made a new low.
		
		// Check if we made a new low recently (Day 1 definition: First up day coming off a low)
		// We look at the Low of the day. Was it a new low? Or was yesterday a new low?
		
		isUpDay := current.IndexValue > prev.IndexValue
		if isUpDay {
			// Check if yesterday or today made a new low (20-day low)
			// Get history
			hist, err := t.repo.GetMarketRegimes(ctx, date.AddDate(0, 0, -30), date)
			if err == nil {
				prices := extractLows(hist)
				// Check if the low of this sequence is recent
				lowestIndex := findLowestIndex(prices)
				daysSinceLow := len(prices) - 1 - lowestIndex
				
				if daysSinceLow <= 1 { // Low was today or yesterday
					// FOUND DAY 1
					day1 := 1
					current.RallyAttemptDay = &day1
					
					// Set baseline low for Day 2+ validation
					currentLow := current.IndexValue
					if current.Low != nil {
						currentLow = *current.Low
					}
					
					// O'Neil methodology: "The low of the rally day is the absolute bottom."
					current.RallyAttemptBaseline = &currentLow
					
					logger.Info().Time("date", date).Msg("Rally Attempt Day 1 Detected")
				}
			}
		}
	} else {
		// Case B: Active Rally (Day 1+)
		// Validate: Price must not undercut the Day 1 Low
		
		// Check for undercut
		// Check Low vs Baseline
		currentLow := current.IndexValue
		if current.Low != nil {
			currentLow = *current.Low
		}

		if currentLow < baselineLow {
			// FAILED - Undercut
			logger.Info().Time("date", date).
				Float64("low", currentLow).
				Float64("baseline", baselineLow).
				Msg("Rally Failed - Undercut Day 1 Low")
			current.RallyAttemptDay = nil
			current.RallyAttemptBaseline = nil
		} else {
			// HELD
			nextDay := currentRallyDay + 1
			current.RallyAttemptDay = &nextDay
			current.RallyAttemptBaseline = &baselineLow // Maintain baseline
			
			// Check for FTD Confirmation (Day 4-7+)
			if nextDay >= 4 {
				isFTD, score := t.scorer.CalculateScore(current, prev, t.repo) // Pass repo to get breadth/etc
				if isFTD {
					current.IsFTD = true
					current.FTDScore = score.Total
					current.FTDStrength = score.Strength
					
					logger.Info().
						Time("date", date).
						Int("score", score.Total).
						Str("strength", score.Strength).
						Msg("🚨 FOLLOW-THROUGH DAY CONFIRMED")
				}
			}
		}
	}

	// 4. Update DB
	return t.repo.UpsertMarketRegime(ctx, current)
}

func extractLows(regimes []*MarketRegime) []float64 {
	lows := make([]float64, len(regimes))
	for i, r := range regimes {
		if r.Low != nil {
			lows[i] = *r.Low
		} else {
			lows[i] = r.IndexValue
		}
	}
	return lows
}

func findLowestIndex(prices []float64) int {
	minIdx := -1
	minVal := math.MaxFloat64
	for i, p := range prices {
		if p < minVal {
			minVal = p
			minIdx = i
		}
	}
	return minIdx
}
