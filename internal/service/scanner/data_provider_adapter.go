package scanner

import (
	"fmt"

	"github.com/nonobeam/golang-stock-trading/internal/analysis/indicators"
	"github.com/nonobeam/golang-stock-trading/internal/signals"
)

// BarCacheDataProvider adapts BarCache to implement signals.DataProvider
type BarCacheDataProvider struct {
	cache *BarCache
}

// NewBarCacheDataProvider creates a new data provider from bar cache
func NewBarCacheDataProvider(cache *BarCache) *BarCacheDataProvider {
	return &BarCacheDataProvider{cache: cache}
}

// GetDailyData converts cached bars to MarketData format
func (p *BarCacheDataProvider) GetDailyData(symbol string) (*signals.MarketData, error) {
	bars := p.cache.Get(symbol)
	if len(bars) == 0 {
		return nil, fmt.Errorf("no data available for %s", symbol)
	}

	// Extract arrays from bars
	opens := make([]float64, len(bars))
	highs := make([]float64, len(bars))
	lows := make([]float64, len(bars))
	closes := make([]float64, len(bars))
	volumes := make([]float64, len(bars))

	for i, bar := range bars {
		opens[i] = bar.Open
		highs[i] = bar.High
		lows[i] = bar.Low
		closes[i] = bar.Close
		volumes[i] = float64(bar.Volume)
	}

	// Latest bar
	latest := bars[len(bars)-1]

	marketData := &signals.MarketData{
		Symbol:        symbol,
		Timestamp:     latest.Timestamp,
		DailySeries:   nil, // Not using Series struct
		CurrentClose:  latest.Close,
		CurrentHigh:   latest.High,
		CurrentLow:    latest.Low,
		CurrentOpen:   latest.Open,
		CurrentVolume: float64(latest.Volume),
		Closes:        closes,
		Highs:         highs,
		Lows:          lows,
		Volumes:       volumes,
		Opens:         opens,
	}

	// Calculate moving averages if enough data
	if len(closes) >= 20 {
		if sma20, err := indicators.CalculateSMA(closes, 20); err == nil {
			marketData.SMA20 = sma20
		}
		if ema20, err := indicators.CalculateEMA(closes, 20); err == nil {
			marketData.EMA20 = ema20
		}
	}

	if len(closes) >= 50 {
		if sma50, err := indicators.CalculateSMA(closes, 50); err == nil {
			marketData.SMA50 = sma50
		}
		if ema50, err := indicators.CalculateEMA(closes, 50); err == nil {
			marketData.EMA50 = ema50
		}
	}

	// Calculate RSI
	if len(closes) >= 14 {
		if rsi, err := indicators.CalculateRSI(closes, 14); err == nil {
			marketData.RSI = rsi
		}
	}

	// Calculate MACD
	if len(closes) >= 26 {
		if result, err := indicators.CalculateMACD(closes, 12, 26, 9); err == nil {
			marketData.MACD = result.MACD
			marketData.MACDSignal = result.Signal
			marketData.MACDHistogram = result.Histogram
		}
	}

	// Calculate ATR
	if len(closes) >= 14 {
		if atr, err := indicators.CalculateATR(highs, lows, closes, 14); err == nil {
			marketData.ATR = atr
		}
	}

	// Calculate Stochastic
	if len(closes) >= 14 {
		if result, err := indicators.CalculateStochastic(highs, lows, closes, 14, 3, 3); err == nil {
			marketData.StochK = result.K
			marketData.StochD = result.D
		}
	}

	// Calculate ADX if enough data
	if len(closes) >= 14 {
		if result, err := indicators.CalculateADX(highs, lows, closes, 14); err == nil {
			marketData.ADX = result.ADX
			marketData.PlusDI = result.PlusDI
			marketData.MinusDI = result.MinusDI
		}
	}

	return marketData, nil
}

// GetVolumeStats calculates volume statistics from cached bars
func (p *BarCacheDataProvider) GetVolumeStats(symbol string) (volumeMA20, volumePercentile float64, error error) {
	bars := p.cache.Get(symbol)
	if len(bars) < 20 {
		return 0, 0, fmt.Errorf("insufficient data for volume stats")
	}

	// Calculate 20-period volume MA
	volumes := make([]float64, len(bars))
	for i, bar := range bars {
		volumes[i] = float64(bar.Volume)
	}

	if vm, err := indicators.CalculateSMA(volumes, 20); err == nil {
		volumeMA20 = vm
	}

	// Calculate percentile (simplified: position in sorted list)
	currentVol := float64(bars[len(bars)-1].Volume)
	lowerCount := 0
	for _, vol := range volumes {
		if vol < currentVol {
			lowerCount++
		}
	}
	volumePercentile = float64(lowerCount) / float64(len(volumes)) * 100

	return volumeMA20, volumePercentile, nil
}

// GetSupportResistance returns support/resistance levels
// For real-time scanner, we'll use a simplified approach
func (p *BarCacheDataProvider) GetSupportResistance(symbol string) (*signals.SRLevels, error) {
	bars := p.cache.Get(symbol)
	if len(bars) < 20 {
		return nil, fmt.Errorf("insufficient data for S/R calculation")
	}

	// Simple S/R: recent swing highs/lows
	var recentHighs, recentLows []float64

	// Find local highs and lows in last 20 bars
	for i := 1; i < len(bars)-1 && i < 20; i++ {
		idx := len(bars) - 1 - i
		if bars[idx].High > bars[idx-1].High && bars[idx].High > bars[idx+1].High {
			recentHighs = append(recentHighs, bars[idx].High)
		}
		if bars[idx].Low < bars[idx-1].Low && bars[idx].Low < bars[idx+1].Low {
			recentLows = append(recentLows, bars[idx].Low)
		}
	}

	// Return top 3 of each
	var resistanceLevels, supportLevels []float64
	if len(recentHighs) > 0 {
		count := 3
		if len(recentHighs) < count {
			count = len(recentHighs)
		}
		resistanceLevels = recentHighs[:count]
	}

	if len(recentLows) > 0 {
		count := 3
		if len(recentLows) < count {
			count = len(recentLows)
		}
		supportLevels = recentLows[:count]
	}

	// Use first resistance/support if available
	var primaryResistance, primarySupport float64
	if len(resistanceLevels) > 0 {
		primaryResistance = resistanceLevels[0]
	}
	if len(supportLevels) > 0 {
		primarySupport = supportLevels[0]
	}

	return &signals.SRLevels{
		Resistance: primaryResistance,
		Support:    primarySupport,
	}, nil
}

// GetMarketRegime returns market regime information
// For real-time scanner, return simplified regime
func (p *BarCacheDataProvider) GetMarketRegime(symbol string) (map[string]interface{}, error) {
	bars := p.cache.Get(symbol)
	if len(bars) < 50 {
		return map[string]interface{}{
			"regime": "unknown",
		}, nil
	}

	// Simple regime detection based on trend
	closes := make([]float64, len(bars))
	for i, bar := range bars {
		closes[i] = bar.Close
	}

	sma20, _ := indicators.CalculateSMA(closes, 20)
	sma50, _ := indicators.CalculateSMA(closes, 50)
	currentClose := bars[len(bars)-1].Close

	regime := "range"
	if currentClose > sma20 && sma20 > sma50 {
		regime = "bull"
	} else if currentClose < sma20 && sma20 < sma50 {
		regime = "bear"
	}

	return map[string]interface{}{
		"regime":       regime,
		"sma20":        sma20,
		"sma50":        sma50,
		"currentClose": currentClose,
	}, nil
}
