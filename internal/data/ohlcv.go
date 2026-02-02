// Package data provides OHLCV data structures for stock price analysis.
// OPEN - HIGH - LOW - CLOSE - VOLUME

package data

import (
	"time"
)

// OHLCV represents a single candlestick bar with Open, High, Low, Close, Volume, and Timestamp.
type OHLCV struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// NewOHLCV creates a new OHLCV bar.
func NewOHLCV(timestamp time.Time, open, high, low, close, volume float64) OHLCV {
	return OHLCV{
		Timestamp: timestamp,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
	}
}

// TypicalPrice returns the typical price (High + Low + Close) / 3.
// Used in VWAP and other volume-weighted calculations.
func (o OHLCV) TypicalPrice() float64 {
	return (o.High + o.Low + o.Close) / 3
}

// Range returns the bar's price range (High - Low).
func (o OHLCV) Range() float64 {
	return o.High - o.Low
}

// IsBullish returns true if close > open (up day).
func (o OHLCV) IsBullish() bool {
	return o.Close > o.Open
}

// IsBearish returns true if close < open (down day).
func (o OHLCV) IsBearish() bool {
	return o.Close < o.Open
}

// Body returns the absolute body size |Close - Open|.
func (o OHLCV) Body() float64 {
	body := o.Close - o.Open
	if body < 0 {
		return -body
	}
	return body
}

// UpperWick returns the upper wick size.
func (o OHLCV) UpperWick() float64 {
	if o.IsBullish() {
		return o.High - o.Close
	}
	return o.High - o.Open
}

// LowerWick returns the lower wick size.
func (o OHLCV) LowerWick() float64 {
	if o.IsBullish() {
		return o.Open - o.Low
	}
	return o.Close - o.Low
}
