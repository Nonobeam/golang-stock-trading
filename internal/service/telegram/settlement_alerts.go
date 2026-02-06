package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/rs/zerolog"
)

// SettlementAlerter handles settlement-related alerts
type SettlementAlerter struct {
	botService *BotService
	posRepo    *repository.PositionRepository
	logger     *zerolog.Logger
}

// NewSettlementAlerter creates a new settlement alerter
func NewSettlementAlerter(botService *BotService, posRepo *repository.PositionRepository) *SettlementAlerter {
	return &SettlementAlerter{
		botService: botService,
		posRepo:    posRepo,
		logger:     logger.Get(),
	}
}

// SendPositionLiquidAlert sends alert when position becomes liquid
func (sa *SettlementAlerter) SendPositionLiquidAlert(ctx context.Context, position *repository.Position, chatID int64) error {
	message := FormatPositionLiquidNotification(position)

	err := sa.botService.SendMessage(chatID, message)
	if err != nil {
		sa.logger.Error().Err(err).
			Str("symbol", position.Symbol).
			Int64("chat_id", chatID).
			Msg("Failed to send position liquid alert")
		return err
	}

	sa.logger.Info().
		Str("symbol", position.Symbol).
		Int64("chat_id", chatID).
		Msg("Sent position liquid alert")

	return nil
}

// SendLockedRiskThresholdAlert sends alert when locked risk approaches threshold
func (sa *SettlementAlerter) SendLockedRiskThresholdAlert(
	ctx context.Context,
	userID int64,
	chatID int64,
	totalLockedRisk float64,
	maxAllowed float64,
	usedPercent float64,
) error {
	message := fmt.Sprintf(
		"⚠️ *Locked Risk Alert*\n\n"+
			"Your locked capital risk is approaching the threshold:\n"+
			"• Current: %s VND\n"+
			"• Threshold: %s VND\n"+
			"• Usage: %.1f%%\n\n"+
			"New purchases may be restricted until existing positions settle.",
		formatVND(totalLockedRisk),
		formatVND(maxAllowed),
		usedPercent,
	)

	err := sa.botService.SendMessage(chatID, message)
	if err != nil {
		sa.logger.Error().Err(err).
			Int64("user_id", userID).
			Int64("chat_id", chatID).
			Msg("Failed to send locked risk threshold alert")
		return err
	}

	sa.logger.Info().
		Int64("user_id", userID).
		Float64("used_percent", usedPercent).
		Msg("Sent locked risk threshold alert")

	return nil
}

// SendStopLossBreachAlert sends alert when stop loss breached but not executable
func (sa *SettlementAlerter) SendStopLossBreachAlert(
	ctx context.Context,
	position *repository.Position,
	chatID int64,
	stopPrice float64,
	actualPrice float64,
	daysUntilExecutable int,
) error {
	settlementStatus := "LOCKED"
	if position.SettlementStatus != nil {
		settlementStatus = *position.SettlementStatus
	}

	canSellDate := "unknown"
	if position.CanSellDate != nil {
		canSellDate = position.CanSellDate.Format("2006-01-02")
	}

	message := fmt.Sprintf(
		"🚨 *Stop Loss Breach (Non-Executable)*\n\n"+
			"Symbol: *%s*\n"+
			"Stop Loss: *%s VND*\n"+
			"Current Price: *%s VND*\n\n"+
			"⚠️ *Cannot execute*: Shares in settlement (%s)\n"+
			"Days until executable: *%d*\n"+
			"Can sell from: *%s*\n\n"+
			"⚠️ Price has hit your stop loss, but shares are locked in settlement period. "+
			"Monitor the price closely. If it recovers before settlement, you may avoid the loss.",
		position.Symbol,
		formatVND(stopPrice),
		formatVND(actualPrice),
		settlementStatus,
		daysUntilExecutable,
		canSellDate,
	)

	err := sa.botService.SendMessage(chatID, message)
	if err != nil {
		sa.logger.Error().Err(err).
			Str("symbol", position.Symbol).
			Int64("chat_id", chatID).
			Msg("Failed to send stop loss breach alert")
		return err
	}

	sa.logger.Warn().
		Str("symbol", position.Symbol).
		Float64("stop_price", stopPrice).
		Float64("actual_price", actualPrice).
		Str("settlement_status", settlementStatus).
		Msg("Sent stop loss breach alert (non-executable)")

	return nil
}

// CheckAndSendDailySettlementAlerts checks for positions transitioning to liquid
func (sa *SettlementAlerter) CheckAndSendDailySettlementAlerts(ctx context.Context) error {
	// This would be called by a daily job after settlement update
	// Get all users with positions (simplified - in production, get from user_config)
	// For now, this is a placeholder for the integration

	sa.logger.Info().Msg("Checking for daily settlement alerts")

	// TODO: Implement user iteration and alert sending
	// 1. Query all users with open positions
	// 2. For each user, check if any positions transitioned to LIQUID today
	// 3. Send alerts for newly liquid positions

	return nil
}

// MonitorLockedRiskThreshold monitors locked risk and sends alerts when approaching threshold
func (sa *SettlementAlerter) MonitorLockedRiskThreshold(ctx context.Context, userID int64, chatID int64, accountValue float64, threshold float64) error {
	totalLockedRisk, err := sa.posRepo.GetTotalLockedRisk(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get total locked risk: %w", err)
	}

	maxAllowed := accountValue * threshold
	usedPercent := (totalLockedRisk / maxAllowed) * 100

	// Send alert if above 80% of threshold
	if usedPercent > 80 {
		return sa.SendLockedRiskThresholdAlert(ctx, userID, chatID, totalLockedRisk, maxAllowed, usedPercent)
	}

	return nil
}

// NotifyPositionTransitionsToLiquid notifies user when positions become liquid
// This should be called by the daily settlement update job
func (sa *SettlementAlerter) NotifyPositionTransitionsToLiquid(ctx context.Context, positions []*repository.Position, chatID int64) error {
	for _, pos := range positions {
		if pos.IsLiquid() {
			// Check if this just transitioned (would need to query tracking table)
			// For now, just send notification
			err := sa.SendPositionLiquidAlert(ctx, pos, chatID)
			if err != nil {
				sa.logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("Failed to send liquid alert")
				// Continue with other positions
			}
		}
	}
	return nil
}

// SendEntryDayWarning sends warning about Thursday/Friday entry restrictions
func (sa *SettlementAlerter) SendEntryDayWarning(chatID int64, ticker string, dayOfWeek time.Weekday) error {
	dayName := dayOfWeek.String()

	message := fmt.Sprintf(
		"⚠️ *Entry Day Warning*\n\n"+
			"Symbol: *%s*\n"+
			"Entry Day: *%s*\n\n"+
			"⚡ Position size recommendation: *50%%*\n\n"+
			"Reason: %s entries extend settlement period over the weekend, "+
			"increasing the locked risk duration. Consider reducing position size to manage risk.",
		ticker,
		dayName,
		dayName,
	)

	return sa.botService.SendMessage(chatID, message)
}
