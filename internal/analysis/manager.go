// Package analysis provides real-time analysis integration with market data streams.
package analysis

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/analysis/indicators"
	"github.com/nonobeam/golang-stock-trading/internal/data"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

// StockAnalyzer manages real-time analysis for a single stock.
type StockAnalyzer struct {
	symbol string
	series *data.Series

	// Indicators
	sma20      *indicators.SMA
	sma50      *indicators.SMA
	ema20      *indicators.EMA
	rsi        *indicators.RSI
	macd       *indicators.MACD
	atr        *indicators.ATR
	bbands     *indicators.BollingerBands
	adx        *indicators.ADX
	stochastic *indicators.Stochastic

	// Weekly indicators
	weeklySMA200 *indicators.SMA
	weeklyRSI    *indicators.RSI
	weeklyStoch  *indicators.Stochastic
	weeklySeries *data.Series

	mu sync.RWMutex
}

// AnalysisResult holds the latest indicator values.
type AnalysisResult struct {
	Symbol    string    `json:"symbol"`
	Timestamp time.Time `json:"timestamp"`

	// Price
	Close float64 `json:"close"`

	// Moving Averages
	SMA20 float64 `json:"sma20"`
	SMA50 float64 `json:"sma50"`
	EMA20 float64 `json:"ema20"`

	// Momentum
	RSI           float64 `json:"rsi"`
	MACD          float64 `json:"macd"`
	MACDSignal    float64 `json:"macdSignal"`
	MACDHistogram float64 `json:"macdHistogram"`

	// Volatility
	ATR       float64 `json:"atr"`
	BBUpper   float64 `json:"bbUpper"`
	BBMiddle  float64 `json:"bbMiddle"`
	BBLower   float64 `json:"bbLower"`
	BBPercentB float64 `json:"bbPercentB"`

	// Trend
	ADX     float64 `json:"adx"`
	PlusDI  float64 `json:"plusDI"`
	MinusDI float64 `json:"minusDI"`

	// Stochastic
	StochK float64 `json:"stochK"`
	StochD float64 `json:"stochD"`

	// Weekly Timeframe Indicators
	WeeklyClose  float64 `json:"weeklyClose"`
	WeeklySMA200 float64 `json:"weeklySMA200"`
	WeeklyRSI    float64 `json:"weeklyRSI"`
	WeeklyStochK float64 `json:"weeklyStochK"`
	WeeklyStochD float64 `json:"weeklyStochD"`
}

// NewStockAnalyzer creates a new analyzer for a symbol.
func NewStockAnalyzer(symbol string, maxBars int) *StockAnalyzer {
	if maxBars <= 0 {
		maxBars = 200 // Default lookback for longest indicator
	}

	return &StockAnalyzer{
		symbol:       symbol,
		series:       data.NewSeries(maxBars),
		sma20:        indicators.NewSMA(20),
		sma50:        indicators.NewSMA(50),
		ema20:        indicators.NewEMA(20), // Changed from ema20 to ema20 as per instruction, assuming typo in instruction and keeping original name
		rsi:          indicators.NewRSI(14),
		macd:         indicators.NewMACD(12, 26, 9),
		atr:          indicators.NewATR(14),
		bbands:       indicators.NewBollingerBands(20, 2.0),
		adx:          indicators.NewADX(14),
		stochastic:   indicators.NewStochastic(14, 3, 3),
		weeklySMA200: indicators.NewSMA(200),
		weeklyRSI:    indicators.NewRSI(14),
		weeklyStoch:  indicators.NewStochastic(14, 3, 3),
		weeklySeries: nil, // Will be computed on-demand
	}
}

// AddBar adds a new OHLCV bar and updates analysis.
func (a *StockAnalyzer) AddBar(bar data.OHLCV) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.series.Append(bar)
	// Invalidate weekly series cache when new daily bar is added
	a.weeklySeries = nil
}

// AddBarFromJSON parses WebSocket payload and adds bar.
func (a *StockAnalyzer) AddBarFromJSON(payload []byte) error {
	var bar struct {
		Timestamp int64   `json:"t"`
		Open      float64 `json:"o"`
		High      float64 `json:"h"`
		Low       float64 `json:"l"`
		Close     float64 `json:"c"`
		Volume    float64 `json:"v"`
	}

	if err := json.Unmarshal(payload, &bar); err != nil {
		return err
	}

	ohlcv := data.NewOHLCV(
		time.Unix(bar.Timestamp/1000, 0),
		bar.Open, bar.High, bar.Low, bar.Close, bar.Volume,
	)

	a.AddBar(ohlcv)
	return nil
}

// Analyze calculates all indicators and returns results.
func (a *StockAnalyzer) Analyze() (*AnalysisResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.series.Len() < 50 {
		return nil, data.ErrInsufficientData
	}

	last, _ := a.series.Last()

	result := &AnalysisResult{
		Symbol:    a.symbol,
		Timestamp: last.Timestamp,
		Close:     last.Close,
	}

	// Calculate indicators (ignore errors for optional indicators)
	if v, err := a.sma20.Calculate(a.series); err == nil {
		result.SMA20 = v
	}
	if v, err := a.sma50.Calculate(a.series); err == nil {
		result.SMA50 = v
	}
	if v, err := a.ema20.Calculate(a.series); err == nil {
		result.EMA20 = v
	}
	if v, err := a.rsi.Calculate(a.series); err == nil {
		result.RSI = v
	}

	if macdResult, err := a.macd.CalculateFull(a.series); err == nil {
		result.MACD = macdResult.MACD
		result.MACDSignal = macdResult.Signal
		result.MACDHistogram = macdResult.Histogram
	}

	if v, err := a.atr.Calculate(a.series); err == nil {
		result.ATR = v
	}

	if bbResult, err := a.bbands.CalculateFull(a.series); err == nil {
		result.BBUpper = bbResult.Upper
		result.BBMiddle = bbResult.Middle
		result.BBLower = bbResult.Lower
		result.BBPercentB = bbResult.PercentB
	}

	if adxResult, err := a.adx.CalculateFull(a.series); err == nil {
		result.ADX = adxResult.ADX
		result.PlusDI = adxResult.PlusDI
		result.MinusDI = adxResult.MinusDI
	}

	// Calculate Stochastic
	if stochResult, err := a.stochastic.CalculateFull(a.series); err == nil {
		result.StochK = stochResult.K
		result.StochD = stochResult.D
	}

	// Calculate weekly indicators
	a.calculateWeeklyIndicators(result)

	return result, nil
}

// calculateWeeklyIndicators computes weekly timeframe indicators.
func (a *StockAnalyzer) calculateWeeklyIndicators(result *AnalysisResult) {
	// Generate or use cached weekly series
	if a.weeklySeries == nil {
		a.weeklySeries = data.AggregateToWeekly(a.series)
	}

	if a.weeklySeries.Len() < 20 {
		// Not enough weekly data
		return
	}

	// Get weekly close
	if lastWeekly, err := a.weeklySeries.Last(); err == nil {
		result.WeeklyClose = lastWeekly.Close
	}

	// Calculate weekly SMA200
	if v, err := a.weeklySMA200.Calculate(a.weeklySeries); err == nil {
		result.WeeklySMA200 = v
	}

	// Calculate weekly RSI
	if v, err := a.weeklyRSI.Calculate(a.weeklySeries); err == nil {
		result.WeeklyRSI = v
	}

	// Calculate weekly Stochastic
	if stochResult, err := a.weeklyStoch.CalculateFull(a.weeklySeries); err == nil {
		result.WeeklyStochK = stochResult.K
		result.WeeklyStochD = stochResult.D
	}
}

// Symbol returns the analyzer's symbol.
func (a *StockAnalyzer) Symbol() string {
	return a.symbol
}

// BarCount returns the number of bars in the series.
func (a *StockAnalyzer) BarCount() int {
	return a.series.Len()
}

// GetSeries returns the underlying time series data.
func (a *StockAnalyzer) GetSeries() *data.Series {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.series
}

// AnalysisManager manages analyzers for multiple symbols.
type AnalysisManager struct {
	analyzers map[string]*StockAnalyzer
	mu        sync.RWMutex
}

// NewAnalysisManager creates a new analysis manager.
func NewAnalysisManager() *AnalysisManager {
	return &AnalysisManager{
		analyzers: make(map[string]*StockAnalyzer),
	}
}

// GetOrCreate returns existing analyzer or creates new one.
func (m *AnalysisManager) GetOrCreate(symbol string) *StockAnalyzer {
	m.mu.Lock()
	defer m.mu.Unlock()

	if analyzer, exists := m.analyzers[symbol]; exists {
		return analyzer
	}

	analyzer := NewStockAnalyzer(symbol, 200)
	m.analyzers[symbol] = analyzer
	logger.Debug().Str("symbol", symbol).Msg("Created new stock analyzer")

	return analyzer
}

// HandleOHLCMessage processes WebSocket OHLC messages.
func (m *AnalysisManager) HandleOHLCMessage(symbol string, payload []byte) error {
	analyzer := m.GetOrCreate(symbol)
	return analyzer.AddBarFromJSON(payload)
}

// GetAnalysis returns analysis for a symbol.
func (m *AnalysisManager) GetAnalysis(symbol string) (*AnalysisResult, error) {
	m.mu.RLock()
	analyzer, exists := m.analyzers[symbol]
	m.mu.RUnlock()

	if !exists {
		return nil, data.ErrInsufficientData
	}

	return analyzer.Analyze()
}

// Symbols returns list of tracked symbols.
func (m *AnalysisManager) Symbols() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	symbols := make([]string, 0, len(m.analyzers))
	for s := range m.analyzers {
		symbols = append(symbols, s)
	}
	return symbols
}
