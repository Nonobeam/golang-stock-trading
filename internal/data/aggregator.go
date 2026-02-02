package data

import (
	"time"
)

// AggregateToWeekly converts a daily OHLCV series to weekly timeframe.
// Week boundaries: Monday = open, Friday = close.
// Returns a new Series containing weekly bars.
func AggregateToWeekly(dailySeries *Series) *Series {
	dailyBars := dailySeries.All()
	if len(dailyBars) == 0 {
		return NewSeries(0)
	}

	weeklyBars := make([]OHLCV, 0)
	var currentWeekBars []OHLCV

	for i, bar := range dailyBars {
		currentWeekBars = append(currentWeekBars, bar)

		// Check if we've reached end of week (Friday) or end of data
		isEndOfWeek := bar.Timestamp.Weekday() == time.Friday
		isLastBar := i == len(dailyBars)-1

		if isEndOfWeek || isLastBar {
			// Aggregate the week
			if len(currentWeekBars) > 0 {
				weeklyBar := aggregateWeek(currentWeekBars)
				weeklyBars = append(weeklyBars, weeklyBar)
				currentWeekBars = make([]OHLCV, 0)
			}
		}
	}

	// Create new series with weekly bars
	weeklySeries := NewSeries(len(weeklyBars))
	for _, bar := range weeklyBars {
		weeklySeries.Append(bar)
	}

	return weeklySeries
}

// aggregateWeek combines multiple daily bars into a single weekly bar.
func aggregateWeek(dailyBars []OHLCV) OHLCV {
	if len(dailyBars) == 0 {
		return OHLCV{}
	}

	// Open: First bar's open
	open := dailyBars[0].Open

	// High: Highest high of the week
	high := dailyBars[0].High
	for _, bar := range dailyBars {
		if bar.High > high {
			high = bar.High
		}
	}

	// Low: Lowest low of the week
	low := dailyBars[0].Low
	for _, bar := range dailyBars {
		if bar.Low < low {
			low = bar.Low
		}
	}

	// Close: Last bar's close
	close := dailyBars[len(dailyBars)-1].Close

	// Volume: Sum of all volumes
	volume := 0.0
	for _, bar := range dailyBars {
		volume += bar.Volume
	}

	// Timestamp: Last bar's timestamp (end of week)
	timestamp := dailyBars[len(dailyBars)-1].Timestamp

	return NewOHLCV(timestamp, open, high, low, close, volume)
}

// AggregateToTimeframe converts daily series to any custom timeframe (in days).
// For example, timeframeDays=5 creates weekly bars, timeframeDays=30 creates monthly bars.
func AggregateToTimeframe(dailySeries *Series, timeframeDays int) *Series {
	if timeframeDays <= 1 {
		// Return copy of daily series
		return dailySeries
	}

	dailyBars := dailySeries.All()
	if len(dailyBars) == 0 {
		return NewSeries(0)
	}

	aggregatedBars := make([]OHLCV, 0)
	currentBatch := make([]OHLCV, 0)

	for _, bar := range dailyBars {
		currentBatch = append(currentBatch, bar)

		if len(currentBatch) >= timeframeDays {
			// Aggregate the batch
			aggregatedBar := aggregateWeek(currentBatch) // Reuse aggregation logic
			aggregatedBars = append(aggregatedBars, aggregatedBar)
			currentBatch = make([]OHLCV, 0)
		}
	}

	// Handle remaining bars
	if len(currentBatch) > 0 {
		aggregatedBar := aggregateWeek(currentBatch)
		aggregatedBars = append(aggregatedBars, aggregatedBar)
	}

	// Create new series
	aggregatedSeries := NewSeries(len(aggregatedBars))
	for _, bar := range aggregatedBars {
		aggregatedSeries.Append(bar)
	}

	return aggregatedSeries
}
