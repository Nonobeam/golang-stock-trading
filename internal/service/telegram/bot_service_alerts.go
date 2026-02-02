package telegram

import (
	"fmt"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
	positionService "github.com/nonobeam/golang-stock-trading/internal/service/position"
)

// SendStopLossAlert sends an alert when a stop-loss is hit.
func (s *BotService) SendStopLossAlert(alert positionService.PositionAlert) {
	msg := fmt.Sprintf(
		"<b>STOP-LOSS HIT</b>\n\n"+
			"Symbol: <b>%s</b>\n"+
			"Current Price: %s VND\n"+
			"Stop Level: %s VND\n"+
			"R-Multiple: %.2fR\n\n"+
			"<b>Recommendation:</b> Consider selling to limit losses\n\n"+
			"%s",
		alert.Symbol,
		formatPrice(alert.CurrentPrice),
		formatPrice(alert.StopPrice),
		alert.RMultiple,
		alert.Message,
	)

	if err := s.Broadcast(msg); err != nil {
		logger.Error().
			Err(err).
			Str("symbol", alert.Symbol).
			Msg("Failed to send stop-loss alert")
	}
}

// SendBreakevenAlert sends an alert when position moves to breakeven.
func (s *BotService) SendBreakevenAlert(alert positionService.PositionAlert) {
	msg := fmt.Sprintf(
		"<b>BREAKEVEN PROTECTION ACTIVATED</b>\n\n"+
			"Symbol: <b>%s</b>\n"+
			"Current Price: %s VND\n"+
			"New Stop: %s VND (Breakeven)\n"+
			"R-Multiple: +%.2fR\n\n"+
			"<b>Your profit is now protected!</b>\n\n"+
			"%s",
		alert.Symbol,
		formatPrice(alert.CurrentPrice),
		formatPrice(alert.StopPrice),
		alert.RMultiple,
		alert.Message,
	)

	if err := s.Broadcast(msg); err != nil {
		logger.Error().
			Err(err).
			Str("symbol", alert.Symbol).
			Msg("Failed to send breakeven alert")
	}
}

// SendTargetAlert sends an alert when a price target is reached.
func (s *BotService) SendTargetAlert(alert positionService.PositionAlert) {
	msg := fmt.Sprintf(
		"<b>TARGET REACHED</b>\n\n"+
			"Symbol: <b>%s</b>\n"+
			"Current Price: %s VND\n"+
			"New Stop: %s VND\n"+
			"R-Multiple: +%.2fR\n\n"+
			"<b>Recommendation:</b> Consider taking partial profits\n\n"+
			"%s",
		alert.Symbol,
		formatPrice(alert.CurrentPrice),
		formatPrice(alert.StopPrice),
		alert.RMultiple,
		alert.Message,
	)

	if err := s.Broadcast(msg); err != nil {
		logger.Error().
			Err(err).
			Str("symbol", alert.Symbol).
			Msg("Failed to send target alert")
	}
}

// SendPositionAlert is a generic dispatcher for position alerts.
func (s *BotService) SendPositionAlert(alert positionService.PositionAlert) {
	switch alert.AlertType {
	case positionService.AlertStopLoss:
		s.SendStopLossAlert(alert)
	case positionService.AlertBreakeven:
		s.SendBreakevenAlert(alert)
	case positionService.AlertTarget:
		s.SendTargetAlert(alert)
	default:
		logger.Warn().
			Str("alertType", string(alert.AlertType)).
			Msg("Unknown alert type")
	}
}
