package risk_test

import (
	"testing"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/position"
	"github.com/nonobeam/golang-stock-trading/internal/risk"
)

// Helper function to create a test position
func createTestPosition(ticker string, entry, stop float64, shares int, posType string) *position.Position {
	pos := &position.Position{
		PositionID:      ticker + "-001",
		Ticker:          ticker,
		EntryDate:       time.Now(),
		EntryPrice:      entry,
		Shares:          shares,
		SharesRemaining: shares,
		StopLoss:        stop,
		PositionType:    posType,
		CurrentPrice:    entry,
		PositionValue:   entry * float64(shares),
		RiskPerShare:    entry - stop,
		TotalRisk:       (entry - stop) * float64(shares),
		Targets:         []position.Target{},
		Exits:           []position.Exit{},
		LastUpdated:     time.Now(),
	}
	pos.Initialize()
	return pos
}

func TestNewPortfolioManager(t *testing.T) {
	pm := risk.NewPortfolioManager(100_000_000, nil)

	if pm == nil {
		t.Fatal("Expected non-nil portfolio manager")
	}

	if pm.InitialCapital != 100_000_000 {
		t.Errorf("Expected initial capital 100M, got %.0f", pm.InitialCapital)
	}

	if pm.Limits == nil {
		t.Fatal("Expected non-nil limits")
	}

	// Check default limits
	if pm.Limits.MaxPositions != 8 {
		t.Errorf("Expected max positions 8, got %d", pm.Limits.MaxPositions)
	}
}

func TestCalculateRisk_EmptyPortfolio(t *testing.T) {
	pm := risk.NewPortfolioManager(100_000_000, nil)

	positions := make(map[string]*position.Position)
	prices := make(map[string]float64)

	snapshot := pm.CalculateRisk(positions, prices)

	if snapshot == nil {
		t.Fatal("Expected non-nil snapshot")
	}

	if snapshot.NumPositions != 0 {
		t.Errorf("Expected 0 positions, got %d", snapshot.NumPositions)
	}

	if snapshot.TotalRiskAmount != 0 {
		t.Errorf("Expected 0 risk, got %.0f", snapshot.TotalRiskAmount)
	}

	if snapshot.UsedCapital != 0 {
		t.Errorf("Expected 0 used capital, got %.0f", snapshot.UsedCapital)
	}

	if snapshot.AvailableCapital != 100_000_000 {
		t.Errorf("Expected 100M available, got %.0f", snapshot.AvailableCapital)
	}
}

func TestCalculateRisk_WithPositions(t *testing.T) {
	pm := risk.NewPortfolioManager(100_000_000, nil)

	// Create test positions
	positions := make(map[string]*position.Position)

	// FPT: 85k entry, 82k stop, 100 shares = 300k risk
	fptPos := createTestPosition("FPT", 85000, 82000, 100, "long")
	positions["FPT-001"] = fptPos

	// VCB: 90k entry, 87k stop, 80 shares = 240k risk
	vcbPos := createTestPosition("VCB", 90000, 87000, 80, "long")
	positions["VCB-001"] = vcbPos

	prices := map[string]float64{
		"FPT": 86000,
		"VCB": 91000,
	}

	snapshot := pm.CalculateRisk(positions, prices)

	if snapshot.NumPositions != 2 {
		t.Errorf("Expected 2 positions, got %d", snapshot.NumPositions)
	}

	// FPT: 86k * 100 = 8.6M
	// VCB: 91k * 80 = 7.28M
	// Total: 15.88M
	expectedUsed := 86000*100 + 91000*80
	if snapshot.UsedCapital != float64(expectedUsed) {
		t.Errorf("Expected %.0f used capital, got %.0f", float64(expectedUsed), snapshot.UsedCapital)
	}

	// Risk: FPT = (86k - 82k) * 100 = 400k
	//       VCB = (91k - 87k) * 80 = 320k
	//       Total = 720k = 0.72% of 100M
	expectedRisk := (86000-82000)*100 + (91000-87000)*80
	if snapshot.TotalRiskAmount != float64(expectedRisk) {
		t.Errorf("Expected %.0f total risk, got %.0f", float64(expectedRisk), snapshot.TotalRiskAmount)
	}

	// Check sector exposure
	if len(snapshot.SectorExposure) == 0 {
		t.Error("Expected sector exposure data")
	}

	// FPT should be in Technology
	if techSector, ok := snapshot.SectorExposure["Technology"]; !ok {
		t.Error("Expected Technology sector")
	} else {
		if techSector.Count != 1 {
			t.Errorf("Expected 1 technology position, got %d", techSector.Count)
		}
	}

	// VCB should be in Banking
	if bankSector, ok := snapshot.SectorExposure["Banking"]; !ok {
		t.Error("Expected Banking sector")
	} else {
		if bankSector.Count != 1 {
			t.Errorf("Expected 1 banking position, got %d", bankSector.Count)
		}
	}
}

func TestCheckRiskLimits_TotalRiskViolation(t *testing.T) {
	limits := risk.DefaultRiskLimits()
	limits.MaxTotalRiskPercent = 1.0 // Very low limit for testing

	pm := risk.NewPortfolioManager(100_000_000, limits)

	positions := make(map[string]*position.Position)

	// Create position with high risk
	// Entry 85k, stop 75k, 200 shares = 2M risk = 2% of capital
	fptPos := createTestPosition("FPT", 85000, 75000, 200, "long")
	positions["FPT-001"] = fptPos

	prices := map[string]float64{
		"FPT": 85000,
	}

	snapshot := pm.CalculateRisk(positions, prices)

	// Should have violation since risk (2%) > limit (1%)
	if len(snapshot.Violations) == 0 {
		t.Error("Expected risk violation")
	}

	foundViolation := false
	for _, v := range snapshot.Violations {
		if v.Type == "TOTAL_RISK_EXCEEDED" {
			foundViolation = true
			break
		}
	}

	if !foundViolation {
		t.Error("Expected TOTAL_RISK_EXCEEDED violation")
	}
}

func TestCheckRiskLimits_SectorConcentration(t *testing.T) {
	limits := risk.DefaultRiskLimits()
	limits.MaxSectorExposurePercent = 10.0 // Low limit for testing

	pm := risk.NewPortfolioManager(100_000_000, limits)

	positions := make(map[string]*position.Position)

	// Create multiple banking positions (VCB, BID)
	// Each 10M = 10% of capital, total banking = 20%
	vcbPos := createTestPosition("VCB", 100000, 97000, 100, "long")
	positions["VCB-001"] = vcbPos

	bidPos := createTestPosition("BID", 100000, 97000, 100, "long")
	positions["BID-001"] = bidPos

	prices := map[string]float64{
		"VCB": 100000,
		"BID": 100000,
	}

	snapshot := pm.CalculateRisk(positions, prices)

	// Should have violation for Banking sector > 10%
	foundViolation := false
	for _, v := range snapshot.Violations {
		if v.Type == "SECTOR_CONCENTRATION" {
			foundViolation = true
			break
		}
	}

	if !foundViolation {
		t.Error("Expected SECTOR_CONCENTRATION violation")
	}
}

func TestCanAddPosition_NoExistingPositions(t *testing.T) {
	pm := risk.NewPortfolioManager(100_000_000, nil)

	result := pm.CanAddPosition(
		10_000_000, // 10M position value
		500_000,    // 500k risk
		"FPT",
		0.0, // no correlation
	)

	if !result.CanAdd {
		t.Error("Expected to be able to add position when portfolio is empty")
	}

	if len(result.Issues) > 0 {
		t.Errorf("Expected no issues, got: %v", result.Issues)
	}
}

func TestCanAddPosition_ExceedsMaxPositions(t *testing.T) {
	limits := risk.DefaultRiskLimits()
	limits.MaxPositions = 2

	pm := risk.NewPortfolioManager(100_000_000, limits)

	// Add 2 positions first
	positions := make(map[string]*position.Position)
	positions["FPT-001"] = createTestPosition("FPT", 85000, 82000, 100, "long")
	positions["VCB-001"] = createTestPosition("VCB", 90000, 87000, 100, "long")

	prices := map[string]float64{
		"FPT": 85000,
		"VCB": 90000,
	}

	pm.CalculateRisk(positions, prices)

	// Try to add 3rd position
	result := pm.CanAddPosition(
		10_000_000,
		500_000,
		"VNM",
		0.0,
	)

	if result.CanAdd {
		t.Error("Expected cannot add position when at max positions")
	}

	if len(result.Issues) == 0 {
		t.Error("Expected issues when exceeding max positions")
	}

	foundIssue := false
	for _, issue := range result.Issues {
		if issue != "" && len(issue) > 0 {
			foundIssue = true
			break
		}
	}

	if !foundIssue {
		t.Error("Expected max positions issue")
	}
}

func TestRecordTradeResult(t *testing.T) {
	pm := risk.NewPortfolioManager(100_000_000, nil)

	// Record a loss
	pm.RecordTradeResult("loss")
	if pm.ConsecutiveLosses != 1 {
		t.Errorf("Expected 1 consecutive loss, got %d", pm.ConsecutiveLosses)
	}
	if pm.ConsecutiveWins != 0 {
		t.Errorf("Expected 0 consecutive wins, got %d", pm.ConsecutiveWins)
	}

	// Record another loss
	pm.RecordTradeResult("loss")
	if pm.ConsecutiveLosses != 2 {
		t.Errorf("Expected 2 consecutive losses, got %d", pm.ConsecutiveLosses)
	}

	// Record a win - should reset losses
	pm.RecordTradeResult("win")
	if pm.ConsecutiveLosses != 0 {
		t.Errorf("Expected 0 consecutive losses after win, got %d", pm.ConsecutiveLosses)
	}
	if pm.ConsecutiveWins != 1 {
		t.Errorf("Expected 1 consecutive win, got %d", pm.ConsecutiveWins)
	}
}

func TestGetDashboardSummary(t *testing.T) {
	pm := risk.NewPortfolioManager(100_000_000, nil)

	// Empty portfolio
	summary := pm.GetDashboardSummary()
	if summary != "No positions - portfolio empty" {
		t.Errorf("Expected empty portfolio message, got: %s", summary)
	}

	// Add position
	positions := make(map[string]*position.Position)
	positions["FPT-001"] = createTestPosition("FPT", 85000, 82000, 100, "long")

	prices := map[string]float64{
		"FPT": 86000,
	}

	pm.CalculateRisk(positions, prices)

	summary = pm.GetDashboardSummary()

	// Check that dashboard contains expected sections
	expectedSections := []string{
		"PORTFOLIO RISK DASHBOARD",
		"CAPITAL METRICS",
		"RISK METRICS",
		"POSITIONS",
		"SECTOR EXPOSURE",
		"PERFORMANCE",
		"STATUS",
	}

	for _, section := range expectedSections {
		if !contains(summary, section) {
			t.Errorf("Expected dashboard to contain '%s'", section)
		}
	}
}

func TestPerformanceMetrics_WithProfitAndLoss(t *testing.T) {
	pm := risk.NewPortfolioManager(100_000_000, nil)

	positions := make(map[string]*position.Position)

	// Winning position: Entry 85k, current 90k, 100 shares = +500k unrealized
	winPos := createTestPosition("FPT", 85000, 82000, 100, "long")
	positions["FPT-001"] = winPos

	// Losing position: Entry 90k, current 88k, 100 shares = -200k unrealized
	losePos := createTestPosition("VCB", 90000, 87000, 100, "long")
	positions["VCB-001"] = losePos

	prices := map[string]float64{
		"FPT": 90000,
		"VCB": 88000,
	}

	snapshot := pm.CalculateRisk(positions, prices)

	// Total unrealized: +500k - 200k = +300k
	expectedPL := (90000-85000)*100 + (88000-90000)*100
	if snapshot.TotalUnrealizedPL != float64(expectedPL) {
		t.Errorf("Expected %.0f total P&L, got %.0f", float64(expectedPL), snapshot.TotalUnrealizedPL)
	}

	// Current capital should be initial + P&L
	expectedCapital := 100_000_000 + float64(expectedPL)
	if pm.CurrentCapital != expectedCapital {
		t.Errorf("Expected %.0f current capital, got %.0f", expectedCapital, pm.CurrentCapital)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
