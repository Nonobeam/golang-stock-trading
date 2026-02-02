package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// UserConfigRepository handles user configuration CRUD operations
type UserConfigRepository struct {
	db *sql.DB
}

// NewUserConfigRepository creates a new user config repository
func NewUserConfigRepository(db *sql.DB) *UserConfigRepository {
	return &UserConfigRepository{db: db}
}

// Create creates a new user configuration
func (r *UserConfigRepository) Create(ctx context.Context, config *UserConfig) error {
	query := `
		INSERT INTO user_config (
			user_id, telegram_chat_id, initial_capital,
			max_positions, risk_per_trade,
			notification_enabled, daily_report_enabled, daily_report_time, timezone
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id) DO NOTHING
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		config.UserID, config.TelegramChatID, config.InitialCapital,
		config.MaxPositions, config.RiskPerTrade,
		config.NotificationEnabled, config.DailyReportEnabled, config.DailyReportTime, config.Timezone,
	).Scan(&config.CreatedAt, &config.UpdatedAt)

	if err == sql.ErrNoRows {
		// User already exists, that's okay
		return nil
	}

	return err
}

// GetByUserID returns user config by user ID
func (r *UserConfigRepository) GetByUserID(ctx context.Context, userID int64) (*UserConfig, error) {
	query := `
		SELECT user_id, telegram_chat_id, initial_capital,
			max_positions, risk_per_trade,
			notification_enabled, daily_report_enabled, daily_report_time, timezone,
			created_at, updated_at
		FROM user_config
		WHERE user_id = $1
	`

	config := &UserConfig{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&config.UserID, &config.TelegramChatID, &config.InitialCapital,
		&config.MaxPositions, &config.RiskPerTrade,
		&config.NotificationEnabled, &config.DailyReportEnabled, &config.DailyReportTime, &config.Timezone,
		&config.CreatedAt, &config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return config, nil
}

// GetByChatID returns user config by Telegram chat ID
func (r *UserConfigRepository) GetByChatID(ctx context.Context, chatID int64) (*UserConfig, error) {
	query := `
		SELECT user_id, telegram_chat_id, initial_capital,
			max_positions, risk_per_trade,
			notification_enabled, daily_report_enabled, daily_report_time, timezone,
			created_at, updated_at
		FROM user_config
		WHERE telegram_chat_id = $1
	`

	config := &UserConfig{}
	err := r.db.QueryRowContext(ctx, query, chatID).Scan(
		&config.UserID, &config.TelegramChatID, &config.InitialCapital,
		&config.MaxPositions, &config.RiskPerTrade,
		&config.NotificationEnabled, &config.DailyReportEnabled, &config.DailyReportTime, &config.Timezone,
		&config.CreatedAt, &config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return config, nil
}



// Update updates user configuration
func (r *UserConfigRepository) Update(ctx context.Context, config *UserConfig) error {
	query := `
		UPDATE user_config
		SET max_positions = $2,
			risk_per_trade = $3,
			notification_enabled = $4,
			daily_report_enabled = $5,
			daily_report_time = $6,
			timezone = $7,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		config.UserID, config.MaxPositions, config.RiskPerTrade,
		config.NotificationEnabled, config.DailyReportEnabled, config.DailyReportTime, config.Timezone,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("user config not found: %d", config.UserID)
	}

	return nil
}

// GetAllWithDailyReports returns all users who have daily reports enabled at a specific time
func (r *UserConfigRepository) GetAllWithDailyReports(ctx context.Context, reportTime string) ([]*UserConfig, error) {
	query := `
		SELECT user_id, telegram_chat_id, initial_capital,
			max_positions, risk_per_trade,
			notification_enabled, daily_report_enabled, daily_report_time, timezone,
			created_at, updated_at
		FROM user_config
		WHERE daily_report_enabled = TRUE 
			AND daily_report_time = $1
		ORDER BY user_id
	`

	rows, err := r.db.QueryContext(ctx, query, reportTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*UserConfig
	for rows.Next() {
		config := &UserConfig{}
		err := rows.Scan(
			&config.UserID, &config.TelegramChatID, &config.InitialCapital,
			&config.MaxPositions, &config.RiskPerTrade,
			&config.NotificationEnabled, &config.DailyReportEnabled, &config.DailyReportTime, &config.Timezone,
			&config.CreatedAt, &config.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	return configs, rows.Err()
}

// FindUserByChatID finds an existing user by their Telegram chat ID
func (r *UserConfigRepository) FindUserByChatID(ctx context.Context, chatID int64) (*UserConfig, error) {
	return r.GetByChatID(ctx, chatID)
}

// CreateUserForChatID creates a new user with the given chat ID and default configuration
func (r *UserConfigRepository) CreateUserForChatID(ctx context.Context, chatID int64) (*UserConfig, error) {
	query := `
		INSERT INTO user_config (
			telegram_chat_id, initial_capital,
			max_positions, risk_per_trade,
			notification_enabled, daily_report_enabled, daily_report_time, timezone
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING user_id, telegram_chat_id, initial_capital,
			max_positions, risk_per_trade,
			notification_enabled, daily_report_enabled, daily_report_time, timezone,
			created_at, updated_at
	`

	config := &UserConfig{}
	err := r.db.QueryRowContext(ctx, query,
		chatID,
		10000000.0,
		3,
		0.01,
		true,
		false,
		"09:00",
		"Asia/Ho_Chi_Minh",
	).Scan(
		&config.UserID, &config.TelegramChatID, &config.InitialCapital,
		&config.MaxPositions, &config.RiskPerTrade,
		&config.NotificationEnabled, &config.DailyReportEnabled, &config.DailyReportTime, &config.Timezone,
		&config.CreatedAt, &config.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user for chat ID %d: %w", chatID, err)
	}

	return config, nil
}

// GetOrCreateUserByChatID returns existing user or creates a new one atomically
func (r *UserConfigRepository) GetOrCreateUserByChatID(ctx context.Context, chatID int64) (*UserConfig, error) {
	user, err := r.FindUserByChatID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by chat ID: %w", err)
	}

	if user != nil {
		return user, nil
	}

	return r.CreateUserForChatID(ctx, chatID)
}

// GetAllActiveUsers returns all users with notifications enabled
func (r *UserConfigRepository) GetAllActiveUsers(ctx context.Context) ([]*UserConfig, error) {
	query := `
		SELECT user_id, telegram_chat_id, initial_capital,
			max_positions, risk_per_trade,
			notification_enabled, daily_report_enabled, daily_report_time, timezone,
			created_at, updated_at
		FROM user_config
		WHERE notification_enabled = TRUE
		ORDER BY user_id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*UserConfig
	for rows.Next() {
		config := &UserConfig{}
		err := rows.Scan(
			&config.UserID, &config.TelegramChatID, &config.InitialCapital,
			&config.MaxPositions, &config.RiskPerTrade,
			&config.NotificationEnabled, &config.DailyReportEnabled, &config.DailyReportTime, &config.Timezone,
			&config.CreatedAt, &config.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	return configs, rows.Err()
}
