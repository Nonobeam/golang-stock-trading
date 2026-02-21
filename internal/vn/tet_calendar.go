// Package vn provides Vietnamese market calendar utilities.
package vn

import (
	"time"
)

// TetHolidayInfo contains information about Tet (Lunar New Year) holiday.
type TetHolidayInfo struct {
	TetDate       time.Time
	MarketClosure int // Days market is closed
	PreTetDate    time.Time // 7-10 days before Tet
}

// GetTetDate returns the Tet (Lunar New Year) date for a given year.
// Vietnamese Lunar New Year typically falls between late January and mid-February.
func GetTetDate(year int) time.Time {
	// Lunar calendar dates for Tet (approximation - in production use lunar calendar library)
	tetDates := map[int]time.Time{
		2024: time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC),
		2025: time.Date(2025, 1, 29, 0, 0, 0, 0, time.UTC),
		2026: time.Date(2026, 2, 17, 0, 0, 0, 0, time.UTC),
		2027: time.Date(2027, 2, 6, 0, 0, 0, 0, time.UTC),
		2028: time.Date(2028, 1, 26, 0, 0, 0, 0, time.UTC),
	}
	
	if date, ok := tetDates[year]; ok {
		return date
	}
	
	// Default fallback (not accurate)
	return time.Date(year, 2, 1, 0, 0, 0, 0, time.UTC)
}

// CheckTetHolidayAdjustment checks if current date is within pre-Tet window.
// Returns adjustment info if positions should be reduced, nil otherwise.
func CheckTetHolidayAdjustment() *TetAdjustment {
	now := time.Now()
	tetDate := GetTetDate(now.Year())
	
	daysToTet := int(tetDate.Sub(now).Hours() / 24)
	
	// If 5-10 days before Tet, trigger position reduction
	if daysToTet >= 5 && daysToTet <= 10 {
		return &TetAdjustment{
			Action:        "REDUCE_POSITIONS",
			TargetSize:    0.50, // Reduce to 50%
			Reason:        "Tet holiday approaching - 7-10 day market closure",
			DaysToTet:     daysToTet,
			TetDate:       tetDate,
		}
	}
	
	return nil
}

// TetAdjustment represents a required adjustment for Tet holiday.
type TetAdjustment struct {
	Action     string
	TargetSize float64    // Target position size as percentage (0.50 = 50%)
	Reason     string
	DaysToTet  int
	TetDate    time.Time
}
