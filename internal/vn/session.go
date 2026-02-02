package vn

import (
	"errors"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

var vnLocation *time.Location

func init() {
	var err error
	// Explicitly load Vietnam timezone
	vnLocation, err = time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		// Fallback to FixedZone if LoadLocation fails (e.g. missing tzdata)
		vnLocation = time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)
		logger.Warn().Err(err).Msg("Failed to load Asia/Ho_Chi_Minh, using FixedZone UTC+7")
	}
}

// Trading session errors
var (
	ErrMarketClosed   = errors.New("market closed")
	ErrLunchBreak     = errors.New("market closed for lunch break")
	ErrInvalidSession = errors.New("invalid session for order type")
)

// SessionType represents the type of trading session
type SessionType string

const (
	SessionTypeAuction    SessionType = "auction"
	SessionTypeContinuous SessionType = "continuous"
)

// TradingSession represents a trading period during the day
type TradingSession struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Type      SessionType
}

// GetCurrentSession detects the current trading session based on Vietnam market hours (HOSE).
// Session timeline:
// - 09:00-09:15: ATO (At The Opening) Auction
// - 09:15-11:30: Continuous Trading (Morning)
// - 11:30-13:00: BREAK (No trading)
// - 13:00-14:30: Continuous Trading (Afternoon)
// - 14:30-14:45: ATC (At The Closing) Auction
func GetCurrentSession() (*TradingSession, error) {
	return GetSessionForTime(time.Now())
}

// GetSessionForTime returns the trading session for a specific time
func GetSessionForTime(t time.Time) (*TradingSession, error) {
	// Convert input time to Vietnam time
	vnTime := t.In(vnLocation)

	// Check if it's a trading day
	if !IsTradingDay(vnTime) {
		return nil, ErrMarketClosed
	}

	hour := vnTime.Hour()
	minute := vnTime.Minute()

	// Convert to minutes since midnight for easier comparison
	minutesSinceMidnight := hour*60 + minute

	// ATO Auction: 09:00-09:15
	if minutesSinceMidnight >= 9*60 && minutesSinceMidnight < 9*60+15 {
		return &TradingSession{
			Name: "ATO Auction",
			Type: SessionTypeAuction,
		}, nil
	}

	// Morning Continuous: 09:15-11:30
	if minutesSinceMidnight >= 9*60+15 && minutesSinceMidnight < 11*60+30 {
		return &TradingSession{
			Name: "Morning Continuous",
			Type: SessionTypeContinuous,
		}, nil
	}

	// Lunch Break: 11:30-13:00
	if minutesSinceMidnight >= 11*60+30 && minutesSinceMidnight < 13*60 {
		return nil, ErrLunchBreak
	}

	// Afternoon Continuous: 13:00-14:30
	if minutesSinceMidnight >= 13*60 && minutesSinceMidnight < 14*60+30 {
		return &TradingSession{
			Name: "Afternoon Continuous",
			Type: SessionTypeContinuous,
		}, nil
	}

	// ATC Auction: 14:30-14:45
	if minutesSinceMidnight >= 14*60+30 && minutesSinceMidnight < 14*60+45 {
		return &TradingSession{
			Name: "ATC Auction",
			Type: SessionTypeAuction,
		}, nil
	}

	// After market hours
	return nil, ErrMarketClosed
}

// ValidateOrderTiming checks if an order type is allowed in the current session.
// Returns (valid, message).
//
// Order Type Restrictions:
// - LO (Limit): Allowed in all sessions
// - MP (Market): Not allowed in auctions, risky in continuous
// - ATO: Only in ATO auction
// - ATC: Only in ATC auction
func ValidateOrderTiming(orderType string, session *TradingSession) (bool, string) {
	if session == nil {
		return false, "market closed"
	}

	switch orderType {
	case "LO":
		// Limit orders allowed in all sessions
		return true, "limit order valid in all sessions"

	case "MP":
		// Market orders not allowed in auctions
		if session.Type == SessionTypeAuction {
			return false, "market orders not allowed in auctions"
		}
		return true, "WARNING: Market orders risky - use limit orders"

	case "ATO":
		// ATO orders only in ATO auction
		if session.Name == "ATO Auction" {
			return true, "valid"
		}
		return false, "ATO only valid during opening auction"

	case "ATC":
		// ATC orders only in ATC auction
		if session.Name == "ATC Auction" {
			return true, "valid"
		}
		return false, "ATC only valid during closing auction"

	default:
		return false, "unknown order type"
	}
}

// IsMarketOpen returns true if the market is currently open (any session)
func IsMarketOpen() bool {
	session, err := GetCurrentSession()
	return err == nil && session != nil
}

// IsAuctionSession returns true if the current session is an auction
func IsAuctionSession(session *TradingSession) bool {
	return session != nil && session.Type == SessionTypeAuction
}

// IsContinuousSession returns true if the current session is continuous trading
func IsContinuousSession(session *TradingSession) bool {
	return session != nil && session.Type == SessionTypeContinuous
}

// GetPreferredEntrySession returns true if the current session is preferred for entry orders
// Preferred: Morning and early afternoon continuous sessions (09:20-10:30, 13:15-14:00)
func GetPreferredEntrySession(t time.Time) bool {
	session, err := GetSessionForTime(t)
	if err != nil || session.Type != SessionTypeContinuous {
		return false
	}

	hour := t.Hour()
	minute := t.Minute()

	// Morning: 09:20-10:30
	if (hour == 9 && minute >= 20) || (hour == 10 && minute <= 30) {
		return true
	}

	// Afternoon: 13:15-14:00
	if (hour == 13 && minute >= 15) || (hour == 14 && minute == 0) {
		return true
	}

	return false
}
