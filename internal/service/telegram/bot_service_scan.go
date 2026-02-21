package telegram

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nonobeam/golang-stock-trading/internal/config"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

// handleScanCommand handles the /scan [date] command.
//
// It invokes the Python weekly portfolio selection pipeline via exec.Command
// and streams the result back to the requester as a Telegram message.
//
// Usage:
//
//	/scan              – use the most recent prediction date in the DB
//	/scan 2026-02-17   – use a specific prediction date (YYYY-MM-DD)
func (s *BotService) handleScanCommand(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	// --- Parse optional date argument ---
	args := strings.Fields(msg.Text)
	var dateArg string
	if len(args) >= 2 {
		dateArg = args[1]
	}

	// --- Load path config ---
	cfg := config.Get()
	pythonPath := cfg.MLPythonPath
	mlServiceDir := cfg.MLServiceDir

	// Resolve to absolute paths (handles relative paths in config)
	if !filepath.IsAbs(pythonPath) {
		if abs, err := filepath.Abs(pythonPath); err == nil {
			pythonPath = abs
		}
	}
	if !filepath.IsAbs(mlServiceDir) {
		if abs, err := filepath.Abs(mlServiceDir); err == nil {
			mlServiceDir = abs
		}
	}

	// Sanity check: verify Python binary exists
	if _, err := os.Stat(pythonPath); err != nil {
		logger.Error().Err(err).Str("pythonPath", pythonPath).Msg("Python binary not found")
		s.SendMessage(chatID, fmt.Sprintf(
			"❌ <b>Scan Failed</b>\n\n"+
				"Python binary not found at:\n<code>%s</code>\n\n"+
				"Set <code>ML_PYTHON_PATH</code> in your .env file.",
			pythonPath,
		))
		return
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

		// Build command: python -m daily.run_weekly_portfolio [--date DATE]
		cmdArgs := []string{"-m", "daily.run_weekly_portfolio"}
		if dateArg != "" {
			cmdArgs = append(cmdArgs, "--date", dateArg)
		}

		cmd := exec.Command(pythonPath, cmdArgs...)
		cmd.Dir = mlServiceDir

		logger.Info().
			Str("cmd", pythonPath).
			Strs("args", cmdArgs).
			Str("cwd", mlServiceDir).
			Msg("Starting portfolio scan")

		output, err := cmd.CombinedOutput()
		elapsed := time.Since(startTime).Round(time.Second)
		outputStr := strings.TrimSpace(string(output))

		if err != nil {
			logger.Error().
				Err(err).
				Str("output", outputStr).
				Msg("Portfolio scan failed")

			// Trim output to avoid Telegram 4096 char limit
			if len(outputStr) > 800 {
				outputStr = "..." + outputStr[len(outputStr)-800:]
			}
			s.SendMessage(cid, fmt.Sprintf(
				"❌ <b>Portfolio Scan Failed</b>\n\n"+
					"Elapsed: %s\nError: %s\n\n"+
					"<pre>%s</pre>",
				elapsed, err.Error(), outputStr,
			))
			return
		}

		logger.Info().Dur("elapsed", elapsed).Msg("Portfolio scan completed successfully")

		// The Python script sends its own Telegram messages with the full report.
		// We just send a brief confirmation here.
		s.SendMessage(cid, fmt.Sprintf(
			"✅ <b>Portfolio Scan Complete</b>\n\n"+
				"Elapsed: %s\n\n"+
				"<i>The full report has been sent above by the ML service.</i>",
			elapsed,
		))
	}(chatID)
}
