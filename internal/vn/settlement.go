package vn

import (
	"fmt"
	"time"
)

// SettlementStatus represents the T+2 settlement status of shares.
type SettlementStatus string

const (
	// LockedT0 means shares were purchased today, cannot sell until T+3.
	LockedT0 SettlementStatus = "LOCKED_T0"

	// LockedT1 means shares are T+1 day after purchase, still cannot sell.
	LockedT1 SettlementStatus = "LOCKED_T1"

	// LockedT2 means shares are T+2 day after purchase, settlement in progress.
	LockedT2 SettlementStatus = "LOCKED_T2"

	// Liquid means shares have settled (T+3+), can be sold.
	Liquid SettlementStatus = "LIQUID"
)

// IsLocked returns true if the status indicates shares cannot be sold.
func (s SettlementStatus) IsLocked() bool {
	return s == LockedT0 || s == LockedT1 || s == LockedT2
}

// IsLiquid returns true if shares can be sold.
func (s SettlementStatus) IsLiquid() bool {
	return s == Liquid
}

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
// Comprehensive 2024-2026 calendar included for accurate settlement calculations.
func IsVNHoliday(date time.Time) bool {
	// Format: YYYY-MM-DD for exact date matching
	dateStr := date.Format("2006-01-02")

	// 2024 Holidays
	holidays2024 := map[string]bool{
		"2024-01-01": true, // New Year's Day
		"2024-02-08": true, // Lunar New Year Eve
		"2024-02-09": true, // Lunar New Year Day 1 (Tet)
		"2024-02-10": true, // Lunar New Year Day 2
		"2024-02-11": true, // Lunar New Year Day 3
		"2024-02-12": true, // Lunar New Year Day 4
		"2024-02-13": true, // Lunar New Year Day 5
		"2024-02-14": true, // Lunar New Year Day 6
		"2024-04-18": true, // Hung Kings' Commemoration Day
		"2024-04-30": true, // Reunification Day
		"2024-05-01": true, // International Labor Day
		"2024-09-02": true, // National Day
		"2024-09-03": true, // National Day
	}

	// 2025 Holidays
	holidays2025 := map[string]bool{
		"2025-01-01": true, // New Year's Day
		"2025-01-28": true, // Lunar New Year Eve
		"2025-01-29": true, // Lunar New Year Day 1 (Tet)
		"2025-01-30": true, // Lunar New Year Day 2
		"2025-01-31": true, // Lunar New Year Day 3
		"2025-02-01": true, // Lunar New Year Day 4
		"2025-02-02": true, // Lunar New Year Day 5
		"2025-04-07": true, // Hung Kings' Commemoration Day
		"2025-04-30": true, // Reunification Day
		"2025-05-01": true, // International Labor Day
		"2025-09-01": true, // National Day
		"2025-09-02": true, // National Day
		"2025-09-03": true, // National Day
	}

	// 2026 Holidays
	holidays2026 := map[string]bool{
		"2026-01-01": true, // New Year's Day
		"2026-02-16": true, // Lunar New Year Eve
		"2026-02-17": true, // Lunar New Year Day 1 (Tet)
		"2026-02-18": true, // Lunar New Year Day 2
		"2026-02-19": true, // Lunar New Year Day 3
		"2026-02-20": true, // Lunar New Year Day 4
		"2026-02-21": true, // Lunar New Year Day 5
		"2026-03-26": true, // Hung Kings' Commemoration Day (Updated)
		"2026-04-30": true, // Reunification Day
		"2026-05-01": true, // International Labor Day
		"2026-09-01": true, // National Day
		"2026-09-02": true, // National Day
		"2026-09-03": true, // National Day
	}

	// Check comprehensive holiday maps first
	if holidays2024[dateStr] || holidays2025[dateStr] || holidays2026[dateStr] {
		return true
	}

	// Fallback to old logic for years beyond 2026
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

	// Only use fallback for years after 2026
	if year > 2026 {
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

// CalculateSettlementStatusFromDates determines settlement status based on purchase date and current date.
// Vietnamese market: T+2 settlement means shares settle 2 trading days after purchase,
// but can only be sold on T+3.
func CalculateSettlementStatusFromDates(purchaseDate, currentDate time.Time) SettlementStatus {
	tradingDaysSince := CountTradingDaysBetween(purchaseDate, currentDate)

	switch tradingDaysSince {
	case 0:
		return LockedT0
	case 1:
		return LockedT1
	case 2:
		return LockedT2
	default:
		// T+3 or later
		return Liquid
	}
}

// CountTradingDaysBetween counts the number of trading days between two dates (inclusive of start, exclusive of end).
// Excludes weekends and Vietnamese public holidays.
func CountTradingDaysBetween(startDate, endDate time.Time) int {
	if startDate.After(endDate) {
		return 0
	}

	// Normalize to start of day
	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())

	count := 0
	current := start

	for current.Before(end) {
		if IsTradingDay(current) {
			count++
		}
		current = current.AddDate(0, 0, 1)
	}

	return count
}

// GetDaysUntilLiquid calculates how many trading days remain until shares become liquid.
// Returns 0 if shares are already liquid.
func GetDaysUntilLiquid(purchaseDate, currentDate time.Time) int {
	status := CalculateSettlementStatusFromDates(purchaseDate, currentDate)
	if status.IsLiquid() {
		return 0
	}

	// Calculate days from purchase to current
	daysSincePurchase := CountTradingDaysBetween(purchaseDate, currentDate)

	// Need 3 trading days total to become liquid (T+0, T+1, T+2 -> liquid on T+3)
	daysNeeded := 3
	return daysNeeded - daysSincePurchase
}

// GetCanSellDate calculates the first date when shares can be sold.
// This is T+3 from purchase date (3 trading days after purchase).
func GetCanSellDate(purchaseDate time.Time) time.Time {
	// Start from purchase date
	current := purchaseDate
	tradingDaysAdded := 0

	// Add 3 trading days
	for tradingDaysAdded < 3 {
		current = current.AddDate(0, 0, 1)
		if IsTradingDay(current) {
			tradingDaysAdded++
		}
	}

	return current
}

// GetSettlementDateT2 calculates the date when shares settle (T+2).
// Note: Shares settle on T+2 but cannot be sold until T+3.
func GetSettlementDateT2(purchaseDate time.Time) time.Time {
	// Use existing CalculateSettlement which computes T+2
	result := CalculateSettlement(purchaseDate)
	return result.SettlementDate
}

// GetEntryDayMultiplier returns position size multiplier based on day of week.
// Thursday/Friday entries get 50% multiplier due to extended weekend lock risk.
func GetEntryDayMultiplier(purchaseDate time.Time) float64 {
	weekday := purchaseDate.Weekday()

	// Thursday or Friday: reduce position size to 50%
	if weekday == time.Thursday || weekday == time.Friday {
		return 0.5
	}

	// Monday, Tuesday, Wednesday: full position size
	return 1.0
}

// GetExchangeFromTicker infers the stock exchange from ticker symbol.
// HOSE: typically tickers like "VNM", "HPG" (3 letters)
// HNX: typically tickers like "ACB", "SHB" (3 letters, but different list)
// UPCOM: typically tickers like "AAM", "ABC" (3 letters, smaller cap)
//
// Note: This is a simplified heuristic. In production, use a lookup table.
func GetExchangeFromTicker(ticker string) string {
	// TODO: Replace with actual exchange lookup from database or API
	// For now, default to HOSE (most common)
	// This should be enhanced with actual ticker-to-exchange mapping
	return "HOSE"
}

// GetFloorLimitForExchange returns the daily floor limit percentage for each exchange.
// HOSE: 7% (most liquid)
// HNX: 10%
// UPCOM: 15% (least liquid, higher volatility)
func GetFloorLimitForExchange(exchange string) float64 {
	switch exchange {
	case "HOSE":
		return 0.07
	case "HNX":
		return 0.10
	case "UPCOM":
		return 0.15
	default:
		// Default to HOSE limit
		return 0.07
	}
}

// GetLockedRiskMultiplierForExchange returns the worst-case risk multiplier for locked shares.
// This accounts for floor-hit scenario plus slippage and fees.
// HOSE: 20% (7% floor + margin)
// HNX: 30% (10% floor + margin)
// UPCOM: 40% (15% floor + margin)
func GetLockedRiskMultiplierForExchange(exchange string) float64 {
	switch exchange {
	case "HOSE":
		return 0.20
	case "HNX":
		return 0.30
	case "UPCOM":
		return 0.40
	default:
		// Default to HOSE multiplier
		return 0.20
	}
}
