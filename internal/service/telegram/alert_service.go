package telegram

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

// AlertService monitors portfolio state and sends risk/execution alerts
type AlertService struct {
	botService      *BotService
	riskManager     RiskManager
	positionTracker PositionTracker
	thresholds      *AlertThresholds

	// Anti-spam: track last alert time per type
	lastAlerts map[string]time.Time
	mu         sync.RWMutex
}

// AlertThresholds defines when to send warnings and critical alerts
type AlertThresholds struct {
	// Portfolio risk thresholds (as percentage, e.g., 0.048 = 4.8%)
	PortfolioRiskWarning  float64 // Warning at 80% of max
	PortfolioRiskCritical float64 // Critical at 90% of max

	// Daily loss thresholds
	DailyLossWarning float64 // Warning at 80% of daily limit

	// Capital utilization
	CapitalUtilWarning float64 // Warning at 90% capital used

	// Anti-spam: minimum time between same alert type
	AlertCooldown time.Duration // Default: 5 minutes
}


// NewAlertService creates a new alert service
func NewAlertService(bot *BotService, riskMgr RiskManager, posMgr PositionTracker, thresholds *AlertThresholds) *AlertService {
	if thresholds.AlertCooldown == 0 {
		thresholds.AlertCooldown = 5 * time.Minute
	}

	return &AlertService{
		botService:      bot,
		riskManager:     riskMgr,
		positionTracker: posMgr,
		thresholds:      thresholds,
		lastAlerts:      make(map[string]time.Time),
	}
}

// StartMonitoring begins periodic monitoring of portfolio and positions
func (a *AlertService) StartMonitoring(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info().Dur("interval", interval).Msg("Alert service monitoring started")

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Alert service monitoring stopped")
			return
		case <-ticker.C:
			// Check risk limits
			a.CheckRiskLimits()

			// Monitor positions
			a.MonitorPositions()
		}
	}
}

// CheckRiskLimits monitors portfolio risk and sends alerts if thresholds exceeded
func (a *AlertService) CheckRiskLimits() {
	// Check portfolio risk
	portfolioRisk := a.riskManager.GetPortfolioRisk()
	maxRisk := a.riskManager.GetMaxPortfolioRisk()

	if portfolioRisk >= a.thresholds.PortfolioRiskCritical {
		if a.shouldSendAlert("portfolio_risk_critical") {
			a.notifyRiskCritical("Portfolio Risk", portfolioRisk, maxRisk)
			a.recordAlert("portfolio_risk_critical")
		}
	} else if portfolioRisk >= a.thresholds.PortfolioRiskWarning {
		if a.shouldSendAlert("portfolio_risk_warning") {
			a.notifyRiskWarning("Portfolio Risk", portfolioRisk, a.thresholds.PortfolioRiskWarning)
			a.recordAlert("portfolio_risk_warning")
		}
	}

	// Check daily loss
	dailyLoss := a.riskManager.GetDailyLoss()
	dailyLimit := a.riskManager.GetDailyLossLimit()

	if dailyLoss >= dailyLimit {
		if a.shouldSendAlert("daily_loss_critical") {
			a.notifyRiskCritical("Daily Loss", dailyLoss, dailyLimit)
			a.recordAlert("daily_loss_critical")
		}
	} else if dailyLoss >= a.thresholds.DailyLossWarning {
		if a.shouldSendAlert("daily_loss_warning") {
			a.notifyRiskWarning("Daily Loss", dailyLoss, a.thresholds.DailyLossWarning)
			a.recordAlert("daily_loss_warning")
		}
	}

	// Check capital utilization
	capitalUtil := a.riskManager.GetCapitalUtilization()

	if capitalUtil >= a.thresholds.CapitalUtilWarning {
		if a.shouldSendAlert("capital_util_warning") {
			a.notifyCapitalWarning(capitalUtil)
			a.recordAlert("capital_util_warning")
		}
	}
}

// MonitorPositions checks active positions for execution alerts
func (a *AlertService) MonitorPositions() {
	positions := a.positionTracker.GetActivePositions()

	for _, pos := range positions {
		// Check if near target (>= 95% progress)
		if pos.TargetProgress >= 95.0 {
			alertKey := fmt.Sprintf("target_approaching_%s", pos.Symbol)
			if a.shouldSendAlert(alertKey) {
				a.notifyTargetApproaching(pos)
				a.recordAlert(alertKey)
			}
		}

		// Check if near stop (<= 2% away)
		if pos.StopDistance <= 2.0 && pos.StopDistance > 0 {
			alertKey := fmt.Sprintf("stop_approaching_%s", pos.Symbol)
			if a.shouldSendAlert(alertKey) {
				a.notifyStopApproaching(pos)
				a.recordAlert(alertKey)
			}
		}

		// Check if held too long (> 10 days)
		timeHeld := time.Since(pos.EntryDate)
		if timeHeld > 10*24*time.Hour {
			alertKey := fmt.Sprintf("time_stop_%s", pos.Symbol)
			if a.shouldSendAlert(alertKey) {
				a.notifyTimeStop(pos, timeHeld)
				a.recordAlert(alertKey)
			}
		}
	}
}

// shouldSendAlert checks if enough time has passed since last alert of this type
func (a *AlertService) shouldSendAlert(alertType string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	lastTime, exists := a.lastAlerts[alertType]
	if !exists {
		return true
	}

	return time.Since(lastTime) > a.thresholds.AlertCooldown
}

// recordAlert records the time of an alert to prevent spam
func (a *AlertService) recordAlert(alertType string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastAlerts[alertType] = time.Now()
}

// Notification methods

func (a *AlertService) notifyRiskWarning(alertType string, current, threshold float64) {
	percentage := current * 100
	thresholdPct := threshold * 100

	msg := fmt.Sprintf(
		"<b>RISK WARNING</b>\n\n"+
			"<b>Type:</b> %s\n"+
			"<b>Current:</b> %.2f%%\n"+
			"<b>Threshold:</b> %.2f%%\n\n"+
			"<i>Approaching limit. Monitor carefully.</i>",
		alertType,
		percentage,
		thresholdPct,
	)

	if err := a.botService.Broadcast(msg); err != nil {
		logger.Error().Err(err).Str("type", alertType).Msg("Failed to send risk warning")
	}
}

func (a *AlertService) notifyRiskCritical(alertType string, current, limit float64) {
	percentage := current * 100
	limitPct := limit * 100

	msg := fmt.Sprintf(
		"<b>CRITICAL RISK ALERT</b>\n\n"+
			"<b>Type:</b> %s\n"+
			"<b>Current:</b> %.2f%%\n"+
			"<b>Maximum:</b> %.2f%%\n\n"+
			"<b>LIMIT EXCEEDED</b>\n"+
			"<i>Reduce positions immediately!</i>",
		alertType,
		percentage,
		limitPct,
	)

	if err := a.botService.Broadcast(msg); err != nil {
		logger.Error().Err(err).Str("type", alertType).Msg("Failed to send critical alert")
	}
}

func (a *AlertService) notifyCapitalWarning(utilization float64) {
	msg := fmt.Sprintf(
		"<b>CAPITAL UTILIZATION WARNING</b>\n\n"+
			"<b>Current:</b> %.1f%%\n"+
			"<b>Threshold:</b> 90%%\n\n"+
			"<i>Limited capital remaining for new positions.</i>",
		utilization*100,
	)

	if err := a.botService.Broadcast(msg); err != nil {
		logger.Error().Err(err).Msg("Failed to send capital warning")
	}
}

func (a *AlertService) notifyTargetApproaching(pos Position) {
	nextTarget := 0.0
	targetNum := 0
	for i, t := range pos.Targets {
		if t > pos.CurrentPrice {
			nextTarget = t
			targetNum = i + 1
			break
		}
	}

	if nextTarget == 0 {
		return // All targets hit
	}

	percentToTarget := ((nextTarget - pos.CurrentPrice) / pos.CurrentPrice) * 100

	msg := fmt.Sprintf(
		"<b>TARGET APPROACHING</b>\n\n"+
			"<b>Symbol:</b> <code>%s</code>\n"+
			"<b>Target:</b> T%d at %s VND\n"+
			"<b>Current:</b> %s VND\n"+
			"<b>Progress:</b> %.1f%%\n"+
			"<b>Distance:</b> +%.1f%%\n\n"+
			"<i>Consider taking partial profit soon.</i>",
		pos.Symbol,
		targetNum,
		formatPrice(nextTarget),
		formatPrice(pos.CurrentPrice),
		pos.TargetProgress,
		percentToTarget,
	)

	if err := a.botService.Broadcast(msg); err != nil {
		logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("Failed to send target alert")
	}
}

func (a *AlertService) notifyStopApproaching(pos Position) {
	msg := fmt.Sprintf(
		"<b>STOP LOSS APPROACHING</b>\n\n"+
			"<b>Symbol:</b> <code>%s</code>\n"+
			"<b>Current:</b> %s VND\n"+
			"<b>Stop:</b> %s VND\n"+
			"<b>Distance:</b> %.1f%% away\n"+
			"<b>Risk:</b> %.2fR\n\n"+
			"<i>Monitor closely. Prepare for exit.</i>",
		pos.Symbol,
		formatPrice(pos.CurrentPrice),
		formatPrice(pos.StopLoss),
		pos.StopDistance,
		pos.RMultiple,
	)

	if err := a.botService.Broadcast(msg); err != nil {
		logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("Failed to send stop alert")
	}
}

func (a *AlertService) notifyTimeStop(pos Position, timeHeld time.Duration) {
	days := int(timeHeld.Hours() / 24)

	msg := fmt.Sprintf(
		"<b>TIME STOP REMINDER</b>\n\n"+
			"<b>Symbol:</b> <code>%s</code>\n"+
			"<b>Held for:</b> %d days\n"+
			"<b>P&L:</b> %.2fR\n"+
			"<b>Current:</b> %s VND\n\n"+
			"<i>Position held too long. Consider closing if no clear progress.</i>",
		pos.Symbol,
		days,
		pos.RMultiple,
		formatPrice(pos.CurrentPrice),
	)

	if err := a.botService.Broadcast(msg); err != nil {
		logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("Failed to send time stop alert")
	}
}

// NotifyTargetHit sends celebration when target is reached
func (a *AlertService) NotifyTargetHit(pos Position, targetNum int, targetPrice float64) error {
	percentGain := ((targetPrice - pos.EntryPrice) / pos.EntryPrice) * 100

	msg := fmt.Sprintf(
		"<b>TARGET HIT!</b>\n\n"+
			"<b>Symbol:</b> <code>%s</code>\n"+
			"<b>Target:</b> T%d at %s VND\n"+
			"<b>Entry:</b> %s VND\n"+
			"<b>Gain:</b> +%.1f%% (+%.2fR)\n\n"+
			"<i>Congratulations! Consider taking profit.</i>",
		pos.Symbol,
		targetNum,
		formatPrice(targetPrice),
		formatPrice(pos.EntryPrice),
		percentGain,
		pos.RMultiple,
	)

	return a.botService.Broadcast(msg)
}

// NotifyStopHit sends notification when stop loss is triggered
func (a *AlertService) NotifyStopHit(pos Position) error {
	loss := ((pos.StopLoss - pos.EntryPrice) / pos.EntryPrice) * 100

	msg := fmt.Sprintf(
		"<b>STOP LOSS HIT</b>\n\n"+
			"<b>Symbol:</b> <code>%s</code>\n"+
			"<b>Stopped at:</b> %s VND\n"+
			"<b>Entry:</b> %s VND\n"+
			"<b>Loss:</b> %.1f%% (-1R)\n\n"+
			"<i>Move on to the next opportunity.</i>",
		pos.Symbol,
		formatPrice(pos.StopLoss),
		formatPrice(pos.EntryPrice),
		loss,
	)

	return a.botService.Broadcast(msg)
}
