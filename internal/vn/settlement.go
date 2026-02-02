package vn

import (
	"fmt"
	"time"
)

// SettlementResult holds settlement calculation info.
type SettlementResult struct {
	TradeDate      time.Time
	SettlementDate time.Time
	TradingDays    int // Number of trading days between
}

// CalculateSettlement calculates T+N settlement date.
// Skips weekends and known Vietnam holidays.
func CalculateSettlement(tradeDate time.Time) SettlementResult {
	settlementDays := GetSettlementDays()
	settlement := tradeDate
	daysAdded := 0

	for daysAdded < settlementDays {
		settlement = settlement.AddDate(0, 0, 1)

		// Skip weekends
		if settlement.Weekday() == time.Saturday || settlement.Weekday() == time.Sunday {
			continue
		}

		// Skip holidays
		if IsVNHoliday(settlement) {
			continue
		}

		daysAdded++
	}

	return SettlementResult{
		TradeDate:      tradeDate,
		SettlementDate: settlement,
		TradingDays:    settlementDays,
	}
}

// IsVNHoliday checks if date is a Vietnam market holiday.
// Major holidays included (dates may shift yearly for lunar calendar holidays).
func IsVNHoliday(date time.Time) bool {
	month := date.Month()
	day := date.Day()

	// Fixed holidays
	holidays := map[string]bool{
		"1-1":  true, // New Year's Day
		"4-30": true, // Reunification Day
		"5-1":  true, // Labour Day
		"9-2":  true, // National Day
	}

	key := formatDate(month, day)
	if holidays[key] {
		return true
	}

	// Lunar holidays need yearly lookup (simplified check)
	// In production, use a proper holiday calendar
	year := date.Year()

	// Tết Nguyên Đán (approximate - varies by year)
	tetDates := getTetDates(year)
	for _, tetDate := range tetDates {
		if date.Month() == tetDate.Month() && date.Day() == tetDate.Day() {
			return true
		}
	}

	// Hung Kings' Commemoration (10th day of 3rd lunar month)
	// Simplified: typically falls in April
	hungKings := getHungKingsDate(year)
	if date.Month() == hungKings.Month() && date.Day() == hungKings.Day() {
		return true
	}

	return false
}

func formatDate(month time.Month, day int) string {
	return time.Date(2000, month, day, 0, 0, 0, 0, time.UTC).Format("1-2")
}

// getTetDates returns approximate Tết dates for a year.
// In production, use a lunar calendar library.
func getTetDates(year int) []time.Time {
	// Approximate dates - Tết typically falls late Jan/early Feb
	// Returns ~5 days off
	tetStart := map[int]time.Time{
		2024: time.Date(2024, 2, 8, 0, 0, 0, 0, time.UTC),
		2025: time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC),
		2026: time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC),
	}

	start, ok := tetStart[year]
	if !ok {
		// Default estimate
		start = time.Date(year, 2, 1, 0, 0, 0, 0, time.UTC)
	}

	var dates []time.Time
	for i := 0; i < 5; i++ { // 5 days off for Tết
		dates = append(dates, start.AddDate(0, 0, i))
	}
	return dates
}

// getHungKingsDate returns Hung Kings' Commemoration date.
func getHungKingsDate(year int) time.Time {
	// Approximate - typically mid-April
	dates := map[int]time.Time{
		2024: time.Date(2024, 4, 18, 0, 0, 0, 0, time.UTC),
		2025: time.Date(2025, 4, 7, 0, 0, 0, 0, time.UTC),
		2026: time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC),
	}

	if date, ok := dates[year]; ok {
		return date
	}
	return time.Date(year, 4, 15, 0, 0, 0, 0, time.UTC) // Default estimate
}

// CashAvailability calculates available cash after pending settlements.
type CashAvailability struct {
	TotalCash     float64
	PendingBuys   float64 // Unsettled buy orders
	PendingSells  float64 // Unsettled sell proceeds
	AvailableCash float64
}

// CalculateCashAvailable computes available trading cash.
func CalculateCashAvailable(totalCash, pendingBuys, pendingSells float64) CashAvailability {
	available := totalCash - pendingBuys + pendingSells

	return CashAvailability{
		TotalCash:     totalCash,
		PendingBuys:   pendingBuys,
		PendingSells:  pendingSells,
		AvailableCash: available,
	}
}

// IsTradingDay checks if a date is a valid trading day.
func IsTradingDay(date time.Time) bool {
	// Weekend check
	if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
		return false
	}

	// Holiday check
	if IsVNHoliday(date) {
		return false
	}

	return true
}

// NextTradingDay returns the next valid trading day.
func NextTradingDay(from time.Time) time.Time {
	next := from.AddDate(0, 0, 1)
	for !IsTradingDay(next) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// PendingSettlement represents a transaction awaiting settlement
type PendingSettlement struct {
	TransactionDate time.Time
	SettlementDate  time.Time
	Amount          float64
	Type            string // "buy" or "sell"
}

// SettlementTracker tracks pending settlements to calculate available buying power
type SettlementTracker struct {
	accountBalance     float64
	pendingSettlements []PendingSettlement
}

// NewSettlementTracker creates a new settlement tracker
func NewSettlementTracker(accountBalance float64) *SettlementTracker {
	return &SettlementTracker{
		accountBalance:     accountBalance,
		pendingSettlements: make([]PendingSettlement, 0),
	}
}

// AddPendingSettlement adds a transaction awaiting settlement
func (st *SettlementTracker) AddPendingSettlement(pending PendingSettlement) {
	st.pendingSettlements = append(st.pendingSettlements, pending)
}

// CalculateAvailableCash computes available trading cash accounting for T+2 settlement
// Available Cash = Account Balance - Unsettled Buys
// Note: Unsettled sells are NOT counted as available (conservative approach)
func (st *SettlementTracker) CalculateAvailableCash() float64 {
	available := st.accountBalance

	now := time.Now()
	for _, pending := range st.pendingSettlements {
		// Only subtract unsettled buy orders
		if pending.Type == "buy" && now.Before(pending.SettlementDate) {
			available -= pending.Amount
		}
	}

	return available
}

// CanAffordPosition validates if a position can be afforded with available cash
// Returns (canAfford, message)
func (st *SettlementTracker) CanAffordPosition(positionValue float64) (bool, string) {
	available := st.CalculateAvailableCash()

	if positionValue > available {
		return false, fmt.Sprintf(
			"Insufficient buying power: Need %.0f VND, have %.0f VND available (T+2)",
			positionValue,
			available,
		)
	}

	// Warning if using >80% of available cash
	usagePercent := (positionValue / available) * 100
	if usagePercent > 80 {
		return true, fmt.Sprintf(
			"WARNING: Using %.0f%% of available buying power",
			usagePercent,
		)
	}

	return true, ""
}

// UpdateAccountBalance updates the account balance
func (st *SettlementTracker) UpdateAccountBalance(newBalance float64) {
	st.accountBalance = newBalance
}

// CleanupSettled removes settled transactions from pending list
func (st *SettlementTracker) CleanupSettled() {
	now := time.Now()
	filtered := make([]PendingSettlement, 0)

	for _, pending := range st.pendingSettlements {
		if now.Before(pending.SettlementDate) {
			filtered = append(filtered, pending)
		}
	}

	st.pendingSettlements = filtered
}
