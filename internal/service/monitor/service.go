package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/service"
	"github.com/nonobeam/golang-stock-trading/internal/service/telegram"
	"github.com/nonobeam/golang-stock-trading/internal/vn"
)

// Constants for configuration
const (
	DropThreshold = -0.03 // -3%
	GainThreshold = 0.05  // +5%
	AlertCooldown = 30 * time.Minute
	CheckInterval = 1 * time.Minute
)

// PriceMonitorService monitors stock prices and sends alerts
type PriceMonitorService struct {
	marketData    *service.MarketDataService
	botService    *telegram.BotService
	watchlistRepo *repository.WatchlistRepository

	lastAlerts map[string]time.Time
	wasOpen    bool // Tracks previous market state
	mu         sync.RWMutex
}

// NewPriceMonitorService creates a new instance of PriceMonitorService
func NewPriceMonitorService(md *service.MarketDataService, bot *telegram.BotService, repo *repository.WatchlistRepository) *PriceMonitorService {
	// Initialize wasOpen correctly based on current state to avoid false start message
	return &PriceMonitorService{
		marketData:    md,
		botService:    bot,
		watchlistRepo: repo,
		lastAlerts:    make(map[string]time.Time),
		wasOpen:       vn.IsMarketOpen(),
	}
}

// Start begins the monitoring loop
func (s *PriceMonitorService) Start(ctx context.Context) {
	ticker := time.NewTicker(CheckInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Info().Msg("Price monitor service stopped")
				return
			case <-ticker.C:
				s.checkMarketStatus()
				
				if vn.IsMarketOpen() {
					s.CheckPrices(ctx)
				}
			}
		}
	}()
	logger.Info().Msg("Price monitor service started")
}

func (s *PriceMonitorService) checkMarketStatus() {
	currentOpen := vn.IsMarketOpen()

	// Transition: Closed -> Open
	if currentOpen && !s.wasOpen {
		s.botService.Broadcast("<b>Market is ON</b>\n\nStart trading now! Good luck.")
	}

	// Transition: Open -> Closed
	if !currentOpen && s.wasOpen {
		// Calculate time in VN timezone to check for Friday
		loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
		if err != nil {
			loc = time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)
		}
		vnTime := time.Now().In(loc)

		if vnTime.Weekday() == time.Friday {
			s.botService.Broadcast("<b>Market is OFF for weekend</b>\n\nCome back on Monday! Have a great weekend.")
		} else {
			s.botService.Broadcast("<b>Market is OFF</b>\n\nRest now. See you next session.")
		}
	}

	s.wasOpen = currentOpen
}

// CheckPrices fetches prices and triggers alerts if thresholds are met
func (s *PriceMonitorService) CheckPrices(ctx context.Context) {
	symbols, err := s.watchlistRepo.GetSymbols(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch watchlist symbols")
		return
	}

	// Ensure we are subscribed to these symbols
	if err := s.marketData.SubscribeStockInfo(symbols); err != nil {
		logger.Error().Err(err).Msg("Failed to subscribe to watchlist symbols")
	}

	for _, sym := range symbols {
		info := s.marketData.GetLatestStockInfo(sym)
		if info == nil || info.LastPrice == 0 || info.Reference == 0 {
			continue
		}

		changePct := (info.LastPrice - info.Reference) / info.Reference

		if changePct <= DropThreshold {
			s.sendAlert(sym, info.LastPrice, changePct, "PRICE_DROP", "Consider selling or checking stop loss.")
		} else if changePct >= GainThreshold {
			s.sendAlert(sym, info.LastPrice, changePct, "PRICE_GAIN", "Consider taking profit.")
		}
	}
}

func (s *PriceMonitorService) sendAlert(symbol string, price, change float64, alertType, advice string) {
	key := symbol + ":" + alertType

	s.mu.Lock()
	lastTime, ok := s.lastAlerts[key]
	if ok && time.Since(lastTime) < AlertCooldown {
		s.mu.Unlock()
		return
	}
	s.lastAlerts[key] = time.Now()
	s.mu.Unlock()

	if err := s.botService.NotifyPriceMonitorAlert(symbol, price, change, alertType, advice); err != nil {
		logger.Error().Err(err).Str("symbol", symbol).Msg("Failed to send price alert")
	}
}
