package market

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/websocket"
)

// IndexDataPoint represents a single market index snapshot
type IndexDataPoint struct {
	Timestamp int64   `json:"timestamp"` // Unix milliseconds
	Value     float64 `json:"value"`
}

// MarketIndex represents current index state
type MarketIndex struct {
	Name          string  `json:"name"`
	Value         float64 `json:"value"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
}

// MarketIndexWithData includes historical data points
type MarketIndexWithData struct {
	MarketIndex
	Data []IndexDataPoint `json:"data"`
}

// MarketIndicesResponse is the complete response for all indices
type MarketIndicesResponse struct {
	VNIndex    MarketIndexWithData `json:"vnIndex"`
	VN30       MarketIndexWithData `json:"vn30"`
	VN100      MarketIndexWithData `json:"vn100"`
	LastUpdate string              `json:"lastUpdate"`
}

// IndexHistoryResponse returns historical data for a specific index
type IndexHistoryResponse struct {
	Name string           `json:"name"`
	Data []IndexDataPoint `json:"data"`
}

// MarketRegimeResponse contains market regime analysis
type MarketRegimeResponse struct {
	Regime       string             `json:"regime"`       // trending-up, trending-down, choppy, volatile
	RegimeScore  int                `json:"regimeScore"`  // 1-10
	Breadth      MarketBreadth      `json:"breadth"`
	MarketStatus string             `json:"marketStatus"` // pre-market, open, closed, ato, atc
	LastUpdate   string             `json:"lastUpdate"`
}

// MarketBreadth represents market breadth metrics
type MarketBreadth struct {
	Advances  int `json:"advances"`
	Declines  int `json:"declines"`
	Unchanged int `json:"unchanged"`
}

// Service manages real-time market data with WebSocket integration
type Service struct {
	mu sync.RWMutex

	// Circular buffers for intraday data (last 390 minutes = 1 trading day)
	vnIndexData  []IndexDataPoint
	vn30Data     []IndexDataPoint
	vn100Data    []IndexDataPoint

	// Current values
	vnIndexCurrent *MarketIndex
	vn30Current    *MarketIndex
	vn100Current   *MarketIndex

	// Regime data
	currentRegime *MarketRegimeResponse

	// Dependencies
	wsClient      *websocket.Client

	// Control
	stopChan chan struct{}
}

// NewService creates and starts market data aggregation
func NewService(wsClient *websocket.Client) *Service {
	svc := &Service{
		vnIndexData: make([]IndexDataPoint, 0, 390),
		vn30Data:    make([]IndexDataPoint, 0, 390),
		vn100Data:   make([]IndexDataPoint, 0, 390),
		wsClient:    wsClient,
		stopChan:    make(chan struct{}),
		currentRegime: &MarketRegimeResponse{
			Regime: "choppy",
			RegimeScore: 5,
			Breadth: MarketBreadth{Advances: 0, Declines: 0, Unchanged: 0},
			MarketStatus: "open",
		},
	}

	// Subscribe to WebSocket index updates if client provided
	if wsClient != nil {
		wsClient.RegisterMarketIndexHandler(svc.handleIndexUpdate)
	}

	// Start periodic snapshots (every 1 minute)
	go svc.snapshotLoop()

	logger.Info().Msg("Market service started")

	return svc
}

// handleIndexUpdate receives real-time index updates from WebSocket
func (s *Service) handleIndexUpdate(data *websocket.MarketIndex) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch data.IndexName {
	case "VNINDEX":
		s.vnIndexCurrent = &MarketIndex{
			Name:          "VNINDEX",
			Value:         data.IndexValue,
			Change:        data.Change,
			ChangePercent: data.ChangeRate,
		}
	case "VN30":
		s.vn30Current = &MarketIndex{
			Name:          "VN30",
			Value:         data.IndexValue,
			Change:        data.Change,
			ChangePercent: data.ChangeRate,
		}
	case "VN100":
		s.vn100Current = &MarketIndex{
			Name:          "VN100",
			Value:         data.IndexValue,
			Change:        data.Change,
			ChangePercent: data.ChangeRate,
		}
	}
}

// snapshotLoop captures index data every minute
func (s *Service) snapshotLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.captureSnapshot()
		case <-s.stopChan:
			logger.Info().Msg("Market service snapshot loop stopped")
			return
		}
	}
}

// captureSnapshot stores current index values in circular buffers
func (s *Service) captureSnapshot() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()

	if s.vnIndexCurrent != nil {
		s.addDataPoint(&s.vnIndexData, now, s.vnIndexCurrent.Value)
	}
	if s.vn30Current != nil {
		s.addDataPoint(&s.vn30Data, now, s.vn30Current.Value)
	}
	if s.vn100Current != nil {
		s.addDataPoint(&s.vn100Data, now, s.vn100Current.Value)
	}
}

// addDataPoint adds to circular buffer (max 390 entries)
func (s *Service) addDataPoint(buffer *[]IndexDataPoint, timestamp int64, value float64) {
	point := IndexDataPoint{Timestamp: timestamp, Value: value}

	if len(*buffer) >= 390 {
		// Remove oldest, add newest (circular buffer)
		*buffer = append((*buffer)[1:], point)
	} else {
		*buffer = append(*buffer, point)
	}
}

// GetAllIndices returns current state of all indices with chart data
func (s *Service) GetAllIndices(ctx context.Context) (*MarketIndicesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Default values if no data yet
	vnIndex := s.vnIndexCurrent
	if vnIndex == nil {
		vnIndex = &MarketIndex{Name: "VNINDEX", Value: 1265.43, Change: 2.56, ChangePercent: 0.20}
	}

	vn30 := s.vn30Current
	if vn30 == nil {
		vn30 = &MarketIndex{Name: "VN30", Value: 1320.15, Change: 4.73, ChangePercent: 0.36}
	}

	vn100 := s.vn100Current
	if vn100 == nil {
		vn100 = &MarketIndex{Name: "VN100", Value: 1185.67, Change: 3.33, ChangePercent: 0.28}
	}

	return &MarketIndicesResponse{
		VNIndex: MarketIndexWithData{
			MarketIndex: *vnIndex,
			Data:        s.vnIndexData,
		},
		VN30: MarketIndexWithData{
			MarketIndex: *vn30,
			Data:        s.vn30Data,
		},
		VN100: MarketIndexWithData{
			MarketIndex: *vn100,
			Data:        s.vn100Data,
		},
		LastUpdate: time.Now().Format(time.RFC3339),
	}, nil
}

// GetIndexHistory returns historical data for a specific index
func (s *Service) GetIndexHistory(ctx context.Context, indexKey string, interval string, limit int) (*IndexHistoryResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var data []IndexDataPoint
	var name string

	switch indexKey {
	case "vnIndex":
		data = s.vnIndexData
		name = "VNINDEX"
	case "vn30":
		data = s.vn30Data
		name = "VN30"
	case "vn100":
		data = s.vn100Data
		name = "VN100"
	default:
		return nil, fmt.Errorf("invalid index key: %s", indexKey)
	}

	// Apply limit
	if limit > 0 && limit < len(data) {
		data = data[len(data)-limit:]
	}

	// TODO: Implement interval filtering (1m, 5m, 15m, 1h) by sampling
	// For now, return all 1-minute data

	return &IndexHistoryResponse{
		Name: name,
		Data: data,
	}, nil
}

// GetMarketRegime returns current market regime analysis
func (s *Service) GetMarketRegime(ctx context.Context) (*MarketRegimeResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If we have a regime detector, use it
	if s.currentRegime != nil {
		// Return stored regime
		return s.currentRegime, nil
	}

	// Return default regime if no detector
	return s.currentRegime, nil
}

// UpdateMarketStatus updates the current market status (open, closed, etc.)
func (s *Service) UpdateMarketStatus(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentRegime.MarketStatus = status
	s.currentRegime.LastUpdate = time.Now().Format(time.RFC3339)
}

// Stop gracefully stops the market service
func (s *Service) Stop() {
	close(s.stopChan)
	logger.Info().Msg("Market service stopped")
}
