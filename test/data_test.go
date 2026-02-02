// Package test contains all unit tests for the GST Trading System.
package test

import (
	"testing"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/data"
)

// --- Data Layer Tests ---

func TestOHLCV_TypicalPrice(t *testing.T) {
	bar := data.NewOHLCV(time.Now(), 100, 110, 90, 105, 1000)
	expected := (110.0 + 90.0 + 105.0) / 3 // 101.67
	got := bar.TypicalPrice()

	if got != expected {
		t.Errorf("TypicalPrice() = %v, want %v", got, expected)
	}
}

func TestOHLCV_IsBullish(t *testing.T) {
	bullish := data.NewOHLCV(time.Now(), 100, 110, 95, 108, 1000)
	bearish := data.NewOHLCV(time.Now(), 100, 105, 90, 92, 1000)
	doji := data.NewOHLCV(time.Now(), 100, 105, 95, 100, 1000)

	if !bullish.IsBullish() {
		t.Error("Expected bullish bar to return true")
	}
	if bullish.IsBearish() {
		t.Error("Bullish bar should not be bearish")
	}
	if !bearish.IsBearish() {
		t.Error("Expected bearish bar to return true")
	}
	if doji.IsBullish() || doji.IsBearish() {
		t.Error("Doji should be neither bullish nor bearish")
	}
}

func TestSeries_Append(t *testing.T) {
	s := data.NewSeries(3)

	for i := 0; i < 5; i++ {
		bar := data.NewOHLCV(time.Now(), float64(i), float64(i+1), float64(i-1), float64(i), 100)
		s.Append(bar)
	}

	if s.Len() != 3 {
		t.Errorf("Series length = %d, want 3", s.Len())
	}

	// First bar should be index 2 (oldest removed)
	first, _ := s.Get(0)
	if first.Open != 2 {
		t.Errorf("First bar Open = %v, want 2", first.Open)
	}
}

func TestSeries_LastN(t *testing.T) {
	s := data.NewSeries(100)
	prices := []float64{50000, 51000, 52000, 53000, 54000}

	for _, p := range prices {
		s.Append(data.NewOHLCV(time.Now(), p, p+100, p-100, p, 1000))
	}

	last3, err := s.LastN(3)
	if err != nil {
		t.Fatalf("LastN(3) error: %v", err)
	}

	if len(last3) != 3 {
		t.Errorf("LastN(3) returned %d bars, want 3", len(last3))
	}

	// Should be [52000, 53000, 54000]
	if last3[0].Close != 52000 || last3[1].Close != 53000 || last3[2].Close != 54000 {
		t.Errorf("LastN(3) returned wrong values: %v, %v, %v", last3[0].Close, last3[1].Close, last3[2].Close)
	}
}

func TestSeries_InsufficientData(t *testing.T) {
	s := data.NewSeries(100)
	s.Append(data.NewOHLCV(time.Now(), 100, 110, 90, 105, 1000))

	_, err := s.LastN(5)
	if err != data.ErrInsufficientData {
		t.Errorf("Expected ErrInsufficientData, got %v", err)
	}
}

func TestSeries_HighestHighLowestLow(t *testing.T) {
	s := data.NewSeries(100)
	bars := []struct{ h, l float64 }{
		{110, 90},
		{115, 95},
		{108, 88},
		{120, 92},
		{112, 85},
	}

	for _, b := range bars {
		s.Append(data.NewOHLCV(time.Now(), 100, b.h, b.l, 100, 1000))
	}

	highest, err := s.HighestHigh(3)
	if err != nil {
		t.Fatalf("HighestHigh error: %v", err)
	}
	if highest != 120 {
		t.Errorf("HighestHigh(3) = %v, want 120", highest)
	}

	lowest, err := s.LowestLow(3)
	if err != nil {
		t.Fatalf("LowestLow error: %v", err)
	}
	if lowest != 85 {
		t.Errorf("LowestLow(3) = %v, want 85", lowest)
	}
}

func TestSeries_Closes(t *testing.T) {
	s := data.NewSeries(100)
	expected := []float64{50000, 51000, 52000}

	for _, c := range expected {
		s.Append(data.NewOHLCV(time.Now(), c, c+100, c-100, c, 1000))
	}

	closes := s.Closes()
	for i, want := range expected {
		if closes[i] != want {
			t.Errorf("Closes()[%d] = %v, want %v", i, closes[i], want)
		}
	}
}
