package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nonobeam/golang-stock-trading/internal/api/dnse"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/websocket"
)

// MarketDataService manages real-time market data subscriptions and caching.
type MarketDataService struct {
	wsClient *websocket.Client
	cache    *MarketDataCache
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// MarketDataCache holds latest market data values.
type MarketDataCache struct {
	StockInfo map[string]*dnse.StockInfoMessage
	TopPrice  map[string]*dnse.TopPriceMessage
	VNIndex   *dnse.IndexMessage
	mu        sync.RWMutex
}

// NewMarketDataService creates a new market data service.
func NewMarketDataService(wsClient *websocket.Client) *MarketDataService {
	ctx, cancel := context.WithCancel(context.Background())
	return &MarketDataService{
		wsClient: wsClient,
		cache: &MarketDataCache{
			StockInfo: make(map[string]*dnse.StockInfoMessage),
			TopPrice:  make(map[string]*dnse.TopPriceMessage),
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start initializes the service and default subscriptions.
func (s *MarketDataService) Start() error {
	// Subscribe to VN-Index by default
	if err := s.SubscribeVNIndex(); err != nil {
		logger.Error().Err(err).Msg("Failed to subscribe to VN-Index")
	}
	return nil
}

// Stop stops the service.
func (s *MarketDataService) Stop() {
	s.cancel()
}

// SubscribeStockInfo subscribes to real-time stock info updates.
func (s *MarketDataService) SubscribeStockInfo(symbols []string) error {
	for _, sym := range symbols {
		topic := fmt.Sprintf("quotes/krx/mdds/v2/stockinfo/%s", sym)
		if err := s.wsClient.Subscribe(topic); err != nil {
			return err
		}

		// Register handler
		s.wsClient.RegisterTopicHandler(topic, func(data []byte) {
			var msg dnse.StockInfoMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				logger.Error().Err(err).Str("symbol", sym).Msg("Failed to parse StockInfo")
				return
			}
			msg.UpdateCeilingFloorStatus()
			s.cache.UpdateStockInfo(&msg)
		})
	}
	return nil
}

// SubscribeTopPrice subscribes to top-of-book price updates.
func (s *MarketDataService) SubscribeTopPrice(symbols []string) error {
	for _, sym := range symbols {
		topic := fmt.Sprintf("quotes/krx/mdds/v2/topprice/%s", sym)
		if err := s.wsClient.Subscribe(topic); err != nil {
			return err
		}

		s.wsClient.RegisterTopicHandler(topic, func(data []byte) {
			var msg dnse.TopPriceMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				logger.Error().Err(err).Str("symbol", sym).Msg("Failed to parse TopPrice")
				return
			}
			s.cache.UpdateTopPrice(&msg)
		})
	}
	return nil
}

// SubscribeVNIndex subscribes to VN-Index updates.
func (s *MarketDataService) SubscribeVNIndex() error {
	topic := "quotes/krx/mdds/v2/index/VNINDEX"
	if err := s.wsClient.Subscribe(topic); err != nil {
		return err
	}

	s.wsClient.RegisterTopicHandler(topic, func(data []byte) {
		var msg dnse.IndexMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			logger.Error().Err(err).Msg("Failed to parse VN-Index")
			return
		}
		s.cache.UpdateVNIndex(&msg)
	})
	return nil
}

// GetLatestStockInfo returns the latest cached stock info.
func (s *MarketDataService) GetLatestStockInfo(symbol string) *dnse.StockInfoMessage {
	return s.cache.GetStockInfo(symbol)
}

// GetLatestSpread returns the latest bid-ask spread percentage.
func (s *MarketDataService) GetLatestSpread(symbol string) float64 {
	info := s.cache.GetTopPrice(symbol)
	if info == nil {
		return 0
	}
	return info.CalculateSpread()
}

// GetVNIndex returns the latest VN-Index value.
func (s *MarketDataService) GetVNIndex() *dnse.IndexMessage {
	return s.cache.GetVNIndex()
}

// --- Cache Methods ---

func (c *MarketDataCache) UpdateStockInfo(msg *dnse.StockInfoMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.StockInfo[msg.Symbol] = msg
}

func (c *MarketDataCache) UpdateTopPrice(msg *dnse.TopPriceMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.TopPrice[msg.Symbol] = msg
}

func (c *MarketDataCache) UpdateVNIndex(msg *dnse.IndexMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.VNIndex = msg
}

func (c *MarketDataCache) GetStockInfo(symbol string) *dnse.StockInfoMessage {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.StockInfo[symbol]
}

func (c *MarketDataCache) GetTopPrice(symbol string) *dnse.TopPriceMessage {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.TopPrice[symbol]
}

func (c *MarketDataCache) GetVNIndex() *dnse.IndexMessage {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.VNIndex
}
