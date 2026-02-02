package dnse

import (
	"time"
)

// StockInfoMessage represents real-time stock information from DNSE WebSocket.
// Stream: quotes/krx/mdds/v2/stockinfo/{symbol}
type StockInfoMessage struct {
	Symbol         string    `json:"symbol"`
	LastPrice      float64   `json:"lastPrice"`
	Change         float64   `json:"change"`
	ChangePercent  float64   `json:"changePercent"`
	Ceiling        float64   `json:"ceiling"`
	Floor          float64   `json:"floor"`
	Reference      float64   `json:"reference"`
	Volume         int64     `json:"volume"`
	Turnover       float64   `json:"turnover"`
	ForeignBuyVol  int64     `json:"foreignBuyVol"`
	ForeignSellVol int64     `json:"foreignSellVol"`
	ForeignRoom    int64     `json:"foreignRoom"`
	HitCeiling     bool      `json:"hitCeiling"` // Derived: LastPrice == Ceiling
	HitFloor       bool      `json:"hitFloor"`   // Derived: LastPrice == Floor
	Timestamp      time.Time `json:"timestamp"`
}

// TopPriceMessage represents top-of-book bid/ask prices.
// Stream: quotes/krx/mdds/v2/topprice/{symbol}
type TopPriceMessage struct {
	Symbol     string    `json:"symbol"`
	BidPrice1  float64   `json:"bidPrice1"`
	BidVolume1 int64     `json:"bidVolume1"`
	AskPrice1  float64   `json:"askPrice1"`
	AskVolume1 int64     `json:"askVolume1"`
	Timestamp  time.Time `json:"timestamp"`
}

// OHLCMessage represents OHLC bar data (1m, 5m, 15m, etc.).
// Stream: quotes/krx/mdds/v2/ohlc/intraday/{interval}/{symbol}
type OHLCMessage struct {
	Symbol    string    `json:"symbol"`
	Interval  string    `json:"interval"` // "1m", "5m", "15m", etc.
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    int64     `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

// IndexMessage represents market index data (VN-Index, VN30, etc.).
// Stream: quotes/krx/mdds/v2/index/{indexName}
type IndexMessage struct {
	IndexName string    `json:"indexName"` // "VNINDEX", "VN30", etc.
	Value     float64   `json:"value"`
	Change    float64   `json:"change"`
	Volume    int64     `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

// CalculateSpread calculates the bid-ask spread percentage.
func (t *TopPriceMessage) CalculateSpread() float64 {
	if t.BidPrice1 == 0 || t.AskPrice1 == 0 {
		return 0
	}
	midPrice := (t.BidPrice1 + t.AskPrice1) / 2
	spread := t.AskPrice1 - t.BidPrice1
	return (spread / midPrice) * 100
}

// UpdateCeilingFloorStatus updates HitCeiling and HitFloor flags.
func (s *StockInfoMessage) UpdateCeilingFloorStatus() {
	tolerance := 0.01 // 1 VND tolerance
	s.HitCeiling = (s.LastPrice >= s.Ceiling-tolerance)
	s.HitFloor = (s.LastPrice <= s.Floor+tolerance)
}
