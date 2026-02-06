package risk

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/vn"
)

// LockedRiskCalculator handles locked capital risk calculations for T+2 settlement
type LockedRiskCalculator struct {
	db *sql.DB
}

// NewLockedRiskCalculator creates a new locked risk calculator
func NewLockedRiskCalculator(db *sql.DB) *LockedRiskCalculator {
	return &LockedRiskCalculator{db: db}
}

// LockedRiskResult contains locked risk analysis for a user's portfolio
type LockedRiskResult struct {
	TotalLockedCapital  float64 // Total value of shares in settlement
	TotalLockedRisk     float64 // Worst-case floor-hit risk
	TotalLiquidCapital  float64 // Total value of liquid shares
	TotalLiquidRisk     float64 // Controlled risk (entry - stop)
	AccountValue        float64 // Total account value
	LockedRiskPercent   float64 // Locked risk as % of account
	LockedRiskThreshold float64 // User's configured threshold
	LockedRiskRemaining float64 // Budget remaining for new purchases
	CanAffordNew        bool    // Whether new purchases are allowed
	Positions           []PositionRiskBreakdown
}

// PositionRiskBreakdown shows risk details for a single position
type PositionRiskBreakdown struct {
	Symbol           string
	SettlementStatus string
	LockedCapital    float64
	LockedRisk       float64
	LiquidCapital    float64
	LiquidRisk       float64
	Exchange         string
	DaysUntilLiquid  int
}

// CalculateLockedRisk computes locked risk for shares based on exchange
func CalculateLockedRisk(shares int, price float64, exchange string) float64 {
	capital := float64(shares) * price
	multiplier := vn.GetLockedRiskMultiplierForExchange(exchange)
	return capital * multiplier
}

// CalculateLiquidRisk computes controlled risk for liquid shares
func CalculateLiquidRisk(shares int, entryPrice, stopLoss float64) float64 {
	if entryPrice <= stopLoss {
		return 0
	}
	return float64(shares) * (entryPrice - stopLoss)
}

// GetTotalLockedRisk calculates total locked risk across all positions
func (c *LockedRiskCalculator) GetTotalLockedRisk(ctx context.Context, userID int64) (float64, error) {
	posRepo := repository.NewPositionRepository(c.db)
	return posRepo.GetTotalLockedRisk(ctx, userID)
}

// GetAvailableLockedRiskBudget calculates remaining budget for new locked positions
func (c *LockedRiskCalculator) GetAvailableLockedRiskBudget(ctx context.Context, userID int64, accountValue float64, threshold float64) (float64, error) {
	totalLockedRisk, err := c.GetTotalLockedRisk(ctx, userID)
	if err != nil {
		return 0, err
	}

	maxAllowed := accountValue * threshold
	remaining := maxAllowed - totalLockedRisk

	if remaining < 0 {
		return 0, nil
	}

	return remaining, nil
}

// CanAffordLockedRisk checks if a new purchase fits within locked risk budget
func (c *LockedRiskCalculator) CanAffordLockedRisk(ctx context.Context, userID int64, shares int, price float64, exchange string, accountValue float64, threshold float64) (bool, string, error) {
	// Calculate locked risk for proposed purchase
	newLockedRisk := CalculateLockedRisk(shares, price, exchange)

	// Get current locked risk
	currentLockedRisk, err := c.GetTotalLockedRisk(ctx, userID)
	if err != nil {
		return false, "", err
	}

	// Check against threshold
	maxAllowed := accountValue * threshold
	totalAfterPurchase := currentLockedRisk + newLockedRisk

	if totalAfterPurchase > maxAllowed {
		return false, fmt.Sprintf(
			"Locked risk budget exceeded: current %.0f VND + new %.0f VND = %.0f VND > max %.0f VND (%.0f%% of account)",
			currentLockedRisk,
			newLockedRisk,
			totalAfterPurchase,
			maxAllowed,
			threshold*100,
		), nil
	}

	// Warn if approaching threshold (80%)
	if totalAfterPurchase > maxAllowed*0.8 {
		return true, fmt.Sprintf(
			"WARNING: Locked risk will be %.0f%% of threshold (%.0f / %.0f VND)",
			(totalAfterPurchase/maxAllowed)*100,
			totalAfterPurchase,
			maxAllowed,
		), nil
	}

	return true, "", nil
}

// GetRiskComposition returns breakdown of locked vs liquid risk
func (c *LockedRiskCalculator) GetRiskComposition(ctx context.Context, userID int64) (*LockedRiskResult, error) {
	posRepo := repository.NewPositionRepository(c.db)

	// Get all open positions with settlement data
	positions, err := posRepo.GetOpenPositions(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := &LockedRiskResult{
		Positions: make([]PositionRiskBreakdown, 0),
	}

	currentTime := time.Now()

	for _, pos := range positions {
		breakdown := PositionRiskBreakdown{
			Symbol: pos.Symbol,
		}

		if pos.SettlementStatus != nil {
			breakdown.SettlementStatus = *pos.SettlementStatus
		}

		if pos.Exchange != nil {
			breakdown.Exchange = *pos.Exchange
		} else {
			breakdown.Exchange = vn.GetExchangeFromTicker(pos.Symbol)
		}

		// Calculate locked vs liquid capital
		if pos.IsLocked() {
			if pos.LockedCapital != nil {
				breakdown.LockedCapital = *pos.LockedCapital
			} else {
				breakdown.LockedCapital = float64(pos.Quantity) * pos.EntryPrice
			}
			breakdown.LockedRisk = CalculateLockedRisk(pos.Quantity, pos.EntryPrice, breakdown.Exchange)
			result.TotalLockedCapital += breakdown.LockedCapital
			result.TotalLockedRisk += breakdown.LockedRisk

			// Calculate days until liquid
			if pos.PurchaseDate != nil {
				breakdown.DaysUntilLiquid = vn.GetDaysUntilLiquid(*pos.PurchaseDate, currentTime)
			}
		} else {
			if pos.LiquidCapital != nil {
				breakdown.LiquidCapital = *pos.LiquidCapital
			} else {
				breakdown.LiquidCapital = float64(pos.Quantity) * pos.EntryPrice
			}
			breakdown.LiquidRisk = CalculateLiquidRisk(pos.Quantity, pos.EntryPrice, pos.StopLoss)
			result.TotalLiquidCapital += breakdown.LiquidCapital
			result.TotalLiquidRisk += breakdown.LiquidRisk
		}

		result.Positions = append(result.Positions, breakdown)
	}

	return result, nil
}

// CheckLockedRiskThreshold validates locked risk is within configured limit
func (c *LockedRiskCalculator) CheckLockedRiskThreshold(ctx context.Context, userID int64, accountValue float64, threshold float64) (bool, string, error) {
	currentLockedRisk, err := c.GetTotalLockedRisk(ctx, userID)
	if err != nil {
		return false, "", err
	}

	maxAllowed := accountValue * threshold
	lockedPercent := (currentLockedRisk / accountValue) * 100

	if currentLockedRisk > maxAllowed {
		return false, fmt.Sprintf(
			"Locked risk threshold exceeded: %.0f VND (%.2f%%) > max %.0f VND (%.0f%%)",
			currentLockedRisk,
			lockedPercent,
			maxAllowed,
			threshold*100,
		), nil
	}

	// Warn if above 80% of threshold
	if currentLockedRisk > maxAllowed*0.8 {
		return true, fmt.Sprintf(
			"WARNING: Locked risk at %.0f%% of threshold (%.0f / %.0f VND)",
			(currentLockedRisk/maxAllowed)*100,
			currentLockedRisk,
			maxAllowed,
		), nil
	}

	return true, "", nil
}

// CalculateMaxSharesForLockedRiskBudget determines max shares purchasable within budget
func (c *LockedRiskCalculator) CalculateMaxSharesForLockedRiskBudget(ctx context.Context, userID int64, price float64, exchange string, accountValue float64, threshold float64) (int, error) {
	availableBudget, err := c.GetAvailableLockedRiskBudget(ctx, userID, accountValue, threshold)
	if err != nil {
		return 0, err
	}

	if availableBudget <= 0 {
		return 0, nil
	}

	// Calculate max capital we can lock
	multiplier := vn.GetLockedRiskMultiplierForExchange(exchange)
	maxCapital := availableBudget / multiplier

	// Convert to shares
	maxShares := int(maxCapital / price)

	// Round down to lot size
	return RoundToLotSize(float64(maxShares), GetLotSize()), nil
}
