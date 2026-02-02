package test

import (
	"testing"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/analysis"
	"github.com/nonobeam/golang-stock-trading/internal/data"
)

func TestWeeklyAggregation(t *testing.T) {
	aggregator := analysis.NewWeeklyAggregator()

	// Create 5 daily bars in one week (Mon-Fri)
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // Monday
	dailyBars := []data.OHLCV{
		{Timestamp: baseDate, Open: 100, High: 105, Low: 99, Close: 103, Volume: 1000},                   // Mon
		{Timestamp: baseDate.AddDate(0, 0, 1), Open: 103, High: 107, Low: 102, Close: 106, Volume: 1200}, // Tue
		{Timestamp: baseDate.AddDate(0, 0, 2), Open: 106, High: 110, Low: 105, Close: 108, Volume: 1500}, // Wed
		{Timestamp: baseDate.AddDate(0, 0, 3), Open: 108, High: 112, Low: 107, Close: 110, Volume: 1300}, // Thu
		{Timestamp: baseDate.AddDate(0, 0, 4), Open: 110, High: 115, Low: 109, Close: 112, Volume: 1600}, // Fri
	}

	weeklyBars, err := aggregator.AggregateToWeekly(dailyBars)
	if err != nil {
		t.Fatalf("AggregateToWeekly() error = %v", err)
	}

	if len(weeklyBars) != 1 {
		t.Errorf("Expected 1 weekly bar, got %d", len(weeklyBars))
	}

	weekly := weeklyBars[0]

	// Verify OHLC
	if weekly.Open != 100 {
		t.Errorf("Weekly Open = %v, want 100 (Monday open)", weekly.Open)
	}
	if weekly.High != 115 {
		t.Errorf("Weekly High = %v, want 115 (max of all highs)", weekly.High)
	}
	if weekly.Low != 99 {
		t.Errorf("Weekly Low = %v, want 99 (min of all lows)", weekly.Low)
	}
	if weekly.Close != 112 {
		t.Errorf("Weekly Close = %v, want 112 (Friday close)", weekly.Close)
	}

	// Verify volume
	expectedVolume := int64(1000 + 1200 + 1500 + 1300 + 1600)
	if weekly.Volume != expectedVolume {
		t.Errorf("Weekly Volume = %v, want %v", weekly.Volume, expectedVolume)
	}

	t.Logf("✓ Weekly bar aggregated correctly: O=%.0f H=%.0f L=%.0f C=%.0f V=%d",
		weekly.Open, weekly.High, weekly.Low, weekly.Close, weekly.Volume)
}

func TestWeeklySMA200(t *testing.T) {
	aggregator := analysis.NewWeeklyAggregator()

	// Create 200 weekly bars with predictable closes
	weeklyBars := make([]analysis.WeeklyBar, 200)
	baseDate := time.Date(2020, 1, 3, 0, 0, 0, 0, time.UTC) // Friday

	for i := 0; i < 200; i++ {
		weeklyBars[i] = analysis.WeeklyBar{
			Symbol:  "TEST",
			WeekEnd: baseDate.AddDate(0, 0, i*7),
			Open:    100,
			High:    105,
			Low:     95,
			Close:   float64(100 + i), // Incrementing closes
			Volume:  10000,
		}
	}

	sma := aggregator.CalculateWeeklySMA200(weeklyBars)

	// Expected SMA of 100, 101, 102, ..., 299
	// SMA = (100 + 101 + ... + 299) / 200
	// This is arithmetic sequence: sum = n/2 * (first + last) = 200/2 * (100 + 299) = 39900
	expectedSMA := 39900.0 / 200.0

	if sma != expectedSMA {
		t.Errorf("Weekly SMA 200 = %v, want %v", sma, expectedSMA)
	}

	t.Logf("✓ Weekly SMA 200 calculated correctly: %.2f", sma)
}

func TestWeeklySMA200_InsufficientData(t *testing.T) {
	aggregator := analysis.NewWeeklyAggregator()

	// Only 100 weekly bars (insufficient for 200-period SMA)
	weeklyBars := make([]analysis.WeeklyBar, 100)
	for i := 0; i < 100; i++ {
		weeklyBars[i] = analysis.WeeklyBar{Close: 100}
	}

	sma := aggregator.CalculateWeeklySMA200(weeklyBars)

	if sma != 0 {
		t.Errorf("Expected SMA = 0 for insufficient data, got %v", sma)
	}

	t.Logf("✓ Correctly returns 0 for insufficient data")
}

func TestGetWeeklySMA200_EndToEnd(t *testing.T) {
	aggregator := analysis.NewWeeklyAggregator()

	// Create ~4 years of daily data (1400 calendar days → ~1000 trading days ≈ 200 weeks)
	dailyBars := make([]data.OHLCV, 0, 1400)
	baseDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 1400; i++ {
		// Skip weekends
		date := baseDate.AddDate(0, 0, i)
		if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
			continue
		}

		dailyBars = append(dailyBars, data.OHLCV{
			Timestamp: date,
			Open:      100,
			High:      105,
			Low:       95,
			Close:     100,
			Volume:    10000,
		})
	}

	sma, err := aggregator.GetWeeklySMA200(dailyBars)
	if err != nil {
		t.Fatalf("GetWeeklySMA200() error = %v", err)
	}

	// Should be close to 100 (all closes are 100)
	if sma < 99 || sma > 101 {
		t.Errorf("Weekly SMA 200 = %v, expected ~100", sma)
	}

	t.Logf("✓ End-to-end weekly SMA 200: %.2f", sma)
}
