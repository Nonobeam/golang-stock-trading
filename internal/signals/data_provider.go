package signals

import (
	"github.com/nonobeam/golang-stock-trading/internal/analysis"
	"github.com/nonobeam/golang-stock-trading/internal/data"
)

// AnalysisDataProvider implements DataProvider using AnalysisManager.
type AnalysisDataProvider struct {
	analysisManager *analysis.AnalysisManager
}

// NewAnalysisDataProvider creates a data provider from AnalysisManager.
func NewAnalysisDataProvider(manager *analysis.AnalysisManager) *AnalysisDataProvider {
	return &AnalysisDataProvider{
		analysisManager: manager,
	}
}

// GetDailyData retrieves market data for a symbol.
func (p *AnalysisDataProvider) GetDailyData(symbol string) (*MarketData, error) {
	// Get analyzer for symbol
	analyzer := p.analysisManager.GetOrCreate(symbol)

	if analyzer.BarCount() < 50 {
		return nil, data.ErrInsufficientData
	}

	// Get analysis result (contains all indicators)
	analysisResult, err := p.analysisManager.GetAnalysis(symbol)
	if err != nil {
		return nil, err
	}

	// Get series for historical data
	series := analyzer.GetSeries() // Note: Need to add GetSeries() method to StockAnalyzer

	// Get current bar
	currentBar, err := series.Last()
	if err != nil {
		return nil, err
	}

	// Extract OHLCV arrays
	closes := series.Closes()
	opens := extractOpens(series) // Helper to get opens
	highs := series.Highs()
	lows := series.Lows()
	volumes := series.Volumes()

	// Build MarketData
	marketData := &MarketData{
		Symbol:    symbol,
		Timestamp: currentBar.Timestamp,

		// Daily series
		DailySeries: series,

		// Current OHLCV
		CurrentClose:  currentBar.Close,
		CurrentHigh:   currentBar.High,
		CurrentLow:    currentBar.Low,
		CurrentOpen:   currentBar.Open,
		CurrentVolume: currentBar.Volume,

		// Daily indicators
		EMA20:         analysisResult.EMA20,
		EMA50:         analysisResult.EMA20, // Note: Need EMA50 in AnalysisResult
		SMA20:         analysisResult.SMA20,
		SMA50:         analysisResult.SMA50,
		RSI:           analysisResult.RSI,
		MACD:          analysisResult.MACD,
		MACDSignal:    analysisResult.MACDSignal,
		MACDHistogram: analysisResult.MACDHistogram,
		ADX:           analysisResult.ADX,
		PlusDI:        analysisResult.PlusDI,
		MinusDI:       analysisResult.MinusDI,
		ATR:           analysisResult.ATR,
		StochK:        analysisResult.StochK,
		StochD:        analysisResult.StochD,

		// Weekly indicators
		WeeklyClose:  analysisResult.WeeklyClose,
		WeeklySMA200: analysisResult.WeeklySMA200,
		WeeklyRSI:    analysisResult.WeeklyRSI,
		WeeklyStochK: analysisResult.WeeklyStochK,
		WeeklyStochD: analysisResult.WeeklyStochD,

		// Historical arrays
		Highs:   highs,
		Lows:    lows,
		Closes:  closes,
		Volumes: volumes,
		Opens:   opens,
	}

	return marketData, nil
}

// GetVolumeStats calculates volume statistics for a symbol.
func (p *AnalysisDataProvider) GetVolumeStats(symbol string) (volumeMA20, volumePercentile float64, err error) {
	analyzer := p.analysisManager.GetOrCreate(symbol)

	if analyzer.BarCount() < 20 {
		return 0, 0, data.ErrInsufficientData
	}

	series := analyzer.GetSeries()
	volumes := series.Volumes()

	if len(volumes) < 20 {
		return 0, 0, data.ErrInsufficientData
	}

	// Calculate 20-day volume MA
	recent20 := volumes[len(volumes)-20:]
	volumeMA20 = average(recent20)

	// Calculate volume percentile (current vs last 60 days)
	lookback := 60
	if len(volumes) < lookback {
		lookback = len(volumes)
	}

	historical := volumes[len(volumes)-lookback : len(volumes)-1] // Exclude current
	currentVolume := volumes[len(volumes)-1]

	volumePercentile = CalculateVolumePercentile(currentVolume, historical)

	return volumeMA20, volumePercentile, nil
}

// extractOpens extracts open prices from a series.
// Helper function since Series doesn't have Opens() method yet.
func extractOpens(series *data.Series) []float64 {
	allBars := series.All()
	opens := make([]float64, len(allBars))

	for i, bar := range allBars {
		opens[i] = bar.Open
	}

	return opens
}

// GetSupportResistance identifies support and resistance levels for a symbol.
func (p *AnalysisDataProvider) GetSupportResistance(symbol string) (*SRLevels, error) {
	analyzer := p.analysisManager.GetOrCreate(symbol)

	if analyzer.BarCount() < 60 {
		return nil, data.ErrInsufficientData
	}

	series := analyzer.GetSeries()

	// Use existing support/resistance detection (60-day lookback)
	supports := FindSupportLevels(series, 60)
	resistances := FindResistanceLevels(series, 60)

	if len(supports) == 0 || len(resistances) == 0 {
		return nil, data.ErrInsufficientData
	}

	// Return highest-confidence levels (first in list)
	return &SRLevels{
		Support:    supports[0].Price,
		Resistance: resistances[0].Price,
	}, nil
}

// GetMarketRegime returns market regime information for validation.
// This is a simplified version for now - in production, this would call
// the regime detector from internal/regime package.
func (p *AnalysisDataProvider) GetMarketRegime(symbol string) (map[string]interface{}, error) {
	// For now, return a neutral regime map
	// TODO: In full implementation, this should call regime.DetectRegime()
	// to get actual VN-Index regime classification

	analyzer := p.analysisManager.GetOrCreate(symbol)

	if analyzer.BarCount() < 20 {
		return nil, data.ErrInsufficientData
	}

	// Get analysis result for ADX
	analysisResult, err := p.analysisManager.GetAnalysis(symbol)
	if err != nil {
		return nil, err
	}

	// Simple regime classification based on ADX
	var regime string
	if analysisResult.ADX >= 25 {
		regime = "mild_bull" // Trending market
	} else if analysisResult.ADX < 20 {
		regime = "range_bound" // Ranging market
	} else {
		regime = "mild_bull" // Default to mild bull
	}

	return map[string]interface{}{
		"vn_market_regime": regime,
	}, nil
}
