package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	pb "github.com/nonobeam/golang-stock-trading/proto/ml"
)

// handleScanCommand handles the /scan [date] command.
//
// It invokes the Python weekly portfolio selection pipeline via gRPC
// and streams the result back to the requester as a Telegram message.
//
// Usage:
//
//	/scan              – use the most recent prediction date in the DB
//	/scan 2026-02-17   – use a specific prediction date (YYYY-MM-DD)
func (s *BotService) handleScanCommand(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	// --- Guard: ML gRPC client must be wired up ---
	if s.mlClient == nil {
		s.SendMessage(chatID,
			"<b>Scan Failed</b>\n\n"+
				"ML service client is not configured.\n"+
				"Please contact the administrator.")
		return
	}

	// --- Parse optional date argument ---
	args := strings.Fields(msg.Text)
	var dateArg string
	if len(args) >= 2 {
		dateArg = args[1]
	}

	// --- Notify user that scan has started ---
	dateDisplay := dateArg
	if dateDisplay == "" {
		dateDisplay = "latest available"
	}
	s.SendMessage(chatID, fmt.Sprintf(
		"🔍 <b>Portfolio Scan Started</b>\n\n"+
			"Prediction date: <code>%s</code>\n\n"+
			"Running the full pipeline (filter → score → optimize → compare)...\n"+
			"This usually takes 10–30 seconds.",
		dateDisplay,
	))

	// --- Run in background goroutine so the bot stays responsive ---
	go func(cid int64) {
		startTime := time.Now()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		logger.Info().
			Str("pred_date", dateArg).
			Msg("Starting portfolio scan via gRPC")

		resp, err := s.mlClient.RunWeeklyPortfolio(ctx, &pb.RunWeeklyPortfolioRequest{
			PredDate: dateArg,
		})
		elapsed := time.Since(startTime).Round(time.Second)

		if err != nil {
			logger.Error().Err(err).Msg("RunWeeklyPortfolio gRPC call failed")
			s.SendMessage(cid, fmt.Sprintf(
				"❌ <b>Portfolio Scan Failed</b>\n\n"+
					"Elapsed: %s\n"+
					"Error: %s\n\n"+
					"<i>Please ensure the ML service is running.</i>",
				elapsed, err.Error(),
			))
			return
		}

		if !resp.Success {
			logger.Error().Str("error", resp.ErrorMessage).Msg("RunWeeklyPortfolio returned failure")
			s.SendMessage(cid, fmt.Sprintf(
				"❌ <b>Portfolio Scan Failed</b>\n\n"+
					"Elapsed: %s\nDate: <code>%s</code>\n"+
					"Error: %s",
				elapsed, resp.PredDate, resp.ErrorMessage,
			))
			return
		}

		logger.Info().
			Dur("elapsed", elapsed).
			Str("pred_date", resp.PredDate).
			Int32("messages_sent", resp.MessagesSent).
			Msg("Portfolio scan completed successfully")

		// The Python selector already sent the detailed Telegram report.
		// We just send a brief confirmation here.
		s.SendMessage(cid, fmt.Sprintf(
			"✅ <b>Portfolio Scan Complete</b>\n\n"+
				"Date: <code>%s</code>\n"+
				"Elapsed: %s\n\n"+
				"<i>The full report has been sent above by the ML service (%d message(s)).</i>",
			resp.PredDate, elapsed, resp.MessagesSent,
		))
	}(chatID)
}
