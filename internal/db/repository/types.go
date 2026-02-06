package repository

import (
	"time"
)

// Position represents a stock position in the database
type Position struct {
	ID     string `db:"id"`
	UserID int64  `db:"user_id"`
	Symbol string `db:"symbol"`

	// Entry details
	EntryDate  time.Time `db:"entry_date"`
	EntryPrice float64   `db:"entry_price"`
	Quantity   int       `db:"quantity"`

	// Risk management
	StopLoss float64  `db:"stop_loss"`
	Target1  *float64 `db:"target_1"`
	Target2  *float64 `db:"target_2"`
	Target3  *float64 `db:"target_3"`

	// Signal metadata
	SignalType *string `db:"signal_type"`
	Score      *int    `db:"score"`
	Notes      *string `db:"notes"`

	// Exit details
	IsClosed   bool       `db:"is_closed"`
	ExitDate   *time.Time `db:"exit_date"`
	ExitPrice  *float64   `db:"exit_price"`
	ExitReason *string    `db:"exit_reason"`

	// Performance metrics
	PnL        *float64 `db:"pnl"`
	PnLPercent *float64 `db:"pnl_percent"`
	RMultiple  *float64 `db:"r_multiple"`

	// Timestamps
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	// Aggregated position tracking
	TotalEntries   int       `db:"total_entries"`
	TotalFeesPaid  float64   `db:"total_fees_paid"`
	FirstEntryDate time.Time `db:"first_entry_date"`
	LastEntryDate  time.Time `db:"last_entry_date"`

	// T+2 Settlement tracking
	SettlementStatus *string    `db:"settlement_status"` // LOCKED_T0, LOCKED_T1, LOCKED_T2, LIQUID
	PurchaseDate     *time.Time `db:"purchase_date"`
	SettlementDate   *time.Time `db:"settlement_date"`
	CanSellDate      *time.Time `db:"can_sell_date"`
	LockedCapital    *float64   `db:"locked_capital"`
	LiquidCapital    *float64   `db:"liquid_capital"`
	Exchange         *string    `db:"exchange"` // HOSE, HNX, UPCOM
}

// IsLocked returns true if the position's shares are in settlement and cannot be sold.
func (p *Position) IsLocked() bool {
	if p.SettlementStatus == nil {
		return false
	}
	status := *p.SettlementStatus
	return status == "LOCKED_T0" || status == "LOCKED_T1" || status == "LOCKED_T2"
}

// IsLiquid returns true if the position's shares have settled and can be sold.
func (p *Position) IsLiquid() bool {
	if p.SettlementStatus == nil {
		return true // Default to liquid for backward compatibility
	}
	return *p.SettlementStatus == "LIQUID"
}

// GetLockedRisk calculates the worst-case floor-hit risk for locked shares.
// Returns 0 if position is liquid or exchange is unknown.
func (p *Position) GetLockedRisk() float64 {
	if p.IsLiquid() || p.Exchange == nil || p.LockedCapital == nil {
		return 0
	}

	exchange := *p.Exchange
	lockedCapital := *p.LockedCapital

	// Get exchange-specific risk multiplier
	var multiplier float64
	switch exchange {
	case "HOSE":
		multiplier = 0.20
	case "HNX":
		multiplier = 0.30
	case "UPCOM":
		multiplier = 0.40
	default:
		multiplier = 0.20 // Default to HOSE
	}

	return lockedCapital * multiplier
}

// PositionEntry represents a single purchase transaction
type PositionEntry struct {
	EntryID         string    `db:"entry_id"`
	UserID          int64     `db:"user_id"`
	Ticker          string    `db:"ticker"`
	EntryDate       time.Time `db:"entry_date"`
	EntryPrice      float64   `db:"entry_price"`
	SharesPurchased int       `db:"shares_purchased"`
	EntryFeePaid    float64   `db:"entry_fee_paid"`
	TransactionType string    `db:"transaction_type"` // BUY_NEW, BUY_MORE
	CreatedAt       time.Time `db:"created_at"`
}

// WatchlistItem represents a stock being monitored
type WatchlistItem struct {
	ID          string     `db:"id"`
	UserID      int64      `db:"user_id"`
	Symbol      string     `db:"symbol"`
	TargetPrice *float64   `db:"target_price"`
	Notes       *string    `db:"notes"`
	SignalTypes []string   // Will be stored as JSONB
	MinScore    int        `db:"min_score"`
	IsActive    bool       `db:"is_active"`
	AlertSent   bool       `db:"alert_sent"`
	LastAlertAt *time.Time `db:"last_alert_at"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

// SignalHistory represents a detected trading signal
type SignalHistory struct {
	ID           string    `db:"id"`
	Symbol       string    `db:"symbol"`
	SignalType   string    `db:"signal_type"`
	Score        int       `db:"score"`
	EntryPrice   float64   `db:"entry_price"`
	StopLoss     float64   `db:"stop_loss"`
	Targets      []float64 // Will be stored as JSONB
	PositionSize *int      `db:"position_size"`
	RiskAmount   *float64  `db:"risk_amount"`
	DetectedAt   time.Time `db:"detected_at"`
	Regime       *string   `db:"regime"`
	SentToUser   bool      `db:"sent_to_user"`
	UserAction   *string   `db:"user_action"`
	UserID       *int64    `db:"user_id"`
	CreatedAt    time.Time `db:"created_at"`
}

// UserConfig represents user trading preferences
type UserConfig struct {
	UserID              int64     `db:"user_id"`
	TelegramChatID      int64     `db:"telegram_chat_id"`
	InitialCapital      float64   `db:"initial_capital"`
	MaxPositions        int       `db:"max_positions"`
	RiskPerTrade        float64   `db:"risk_per_trade"`
	NotificationEnabled bool      `db:"notification_enabled"`
	DailyReportEnabled  bool      `db:"daily_report_enabled"`
	DailyReportTime     string    `db:"daily_report_time"`
	Timezone            string    `db:"timezone"`
	LockedRiskThreshold *float64  `db:"locked_risk_threshold"` // Max locked capital risk as % of account (default 0.10)
	CreatedAt           time.Time `db:"created_at"`
	UpdatedAt           time.Time `db:"updated_at"`
}

// GetLockedRiskThreshold returns the locked risk threshold, defaulting to 10% if not set.
func (u *UserConfig) GetLockedRiskThreshold() float64 {
	if u.LockedRiskThreshold == nil {
		return 0.10 // Default to 10%
	}
	return *u.LockedRiskThreshold
}

// StockSignalPreference represents per-stock minimum signal score preferences
type StockSignalPreference struct {
	ID             int       `db:"id"`
	UserID         int64     `db:"user_id"`
	Symbol         string    `db:"symbol"`
	MinSignalScore int       `db:"min_signal_score"`
	Notes          *string   `db:"notes"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// PositionSettlementTracking represents daily settlement status snapshots
type PositionSettlementTracking struct {
	TrackingID         string    `db:"tracking_id"`
	PositionID         string    `db:"position_id"`
	CheckDate          time.Time `db:"check_date"`
	SettlementStatus   string    `db:"settlement_status"` // LOCKED_T0, LOCKED_T1, LOCKED_T2, LIQUID
	DaysUntilLiquid    int       `db:"days_until_liquid"`
	LockedValue        float64   `db:"locked_value"`
	LockedRisk         float64   `db:"locked_risk"`
	RiskClassification string    `db:"risk_classification"` // HIGH_RISK_LOCKED, MODERATE_RISK_NEAR_LIQUID, LOW_RISK_LIQUID
	CreatedAt          time.Time `db:"created_at"`
}

// TheoreticalStopBreach represents stop losses triggered but not executable
type TheoreticalStopBreach struct {
	BreachID           string    `db:"breach_id"`
	PositionID         string    `db:"position_id"`
	BreachDate         time.Time `db:"breach_date"`
	StopPrice          float64   `db:"stop_price"`
	ActualPrice        float64   `db:"actual_price"`
	SettlementStatus   string    `db:"settlement_status"`
	DaysUntilExecutable int      `db:"days_until_executable"`
	CreatedAt          time.Time `db:"created_at"`
}
