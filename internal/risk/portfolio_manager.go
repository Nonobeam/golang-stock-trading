package risk

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/position"
)

// RiskSnapshot represents a point-in-time snapshot of portfolio risk metrics.
type RiskSnapshot struct {
	Timestamp time.Time `json:"timestamp"`

	// Capital metrics
	TotalCapital       float64 `json:"total_capital"`
	UsedCapital        float64 `json:"used_capital"`
	AvailableCapital   float64 `json:"available_capital"`
	UsedCapitalPercent float64 `json:"used_capital_percent"`

	// Risk metrics
	TotalRiskAmount  float64 `json:"total_risk_amount"`  // Sum of all position risks
	TotalRiskPercent float64 `json:"total_risk_percent"` // As % of capital
	MaxRiskAllowed   float64 `json:"max_risk_allowed"`
	RiskUtilization  float64 `json:"risk_utilization"` // % of max risk used

	// Position metrics
	NumPositions        int             `json:"num_positions"`
	MaxPositionsAllowed int             `json:"max_positions_allowed"`
	PositionValues      []PositionValue `json:"position_values"`

	// Sector exposure
	SectorExposure           map[string]SectorData `json:"sector_exposure"`
	MaxSectorExposurePercent float64               `json:"max_sector_exposure_percent"`

	// Correlation metrics
	AvgPortfolioCorrelation float64          `json:"avg_portfolio_correlation"`
	MaxPairwiseCorrelation  float64          `json:"max_pairwise_correlation"`
	HighlyCorrelatedPairs   []CorrelatedPair `json:"highly_correlated_pairs"`

	// Performance metrics
	TotalUnrealizedPL float64 `json:"total_unrealized_pl"`
	TotalRealizedPL   float64 `json:"total_realized_pl"`
	TotalPL           float64 `json:"total_pl"`
	DrawdownFromPeak  float64 `json:"drawdown_from_peak"`

	// Risk warnings and violations
	Warnings   []RiskWarning   `json:"warnings"`
	Violations []RiskViolation `json:"violations"`
}

// PositionValue represents value and percentage of a single position.
type PositionValue struct {
	PositionID string  `json:"position_id"`
	Ticker     string  `json:"ticker"`
	Value      float64 `json:"value"`
	Percent    float64 `json:"percent"`
}

// SectorData contains sector exposure information.
type SectorData struct {
	Value     float64  `json:"value"`
	Percent   float64  `json:"percent"`
	Positions []string `json:"positions"`
	Count     int      `json:"count"`
}

// CorrelatedPair represents two highly correlated positions.
type CorrelatedPair struct {
	Ticker1     string  `json:"ticker1"`
	Ticker2     string  `json:"ticker2"`
	Correlation float64 `json:"correlation"`
}

// RiskWarning represents a risk warning (not critical).
type RiskWarning struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Action   string `json:"action"`
}

// RiskViolation represents a critical risk limit violation.
type RiskViolation struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Action   string `json:"action"`
}

// RiskLimits defines configurable portfolio risk limits.
type RiskLimits struct {
	// Position limits
	MaxPositions           int
	MaxPositionSizePercent float64 // Max % of capital per position

	// Risk limits
	MaxTotalRiskPercent float64 // Max aggregate risk
	MaxRiskPerTrade     float64 // Max risk on single trade

	// Loss limits
	DailyLossLimitPercent   float64
	WeeklyLossLimitPercent  float64
	MonthlyLossLimitPercent float64
	MaxDrawdownPercent      float64

	// Sector limits
	MaxSectorExposurePercent float64
	MaxPositionsPerSector    int

	// Correlation limits
	MaxPairwiseCorrelation     float64 // Don't add if correlation > this
	MaxAvgPortfolioCorrelation float64

	// Consecutive loss limits
	MaxConsecutiveLosses  int
	ReduceSizeAfterLosses int
}

// DefaultRiskLimits returns sensible default risk limits.
func DefaultRiskLimits() *RiskLimits {
	return &RiskLimits{
		MaxPositions:               8,
		MaxPositionSizePercent:     20.0,
		MaxTotalRiskPercent:        6.0,
		MaxRiskPerTrade:            2.0,
		DailyLossLimitPercent:      2.0,
		WeeklyLossLimitPercent:     5.0,
		MonthlyLossLimitPercent:    10.0,
		MaxDrawdownPercent:         20.0,
		MaxSectorExposurePercent:   40.0,
		MaxPositionsPerSector:      3,
		MaxPairwiseCorrelation:     0.85,
		MaxAvgPortfolioCorrelation: 0.60,
		MaxConsecutiveLosses:       5,
		ReduceSizeAfterLosses:      3,
	}
}

// VietnamConservativeLimits returns conservative risk limits tailored to Vietnam market.
// Vietnam market has higher volatility, gap risk from ±7% daily limits, and lower liquidity
// requiring more conservative position management.
func VietnamConservativeLimits() *RiskLimits {
	limits := DefaultRiskLimits()
	limits.MaxTotalRiskPercent = 4.0   // Lower than default 6.0% due to gap risk
	limits.MaxPositions = 6            // Fewer positions than default 8 for focus
	limits.DailyLossLimitPercent = 1.5 // Tighter than default 2.0% for control
	return limits
}

// PortfolioManager manages aggregate portfolio-level risk.
type PortfolioManager struct {
	InitialCapital float64
	CurrentCapital float64
	PeakCapital    float64

	Limits *RiskLimits

	// Track daily/weekly/monthly P&L
	DailyStartingCapital   float64
	WeeklyStartingCapital  float64
	MonthlyStartingCapital float64

	DailyStartDate   time.Time
	WeeklyStartDate  time.Time
	MonthlyStartDate time.Time

	// Track consecutive losses
	ConsecutiveLosses int
	ConsecutiveWins   int

	// Store snapshots for history
	RiskSnapshots []RiskSnapshot

	// Sector mapping (Vietnam specific)
	sectorMap map[string]string
}

// NewPortfolioManager creates a new portfolio risk manager.
func NewPortfolioManager(initialCapital float64, limits *RiskLimits) *PortfolioManager {
	if limits == nil {
		limits = DefaultRiskLimits()
	}

	now := time.Now()

	pm := &PortfolioManager{
		InitialCapital:         initialCapital,
		CurrentCapital:         initialCapital,
		PeakCapital:            initialCapital,
		Limits:                 limits,
		DailyStartingCapital:   initialCapital,
		WeeklyStartingCapital:  initialCapital,
		MonthlyStartingCapital: initialCapital,
		DailyStartDate:         now,
		WeeklyStartDate:        getWeekStart(now),
		MonthlyStartDate:       getMonthStart(now),
		RiskSnapshots:          []RiskSnapshot{},
	}

	pm.sectorMap = pm.initializeSectorMap()

	return pm
}

// CanAddResult contains the result of checking if a new position can be added.
type CanAddResult struct {
	CanAdd             bool            `json:"can_add"`
	Issues             []string        `json:"issues"`
	Warnings           []RiskWarning   `json:"warnings"`
	RecommendedMaxSize RecommendedSize `json:"recommended_max_size"`
}

// RecommendedSize contains recommended position sizing adjusted for portfolio state.
type RecommendedSize struct {
	RecommendedRiskVND     float64 `json:"recommended_risk_vnd"`
	RecommendedRiskPercent float64 `json:"recommended_risk_percent"`
	SizeMultiplier         float64 `json:"size_multiplier"`
	Reason                 string  `json:"reason"`
}

// initializeSectorMap creates the Vietnam stock sector mapping.
func (pm *PortfolioManager) initializeSectorMap() map[string]string {
	return map[string]string{
		// Banking
		"VCB": "Banking", "BID": "Banking", "CTG": "Banking", "MBB": "Banking",
		"TCB": "Banking", "VPB": "Banking", "ACB": "Banking", "STB": "Banking",
		"HDB": "Banking", "TPB": "Banking", "SHB": "Banking", "MSB": "Banking",

		// Real Estate
		"VHM": "Real Estate", "VIC": "Real Estate", "NVL": "Real Estate",
		"PDR": "Real Estate", "DXG": "Real Estate", "KDH": "Real Estate",
		"NLG": "Real Estate", "DIG": "Real Estate", "HDG": "Real Estate",

		// Steel
		"HPG": "Steel", "HSG": "Steel", "NKG": "Steel",
		"POM": "Steel", "TLH": "Steel",

		// Rubber
		"GVR": "Rubber", "DPR": "Rubber", "PHR": "Rubber",

		// Fertilizer
		"DPM": "Fertilizer", "DCM": "Fertilizer",

		// Consumer
		"VNM": "Consumer", "SAB": "Consumer", "MSN": "Consumer",

		// Retail
		"MWG": "Retail", "FRT": "Retail", "PNJ": "Retail",

		// Energy
		"GAS": "Energy", "PLX": "Energy", "POW": "Energy",
		"NT2": "Energy", "PVS": "Energy", "PVD": "Energy",
		"PVT": "Energy", "PVG": "Energy",

		// Technology
		"FPT": "Technology", "CMG": "Technology", "VGI": "Technology",

		// Transportation
		"HVN": "Transportation", "VJC": "Transportation", "ACV": "Transportation",
		"GMD": "Logistics", "STG": "Logistics",

		// Agriculture
		"HNG": "Agriculture", "BAF": "Agriculture", "SBT": "Agriculture",

		// Healthcare/Pharma
		"DHG": "Healthcare", "DMC": "Healthcare", "IMP": "Healthcare",
		"TRA": "Healthcare", "DHT": "Healthcare",

		// Securities/Finance
		"SSI": "Securities", "VND": "Securities", "HCM": "Securities",
		"VCI": "Securities", "MBS": "Securities", "SHS": "Securities",
	}
}

// getSector returns the sector for a given ticker.
func (pm *PortfolioManager) getSector(ticker string) string {
	if sector, ok := pm.sectorMap[ticker]; ok {
		return sector
	}
	return "Other"
}

// getWeekStart returns the Monday of the current week.
func getWeekStart(t time.Time) time.Time {
	weekday := t.Weekday()
	offset := int(weekday - time.Monday)
	if offset < 0 {
		offset += 7
	}
	return t.AddDate(0, 0, -offset).Truncate(24 * time.Hour)
}

// getMonthStart returns the first day of the current month.
func getMonthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// updatePeriodTracking updates daily/weekly/monthly starting capital if period changed.
func (pm *PortfolioManager) updatePeriodTracking() {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)

	// Check if new day
	if today.After(pm.DailyStartDate) {
		pm.DailyStartingCapital = pm.CurrentCapital
		pm.DailyStartDate = today
	}

	// Check if new week
	weekStart := getWeekStart(now)
	if weekStart.After(pm.WeeklyStartDate) {
		pm.WeeklyStartingCapital = pm.CurrentCapital
		pm.WeeklyStartDate = weekStart
	}

	// Check if new month
	monthStart := getMonthStart(now)
	if monthStart.After(pm.MonthlyStartDate) {
		pm.MonthlyStartingCapital = pm.CurrentCapital
		pm.MonthlyStartDate = monthStart
	}
}

// RecordTradeResult records win/loss for consecutive tracking.
func (pm *PortfolioManager) RecordTradeResult(result string) {
	if result == "win" {
		pm.ConsecutiveWins++
		pm.ConsecutiveLosses = 0
	} else if result == "loss" {
		pm.ConsecutiveLosses++
		pm.ConsecutiveWins = 0
	}
}

// GetLatestSnapshot returns the most recent risk snapshot, or nil if none exist.
func (pm *PortfolioManager) GetLatestSnapshot() *RiskSnapshot {
	if len(pm.RiskSnapshots) == 0 {
		return nil
	}
	return &pm.RiskSnapshots[len(pm.RiskSnapshots)-1]
}

// GetDashboardSummary generates a human-readable portfolio risk dashboard.
func (pm *PortfolioManager) GetDashboardSummary() string {
	if len(pm.RiskSnapshots) == 0 {
		return "No positions - portfolio empty"
	}

	latest := pm.RiskSnapshots[len(pm.RiskSnapshots)-1]

	var sb strings.Builder

	// Header
	sb.WriteString(strings.Repeat("=", 70) + "\n")
	sb.WriteString("PORTFOLIO RISK DASHBOARD\n")
	sb.WriteString(strings.Repeat("=", 70) + "\n\n")

	// Capital section
	sb.WriteString("CAPITAL METRICS\n")
	sb.WriteString(strings.Repeat("-", 70) + "\n")
	sb.WriteString(fmt.Sprintf("Total Capital:     %15.0f VND\n", latest.TotalCapital))
	sb.WriteString(fmt.Sprintf("Used Capital:      %15.0f VND (%.1f%%)\n", latest.UsedCapital, latest.UsedCapitalPercent))
	sb.WriteString(fmt.Sprintf("Available:         %15.0f VND\n\n", latest.AvailableCapital))

	// Risk section
	sb.WriteString("RISK METRICS\n")
	sb.WriteString(strings.Repeat("-", 70) + "\n")
	sb.WriteString(fmt.Sprintf("Total Risk:        %15.0f VND (%.1f%%)\n", latest.TotalRiskAmount, latest.TotalRiskPercent))
	sb.WriteString(fmt.Sprintf("Max Allowed:       %15.0f VND (%.0f%%)\n", latest.MaxRiskAllowed, pm.Limits.MaxTotalRiskPercent))
	sb.WriteString(fmt.Sprintf("Risk Utilization:  %15.1f%%\n", latest.RiskUtilization))

	// Risk utilization bar
	utilBars := int(latest.RiskUtilization / 5)
	if utilBars > 20 {
		utilBars = 20
	}
	sb.WriteString(fmt.Sprintf("                   [%s%s]\n\n",
		strings.Repeat("█", utilBars),
		strings.Repeat("░", 20-utilBars)))

	// Positions section
	sb.WriteString("POSITIONS\n")
	sb.WriteString(strings.Repeat("-", 70) + "\n")
	sb.WriteString(fmt.Sprintf("Open Positions:    %d / %d\n\n", latest.NumPositions, latest.MaxPositionsAllowed))

	if len(latest.PositionValues) > 0 {
		sb.WriteString(fmt.Sprintf("%-10s %15s %10s %15s\n", "Ticker", "Value", "% Capital", "Sector"))
		sb.WriteString(strings.Repeat("-", 70) + "\n")
		for _, pv := range latest.PositionValues {
			sb.WriteString(fmt.Sprintf("%-10s %15.0f %9.1f%% %15s\n",
				pv.Ticker, pv.Value, pv.Percent, pm.getSector(pv.Ticker)))
		}
		sb.WriteString("\n")
	}

	// Sector exposure
	if len(latest.SectorExposure) > 0 {
		sb.WriteString("SECTOR EXPOSURE\n")
		sb.WriteString(strings.Repeat("-", 70) + "\n")
		sb.WriteString(fmt.Sprintf("%-20s %15s %10s %10s\n", "Sector", "Value", "% Capital", "Positions"))
		sb.WriteString(strings.Repeat("-", 70) + "\n")
		for sector, data := range latest.SectorExposure {
			sb.WriteString(fmt.Sprintf("%-20s %15.0f %9.1f%% %10d\n",
				sector, data.Value, data.Percent, data.Count))

			if data.Percent > pm.Limits.MaxSectorExposurePercent {
				sb.WriteString(fmt.Sprintf("  ⚠️  EXCEEDS LIMIT (%.0f%%)\n", pm.Limits.MaxSectorExposurePercent))
			}
		}
		sb.WriteString("\n")
	}

	// Performance section
	sb.WriteString("PERFORMANCE\n")
	sb.WriteString(strings.Repeat("-", 70) + "\n")
	sb.WriteString(fmt.Sprintf("Unrealized P&L:    %15.0f VND\n", latest.TotalUnrealizedPL))
	sb.WriteString(fmt.Sprintf("Realized P&L:      %15.0f VND\n", latest.TotalRealizedPL))
	sb.WriteString(fmt.Sprintf("Total P&L:         %15.0f VND\n", latest.TotalPL))
	sb.WriteString(fmt.Sprintf("Drawdown:          %15.1f%%\n\n", latest.DrawdownFromPeak))

	// Period P&L
	dailyPL := pm.CurrentCapital - pm.DailyStartingCapital
	dailyPLPct := (dailyPL / pm.DailyStartingCapital) * 100
	weeklyPL := pm.CurrentCapital - pm.WeeklyStartingCapital
	weeklyPLPct := (weeklyPL / pm.WeeklyStartingCapital) * 100

	sb.WriteString("PERIOD P&L\n")
	sb.WriteString(strings.Repeat("-", 70) + "\n")
	sb.WriteString(fmt.Sprintf("Daily:             %15.0f VND (%+.2f%%)\n", dailyPL, dailyPLPct))
	sb.WriteString(fmt.Sprintf("Weekly:            %15.0f VND (%+.2f%%)\n\n", weeklyPL, weeklyPLPct))

	// Streaks
	if pm.ConsecutiveWins > 0 || pm.ConsecutiveLosses > 0 {
		sb.WriteString("STREAKS\n")
		sb.WriteString(strings.Repeat("-", 70) + "\n")
		if pm.ConsecutiveWins > 0 {
			sb.WriteString(fmt.Sprintf("Consecutive Wins:  %d\n", pm.ConsecutiveWins))
		}
		if pm.ConsecutiveLosses > 0 {
			sb.WriteString(fmt.Sprintf("Consecutive Losses: %d\n", pm.ConsecutiveLosses))
			if pm.ConsecutiveLosses >= pm.Limits.ReduceSizeAfterLosses {
				sb.WriteString("  ⚠️  REDUCE POSITION SIZES\n")
			}
		}
		sb.WriteString("\n")
	}

	// Warnings section
	if len(latest.Warnings) > 0 {
		sb.WriteString("⚠️  WARNINGS\n")
		sb.WriteString(strings.Repeat("-", 70) + "\n")
		for _, warning := range latest.Warnings {
			sb.WriteString(fmt.Sprintf("[%s] %s\n", warning.Severity, warning.Message))
			sb.WriteString(fmt.Sprintf("  → %s\n\n", warning.Action))
		}
	}

	// Violations section
	if len(latest.Violations) > 0 {
		sb.WriteString("🚨 VIOLATIONS\n")
		sb.WriteString(strings.Repeat("-", 70) + "\n")
		for _, violation := range latest.Violations {
			sb.WriteString(fmt.Sprintf("[%s] %s\n", violation.Severity, violation.Message))
			sb.WriteString(fmt.Sprintf("  → %s\n\n", violation.Action))
		}
	}

	// Overall status
	sb.WriteString(strings.Repeat("=", 70) + "\n")
	status := "HEALTHY - WITHIN LIMITS"
	if len(latest.Violations) > 0 {
		status = "🚨 CRITICAL - IMMEDIATE ACTION REQUIRED"
	} else if len(latest.Warnings) > 0 {
		status = "⚠️  WARNING - CAUTION ADVISED"
	}
	sb.WriteString(fmt.Sprintf("STATUS: %s\n", status))
	sb.WriteString(strings.Repeat("=", 70) + "\n")

	return sb.String()
}

// CalculateRisk calculates comprehensive portfolio risk snapshot from current positions.
func (pm *PortfolioManager) CalculateRisk(
	positions map[string]*position.Position,
	currentPrices map[string]float64,
) *RiskSnapshot {
	timestamp := time.Now()

	// Update daily/weekly/monthly tracking
	pm.updatePeriodTracking()

	// Calculate all metrics
	capitalMetrics := pm.calculateCapitalMetrics(positions, currentPrices)
	riskMetrics := pm.calculateRiskMetrics(positions, currentPrices)
	sectorMetrics := pm.calculateSectorExposure(positions, currentPrices)
	performanceMetrics := pm.calculatePerformanceMetrics(positions, currentPrices)

	// Check for warnings and violations
	warnings, violations := pm.checkRiskLimits(
		capitalMetrics,
		riskMetrics,
		sectorMetrics,
		performanceMetrics,
	)

	// Create snapshot
	snapshot := RiskSnapshot{
		Timestamp:                timestamp,
		TotalCapital:             pm.CurrentCapital,
		UsedCapital:              capitalMetrics.usedCapital,
		AvailableCapital:         capitalMetrics.availableCapital,
		UsedCapitalPercent:       capitalMetrics.usedCapitalPercent,
		TotalRiskAmount:          riskMetrics.totalRiskAmount,
		TotalRiskPercent:         riskMetrics.totalRiskPercent,
		MaxRiskAllowed:           pm.CurrentCapital * (pm.Limits.MaxTotalRiskPercent / 100),
		RiskUtilization:          riskMetrics.riskUtilization,
		NumPositions:             len(positions),
		MaxPositionsAllowed:      pm.Limits.MaxPositions,
		PositionValues:           capitalMetrics.positionValues,
		SectorExposure:           sectorMetrics,
		MaxSectorExposurePercent: pm.Limits.MaxSectorExposurePercent,
		AvgPortfolioCorrelation:  0, // Not implemented yet
		MaxPairwiseCorrelation:   0, // Not implemented yet
		HighlyCorrelatedPairs:    []CorrelatedPair{},
		TotalUnrealizedPL:        performanceMetrics.totalUnrealizedPL,
		TotalRealizedPL:          performanceMetrics.totalRealizedPL,
		TotalPL:                  performanceMetrics.totalPL,
		DrawdownFromPeak:         performanceMetrics.drawdownFromPeak,
		Warnings:                 warnings,
		Violations:               violations,
	}

	// Store snapshot
	pm.RiskSnapshots = append(pm.RiskSnapshots, snapshot)

	// Keep only last 1000 snapshots
	if len(pm.RiskSnapshots) > 1000 {
		pm.RiskSnapshots = pm.RiskSnapshots[len(pm.RiskSnapshots)-1000:]
	}

	return &snapshot
}

// capitalMetricsResult contains results from capital metrics calculation.
type capitalMetricsResult struct {
	usedCapital        float64
	availableCapital   float64
	usedCapitalPercent float64
	positionValues     []PositionValue
}

// calculateCapitalMetrics calculates capital utilization metrics.
func (pm *PortfolioManager) calculateCapitalMetrics(
	positions map[string]*position.Position,
	currentPrices map[string]float64,
) capitalMetricsResult {
	positionValues := []PositionValue{}
	totalUsed := 0.0

	for posID, pos := range positions {
		currentPrice := currentPrices[pos.Ticker]
		if currentPrice == 0 {
			currentPrice = pos.CurrentPrice
		}

		positionValue := float64(pos.SharesRemaining) * currentPrice
		positionPercent := (positionValue / pm.CurrentCapital) * 100

		positionValues = append(positionValues, PositionValue{
			PositionID: posID,
			Ticker:     pos.Ticker,
			Value:      positionValue,
			Percent:    positionPercent,
		})

		totalUsed += positionValue
	}

	available := pm.CurrentCapital - totalUsed
	usedPercent := (totalUsed / pm.CurrentCapital) * 100

	return capitalMetricsResult{
		usedCapital:        totalUsed,
		availableCapital:   available,
		usedCapitalPercent: usedPercent,
		positionValues:     positionValues,
	}
}

// riskMetricsResult contains results from risk metrics calculation.
type riskMetricsResult struct {
	totalRiskAmount  float64
	totalRiskPercent float64
	riskUtilization  float64
}

// calculateRiskMetrics calculates aggregate risk metrics.
func (pm *PortfolioManager) calculateRiskMetrics(
	positions map[string]*position.Position,
	currentPrices map[string]float64,
) riskMetricsResult {
	totalRisk := 0.0

	for _, pos := range positions {
		currentPrice := currentPrices[pos.Ticker]
		if currentPrice == 0 {
			currentPrice = pos.CurrentPrice
		}

		// Calculate current risk (distance to stop)
		var riskPerShare float64
		if pos.PositionType == "long" {
			riskPerShare = currentPrice - pos.StopLoss
		} else {
			riskPerShare = pos.StopLoss - currentPrice
		}

		if riskPerShare < 0 {
			riskPerShare = 0 // Stop already hit
		}

		positionRisk := riskPerShare * float64(pos.SharesRemaining)
		totalRisk += positionRisk
	}

	totalRiskPercent := (totalRisk / pm.CurrentCapital) * 100
	riskUtilization := (totalRiskPercent / pm.Limits.MaxTotalRiskPercent) * 100

	return riskMetricsResult{
		totalRiskAmount:  totalRisk,
		totalRiskPercent: totalRiskPercent,
		riskUtilization:  riskUtilization,
	}
}

// calculateSectorExposure calculates exposure by sector.
func (pm *PortfolioManager) calculateSectorExposure(
	positions map[string]*position.Position,
	currentPrices map[string]float64,
) map[string]SectorData {
	sectorExposure := make(map[string]SectorData)

	for _, pos := range positions {
		sector := pm.getSector(pos.Ticker)
		currentPrice := currentPrices[pos.Ticker]
		if currentPrice == 0 {
			currentPrice = pos.CurrentPrice
		}

		positionValue := float64(pos.SharesRemaining) * currentPrice

		if data, ok := sectorExposure[sector]; ok {
			data.Value += positionValue
			data.Positions = append(data.Positions, pos.Ticker)
			data.Count++
			sectorExposure[sector] = data
		} else {
			sectorExposure[sector] = SectorData{
				Value:     positionValue,
				Positions: []string{pos.Ticker},
				Count:     1,
			}
		}
	}

	// Calculate percentages
	for sector, data := range sectorExposure {
		data.Percent = (data.Value / pm.CurrentCapital) * 100
		sectorExposure[sector] = data
	}

	return sectorExposure
}

// performanceMetricsResult contains results from performance metrics calculation.
type performanceMetricsResult struct {
	totalUnrealizedPL float64
	totalRealizedPL   float64
	totalPL           float64
	drawdownFromPeak  float64
}

// calculatePerformanceMetrics calculates portfolio performance metrics.
func (pm *PortfolioManager) calculatePerformanceMetrics(
	positions map[string]*position.Position,
	currentPrices map[string]float64,
) performanceMetricsResult {
	totalUnrealizedPL := 0.0
	totalRealizedPL := 0.0

	for _, pos := range positions {
		currentPrice := currentPrices[pos.Ticker]
		if currentPrice == 0 {
			currentPrice = pos.CurrentPrice
		}

		// Unrealized P&L
		var unrealizedPL float64
		if pos.PositionType == "long" {
			unrealizedPL = (currentPrice - pos.EntryPrice) * float64(pos.SharesRemaining)
		} else {
			unrealizedPL = (pos.EntryPrice - currentPrice) * float64(pos.SharesRemaining)
		}

		totalUnrealizedPL += unrealizedPL

		// Realized P&L from partial exits
		for _, exit := range pos.Exits {
			var realizedPL float64
			if pos.PositionType == "long" {
				realizedPL = (exit.Price - pos.EntryPrice) * float64(exit.Shares)
			} else {
				realizedPL = (pos.EntryPrice - exit.Price) * float64(exit.Shares)
			}
			totalRealizedPL += realizedPL
		}
	}

	totalPL := totalUnrealizedPL + totalRealizedPL

	// Update current capital
	pm.CurrentCapital = pm.InitialCapital + totalPL

	// Update peak and calculate drawdown
	if pm.CurrentCapital > pm.PeakCapital {
		pm.PeakCapital = pm.CurrentCapital
	}

	drawdown := 0.0
	if pm.PeakCapital > 0 {
		drawdown = ((pm.PeakCapital - pm.CurrentCapital) / pm.PeakCapital) * 100
	}

	return performanceMetricsResult{
		totalUnrealizedPL: totalUnrealizedPL,
		totalRealizedPL:   totalRealizedPL,
		totalPL:           totalPL,
		drawdownFromPeak:  drawdown,
	}
}

// checkRiskLimits checks all risk limits and generates warnings/violations.
func (pm *PortfolioManager) checkRiskLimits(
	capitalMetrics capitalMetricsResult,
	riskMetrics riskMetricsResult,
	sectorMetrics map[string]SectorData,
	performanceMetrics performanceMetricsResult,
) ([]RiskWarning, []RiskViolation) {
	warnings := []RiskWarning{}
	violations := []RiskViolation{}

	// Check 1: Total risk
	if riskMetrics.totalRiskPercent > pm.Limits.MaxTotalRiskPercent {
		violations = append(violations, RiskViolation{
			Type:     "TOTAL_RISK_EXCEEDED",
			Severity: "HIGH",
			Message:  fmt.Sprintf("Total risk %.1f%% exceeds limit %.0f%%", riskMetrics.totalRiskPercent, pm.Limits.MaxTotalRiskPercent),
			Action:   "Close positions or reduce sizes immediately",
		})
	} else if riskMetrics.totalRiskPercent > pm.Limits.MaxTotalRiskPercent*0.9 {
		warnings = append(warnings, RiskWarning{
			Type:     "TOTAL_RISK_HIGH",
			Severity: "MEDIUM",
			Message:  fmt.Sprintf("Total risk %.1f%% near limit %.0f%%", riskMetrics.totalRiskPercent, pm.Limits.MaxTotalRiskPercent),
			Action:   "Avoid new positions",
		})
	}

	// Check 2: Number of positions
	numPositions := len(capitalMetrics.positionValues)
	if numPositions > pm.Limits.MaxPositions {
		violations = append(violations, RiskViolation{
			Type:     "TOO_MANY_POSITIONS",
			Severity: "MEDIUM",
			Message:  fmt.Sprintf("%d positions exceeds limit %d", numPositions, pm.Limits.MaxPositions),
			Action:   "Close weakest positions",
		})
	}

	// Check 3: Sector concentration
	for sector, data := range sectorMetrics {
		if data.Percent > pm.Limits.MaxSectorExposurePercent {
			violations = append(violations, RiskViolation{
				Type:     "SECTOR_CONCENTRATION",
				Severity: "MEDIUM",
				Message:  fmt.Sprintf("%s exposure %.1f%% exceeds %.0f%%", sector, data.Percent, pm.Limits.MaxSectorExposurePercent),
				Action:   fmt.Sprintf("Reduce %s exposure", sector),
			})
		}

		if data.Count > pm.Limits.MaxPositionsPerSector {
			warnings = append(warnings, RiskWarning{
				Type:     "SECTOR_POSITION_COUNT",
				Severity: "LOW",
				Message:  fmt.Sprintf("%d positions in %s (limit %d)", data.Count, sector, pm.Limits.MaxPositionsPerSector),
				Action:   "Diversify across sectors",
			})
		}
	}

	// Check 4: Daily loss limit
	dailyPL := pm.CurrentCapital - pm.DailyStartingCapital
	dailyLossPercent := (dailyPL / pm.DailyStartingCapital) * 100

	if dailyLossPercent < -pm.Limits.DailyLossLimitPercent {
		violations = append(violations, RiskViolation{
			Type:     "DAILY_LOSS_LIMIT",
			Severity: "HIGH",
			Message:  fmt.Sprintf("Daily loss %.1f%% exceeds limit %.0f%%", dailyLossPercent, pm.Limits.DailyLossLimitPercent),
			Action:   "STOP TRADING - Close all positions or stop new trades",
		})
	} else if dailyLossPercent < -pm.Limits.DailyLossLimitPercent*0.75 {
		warnings = append(warnings, RiskWarning{
			Type:     "DAILY_LOSS_WARNING",
			Severity: "HIGH",
			Message:  fmt.Sprintf("Daily loss %.1f%% approaching limit", dailyLossPercent),
			Action:   "Reduce risk, no new positions",
		})
	}

	// Check 5: Weekly loss limit
	weeklyPL := pm.CurrentCapital - pm.WeeklyStartingCapital
	weeklyLossPercent := (weeklyPL / pm.WeeklyStartingCapital) * 100

	if weeklyLossPercent < -pm.Limits.WeeklyLossLimitPercent {
		violations = append(violations, RiskViolation{
			Type:     "WEEKLY_LOSS_LIMIT",
			Severity: "HIGH",
			Message:  fmt.Sprintf("Weekly loss %.1f%% exceeds limit %.0f%%", weeklyLossPercent, pm.Limits.WeeklyLossLimitPercent),
			Action:   "STOP TRADING - Reduce sizes by 50%",
		})
	}

	// Check 6: Drawdown
	if performanceMetrics.drawdownFromPeak > pm.Limits.MaxDrawdownPercent {
		violations = append(violations, RiskViolation{
			Type:     "MAX_DRAWDOWN_EXCEEDED",
			Severity: "CRITICAL",
			Message:  fmt.Sprintf("Drawdown %.1f%% exceeds limit %.0f%%", performanceMetrics.drawdownFromPeak, pm.Limits.MaxDrawdownPercent),
			Action:   "STOP ALL TRADING - Capital preservation mode",
		})
	}

	// Check 7: Consecutive losses
	if pm.ConsecutiveLosses >= pm.Limits.MaxConsecutiveLosses {
		violations = append(violations, RiskViolation{
			Type:     "CONSECUTIVE_LOSSES",
			Severity: "HIGH",
			Message:  fmt.Sprintf("%d consecutive losses (limit %d)", pm.ConsecutiveLosses, pm.Limits.MaxConsecutiveLosses),
			Action:   "Stop trading, review system",
		})
	} else if pm.ConsecutiveLosses >= pm.Limits.ReduceSizeAfterLosses {
		warnings = append(warnings, RiskWarning{
			Type:     "LOSING_STREAK",
			Severity: "MEDIUM",
			Message:  fmt.Sprintf("%d consecutive losses", pm.ConsecutiveLosses),
			Action:   "Reduce position sizes by 50%",
		})
	}

	return warnings, violations
}

// CanAddPosition checks if a new position can be added within limits.
func (pm *PortfolioManager) CanAddPosition(
	newPositionValue float64,
	newPositionRisk float64,
	ticker string,
	correlationWithExisting float64,
) *CanAddResult {
	// Get latest snapshot
	if len(pm.RiskSnapshots) == 0 {
		// No positions yet, can add
		return &CanAddResult{
			CanAdd:   true,
			Issues:   nil,
			Warnings: nil,
			RecommendedMaxSize: RecommendedSize{
				RecommendedRiskVND:     newPositionRisk,
				RecommendedRiskPercent: (newPositionRisk / pm.CurrentCapital) * 100,
				SizeMultiplier:         1.0,
				Reason:                 "No existing positions",
			},
		}
	}

	latest := pm.RiskSnapshots[len(pm.RiskSnapshots)-1]

	issues := []string{}

	// Check 1: Position count
	if latest.NumPositions >= pm.Limits.MaxPositions {
		issues = append(issues, fmt.Sprintf("Already at max positions (%d)", pm.Limits.MaxPositions))
	}

	// Check 2: Position size
	newPositionPercent := (newPositionValue / pm.CurrentCapital) * 100
	if newPositionPercent > pm.Limits.MaxPositionSizePercent {
		issues = append(issues, fmt.Sprintf("Position size %.1f%% exceeds limit %.0f%%", newPositionPercent, pm.Limits.MaxPositionSizePercent))
	}

	// Check 3: Total risk
	newTotalRiskPercent := latest.TotalRiskPercent + ((newPositionRisk / pm.CurrentCapital) * 100)
	if newTotalRiskPercent > pm.Limits.MaxTotalRiskPercent {
		issues = append(issues, fmt.Sprintf("Would push total risk to %.1f%% (limit %.0f%%)", newTotalRiskPercent, pm.Limits.MaxTotalRiskPercent))
	}

	// Check 4: Sector concentration
	sector := pm.getSector(ticker)
	sectorData, sectorExists := latest.SectorExposure[sector]
	if !sectorExists {
		sectorData = SectorData{Percent: 0, Count: 0}
	}

	newSectorPercent := sectorData.Percent + newPositionPercent
	newSectorCount := sectorData.Count + 1

	if newSectorPercent > pm.Limits.MaxSectorExposurePercent {
		issues = append(issues, fmt.Sprintf("Would push %s to %.1f%% (limit %.0f%%)", sector, newSectorPercent, pm.Limits.MaxSectorExposurePercent))
	}

	if newSectorCount > pm.Limits.MaxPositionsPerSector {
		issues = append(issues, fmt.Sprintf("Would be %dth position in %s (limit %d)", newSectorCount, sector, pm.Limits.MaxPositionsPerSector))
	}

	// Check 5: Correlation
	if correlationWithExisting > pm.Limits.MaxPairwiseCorrelation {
		issues = append(issues, fmt.Sprintf("Correlation %.2f too high (limit %.2f)", correlationWithExisting, pm.Limits.MaxPairwiseCorrelation))
	}

	// Check 6: Consecutive losses
	if pm.ConsecutiveLosses >= pm.Limits.ReduceSizeAfterLosses {
		issues = append(issues, fmt.Sprintf("In losing streak (%d losses) - should reduce size", pm.ConsecutiveLosses))
	}

	// Check 7: Violations present
	if len(latest.Violations) > 0 {
		issues = append(issues, fmt.Sprintf("%d active violations - fix before adding", len(latest.Violations)))
	}

	canAdd := len(issues) == 0

	recommendedSize := pm.calculateRecommendedSize(&latest, newPositionRisk)

	return &CanAddResult{
		CanAdd:             canAdd,
		Issues:             issues,
		Warnings:           latest.Warnings,
		RecommendedMaxSize: recommendedSize,
	}
}

// calculateRecommendedSize calculates recommended position size given current portfolio state.
func (pm *PortfolioManager) calculateRecommendedSize(
	latestSnapshot *RiskSnapshot,
	newPositionRisk float64,
) RecommendedSize {
	// Available risk budget
	availableRisk := pm.Limits.MaxTotalRiskPercent - latestSnapshot.TotalRiskPercent
	availableRiskVND := (availableRisk / 100) * pm.CurrentCapital

	// Adjust for consecutive losses
	sizeMultiplier := 1.0
	if pm.ConsecutiveLosses >= pm.Limits.ReduceSizeAfterLosses {
		sizeMultiplier = 0.5
	} else if pm.ConsecutiveLosses > 0 {
		sizeMultiplier = 0.75
	}

	// Adjust for violations
	if len(latestSnapshot.Violations) > 0 {
		sizeMultiplier *= 0.5
	}

	recommendedRisk := math.Min(newPositionRisk, availableRiskVND*sizeMultiplier)
	recommendedRiskPercent := (recommendedRisk / pm.CurrentCapital) * 100

	reason := pm.explainSizeRecommendation(sizeMultiplier, latestSnapshot)

	return RecommendedSize{
		RecommendedRiskVND:     recommendedRisk,
		RecommendedRiskPercent: recommendedRiskPercent,
		SizeMultiplier:         sizeMultiplier,
		Reason:                 reason,
	}
}

// explainSizeRecommendation explains why size was adjusted.
func (pm *PortfolioManager) explainSizeRecommendation(
	multiplier float64,
	snapshot *RiskSnapshot,
) string {
	if multiplier >= 1.0 {
		return "Normal sizing - portfolio healthy"
	}

	reasons := []string{}

	if pm.ConsecutiveLosses >= pm.Limits.ReduceSizeAfterLosses {
		reasons = append(reasons, fmt.Sprintf("Losing streak (%d losses)", pm.ConsecutiveLosses))
	}

	if len(snapshot.Violations) > 0 {
		reasons = append(reasons, fmt.Sprintf("%d active violations", len(snapshot.Violations)))
	}

	if snapshot.RiskUtilization > 80 {
		reasons = append(reasons, fmt.Sprintf("Risk utilization at %.0f%%", snapshot.RiskUtilization))
	}

	return fmt.Sprintf("Reduced to %.0f%% due to: %s", multiplier*100, strings.Join(reasons, ", "))
}
