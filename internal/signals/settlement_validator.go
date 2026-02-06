// Package signals provides entry signal detection for trading setups.
package signals

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/risk"
	"github.com/nonobeam/golang-stock-trading/internal/vn"
)

// RejectionReason represents why a signal was rejected
type RejectionReason string

const (
	RejectionNone                RejectionReason = ""
	RejectionLockedRiskBudget    RejectionReason = "LOCKED_RISK_BUDGET_EXCEEDED"
	RejectionEntryDayRestriction RejectionReason = "ENTRY_DAY_RESTRICTION"
	RejectionScoreTooLow         RejectionReason = "SCORE_TOO_LOW"
)

// SettlementValidator validates signals against T+2 settlement constraints
type SettlementValidator struct {
	db                  *sql.DB
	lockedRiskCalc      *risk.LockedRiskCalculator
	applyEntryDayLimits bool // Whether to enforce Thursday/Friday restrictions
}

// NewSettlementValidator creates a new settlement validator
func NewSettlementValidator(db *sql.DB, applyEntryDayLimits bool) *SettlementValidator {
	return &SettlementValidator{
		db:                  db,
		lockedRiskCalc:      risk.NewLockedRiskCalculator(db),
		applyEntryDayLimits: applyEntryDayLimits,
	}
}

// ValidationResult contains the result of settlement validation
type ValidationResult struct {
	Approved                bool
	RejectionReason         RejectionReason
	RejectionMessage        string
	OriginalPositionSize    int
	AdjustedPositionSize    int
	PositionSizeMultiplier  float64
	LockedRiskImpact        float64
	LockedRiskAfterPurchase float64
	LockedRiskBudgetUsed    float64
	EntryDayWarning         string
}

// ValidateSignal validates a signal against locked risk budget and entry day restrictions
func (v *SettlementValidator) ValidateSignal(
	ctx context.Context,
	userID int64,
	ticker string,
	positionSize int,
	entryPrice float64,
	accountValue float64,
	userConfig *repository.UserConfig,
) (*ValidationResult, error) {

	result := &ValidationResult{
		Approved:               true,
		OriginalPositionSize:   positionSize,
		AdjustedPositionSize:   positionSize,
		PositionSizeMultiplier: 1.0,
	}

	// Get locked risk threshold from user config
	threshold := userConfig.GetLockedRiskThreshold()

	// Determine exchange from ticker
	exchange := vn.GetExchangeFromTicker(ticker)

	// Step 1: Check locked risk budget
	canAfford, message, err := v.lockedRiskCalc.CanAffordLockedRisk(
		ctx, userID, positionSize, entryPrice, exchange, accountValue, threshold,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to check locked risk budget: %w", err)
	}

	if !canAfford {
		result.Approved = false
		result.RejectionReason = RejectionLockedRiskBudget
		result.RejectionMessage = message
		result.AdjustedPositionSize = 0
		return result, nil
	}

	// If there's a warning (approaching threshold), store it
	if message != "" {
		result.EntryDayWarning = message
	}

	// Calculate locked risk impact
	result.LockedRiskImpact = risk.CalculateLockedRisk(positionSize, entryPrice, exchange)
	currentLockedRisk, _ := v.lockedRiskCalc.GetTotalLockedRisk(ctx, userID)
	result.LockedRiskAfterPurchase = currentLockedRisk + result.LockedRiskImpact
	result.LockedRiskBudgetUsed = (result.LockedRiskAfterPurchase / (accountValue * threshold)) * 100

	// Step 2: Apply entry day restrictions if enabled
	if v.applyEntryDayLimits {
		multiplier := v.GetPositionSizeMultiplier(time.Now())
		if multiplier < 1.0 {
			result.AdjustedPositionSize = int(float64(positionSize) * multiplier)
			result.AdjustedPositionSize = risk.RoundToLotSize(float64(result.AdjustedPositionSize), risk.GetLotSize())
			result.PositionSizeMultiplier = multiplier

			// Recalculate locked risk with adjusted size
			result.LockedRiskImpact = risk.CalculateLockedRisk(result.AdjustedPositionSize, entryPrice, exchange)
			result.LockedRiskAfterPurchase = currentLockedRisk + result.LockedRiskImpact
			result.LockedRiskBudgetUsed = (result.LockedRiskAfterPurchase / (accountValue * threshold)) * 100

			dayName := time.Now().Weekday().String()
			result.EntryDayWarning = fmt.Sprintf(
				"Position size reduced to %.0f%% (%d shares) due to %s entry (weekend lock risk)",
				multiplier*100,
				result.AdjustedPositionSize,
				dayName,
			)
		}
	}

	return result, nil
}

// ValidateLockedRiskBudget checks if a purchase would exceed locked risk budget
func (v *SettlementValidator) ValidateLockedRiskBudget(
	ctx context.Context,
	userID int64,
	ticker string,
	positionSize int,
	entryPrice float64,
	accountValue float64,
	threshold float64,
) (bool, string, error) {

	exchange := vn.GetExchangeFromTicker(ticker)

	return v.lockedRiskCalc.CanAffordLockedRisk(
		ctx, userID, positionSize, entryPrice, exchange, accountValue, threshold,
	)
}

// ApplyEntryDayRestrictions reduces position size for Thursday/Friday entries
func (v *SettlementValidator) ApplyEntryDayRestrictions(positionSize int, entryDate time.Time) (int, string) {
	multiplier := v.GetPositionSizeMultiplier(entryDate)

	if multiplier >= 1.0 {
		return positionSize, ""
	}

	adjustedSize := int(float64(positionSize) * multiplier)
	adjustedSize = risk.RoundToLotSize(float64(adjustedSize), risk.GetLotSize())

	dayName := entryDate.Weekday().String()
	message := fmt.Sprintf(
		"Position size reduced to %.0f%% due to %s entry (weekend lock risk extends settlement period)",
		multiplier*100,
		dayName,
	)

	return adjustedSize, message
}

// GetPositionSizeMultiplier returns position size multiplier based on day of week
// Monday-Wednesday: 1.0 (full size)
// Thursday-Friday: 0.5 (50% size due to weekend lock risk)
func (v *SettlementValidator) GetPositionSizeMultiplier(entryDate time.Time) float64 {
	return vn.GetEntryDayMultiplier(entryDate)
}

// CalculateMaxAffordableShares determines maximum shares purchasable within locked risk budget
func (v *SettlementValidator) CalculateMaxAffordableShares(
	ctx context.Context,
	userID int64,
	ticker string,
	price float64,
	accountValue float64,
	threshold float64,
) (int, error) {

	exchange := vn.GetExchangeFromTicker(ticker)

	return v.lockedRiskCalc.CalculateMaxSharesForLockedRiskBudget(
		ctx, userID, price, exchange, accountValue, threshold,
	)
}

// GetLockedRiskSummary returns current locked risk status for user
func (v *SettlementValidator) GetLockedRiskSummary(
	ctx context.Context,
	userID int64,
	accountValue float64,
	threshold float64,
) (map[string]interface{}, error) {

	totalLockedRisk, err := v.lockedRiskCalc.GetTotalLockedRisk(ctx, userID)
	if err != nil {
		return nil, err
	}

	available, err := v.lockedRiskCalc.GetAvailableLockedRiskBudget(ctx, userID, accountValue, threshold)
	if err != nil {
		return nil, err
	}

	maxAllowed := accountValue * threshold
	usedPercent := (totalLockedRisk / maxAllowed) * 100

	return map[string]interface{}{
		"total_locked_risk_vnd": totalLockedRisk,
		"max_allowed_vnd":       maxAllowed,
		"available_budget_vnd":  available,
		"used_percent":          usedPercent,
		"threshold_percent":     threshold * 100,
		"account_value":         accountValue,
	}, nil
}
