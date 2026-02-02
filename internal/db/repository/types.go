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
	CreatedAt           time.Time `db:"created_at"`
	UpdatedAt           time.Time `db:"updated_at"`
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
