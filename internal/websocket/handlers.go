package websocket

import (
	"encoding/json"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

const (
	TypeMarketIndex = "MARKET_INDEX"
	TypeStockInfo   = "STOCK_INFO"
	TypeTopPrice    = "TOP_PRICE"
	TypeBoardEvent  = "BOARD_EVENT"
	TypeOHLC        = "OHLC"
	TypeTick        = "TICK"
)

type MarketIndex struct {
	IndexName    string  `json:"indexName"`
	IndexValue   float64 `json:"indexValue"`
	Change       float64 `json:"change"`
	ChangeRate   float64 `json:"changeRate"`
	TotalVolume  int64   `json:"totalVolume"`
	TotalValue   float64 `json:"totalValue"`
	TradingTime  string  `json:"tradingTime"`
}

type StockInfo struct {
	Symbol        string  `json:"symbol"`
	RefPrice      float64 `json:"refPrice"`
	CeilingPrice  float64 `json:"ceilingPrice"`
	FloorPrice    float64 `json:"floorPrice"`
	MatchPrice    float64 `json:"matchPrice"`
	MatchVolume   int64   `json:"matchVolume"`
	TotalVolume   int64   `json:"totalVolume"`
	TotalValue    float64 `json:"totalValue"`
	Change        float64 `json:"change"`
	ChangeRate    float64 `json:"changeRate"`
	TradingTime   string  `json:"tradingTime"`
}

type TopPrice struct {
	Symbol     string    `json:"symbol"`
	BidPrices  []float64 `json:"bidPrices"`
	BidVolumes []int64   `json:"bidVolumes"`
	AskPrices  []float64 `json:"askPrices"`
	AskVolumes []int64   `json:"askVolumes"`
}

type BoardEvent struct {
	MarketId    string `json:"marketId"`
	ProductId   string `json:"productId"`
	SessionCode string `json:"sessionCode"`
	SessionName string `json:"sessionName"`
	EventTime   string `json:"eventTime"`
}

type OHLC struct {
	Symbol     string  `json:"symbol"`
	Resolution string  `json:"resolution"`
	Open       float64 `json:"open"`
	High       float64 `json:"high"`
	Low        float64 `json:"low"`
	Close      float64 `json:"close"`
	Volume     int64   `json:"volume"`
	Timestamp  int64   `json:"timestamp"`
}

type Tick struct {
	Symbol     string  `json:"symbol"`
	MatchPrice float64 `json:"matchPrice"`
	MatchVol   int64   `json:"matchVol"`
	Side       string  `json:"side"`
	Time       string  `json:"time"`
}

type (
	MarketIndexHandler func(data *MarketIndex)
	StockInfoHandler   func(data *StockInfo)
	TopPriceHandler    func(data *TopPrice)
	BoardEventHandler  func(data *BoardEvent)
	OHLCHandler        func(data *OHLC)
	TickHandler        func(data *Tick)
)

func (c *Client) RegisterMarketIndexHandler(handler MarketIndexHandler) {
	c.RegisterHandler(TypeMarketIndex, func(msgType string, payload []byte) {
		var data MarketIndex
		if err := json.Unmarshal(payload, &data); err != nil {
			logger.Warn().Err(err).Msg("Failed to parse MarketIndex")
			return
		}
		handler(&data)
	})
}

func (c *Client) RegisterStockInfoHandler(handler StockInfoHandler) {
	c.RegisterHandler(TypeStockInfo, func(msgType string, payload []byte) {
		var data StockInfo
		if err := json.Unmarshal(payload, &data); err != nil {
			logger.Warn().Err(err).Msg("Failed to parse StockInfo")
			return
		}
		handler(&data)
	})
}

func (c *Client) RegisterTopPriceHandler(handler TopPriceHandler) {
	c.RegisterHandler(TypeTopPrice, func(msgType string, payload []byte) {
		var data TopPrice
		if err := json.Unmarshal(payload, &data); err != nil {
			logger.Warn().Err(err).Msg("Failed to parse TopPrice")
			return
		}
		handler(&data)
	})
}

func (c *Client) RegisterBoardEventHandler(handler BoardEventHandler) {
	c.RegisterHandler(TypeBoardEvent, func(msgType string, payload []byte) {
		var data BoardEvent
		if err := json.Unmarshal(payload, &data); err != nil {
			logger.Warn().Err(err).Msg("Failed to parse BoardEvent")
			return
		}
		handler(&data)
	})
}

func (c *Client) RegisterOHLCHandler(handler OHLCHandler) {
	c.RegisterHandler(TypeOHLC, func(msgType string, payload []byte) {
		var data OHLC
		if err := json.Unmarshal(payload, &data); err != nil {
			logger.Warn().Err(err).Msg("Failed to parse OHLC")
			return
		}
		handler(&data)
	})
}

func (c *Client) RegisterTickHandler(handler TickHandler) {
	c.RegisterHandler(TypeTick, func(msgType string, payload []byte) {
		var data Tick
		if err := json.Unmarshal(payload, &data); err != nil {
			logger.Warn().Err(err).Msg("Failed to parse Tick")
			return
		}
		handler(&data)
	})
}
