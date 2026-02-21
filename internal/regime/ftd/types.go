package ftd

import (
	"time"
)

// MarketRegime represents a daily record in market_regime_tracking table.
type MarketRegime struct {
	ID                     int64      `json:"id"`
	Date                   time.Time  `json:"date"`
	IndexValue             float64    `json:"index_value"`
	Open                   *float64   `json:"open"`
	High                   *float64   `json:"high"`
	Low                    *float64   `json:"low"`
	Volume                 int64      `json:"volume"`
	VolumeVsAvg20d         *float64   `json:"volume_vs_avg_20d"` // Percentage
	RallyAttemptDay        *int       `json:"rally_attempt_day"` // NULL, 1, 2, 3, 4-7
	RallyAttemptBaseline   *float64   `json:"rally_attempt_baseline"`
	IsFTD                  bool       `json:"is_ftd"`
	FTDStrength            string     `json:"ftd_strength"` // 'weak', 'moderate', 'strong'
	FTDScore               int        `json:"ftd_score"`
	BreadthRatio           *float64   `json:"breadth_ratio" db:"breadth_ratio"`
	LeaderParticipationScore int      `json:"leader_participation_score" db:"leader_participation_score"`
	DistributionDayCount   int        `json:"distribution_day_count" db:"distribution_day_count"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// FTDEvent represents a record in ftd_events table.
type FTDEvent struct {
	ID                        int64     `json:"id"`
	EventDate                 time.Time `json:"event_date"`
	RallyAttemptStartDate     time.Time `json:"rally_attempt_start_date"`
	DaysToFTD                 int       `json:"days_to_ftd"`
	FTDStrength               string    `json:"ftd_strength"`
	FTDScore                  int       `json:"ftd_score"`
	PatternType               string    `json:"pattern_type"`
	PriceGainPct              float64   `json:"price_gain_pct"`
	VolumeRatio               float64   `json:"volume_ratio"`
	BreadthRatio              float64   `json:"breadth_ratio"`
	LeaderScore               int       `json:"leader_score"`
	Success7d                 *float64  `json:"success_7d"`
	Success14d                *float64  `json:"success_14d"`
	Success30d                *float64  `json:"success_30d"`
	IsValidated               bool      `json:"is_validated"`
	InvalidatedByDistribution bool      `json:"invalidated_by_distribution"`
	InvalidationDate          *time.Time `json:"invalidation_date"`
	CreatedAt                 time.Time `json:"created_at"`
}

// MarketBreadth represents a daily record in market_breadth_daily table.
type MarketBreadth struct {
	ID              int64     `json:"id"`
	Date            time.Time `json:"date"`
	AdvancingStocks int       `json:"advancing_stocks"`
	DecliningStocks int       `json:"declining_stocks"`
	UnchangedStocks int       `json:"unchanged_stocks"`
	NewHighs        int       `json:"new_highs"`
	NewLows         int       `json:"new_lows"`
	SectorLeaders   string    `json:"sector_leaders"` // JSON string
	CreatedAt       time.Time `json:"created_at"`
}
