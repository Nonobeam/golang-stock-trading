package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

// Config holds database configuration
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	Schema          string
	SSLMode         string
	MaxConnections  int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Database wraps sql.DB with additional methods
type Database struct {
	*sql.DB
	config *Config
}

// NewDatabase creates a new database connection
func NewDatabase(cfg *Config) (*Database, error) {
	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	// Add schema to search_path if specified
	if cfg.Schema != "" {
		connStr += fmt.Sprintf(" search_path=%s", cfg.Schema)
	}

	// Open connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(cfg.MaxConnections)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.DBName).
		Msg("Database connected successfully")

	return &Database{
		DB:     db,
		config: cfg,
	}, nil
}

// Close closes the database connection
func (db *Database) Close() error {
	logger.Info().Msg("Closing database connection")
	return db.DB.Close()
}

// HealthCheck verifies database connection is alive
func (db *Database) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.PingContext(ctx)
}
