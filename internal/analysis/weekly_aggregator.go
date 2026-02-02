package analysis

import (
	"fmt"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/data"
)

// WeeklyBar represents aggregated weekly OHLCV data.
type WeeklyBar struct {
	Symbol  string
	WeekEnd time.Time // Friday or last trading day of week
	Open    float64
	High    float64
	Low     float64
	Close   float64
	Volume  int64
}

// WeeklyAggregator converts daily bars to weekly bars.
type WeeklyAggregator struct{}

// NewWeeklyAggregator creates a new weekly aggregator.
func NewWeeklyAggregator() *WeeklyAggregator {
	return &WeeklyAggregator{}
}

// AggregateToWeekly converts daily OHLCV bars to weekly bars.
// Groups by ISO week, using Friday close (or last trading day of week).
func (a *WeeklyAggregator) AggregateToWeekly(dailyBars []data.OHLCV) ([]WeeklyBar, error) {
	if len(dailyBars) == 0 {
		return nil, fmt.Errorf("no daily bars provided")
	}

	// Group bars by ISO week
	weekMap := make(map[string]*WeeklyBar)
	weekOrder := []string{} // Track order of weeks

	for _, bar := range dailyBars {
		// Get ISO week identifier (YYYY-WW format)
		year, week := bar.Timestamp.ISOWeek()
		weekKey := fmt.Sprintf("%d-W%02d", year, week)

		weekBar, exists := weekMap[weekKey]
		if !exists {
			// Create new weekly bar
			weekBar = &WeeklyBar{
				Symbol:  "", // Will be set from first bar
				WeekEnd: getWeekEnd(bar.Timestamp),
				Open:    bar.Open,
				High:    bar.High,
				Low:     bar.Low,
				Close:   bar.Close,
				Volume:  int64(bar.Volume),
			}
			weekMap[weekKey] = weekBar
			weekOrder = append(weekOrder, weekKey)
		} else {
			// Update existing weekly bar
			if bar.High > weekBar.High {
				weekBar.High = bar.High
			}
			if bar.Low < weekBar.Low {
				weekBar.Low = bar.Low
			}
			// Close = last day's close
			weekBar.Close = bar.Close
			weekBar.Volume += int64(bar.Volume)

			// Update week end to latest date
			if bar.Timestamp.After(weekBar.WeekEnd) {
				weekBar.WeekEnd = bar.Timestamp
			}
		}
	}

	// Convert map to ordered slice
	weeklyBars := make([]WeeklyBar, 0, len(weekOrder))
	for _, weekKey := range weekOrder {
		weeklyBars = append(weeklyBars, *weekMap[weekKey])
	}

	return weeklyBars, nil
}

// CalculateWeeklySMA200 calculates 200-period SMA on weekly closes.
// Requires 200 weeks ≈ 3.8 years of data.
// Returns 0 if insufficient data.
func (a *WeeklyAggregator) CalculateWeeklySMA200(weeklyBars []WeeklyBar) float64 {
	if len(weeklyBars) < 200 {
		return 0
	}

	// Extract last 200 weekly closes
	closes := make([]float64, 200)
	startIdx := len(weeklyBars) - 200
	for i := 0; i < 200; i++ {
		closes[i] = weeklyBars[startIdx+i].Close
	}

	// Calculate SMA
	sum := 0.0
	for _, close := range closes {
		sum += close
	}

	return sum / 200.0
}

// GetWeeklySMA200 is a convenience method that aggregates and calculates in one call.
func (a *WeeklyAggregator) GetWeeklySMA200(dailyBars []data.OHLCV) (float64, error) {
	weeklyBars, err := a.AggregateToWeekly(dailyBars)
	if err != nil {
		return 0, err
	}

	sma := a.CalculateWeeklySMA200(weeklyBars)
	if sma == 0 {
		return 0, fmt.Errorf("insufficient weekly data for SMA 200 (need 200 weeks, have %d)", len(weeklyBars))
	}

	return sma, nil
}

// getWeekEnd returns Friday of the week, or the actual date if it's the last trading day.
func getWeekEnd(t time.Time) time.Time {
	// Find Friday of this week
	weekday := t.Weekday()
	daysUntilFriday := (time.Friday - weekday + 7) % 7

	if daysUntilFriday == 0 {
		// Already Friday
		return t
	}

	// If we're past Friday, use actual date (holiday case)
	if weekday == time.Saturday || weekday == time.Sunday {
		return t
	}

	// Move to Friday
	friday := t.AddDate(0, 0, int(daysUntilFriday))
	return friday
}
